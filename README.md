# Go API Template

Template de démarrage pour API REST avec Go, Gin, GORM, PostgreSQL, Redis.

## Quick Start

### Prérequis

- [Go 1.22+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/) + Docker Compose
- `make`

### 1. Créer votre projet depuis ce template

```bash
# Option A — bouton "Use this template" sur GitHub, puis :
git clone https://github.com/<vous>/<votre-projet>.git
cd <votre-projet>

# Option B — clone direct
git clone https://github.com/Balr0g404/go-api-skeletton.git mon-api
cd mon-api
rm -rf .git && git init
```

### 2. Lancer le script d'installation

```bash
make setup
```

Le script interactif vous guide à travers :
- le renommage du module Go (`go.mod` + tous les fichiers `.go`)
- la création du `.env` avec les variables obligatoires
- la génération automatique du `JWT_SECRET` (50 chars) ou saisie manuelle
- les credentials de l'admin de seed

> Pour le détail des variables disponibles, voir [`.env.example`](.env.example).

### 3. Lancer le serveur

```bash
make dev        # démarre PostgreSQL + Redis + le serveur avec hot reload (Air)
```

Vérifiez que tout fonctionne :

```bash
curl http://localhost:8080/health
# {"success":true,"data":{"status":"ok"}}
```

La documentation Swagger est disponible sur `http://localhost:8080/swagger/index.html`.

### 5. Ajouter votre premier domaine métier

Générez le squelette d'un nouveau module en une commande :

```bash
make scaffold NAME=post
```

Cela crée :
- `internal/models/post.go` — struct GORM + réponse JSON
- `internal/repositories/post.go` — CRUD de base
- `internal/services/post.go` — interface + service
- `internal/handlers/post.go` — handlers HTTP avec annotations Swagger

Le script affiche ensuite les 3 lignes à ajouter dans `cmd/server/main.go` et `internal/router/` pour brancher les routes.

### 6. Créer la migration SQL

```bash
make migrate-create NAME=create_posts_table
```

Éditez le fichier `migrations/<timestamp>_create_posts_table.up.sql` généré, puis relancez `make dev` — les migrations s'appliquent automatiquement au démarrage.

### 7. Lancer les tests

```bash
make test                 # tests unitaires (aucune dépendance externe)
make test-integration     # tests d'intégration (requiert Docker en cours)
make test-coverage        # rapport HTML de couverture
```

---

## Stack technique

| Couche | Choix | Justification |
|--------|-------|---------------|
| HTTP | **Gin** | Performant, middleware ecosystem riche |
| ORM | **GORM** + PostgreSQL 16 | Relations, soft delete, hooks |
| Migrations | **golang-migrate** (fichiers SQL versionnés) | Rollback possible, migration as code |
| Cache / Blacklist | **Redis 7** | Blacklist JWT, rate limiting par IP |
| Auth | **JWT** access + refresh tokens | Stateless, logout réel via blacklist Redis |
| Documentation | **Swagger** (swaggo/gin-swagger) | UI interactive générée depuis les annotations |
| Tests unitaires | **testify/mock** + **miniredis** | Isolation totale, pas de dépendances externes |
| Tests d'intégration | build tag `integration` + **TEST_DATABASE_URL** | Séparés des tests unitaires, skip gracieux |
| Logging | **zerolog** | JSON structuré en prod, pretty en dev, level par env |
| Hot reload | **Air** | Rebuild automatique en dev |
| Conteneurs | **Docker Compose** dev & prod | Parité dev/prod, build multi-stage |

## Structure du projet

```
├── cmd/server/          → Point d'entrée
├── internal/
│   ├── config/          → Configuration (env vars)
│   ├── database/        → Connexions PostgreSQL & Redis, RunMigrations
│   ├── handlers/        → Contrôleurs HTTP
│   ├── middleware/      → Auth JWT, CORS, Security, Timeout, Recovery, Rate Limiting, Request ID, Logger
│   ├── mocks/           → Mocks testify/mock générés manuellement
│   ├── models/          → Modèles GORM
│   ├── repositories/    → Couche d'accès aux données
│   ├── router/          → Définition des routes
│   ├── services/        → Logique métier + interfaces
│   └── testutil/        → Helpers partagés pour les tests
├── migrations/          → Fichiers SQL versionnés (golang-migrate)
├── pkg/
│   ├── auth/            → JWT (génération, validation)
│   ├── email/           → Abstraction d'envoi d'emails (SMTP, Resend, noop) + templates
│   ├── filtering/       → Parse et validation de ?sort=, ?order=, ?filter[field]=value
│   ├── logger/          → Initialisation zerolog (JSON prod / pretty dev)
│   ├── pagination/      → Encode/decode cursor (base64url) pour la pagination curseur
│   └── response/        → Helpers de réponse API uniformes
├── docker/              → Dockerfiles dev & prod
├── docker-compose.dev.yml
├── docker-compose.prod.yml
└── Makefile
```

