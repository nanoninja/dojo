# Paiements

## Architecture

Le système de paiement repose sur une interface `payment.Provider` qui découple la logique métier de tout prestataire de paiement spécifique. Stripe est le seul provider actif, mais l'interface permet d'ajouter PayPal ou n'importe quelle autre passerelle sans toucher à la couche service.

```
internal/payment/
├── provider.go          # Interface Provider + types partagés
└── stripe/
    └── client.go        # Implémentation Stripe
```

## Machine à états d'un achat

```
                  ┌─────────────────────────────────┐
                  │                                 │
                  ▼                                 │ déjà traité
            ┌─────────┐    webhook confirmé    ┌───────────┐
   achat ──►│ pending │───────────────────────►│ completed │──► refunded
            └─────────┘                        └───────────┘
                  │
                  │ paiement refusé / session expirée
                  ▼
            ┌────────┐
            │ failed │
            └────────┘
```

| Statut      | Signification                                                    |
|-------------|------------------------------------------------------------------|
| `pending`   | Session Stripe créée, en attente de confirmation de paiement     |
| `completed` | Paiement confirmé, enrollment(s) actif(s)                        |
| `failed`    | Paiement refusé ou session Stripe expirée                        |
| `refunded`  | Remboursement effectué, enrollment(s) annulé(s)                  |
| `disputed`  | Contestation (chargeback) ouverte par l'utilisateur              |

## Flux complet : de l'achat à l'enrollment

### Vue d'ensemble

```
  Client          API Dojo              Stripe           Webhook
    │                │                    │                  │
    │ POST /purchases│                    │                  │
    │────────────────►                    │                  │
    │                │ INSERT purchase    │                  │
    │                │ (status=pending)   │                  │
    │                │                    │                  │
    │                │ CreateCheckout ──► │                  │
    │                │ ◄──────────────────│                  │
    │                │  {id, url}         │                  │
    │                │                    │                  │
    │  {checkout_url}│                    │                  │
    │◄───────────────│                    │                  │
    │                │                    │                  │
    │ Redirige vers ►│                    │                  │
    │ Stripe         │                    │                  │
    │───────────────────────────────────► │                  │
    │                │                    │  Paiement OK     │
    │                │                    │─────────────────►│
    │                │                    │                  │ POST /webhooks/stripe
    │                │◄──────────────────────────────────────│
    │                │ ConfirmPayment     │                  │
    │                │ (completed +       │                  │
    │                │  enrollment créé)  │                  │
    │                │                    │                  │
    │ Redirige vers  │                    │                  │
    │ STRIPE_SUCCESS │                    │                  │
    │◄────────────────────────────────────│                  │
```

### Étape 1 — Le client initie l'achat

```
POST /api/v1/purchases/courses
Authorization: Bearer <token>

{
  "course_id":    "01966b0a-...",
  "amount_cents": 1999,
  "currency":     "EUR"
}
```

### Étape 2 — `BuyCourse` crée une purchase en attente

```
┌─────────────────────────────────────────────────────┐
│ WithTx                                              │
│  INSERT purchases (status='pending', provider=      │
│  'stripe')                                          │
└─────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────┐
│ Hors transaction                                    │
│  provider.CreateCheckout({purchase_id: p.ID, ...})  │
│                                                     │
│  En cas d'erreur ──► CancelPending(p.ID)            │
│  En cas de succès ──► UPDATE provider_session_id    │
└─────────────────────────────────────────────────────┘
```

Réponse retournée au client :

```json
{
  "id":           "01966b0a-...",
  "status":       "pending",
  "checkout_url": "https://checkout.stripe.com/pay/cs_test_...",
  ...
}
```

### Étape 3 — L'utilisateur paie sur Stripe

Le frontend redirige l'utilisateur vers `checkout_url`. Stripe héberge la page de paiement. En cas de succès, Stripe redirige vers `STRIPE_SUCCESS_URL` ; en cas d'annulation, vers `STRIPE_CANCEL_URL`.

### Étape 4 — Stripe envoie un webhook

