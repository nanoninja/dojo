# Stratégie de versioning de l'API

---

## Routes actuelles

Toutes les routes métier sont préfixées `/api/v1/...`. Le router chi est déjà structuré
pour accueillir une `v2` sans toucher à la `v1`.

---

## Qu'est-ce qu'un breaking change ?

Un breaking change oblige les clients existants à modifier leur code. Toute modification
de ce type justifie une nouvelle version majeure.

**Breaking :**
- Supprimer ou renommer un champ de réponse JSON
- Changer le type d'un champ (`string` → `int`, `object` → `array`)
- Renommer ou supprimer une route
- Changer le code HTTP d'une réponse (`200` → `204`, `400` → `422`)
- Rendre obligatoire un champ qui était optionnel
- Changer la sémantique d'un paramètre existant

**Non-breaking (rétrocompatible) :**
- Ajouter un nouveau champ optionnel dans une réponse
- Ajouter une nouvelle route
- Ajouter un nouveau paramètre optionnel
- Réduire les contraintes de validation (accepter plus de valeurs)

---

## Coexistence v1 / v2 dans chi

Quand une `v2` est nécessaire, on ouvre un nouveau groupe de routes dans `router.go`
sans toucher au groupe `v1` existant. Les deux coexistent dans le même binaire.

```go
// v1 — inchangée, marquée deprecated
r.Route("/api/v1", func(r chi.Router) {
    r.Use(mw.Deprecated("Sat, 01 Jan 2027 00:00:00 GMT"))
    // ... routes v1 existantes
})

// v2 — nouvelles signatures
r.Route("/api/v2", func(r chi.Router) {
    // ... routes v2
})
```

Le middleware `Deprecated` injecte deux headers standards dans chaque réponse v1 :

```go
// internal/middleware/deprecated.go
func Deprecated(sunset string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Deprecation", "true")
            w.Header().Set("Sunset", sunset)
            next.ServeHTTP(w, r)
        })
    }
}
```

- `Deprecation: true` — signale que la route est en fin de vie
- `Sunset` — date à laquelle la route sera supprimée (RFC 8594)

Les clients qui lisent ces headers peuvent alerter leurs développeurs automatiquement.

---

## Procédure de lancement d'une v2

1. **Identifier les breaking changes** — lister explicitement ce qui change et pourquoi.

2. **Créer le groupe `/api/v2`** dans `router.go` avec les nouveaux handlers ou les
   nouvelles signatures.

3. **Marquer `/api/v1` comme deprecated** avec le middleware `Deprecated` et une date
   `Sunset` réaliste (minimum 3 mois après le lancement de v2).

4. **Documenter la migration** dans `docs/api.md` : quelles routes changent, comment
   migrer, exemples de requêtes avant/après.

5. **À la date Sunset** — supprimer le groupe `/api/v1` et le middleware `Deprecated`.

---

## Durée minimale de coexistence

| Contexte | Durée recommandée |
|---|---|
| API interne (clients contrôlés) | 1 mois |
| API externe (clients tiers) | 3 à 6 mois |

---

## Ce qu'on ne fait pas

- **Pas de versioning par header** (`Accept: application/vnd.api+json;version=2`) —
  plus difficile à documenter, à tester, et à router.
- **Pas de versioning par paramètre** (`?v=2`) — mélange la version avec les paramètres
  métier, cache mal.
- **Pas de v1.1, v1.2** — seules les versions majeures sont versionnées dans l'URL.
  Les ajouts rétrocompatibles sont livrés dans la version courante sans préavis.
