# Chiffrement des données sensibles

Ce dojo intègre un système de chiffrement **AES-256-GCM** pour protéger les données personnelles sensibles stockées en base de données (adresses, dates de naissance, IPs de connexion, etc.).

## Sommaire

- [Package security](#package-security)
- [Helpers store](#helpers-store)
- [Champs chiffrés dans la table users](#champs-chiffrés-dans-la-table-users)
- [Ajouter un champ chiffré dans un nouveau store](#ajouter-un-champ-chiffré-dans-un-nouveau-store)
- [Calculer la taille VARCHAR](#1-choisir-la-taille-varchar)

---

## Package security

Le package `internal/platform/security` expose le type `Cipher` qui wrape AES-256-GCM.

### Créer un Cipher

La clé doit faire exactement **32 bytes** (AES-256).

```go
cipher, err := security.NewCipher(cfg.Security.EncryptionKey)
if err != nil {
    // ErrInvalidKeySize si la clé ne fait pas 32 bytes
}
```

### Chiffrer une valeur

```go
encrypted, err := cipher.Encrypt("données sensibles")
// encrypted est une string base64 : nonce (12 bytes) + ciphertext + tag (16 bytes)
```

### Déchiffrer une valeur

```go
plaintext, err := cipher.Decrypt(encrypted)
// plaintext == "données sensibles"
```

### Propriétés

- **Authentifié** : toute altération du ciphertext est détectée (GCM tag).
- **Non-déterministe** : deux chiffrements du même texte produisent des résultats différents grâce au nonce aléatoire — impossible de deviner une valeur par comparaison en base.
- **Thread-safe** : une instance `Cipher` peut être partagée entre goroutines.

### Erreurs

| Erreur | Cause |
|---|---|
| `ErrInvalidKeySize` | Clé différente de 32 bytes |
| `ErrCiphertextTooShort` | Données trop courtes pour contenir un nonce |
| `ErrDecryptionFailed` | Données altérées ou mauvaise clé |

---

## Helpers store

Le package `internal/store` expose quatre fonctions utilitaires à utiliser dans n'importe quel store.

### `encrypt` / `decrypt`

Pour les champs `*string` :

```go
// Chiffrement — retourne nil si val est nil
enc, err := encrypt(cipher, u.PhoneNumber)  // *string → *string chiffrée

// Déchiffrement
dec, err := decrypt(cipher, enc)            // *string chiffrée → *string
```

### `encryptTime` / `decryptTime`

Pour les champs `*time.Time` (stockés au format `2006-01-02`) :

```go
// Chiffrement — retourne nil si t est nil
enc, err := encryptTime(cipher, u.BirthDate)  // *time.Time → *string chiffrée

// Déchiffrement
t, err := decryptTime(cipher, enc)            // *string chiffrée → *time.Time
```

---

## Champs chiffrés dans la table users

Les colonnes suivantes sont stockées chiffrées en base. Leur type SQL est `VARCHAR` (et non `DATE` ou `VARCHAR` standard) car elles contiennent du base64 AES-GCM.

| Colonne | Type SQL | Type Go | Raison |
|---|---|---|---|
| `birth_date` | `VARCHAR(64)` | `*time.Time` | Donnée personnelle sensible (RGPD) |
| `address_line1` | `VARCHAR(512)` | `*string` | Donnée personnelle sensible (RGPD) |
| `address_line2` | `VARCHAR(512)` | `*string` | Donnée personnelle sensible (RGPD) |
| `vat_number` | `VARCHAR(128)` | `*string` | Donnée financière sensible |
| `last_login_ip` | `VARCHAR(128)` | `*string` | Donnée de traçabilité sensible |

---

## Ajouter un champ chiffré dans un nouveau store

### 1. Choisir la taille VARCHAR

La sortie d'AES-256-GCM est encodée en base64. La taille finale dépend de la longueur du texte en clair :

```
taille_chiffrée = ceil((plaintext + 28) / 3) * 4
```

Les 28 bytes fixes correspondent au **nonce** (12 bytes) + **GCM tag** (16 bytes), présents dans chaque valeur chiffrée quelle que soit la donnée.

Exemples courants :

| Donnée | Plaintext max | Taille chiffrée | VARCHAR recommandé |
|---|---|---|---|
| Date (`2006-01-02`) | 10 | 52 | `VARCHAR(64)` |
| Code OTP (6 chiffres) | 6 | 48 | `VARCHAR(64)` |
| Numéro de téléphone | 20 | 64 | `VARCHAR(64)` |
| Numéro TVA | 50 | 104 | `VARCHAR(128)` |
| Adresse IP (IPv6) | 45 | 100 | `VARCHAR(128)` |
| IBAN | 34 | 84 | `VARCHAR(128)` |
| Ligne d'adresse | 255 | 380 | `VARCHAR(512)` |
| Champ libre / bio | 1000 | 1368 | `VARCHAR(1500)` |

> Toujours prévoir une marge raisonnable au-dessus de la taille calculée.

### 2. Déclarer la colonne en VARCHAR dans la migration

```sql
phone_number VARCHAR(64) DEFAULT NULL, -- encrypted at rest
```

### 2. Garder le type Go standard dans le model

Le model reste inchangé — le chiffrement est une responsabilité du store, pas du model.

```go
// internal/model/contact.go
type Contact struct {
    ID          string  `db:"id"`
    PhoneNumber *string `db:"phone_number"` // valeur déchiffrée en mémoire
}
```

### 3. Créer un type de scan dans le store

Déclarer un type qui embarque le model et redéclare les champs chiffrés en `*string` pour le scan sqlx.

```go
// internal/store/contact.go
type contactRow struct {
    model.Contact
    PhoneNumber *string `db:"phone_number"` // masque Contact.PhoneNumber pour sqlx
}
```

### 4. Chiffrer à l'écriture

```go
func (s *contactStore) Create(ctx context.Context, c *model.Contact) error {
    phone, err := encrypt(s.cipher, c.PhoneNumber)
    if err != nil {
        return err
    }
    _, err = s.db.ExecContext(ctx,
        `INSERT INTO contacts (id, phone_number) VALUES (UUID_V7(), ?)`,
        phone,
    )
    return err
}
```

### 5. Déchiffrer à la lecture

```go
func (s *contactStore) FindByID(ctx context.Context, id string) (*model.Contact, error) {
    var row contactRow
    err := s.db.GetContext(ctx, &row,
        `SELECT id, phone_number FROM contacts WHERE id = ?`, id)
    if err != nil {
        return nil, err
    }

    c := row.Contact
    c.PhoneNumber, err = decrypt(s.cipher, row.PhoneNumber)
    if err != nil {
        return nil, err
    }

    return &c, nil
}
```

### Pourquoi ce pattern ?

sqlx scanne les colonnes SQL vers les champs Go en se basant sur les tags `db`. Lorsqu'un champ existe à deux niveaux (model embarqué + struct externe), **Go applique la règle du champ le moins profond** : le champ `*string` de `contactRow` prend la priorité sur le champ `*string` de `model.Contact`. Cela permet à sqlx de recevoir le base64 brut, sans tenter de l'interpréter comme un type Go final.
