# Emails transactionnels

Ce dojo intègre un système d'envoi d'emails structuré en trois couches :

```
AuthService  →  AuthMailer  →  Mailer
(logique)       (templates)    (transport)
```

## Sommaire

- [Architecture](#architecture)
- [Ajouter un transport](#ajouter-un-transport)
- [Templates HTML](#templates-html)
- [Personnaliser les templates](#personnaliser-les-templates)
- [Internationalisation](#internationalisation)

---

## Architecture

### Mailer

Interface de transport générique — responsable uniquement d'envoyer un message.

```go
type Mailer interface {
    Send(ctx context.Context, msg MailMessage) error
}
```

Implémentations disponibles :
- `platform/mailer/mock.go` — mock pour les tests (enregistre les messages sans les envoyer)
- `platform/mailer/smtp.go` — transport SMTP avec support multipart HTML/texte

### AuthMailer

Interface qui définit les emails liés à l'authentification.

```go
type AuthMailer interface {
    SendAccountVerification(ctx context.Context, to, token string) error
    SendPasswordReset(ctx context.Context, to, token string) error
    SendOTP(ctx context.Context, to, code string) error
}
```

L'implémentation concrète `authMailer` construit le `MailMessage` et délègue l'envoi au `Mailer`.

### AuthService

Utilise `AuthMailer` sans connaître le transport ni le contenu des messages :

```go
// AuthService délègue simplement
s.mailer.SendAccountVerification(ctx, user.Email, rawToken)
```

---

## Ajouter un transport

Créer un fichier dans `internal/platform/mailer/` qui implémente `service.Mailer` :

```go
// internal/platform/mailer/smtp.go
type SMTPMailer struct { ... }

func (m *SMTPMailer) Send(ctx context.Context, msg service.MailMessage) error {
    // envoi SMTP
}
```

Puis l'injecter dans `main.go` :

```go
mailer := mailer.NewSMTPMailer(cfg.SMTP)
authMailer := service.NewAuthMailer(mailer)
authService := service.NewAuthService(..., authMailer, cfg.JWT)
```

---

## Templates HTML

Les templates HTML sont situés dans `internal/platform/mailer/templates/` et sont
**embarqués dans le binaire** via `embed.FS` — aucun fichier à déployer séparément.

| Fichier | Utilisé par |
|---------|------------|
| `verification.html` | `SendAccountVerification` |
| `password_reset.html` | `SendPasswordReset` |
| `otp.html` | `SendOTP` |

Chaque template reçoit une `map[string]string` comme données :

```
verification.html  →  {{ .Token }}
password_reset.html → {{ .Token }}
otp.html           →  {{ .Code }}
```

Les templates sont chargés une seule fois au démarrage via `mailer.ParseTemplates()` :

```go
// cmd/api/main.go
authMailer := service.NewAuthMailer(mailer.NewSMTP(...), mailer.ParseTemplates())
```

Chaque email est envoyé en **multipart/alternative** : le corps HTML est accompagné
d'un fallback texte brut pour les clients sans support HTML.

---

## Personnaliser les templates

Les sujets sont définis dans chaque méthode de `authMailer` (`internal/service/mailer.go`).
Les corps HTML sont dans les fichiers `.html` de `internal/platform/mailer/templates/`.

Pour modifier un template, éditez directement le fichier HTML correspondant.
Les données disponibles dans chaque template sont listées dans le tableau ci-dessus.

Pour des mises en page plus élaborées, le package `html/template` de la librairie standard
supporte les conditions, boucles et imbrications de templates.

---

## Internationalisation

Les textes des emails sont actuellement en anglais, en dur dans `internal/service/mailer.go`.

Pour une internationalisation complète, remplacez les chaînes par un système de templates
externalisés (fichiers `templates/email/*.html`) ou une lib i18n comme
[go-i18n](https://github.com/nicksnyder/go-i18n).

Cette évolution n'est pas intégrée dans le dojo de base — elle est laissée
à la discrétion de chaque projet.
