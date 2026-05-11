# Journalisation et purge des audits de connexion

Ce dojo enregistre chaque tentative de connexion (succès et échec) dans la table `login_audit_logs`. Une goroutine de fond purge automatiquement les entrées expirées en lots pour limiter la pression sur la base de données.

## Sommaire

- [Table login_audit_logs](#table-login_audit_logs)
- [Statuts de connexion](#statuts-de-connexion)
- [Données chiffrées](#données-chiffrées)
- [Purge automatique](#purge-automatique)
- [Configuration](#configuration)
- [Déploiement multi-instances](#déploiement-multi-instances)

---

## Table login_audit_logs

La table est en **append-only** — les lignes ne sont jamais mises à jour après insertion. Elle sert uniquement à la traçabilité et à la détection d'abus.

| Colonne      | Type          | Description                                      |
|--------------|---------------|--------------------------------------------------|
| `id`         | UUID v7       | Identifiant unique, ordonné chronologiquement    |
| `user_id`    | UUID nullable | Référence vers l'utilisateur (null si inconnu)   |
| `email`      | VARCHAR       | Email de la tentative (en clair)                 |
| `ip_address` | VARCHAR       | IP chiffrée (AES-256-GCM)                        |
| `user_agent` | TEXT          | User-Agent de la requête                         |
| `status`     | login_status  | Résultat de la tentative (voir ci-dessous)       |
| `created_at` | TIMESTAMPTZ   | Horodatage de la tentative                       |

---

## Statuts de connexion

| Valeur                | Signification                                              |
|-----------------------|------------------------------------------------------------|
| `success`             | Authentification réussie                                   |
| `failed_password`     | Mot de passe incorrect                                     |
| `failed_locked`       | Compte temporairement verrouillé (trop de tentatives)      |
| `failed_not_found`    | Aucun compte trouvé pour cet email                         |
| `failed_unverified`   | Compte non vérifié par email                               |

---

## Données chiffrées

L'adresse IP est chiffrée en base (AES-256-GCM) avant insertion et déchiffrée transparentement à la lecture. Ce champ ne peut donc **pas** être utilisé comme critère de recherche SQL directe.

**L'email est stocké en clair**, ce qui permet des recherches directes (`WHERE email = ?`) ou des statistiques par utilisateur sans impact sur les performances. Pour des analyses basées sur l'IP (regroupement, filtrage), il reste nécessaire de charger les données et de filtrer côté application après déchiffrement.

---

## Purge automatique

La purge est orchestrée par la goroutine `runAuditPurge` démarrée au lancement du serveur (`cmd/api/purge.go`). Elle suit ce cycle :

```
Chaque jour à l'heure configurée (défaut : 2h00)
  └── Acquérir l'advisory lock PostgreSQL
        ├── Échec → une autre instance tourne, on passe
        └── Succès →
              Boucle :
                DELETE ... WHERE created_at < NOW() - retention LIMIT batchSize
                Si 0 lignes supprimées → stop
                Sinon → attendre batchPause, recommencer
              Libérer le lock
```

### Pourquoi des lots ?

Un `DELETE` massif sur des millions de lignes bloque les écritures concurrentes et génère un pic d'I/O. En supprimant par petits lots avec une pause entre chaque, la purge est quasi-invisible pour le serveur en production.

### Requête SQL exécutée à chaque lot

```sql
DELETE FROM login_audit_logs
WHERE id IN (
    SELECT id FROM login_audit_logs
    WHERE created_at < NOW() - ($1 * INTERVAL '1 second')
    LIMIT $2
)
```

Le sous-`SELECT` est nécessaire car PostgreSQL n'accepte pas `DELETE ... LIMIT` directement. `$1` est la rétention en secondes, `$2` est la taille du lot.

---

## Configuration

Toutes les valeurs sont surchargeables via variables d'environnement.

| Variable                              | Défaut | Description                                         |
|---------------------------------------|--------|-----------------------------------------------------|
| `AUDIT_PURGE_ENABLED`                 | `true` | Active ou désactive la purge automatique            |
| `AUDIT_PURGE_RETENTION_DAYS`          | `90`   | Rétention en jours avant suppression                |
| `AUDIT_PURGE_BATCH_SIZE`              | `100`  | Nombre de lignes supprimées par itération           |
| `AUDIT_PURGE_BATCH_PAUSE_SECONDS`     | `300`  | Pause en secondes entre deux lots (5 min)           |
| `AUDIT_PURGE_SCHEDULE_HOUR`           | `2`    | Heure de déclenchement quotidien (0-23, heure locale) |

### Exemple `.env`

```env
AUDIT_PURGE_ENABLED=true
AUDIT_PURGE_RETENTION_DAYS=90
AUDIT_PURGE_BATCH_SIZE=100
AUDIT_PURGE_BATCH_PAUSE_SECONDS=300
AUDIT_PURGE_SCHEDULE_HOUR=2
```

---

## Déploiement multi-instances

En environnement Kubernetes ou Docker Swarm avec plusieurs réplicas, chaque instance démarrerait sa purge simultanément sans protection. Pour éviter les `DELETE` concurrents sur les mêmes lignes, la goroutine acquiert un **advisory lock PostgreSQL** avant de démarrer :

```sql
SELECT pg_try_advisory_lock(hashtext('audit_purge'))
```

- Si le lock est obtenu → la purge démarre, le lock est libéré à la fin.
- Si le lock est déjà tenu par une autre instance → la purge est ignorée silencieusement pour ce cycle.

Le lock est **au niveau session** : PostgreSQL le libère automatiquement si la connexion se ferme, il ne peut donc pas rester bloqué indéfiniment.