```
POST /webhooks/stripe
Stripe-Signature: t=...,v1=...
<payload brut>
```

Cette route est **publique** (pas d'auth JWT). La sécurité repose sur la validation de la signature HMAC via `STRIPE_WEBHOOK_SECRET`.

### Étape 5 — `ConfirmPayment` finalise l'achat

```
┌─────────────────────────────────────────────────────────┐
│ Hors transaction (lecture)                              │
│  FindByID(purchaseID)                                   │
│  Vérifier status == 'pending'  ──► ErrAlreadyProcessed  │
│  (idempotence webhook)                                  │
└─────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────┐
│ WithTx (écriture atomique)                              │
│  UPDATE purchases SET status='completed',               │
│         provider_payment_id=...                         │
│  INSERT course_enrollments (status='active',            │
│         source='purchase')                              │
└─────────────────────────────────────────────────────────┘
```

> **Important** : l'enrollment n'est créé qu'ici, après confirmation réelle du paiement. `BuyCourse` ne crée intentionnellement aucun enrollment — l'utilisateur ne doit pas avoir accès au cours avant d'avoir payé.

### Étape 6 — L'utilisateur accède au contenu

```
GET /api/v1/chapters/{id}  ou  GET /api/v1/lessons/{id}
         │
         ▼
  AccessService.CanAccess(ctx, userID, courseID)
         │
         ├── subscription active ? ──► accès autorisé
         │
         └── enrollment actif ?    ──► accès autorisé
                                   sinon ──► 403 Forbidden
```

## Flux de remboursement

```
POST /api/v1/purchases/{id}/refund
         │
         ▼
┌─────────────────────────────────────────────────────────┐
│ Hors transaction (lecture)                              │
│  FindByID(purchaseID)                                   │
│  Si ProviderPaymentID != "" ──► provider.Refund(...)    │
│  Échec Stripe ──► erreur retournée, DB non modifiée     │
└─────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────┐
│ WithTx (écriture atomique)                              │
│  UPDATE purchases SET status='refunded',                │
│         refunded_at=now                                 │
│  UPDATE course_enrollments SET status='cancelled'       │
└─────────────────────────────────────────────────────────┘
```

> **Idempotence** : si Stripe confirme le remboursement via webhook (`charge.refunded` → `EventRefundSucceeded`), le handler appelle également `Refund`. La logique est identique — la double exécution est sans effet si le statut est déjà `refunded`.

## Le webhook Stripe en détail

### Pourquoi un webhook ?

Le paiement Stripe se déroule entièrement sur les serveurs de Stripe, hors de portée de notre API. Notre serveur ne peut donc pas savoir si le paiement a réussi en temps réel — il doit attendre que Stripe le lui notifie. C'est le rôle du webhook : Stripe rappelle notre API dès qu'un événement important se produit (paiement confirmé, refusé, remboursé, etc.).

```
  Sans webhook                        Avec webhook
  ──────────────────────────────      ──────────────────────────────
  Client ──► API ──► Stripe           Client ──► API ──► Stripe
                       │                                    │
  API ne sait pas      │              Stripe ──────────────►│ API
  si le paiement       │              rappelle l'API dès    │
  a abouti             │              que le paiement est   │
                       │              confirmé              │
```

Sans webhook, nous aurions besoin de sonder Stripe toutes les N secondes pour savoir si le paiement est passé (polling). C'est fragile, coûteux en appels API, et non fiable.

### Ce que reçoit le webhook

Stripe envoie une requête `POST` sur `/webhooks/stripe` avec :
- un **corps JSON** décrivant l'événement (type, données, métadonnées)
- un header `Stripe-Signature` contenant la signature HMAC de la requête

```
POST /webhooks/stripe
Stripe-Signature: t=1749123456,v1=abc123...

{
  "type": "checkout.session.completed",
  "data": {
    "object": {
      "id":              "cs_test_...",
      "payment_intent":  "pi_test_...",
      "metadata": {
        "purchase_id": "01966b0a-..."
      }
    }
  }
}
```

### Événements traités

| Événement Stripe                    | EventType normalisé       | Action déclenchée                         |
|-------------------------------------|---------------------------|-------------------------------------------|
| `checkout.session.completed`        | `payment_succeeded`       | `ConfirmPayment` → enrollment créé        |
| `checkout.session.expired`          | `payment_failed`          | `CancelPending` → purchase → `failed`     |
| `charge.refunded`                   | `refund_succeeded`        | `Refund` → purchase `refunded`, enrollments annulés |

### Sécurité : validation de la signature

La route `/webhooks/stripe` est publique (pas de JWT). N'importe qui pourrait envoyer une fausse requête. Stripe protège cela via une signature HMAC-SHA256 incluse dans le header `Stripe-Signature`.

```
┌──────────────────────────────────────────────────────────┐
│ WebhookHandler.Stripe                                    │
│                                                          │
│  1. Lire le corps brut (limité à 64 Ko)                  │
│  2. Vérifier que Stripe-Signature est présent            │
│  3. webhook.ConstructEvent(payload, sig, secret)         │
│      ├── Signature invalide ──► 400 Bad Request          │
│      └── Signature valide   ──► traitement de l'événement│
└──────────────────────────────────────────────────────────┘
```

> **Important** : le corps de la requête doit être lu **brut** (avant tout décodage JSON) pour que la vérification de signature fonctionne. C'est pourquoi le handler lit `io.ReadAll(r.Body)` directement, et non `json.Decode`.

### Idempotence

Stripe peut renvoyer le même webhook plusieurs fois (en cas d'échec de livraison ou de retry). Le handler est idempotent : si la purchase n'est plus en statut `pending`, `ConfirmPayment` retourne `ErrPurchaseAlreadyProcessed` et le webhook répond `204` sans erreur. Stripe ne retentera pas.

```
  1er webhook ──► purchase pending ──► completed + enrollment créé
  2e webhook  ──► purchase completed ──► ErrAlreadyProcessed ──► 204 (ignoré)
```

### Configuration locale avec Stripe CLI

Pour tester les webhooks en développement, utiliser la Stripe CLI :

```bash
stripe listen --forward-to localhost:8000/webhooks/stripe
```

La CLI affiche le `STRIPE_WEBHOOK_SECRET` à utiliser dans `.env` pour la session de test.

## Coupons et promotions

Les coupons sont entièrement gérés par Stripe — expiration, usage unique, pourcentage ou montant fixe. Il n'y a pas de table `coupons` en base. Si un second provider (ex. PayPal) est ajouté, ses coupons seront gérés par ce provider indépendamment. C'est un compromis accepté pour éviter une abstraction prématurée.

## Panier

Le panier est stateless et côté client. Le frontend accumule les articles et envoie la liste en une seule requête `BuyBundle` (pour les bundles) ou `BuyCourse` (pour les cours unitaires). Il n'y a pas de table `cart`. Si la persistance du panier devient nécessaire, elle peut être ajoutée sans modifier le flow de paiement ni d'enrollment.

## Variables d'environnement

| Variable                | Description                                               |
|-------------------------|-----------------------------------------------------------|
| `STRIPE_SECRET_KEY`     | Clé secrète Stripe (`sk_live_...` ou `sk_test_...`)       |
| `STRIPE_WEBHOOK_SECRET` | Secret de signature des webhooks Stripe (`whsec_...`)     |
| `STRIPE_SUCCESS_URL`    | URL de redirection après paiement réussi                  |
| `STRIPE_CANCEL_URL`     | URL de redirection après annulation du paiement           |

## Ajouter un nouveau provider

```
1. Créer internal/payment/<provider>/client.go
   └── Implémenter payment.Provider :
       ├── CreateCheckout(ctx, Order) (Session, error)
       ├── HandleWebhook(payload, signature) (Event, error)
       └── Refund(ctx, paymentID, amountCents) error

2. Ajouter une route webhook dédiée
   └── POST /webhooks/<provider>  (publique, validation signature propre)

3. Câbler dans cmd/api/main.go
   └── Injecter le nouveau provider dans NewPurchaseService
```

La `PurchaseService` et toute la logique métier restent inchangées.
