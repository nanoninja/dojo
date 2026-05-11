# API Reference

Base URL: `http://localhost:8000`

---

## Routes publiques

Ces routes ne nécessitent pas d'authentification.

### Authentification

| Méthode | Route | Description | Rate limit |
|---------|-------|-------------|------------|
| `POST` | `/auth/register` | Créer un compte | 3/min par IP |
| `POST` | `/auth/login` | Se connecter | 5/min par IP |
| `POST` | `/auth/verify` | Vérifier l'adresse email | 5/min par IP |
| `POST` | `/auth/verify/resend` | Renvoyer l'email de vérification | 5/min par IP |
| `POST` | `/auth/otp/verify` | Valider le code 2FA | 5/min par IP |
| `POST` | `/auth/otp/resend` | Renvoyer le code OTP | 5/min par IP |
| `POST` | `/auth/password/reset` | Demander un reset de mot de passe | 3/min par IP |
| `POST` | `/auth/password/new` | Définir un nouveau mot de passe | 3/min par IP |

### Santé et monitoring

| Méthode | Route | Description |
|---------|-------|-------------|
| `GET` | `/health` | État global (version, env, checks DB et cache) |
| `GET` | `/livez` | Liveness probe — le serveur tourne |
| `GET` | `/readyz` | Readiness probe — DB et cache accessibles |
| `GET` | `/metrics` | Métriques Prometheus (accès restreint par IP si `METRICS_ALLOWED_IPS` est défini) |
| `GET` | `/swagger/*` | Interface Swagger UI (dev et test uniquement) |

---

## Modes de transport (`AUTH_TRANSPORT_MODE`)

| Mode | Token envoyé via | CSRF requis | Usage recommandé |
|------|-----------------|-------------|-----------------|
| `bearer` | Header `Authorization: Bearer <token>` | Non | Dev local, Swagger, Postman, clients mobiles |
| `cookie` | Cookie HttpOnly `access_token` | Oui (`X-CSRF-Token`) | Frontend browser uniquement |
| `dual` | Header **et** cookie | Oui (`X-CSRF-Token`) | Production avec frontend browser |

> En mode `bearer`, aucun cookie n'est émis et aucun header CSRF n'est attendu.
> En mode `cookie` ou `dual`, toutes les requêtes mutantes (POST/PUT/DELETE) sur les routes protégées doivent inclure le header `X-CSRF-Token` avec la valeur du cookie `csrf_token`.

**Pour tester avec Swagger ou Postman** : utiliser `AUTH_TRANSPORT_MODE=bearer` dans `.env`, cliquer sur **Authorize** dans Swagger UI et coller le token JWT.

---

## Routes protégées

Ces routes nécessitent un header `Authorization: Bearer <access_token>` (mode `bearer` ou `dual`) ou un cookie d'accès (mode `cookie` ou `dual`).

### Authentification

| Méthode | Route | Description |
|---------|-------|-------------|
| `POST` | `/auth/logout` | Révoquer tous les refresh tokens |
| `POST` | `/auth/token/refresh` | Renouveler les tokens (rotation) |

> `/auth/token/refresh` et `/auth/logout` requièrent le header `X-CSRF-Token` en mode `cookie` ou `dual`.

### Utilisateurs — profil propre

| Méthode | Route | Description | Rôle minimum |
|---------|-------|-------------|--------------|
| `GET` | `/api/v1/users/me` | Récupérer son propre profil complet | `user` |
| `GET` | `/api/v1/users/me/login-history` | Historique de connexions (paramètre `limit`, défaut 20, max 100) | `user` |
| `PUT` | `/api/v1/users/{id}/profile` | Mettre à jour son propre profil | `user` |
| `PUT` | `/api/v1/users/{id}/password` | Changer son propre mot de passe | `user` |
| `DELETE` | `/api/v1/users/{id}` | Supprimer son propre compte | `user` |

> Les routes `{id}` vérifient que l'appelant agit sur son propre compte — une tentative sur l'ID d'un autre utilisateur retourne `403`.

### Utilisateurs — administration

| Méthode | Route | Description | Rôle minimum |
|---------|-------|-------------|--------------|
| `GET` | `/api/v1/users` | Lister les utilisateurs (paginé, filtrable) | `admin` |
| `GET` | `/api/v1/users/{id}` | Récupérer un utilisateur par ID | `admin` |

#### Paramètres de `GET /api/v1/users`

| Paramètre | Type | Défaut | Description |
|-----------|------|--------|-------------|
| `page` | int | `1` | Numéro de page |
| `limit` | int | `20` | Résultats par page (max 100) |
| `status` | string | — | Filtrer par statut : `pending`, `active`, `suspended`, `banned` |
| `search` | string | — | Recherche sur l'email, le prénom ou le nom |
| `sort` | string | `desc` | Ordre de tri par date de création : `asc` ou `desc` |

```
GET /api/v1/users?page=2&limit=10&status=active&search=john&sort=asc
```

```json
{
  "data": [...],
  "meta": {
    "page": 2,
    "limit": 10,
    "total": 42
  }
}
```

---

## Flux d'authentification

### Inscription

```
POST /auth/register
→ 201 { "id": "..." }
→ Email de vérification envoyé automatiquement

POST /auth/verify  { "user_id": "...", "token": "..." }
→ 204
```

### Connexion sans 2FA

```
POST /auth/login
→ 200 { "access_token": "...", "refresh_token": "..." }   (mode bearer/dual)
→ 200 { "authenticated": true }                           (mode cookie)
```

### Connexion avec 2FA activée

```
POST /auth/login
→ 200 { "otp_required": true, "user_id": "..." }
→ Code OTP envoyé par email

POST /auth/otp/verify  { "user_id": "...", "code": "123456" }
→ 200 { "access_token": "...", "refresh_token": "..." }
```

### Renouvellement de token

```
POST /auth/token/refresh  { "refresh_token": "..." }
→ 200 { "access_token": "...", "refresh_token": "..." }   (mode bearer/dual)
→ 200 { "refreshed": true }                               (mode cookie)
```

> Le refresh token est **rotatif** : l'ancien est révoqué à chaque renouvellement.

### Reset de mot de passe

```
POST /auth/password/reset  { "email": "..." }
→ 204  (toujours, même si l'email n'existe pas)

POST /auth/password/new  { "user_id": "...", "token": "...", "new_password": "..." }
→ 204
```

---

## Codes de réponse

| Code | Signification |
|------|--------------|
| `200` | Succès avec corps |
| `201` | Ressource créée |
| `204` | Succès sans corps |
| `400` | Données invalides |
| `401` | Non authentifié |
| `403` | Accès interdit |
| `404` | Ressource introuvable |
| `409` | Conflit (ex: email déjà pris) |
| `429` | Trop de requêtes |
| `500` | Erreur serveur |

---

## Format des erreurs

```json
{
  "error": "message décrivant le problème"
}
```
