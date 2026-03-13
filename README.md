# Go API Template

Template de démarrage pour API REST avec Go, Gin, GORM, PostgreSQL, Redis.

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
│   ├── middleware/      → Auth JWT, CORS, Rate Limiting, Request ID, Logger
│   ├── mocks/           → Mocks testify/mock générés manuellement
│   ├── models/          → Modèles GORM
│   ├── repositories/    → Couche d'accès aux données
│   ├── router/          → Définition des routes
│   ├── services/        → Logique métier + interfaces
│   └── testutil/        → Helpers partagés pour les tests
├── migrations/          → Fichiers SQL versionnés (golang-migrate)
├── pkg/
│   ├── auth/            → JWT (génération, validation)
│   ├── logger/          → Initialisation zerolog (JSON prod / pretty dev)
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

| Méthode | Route                   | Description         |
|---------|-------------------------|---------------------|
| GET     | `/health`               | Health check        |
| POST    | `/api/v1/auth/register` | Inscription         |
| POST    | `/api/v1/auth/login`    | Connexion           |
| POST    | `/api/v1/auth/refresh`  | Rafraîchir le token |

### Authentifiés

| Méthode | Route                      | Description             |
|---------|----------------------------|-------------------------|
| POST    | `/api/v1/auth/logout`      | Déconnexion             |
| GET     | `/api/v1/profile`          | Profil utilisateur      |
| PUT     | `/api/v1/profile`          | Modifier le profil      |
| PUT     | `/api/v1/profile/password` | Changer le mot de passe |

### Admin uniquement

| Méthode | Route                          | Description             |
|---------|--------------------------------|-------------------------|
| GET     | `/api/v1/admin/users`          | Lister les utilisateurs |
| PUT     | `/api/v1/admin/users/:id/role` | Changer le rôle         |

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
- CORS configurable
- Migrations SQL versionnées avec rollback
- Soft delete sur les utilisateurs
- Pagination sur les listings
- Réponses API uniformes
- Logging structuré JSON (zerolog) avec Request ID tracé sur chaque log
- Tests unitaires isolés (mocks + miniredis)
- Tests d'intégration avec vraie base de données
- Hot reload en développement
- Build multi-stage optimisé pour la production
- Health check avec dépendances Docker