## Démarrage rapide

```bash
# Lancer en mode développement (hot reload)
make dev

# Lancer en mode production
make prod

# Voir les logs
make logs-dev

# Arrêter
make down-dev

# Arrêter et supprimer les volumes
make down-dev-v
```

## Migrations

Les migrations SQL sont versionnées dans `migrations/` et compilées dans le binaire via `embed.FS`.
Elles sont appliquées automatiquement au démarrage.

```bash
# Appliquer toutes les migrations (depuis l'hôte, Docker doit tourner)
make migrate-up

# Rollback de la dernière migration
make migrate-down

# Créer une nouvelle migration
make migrate-create NAME=add_posts_table
```

> Requiert le CLI [`migrate`](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate).

## Logging

Les logs utilisent [zerolog](https://github.com/rs/zerolog) via un middleware Gin dédié.

**En développement** — sortie texte colorée :
```
INF request method=GET path=/health status=200 latency=1ms ip=127.0.0.1 request_id=a1b2c3d4-...
```

**En production** — JSON structuré sur stderr :
```json
{"level":"info","time":"2026-03-13T10:00:00Z","request_id":"a1b2c3d4-...","method":"POST","path":"/api/v1/auth/login","status":200,"latency":"12ms","ip":"172.18.0.1","message":"request"}
```

Chaque requête reçoit un `X-Request-ID` UUID (généré si absent du header entrant) propagé dans les logs et les réponses. Le niveau de log est automatiquement adapté au status HTTP : `info` (2xx), `warn` (4xx), `error` (5xx).

## Middleware chain

Chaque requête traverse les middlewares dans cet ordre :

| # | Middleware | Rôle |
|---|-----------|------|
| 1 | **RequestID** | Génère ou propage `X-Request-ID` (UUID) |
| 2 | **Recovery** | Capture les panics, logue avec request_id + stack trace, retourne 500 JSON |
| 3 | **Logger** | Log structuré zerolog après la réponse (method, path, status, latency) |
| 4 | **Timeout** | Annule la requête après 30s, retourne 503 JSON |
| 5 | **SecurityHeaders** | X-Frame-Options, X-Content-Type-Options, Referrer-Policy, HSTS (prod) |
| 6 | **CORS** | Origins configurables via `CORS_ALLOWED_ORIGINS`, matching exact en prod |
| 7 | **RateLimit** | 100 req/min par IP via Redis, headers `X-RateLimit-*` |
| 8 | **AuthRequired** | Validation JWT, vérification blacklist Redis (routes protégées) |
| 9 | **RoleRequired** | Vérification du rôle dans le contexte (routes admin) |

## Linter

```bash
make lint
```

La configuration [`.golangci.yml`](.golangci.yml) active : `errcheck`, `govet`, `staticcheck`, `gofmt`, `goimports`, `revive`, `gosec`, `misspell`, `unconvert`, `unparam`, `copylock`, `exhaustive`, `noctx`, `bodyclose`.

> Requiert [`golangci-lint`](https://golangci-lint.run/usage/install/) : `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`

## Tests

```bash
# Tests unitaires (pas de dépendances externes)
make test

# Tests d'intégration (requiert PostgreSQL sur DB_PORT)
make test-integration

# Rapport de couverture (coverage.html)
make test-coverage
```

Les tests d'intégration sont isolés via le build tag `//go:build integration`.
Ils skippent automatiquement si `TEST_DATABASE_URL` n'est pas défini.
Chaque test nettoie ses données via `t.Cleanup`.

## Documentation API (Swagger)

La documentation est disponible à `http://localhost:8080/swagger/index.html` une fois le serveur lancé.

Pour régénérer après modification des annotations :

```bash
make swagger
```

> Requiert le CLI [`swag`](https://github.com/swaggo/swag) : `go install github.com/swaggo/swag/cmd/swag@latest`

Les annotations Swagger se trouvent sur les handlers dans `internal/handlers/`.

## Endpoints API

### Publics

| Méthode | Route                              | Description                   |
|---------|------------------------------------|-------------------------------|
| GET     | `/health`                          | Health check                  |
| POST    | `/api/v1/auth/register`            | Inscription                   |
| POST    | `/api/v1/auth/login`               | Connexion                     |
| POST    | `/api/v1/auth/refresh`             | Rafraîchir le token           |
| POST    | `/api/v1/auth/forgot-password`     | Demander un lien de reset     |
| POST    | `/api/v1/auth/reset-password`      | Réinitialiser le mot de passe |

### Authentifiés

| Méthode | Route                      | Description             |
|---------|----------------------------|-------------------------|
| POST    | `/api/v1/auth/logout`      | Déconnexion             |
| GET     | `/api/v1/profile`          | Profil utilisateur      |
| PUT     | `/api/v1/profile`          | Modifier le profil      |
| PUT     | `/api/v1/profile/password` | Changer le mot de passe |

### Admin uniquement

| Méthode | Route                               | Description                      |
|---------|-------------------------------------|----------------------------------|
| GET     | `/api/v1/admin/users`               | Lister les utilisateurs (offset) |
| GET     | `/api/v1/admin/users/cursor`        | Lister les utilisateurs (cursor) |
| PUT     | `/api/v1/admin/users/:id/role`      | Changer le rôle                  |

## Pagination curseur

L'endpoint `/api/v1/admin/users/cursor` implémente une pagination par curseur, plus performante que l'offset sur les grandes tables (pas de `OFFSET` SQL, index sur l'ID).

**Paramètres** : `cursor` (optionnel, opaque base64url), `limit` (1–100, défaut 20).

```bash
# Première page
curl http://localhost:8080/api/v1/admin/users/cursor?limit=20 \
  -H "Authorization: Bearer <token>"

# Page suivante (next_cursor de la réponse précédente)
curl "http://localhost:8080/api/v1/admin/users/cursor?cursor=<next_cursor>&limit=20" \
  -H "Authorization: Bearer <token>"
```

**Réponse** :
```json
{
  "success": true,
  "data": {
    "users": [...],
    "next_cursor": "MTIz",
    "has_next": true,
    "limit": 20
  }
}
```

Quand `has_next` est `false`, le champ `next_cursor` est absent : c'est la dernière page.

## Service email

L'envoi d'emails est géré par `pkg/email/` avec une interface `Sender` commune à tous les providers.

### Providers disponibles

| `EMAIL_PROVIDER` | Description |
|-----------------|-------------|
| `noop` (défaut) | Discarde silencieusement les emails — idéal en développement |
| `smtp` | Envoi via SMTP (STARTTLS sur 587, TLS sur 465) |
| `resend` | Envoi via l'API REST [Resend](https://resend.com) |

### Variables d'environnement

```bash
EMAIL_PROVIDER=smtp          # smtp | resend | noop
EMAIL_FROM=noreply@example.com
APP_BASE_URL=https://example.com   # utilisé dans les liens de reset

# SMTP
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=user@example.com
SMTP_PASSWORD=secret

# Resend
RESEND_API_KEY=re_xxxxxxxxxxxx
```

### Emails envoyés automatiquement

| Déclencheur | Email |
|-------------|-------|
| `POST /auth/register` | Email de bienvenue |
| `POST /auth/forgot-password` | Lien de réinitialisation (TTL 1h) |

### Flux reset password

```bash
# 1. Demander le reset (toujours 200, ne révèle pas si l'email existe)
curl -X POST http://localhost:8080/api/v1/auth/forgot-password \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com"}'

# 2. Utiliser le token reçu par email
curl -X POST http://localhost:8080/api/v1/auth/reset-password \
  -H "Content-Type: application/json" \
  -d '{"token":"<token>","password":"newpassword123"}'
```

## Filtrage et tri

Les endpoints de liste (`/api/v1/admin/users` et `/api/v1/admin/users/cursor`) acceptent des paramètres de filtrage et de tri.

### Paramètres

| Paramètre | Valeurs | Défaut | Description |
|-----------|---------|--------|-------------|
| `sort` | `id`, `created_at`, `email`, `first_name`, `last_name`, `role` | `id` | Champ de tri |
| `order` | `asc`, `desc` | `asc` | Sens du tri |
| `filter[email]` | chaîne | — | Filtre exact sur l'email |
| `filter[role]` | `user`, `admin` | — | Filtre exact sur le rôle |
| `filter[active]` | `true`, `false` | — | Filtre sur le statut actif |

Un champ `sort` inconnu est silencieusement ramené au défaut (`id`). Un filtre sur un champ non whitelisté est ignoré (protection SQL injection).

### Exemples

```bash
# Trier par email décroissant
curl "http://localhost:8080/api/v1/admin/users?sort=email&order=desc" \
  -H "Authorization: Bearer <token>"

# Filtrer les admins
curl "http://localhost:8080/api/v1/admin/users?filter[role]=admin" \
  -H "Authorization: Bearer <token>"

# Combiner : admins actifs triés par date de création
curl "http://localhost:8080/api/v1/admin/users?sort=created_at&order=desc&filter[role]=admin&filter[active]=true" \
  -H "Authorization: Bearer <token>"

# Pagination curseur avec filtre
curl "http://localhost:8080/api/v1/admin/users/cursor?filter[role]=admin&limit=10" \
  -H "Authorization: Bearer <token>"
```

## Exemples d'utilisation

```bash
# Inscription
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123","first_name":"John","last_name":"Doe"}'

# Connexion
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'

# Profil (avec token)
curl http://localhost:8080/api/v1/profile \
  -H "Authorization: Bearer <access_token>"

# Rafraîchir le token
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh_token>"}'
```

## CI/CD GitHub Actions

Le pipeline est défini dans [.github/workflows/ci.yml](.github/workflows/ci.yml) :

| Job | Déclencheur | Description |
|-----|-------------|-------------|
| **lint** | push + PR | `golangci-lint` avec la config `.golangci.yml` |
| **test** | push + PR | `go test ./... -race` + rapport de couverture (Codecov) |
| **build** | après lint + test | Compilation du binaire en release, upload artifact |
| **docker** | push sur `main`/`master` seulement | Build + push de l'image vers GHCR (`ghcr.io/<owner>/<repo>`) |

Le job Docker utilise `GITHUB_TOKEN` (automatique) — aucun secret à configurer pour GHCR.

## Scaffold d'un nouveau module

```bash
make scaffold NAME=post
```

Génère en une commande :

| Fichier | Contenu |
|---------|---------|
| `internal/models/post.go` | Struct GORM + `ToResponse()` |
| `internal/repositories/post.go` | CRUD de base (Create, FindByID, Update, Delete) |
| `internal/services/post.go` | Interface repository + service avec logique métier |
| `internal/handlers/post.go` | Handlers HTTP avec annotations Swagger |

Le script affiche les instructions de câblage (main.go + router) à la fin.

## Personnalisation

1. Renommer le module dans `go.mod` : remplacer `github.com/Balr0g404/go-api-skeletton`
2. Rechercher/remplacer globalement le même chemin dans tous les fichiers `.go`
3. Lancer `go mod tidy`
4. Adapter le `.env` selon l'environnement cible

## Fonctionnalités incluses

- Authentification JWT (access + refresh tokens)
- Blacklist de tokens via Redis (logout réel)
- Autorisation par rôle (`user`, `admin`)
- Rate limiting par IP via Redis
- CORS configurable via `CORS_ALLOWED_ORIGINS`
- Headers de sécurité (X-Frame-Options, CSP, HSTS en prod)
- Timeout par requête (30s) avec réponse structurée
- Panic recovery avec log zerolog + request_id + stack trace
- Config linter `.golangci.yml` prête à l'emploi
- Migrations SQL versionnées avec rollback
- Soft delete sur les utilisateurs
- Pagination offset sur les listings
- Pagination curseur (base64url) pour les grandes tables
- Filtrage et tri générique (`?sort=`, `?order=`, `?filter[field]=value`) avec whitelist anti-injection
- Service email avec provider configurable (SMTP, Resend, noop) et templates HTML
- Reset password par email (token Redis TTL 1h, endpoint public sécurisé)
- Réponses API uniformes
- Logging structuré JSON (zerolog) avec Request ID tracé sur chaque log
- Tests unitaires isolés (mocks + miniredis)
- Tests d'intégration avec vraie base de données
- Hot reload en développement
- Build multi-stage optimisé pour la production
- Health check avec dépendances Docker
- Pipeline CI/CD GitHub Actions (lint → test → build → push GHCR)
- Script de scaffold (`make scaffold NAME=<module>`) pour générer modèle + repository + service + handler
