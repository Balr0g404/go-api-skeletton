# TODO — Améliorations du skeleton

## Critique

- [x] **Migrations de BDD** (`golang-migrate`) — migrations SQL versionnées embarquées dans le binaire.
- [x] **Tests** :
  - [x] Tests unitaires des services avec mocks (`testify/mock` + `miniredis`)
  - [x] Tests d'intégration des repositories (`//go:build integration` + `TEST_DATABASE_URL`)
  - [x] Helpers/fixtures partagés (`internal/testutil/`)
  - [x] Couverture de code (`make test-coverage`)
- [x] **Logging structuré** — remplacer le logger Gin par `zerolog` avec :
  - [x] Middleware de logging des requêtes (méthode, path, status, latence, request ID)
  - [x] Output JSON en production
- [x] **Arrêt gracieux** — le `main.go` doit écouter `SIGTERM`/`SIGINT` et attendre la fin des requêtes en cours avant de s'arrêter.

## Important

- [x] **Middleware Request ID** — générer un `X-Request-ID` UUID par requête, le propager dans les logs et les réponses.
- [x] **Headers de sécurité** — middleware type `helmet` :
  - `X-Frame-Options: DENY`
  - `X-Content-Type-Options: nosniff`
  - `Referrer-Policy: strict-origin-when-cross-origin`
  - `Strict-Transport-Security` (HSTS, prod seulement)
- [x] **CORS configurable** — sortir `*` du hardcode, lire `CORS_ALLOWED_ORIGINS` depuis `.env`.
- [x] **Middleware de timeout** — limiter la durée des requêtes (ex : 30s) pour éviter les goroutine leaks.
- [x] **Panic recovery personnalisée** — remplacer le `gin.Recovery()` par un handler custom qui logue avec le request ID et retourne une réponse structurée.
- [x] **Config du linter** — ajouter `.golangci.yml` avec les règles standards (`errcheck`, `govet`, `staticcheck`, `gofmt`, etc.).

## Scalabilité

- [ ] **Métriques** — instrumenter les handlers avec Prometheus (`/metrics`) ou OpenTelemetry.
- [x] **Pagination curseur** — compléter la pagination offset par une pagination curseur pour les grandes tables.
- [x] **Filtrage/tri générique** — supporter `?sort=`, `?order=`, `?filter[field]=value` sur les endpoints de liste.
- [x] **Service email** — base pour l'envoi d'emails (reset password, vérification) avec un provider configurable (SMTP / Resend / Mailgun).

## Workflow développeur

- [x] **CI/CD GitHub Actions** — pipeline avec : lint → test → build → push image Docker.
- [x] **`make migrate-create`** — générer un nouveau fichier de migration SQL numéroté.
- [x] **Script de génération de module** — script pour scaffolder rapidement modèle + repository + service + handler + routes.

---

## Backlog

### Critique

- [x] **Sécuriser le seed admin** — passer les credentials via env vars, exécution dev-only.
- [x] **Refuser de démarrer sans `JWT_SECRET` en prod** — valider la config au démarrage et `log.Fatal` si absent en production.
- [ ] **Rate limiter atomique** — remplacer l'implémentation actuelle par un script Lua Redis (INCR + EXPIRE dans une seule opération atomique).
- [ ] **Résilience Redis (SPOF)** — définir une stratégie explicite par endpoint si Redis est indisponible :
  - Rate limit → fall-back sur rate limiter en mémoire
  - Blacklist/logout/login → fail-closed (refuser) avec log error
  - Reset password → fail-closed
  - Ajouter des logs structurés dédiés pour chaque cas.
- [ ] **`NewPostgres` / `NewRedis` sans `log.Fatal`** — remonter l'erreur au `main` pour un exit gracieux et contrôlé.

### Haute priorité

- [ ] **README : section `cmd/server/`** — la section est annoncée mais le dossier n'apparaît pas dans le repo (`.gitignore` ou oubli).
- [ ] **Doublon email via contrainte DB** — ajouter un index unique sur `users.email` en base plutôt que de ne gérer le doublon qu'au niveau applicatif.
- [ ] **Propager `context.Context`** — toutes les méthodes de services et repositories doivent accepter un `ctx context.Context` en premier paramètre.
- [ ] **Logger les erreurs Redis** — toutes les opérations Redis sans check d'erreur (`Set`, `Del`, `Expire` sur blacklist) doivent logger en `warn`/`error`.
- [ ] **Constructeurs DB/Redis retournant `error`** — remplacer `log.Fatalf` dans `NewPostgres` et `NewRedis` par un `return nil, err`.
- [ ] **Architecture** — La structure du dossier internal est trop plate et perds en lisibilité. Rajouter un découpage par domaine (feature-based) auth, user et common.

### Moyenne priorité

- [ ] **Extraire `UserService` de `AuthService`** — séparer la gestion des utilisateurs (liste, rôle, profil) de la logique d'authentification pure.
- [ ] **Interfaces pour les dépendances des handlers** — les handlers dépendent de structs concrètes ; introduire des interfaces pour faciliter les tests et l'injection.
- [ ] **Endpoint `/ready` (readiness)** — endpoint séparé de `/health` qui vérifie les dépendances critiques (DB, Redis) avant de signaler la disponibilité.
- [ ] **Versionnage applicatif des erreurs** — inclure un code d'erreur applicatif et la version API dans les réponses d'erreur structurées.

### Basse priorité

- [ ] **Timeout client HTTP Resend** — configurer un `http.Client` avec timeout explicite dans `resend.go`.
- [ ] **Status code du timeout middleware** — vérifier et corriger le code HTTP retourné en cas de timeout (doit être `503` ou `504`).
- [ ] **Tests de charge basiques** — ajouter des benchmarks ou tests de charge sur le rate limit et la pagination curseur.
