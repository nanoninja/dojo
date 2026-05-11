# Roles and Permissions

## Hierarchy

Roles are ordered by privilege level. A higher role inherits access to everything a lower role can do.

| Role | Value | Description |
|------|-------|-------------|
| `user` | 10 | Standard authenticated user |
| `moderator` | 20 | Content moderation, extended read access |
| `manager` | 30 | User and resource management |
| `admin` | 40 | Full user administration |
| `superadmin` | 50 | Irreversible or critical operations |
| `system` | 100 | Internal service accounts (jobs, workers) — not assigned to human users |

The comparison is numeric: `system (100) > superadmin (50) > admin (40) > ...`

A middleware check for `RoleAdmin` will allow `admin`, `superadmin`, and `system`, but reject `user`, `moderator`, and `manager`.

## Route Protection

| Method | Route | Minimum role | Notes |
|--------|-------|--------------|-------|
| `GET` | `/api/v1/users` | `admin` | |
| `GET` | `/api/v1/users/{id}` | `admin` | |
| `GET` | `/api/v1/users/me` | `user` | Own profile only |
| `GET` | `/api/v1/users/me/login-history` | `user` | Own history only |
| `PUT` | `/api/v1/users/{id}/profile` | `user` | Own profile only — 403 if `id` ≠ caller |
| `PUT` | `/api/v1/users/{id}/password` | `user` | Own password only — 403 if `id` ≠ caller |
| `DELETE` | `/api/v1/users/{id}` | `user` | Own account only — 403 if `id` ≠ caller |

All protected routes also require a valid JWT (`Authorization: Bearer <token>`).

## Role in JWT

The role is embedded in the JWT access token at login as the `role` claim:

```json
{
  "sub": "01966b3c-...",
  "role": "admin",
  "iat": 1234567890,
  "exp": 1234571490
}
```

> **Note:** Changing a user's role in the database takes effect on the next login or token refresh — existing tokens retain the role they were issued with until they expire.

## Default Role

Every new account is created with the `user` role. Role changes must be done directly in the database or via a future admin API.
