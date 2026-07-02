# Threadly Backend

Threadly is a production-style backend for a Twitter/X-style social media platform. It is built as a Go microservices monorepo with a public REST API Gateway, internal gRPC communication, Supabase PostgreSQL, Redis caching, RabbitMQ event messaging, Docker Compose, GitHub Actions CI, and manual deployment workflows prepared for future deployment.

This repository is designed as a portfolio project to demonstrate backend architecture, microservice design, authentication, service-to-service communication, database modelling, caching, event-driven workflows, testing, CI/CD, containerisation, and documentation.

## Table of contents

- [Project status](#project-status)
- [Tech stack](#tech-stack)
- [System architecture](#system-architecture)
- [Services](#services)
- [Database schema](#database-schema)
- [Core workflows](#core-workflows)
- [Repository structure](#repository-structure)
- [Environment files](#environment-files)
- [Running with Docker Compose](#running-with-docker-compose)
- [Running individual services locally](#running-individual-services-locally)
- [Database migrations](#database-migrations)
- [Testing](#testing)
- [CI/CD](#cicd)
- [API documentation](#api-documentation)
- [Security design](#security-design)
- [Portfolio highlights](#portfolio-highlights)
- [Roadmap](#roadmap)

## Project status

The backend currently includes:

- API Gateway with REST endpoints using Gin
- User Service for account, authentication, refresh-token sessions, and profile operations
- Content Service for posts, replies, likes, bookmarks, reposts, follows, followers, following, and timelines
- Feed Service for home feed generation and Redis-backed feed caching
- Notification Service for notification APIs and RabbitMQ event consumption
- Storage Service for media upload metadata and Supabase Storage/S3 integration
- Shared PostgreSQL schema managed through `db/migrations` using `golang-migrate`
- Docker Compose local runtime for backend services, Redis, and RabbitMQ
- GitHub Actions workflows for service-specific CI and manual deployment placeholders

## Tech stack

| Area | Technology |
|---|---|
| Language | Go |
| HTTP API | Gin |
| Internal RPC | gRPC + Protocol Buffers |
| Database | Supabase PostgreSQL |
| ORM | GORM |
| Migrations | golang-migrate |
| Cache | Redis |
| Messaging | RabbitMQ |
| Storage | Supabase Storage / S3-compatible API |
| Authentication | JWT access tokens + refresh-token sessions |
| Password security | Password hashing in User Service |
| Logging | zerolog |
| Local runtime | Docker Compose |
| CI/CD | GitHub Actions |
| API documentation | Swagger / OpenAPI via swaggo |
| Testing | Go test, testify, miniredis, package/service unit tests |

## System architecture

Threadly follows a pragmatic microservice architecture. The frontend communicates only with the API Gateway through REST APIs. The API Gateway validates requests, extracts authentication context, and calls backend services through gRPC.

Backend services currently use one shared Supabase PostgreSQL database for faster development and simpler local setup. The database schema is centralised under `db/`, while each service keeps its own repository/service layer.

```mermaid
flowchart LR
    FE["React + Vite + TypeScript<br/>Frontend"] -->|REST JSON| GW["API Gateway<br/>Go + Gin"]

    GW -->|gRPC| US["User Service<br/>auth, accounts, sessions"]
    GW -->|gRPC| CS["Content Service<br/>posts, replies, interactions"]
    GW -->|gRPC| FS["Feed Service<br/>home feed and cache"]
    GW -->|gRPC| NS["Notification Service<br/>notifications"]
    GW -->|gRPC| SS["Storage Service<br/>media upload"]

    US --> DB[(Supabase PostgreSQL)]
    CS --> DB
    FS --> DB
    NS --> DB
    SS --> DB

    FS --> REDIS[(Redis<br/>feed cache)]
    CS -->|publish events| MQ[(RabbitMQ)]
    MQ -->|consume events| FS
    MQ -->|consume events| NS
    SS --> OBJ[(Supabase Storage<br/>S3 compatible)]
```

Diagram sources are also stored in [`docs/diagrams`](docs/diagrams) as Mermaid files.

## Services

### API Gateway

The API Gateway is the public REST entry point.

Responsibilities:

- Exposes public REST APIs
- Validates JWT access tokens for protected routes
- Extracts `user_id` and role from token claims
- Calls User, Content, Feed, Notification, and Storage services via gRPC
- Converts gRPC errors into HTTP responses
- Hosts Swagger/OpenAPI documentation

### User Service

The User Service owns authentication and account data.

Responsibilities:

- User registration
- Login
- Password hashing and verification
- JWT access token issuing
- Refresh-token session creation and revocation
- Refresh-token validation flow
- Logout
- Current-user profile retrieval
- Profile update
- User soft deletion

### Content Service

The Content Service owns social content and social interactions.

Responsibilities:

- Create posts
- Get post details
- Update/delete own posts
- Create replies using `posts.reply_to_post_id`
- Get replies for a post
- Like/unlike posts
- Bookmark/unbookmark posts
- Repost/undo repost
- Follow/unfollow users
- Get followers/following
- Get user timeline
- Publish domain events to RabbitMQ

### Feed Service

The Feed Service builds and serves the home feed.

Responsibilities:

- Generate home feed from followed users and the current user
- Cache feed responses in Redis
- Invalidate feed cache when relevant RabbitMQ events arrive
- Provide feed health/status through gRPC

### Notification Service

The Notification Service stores and serves notifications.

Responsibilities:

- Consume RabbitMQ events such as likes, follows, replies, and reposts
- Convert events into notification rows
- Retrieve notifications for authenticated users
- Mark notifications as read

### Storage Service

The Storage Service handles media upload integration and media metadata.

Responsibilities:

- Upload media through Supabase Storage/S3-compatible provider
- Store media metadata in PostgreSQL
- Return public media URLs for posts and profile usage

## Database schema

Threadly uses one shared PostgreSQL database. The schema is centralised under `db/migrations`, while each service keeps its own repository layer.

Core tables:

| Table | Purpose |
|---|---|
| `users` | User accounts, profile data, roles, soft deletion |
| `sessions` | Hashed refresh-token sessions, expiration, revocation |
| `posts` | Posts and replies; replies use nullable `reply_to_post_id` |
| `media` | Uploaded media metadata linked to users/posts |
| `follows` | Follower/following relationships |
| `likes` | User likes on posts |
| `bookmarks` | Saved posts |
| `reposts` | Repost relationships |
| `notifications` | User notifications generated from events |

```mermaid
erDiagram
    USERS {
        uuid id PK
        string email UK
        string username UK
        string password_hash
        string display_name
        string bio
        string avatar_url
        string banner_url
        string location
        string website_url
        string role
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    SESSIONS {
        uuid id PK
        uuid user_id FK
        string refresh_token_hash
        string user_agent
        string ip_address
        timestamp expires_at
        timestamp revoked_at
        timestamp created_at
    }

    POSTS {
        uuid id PK
        uuid user_id FK
        uuid reply_to_post_id FK
        string content
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    MEDIA {
        uuid id PK
        uuid user_id FK
        uuid post_id FK
        string media_url
        string media_type
        string storage_key
        timestamp created_at
    }

    FOLLOWS {
        uuid follower_id FK
        uuid following_id FK
        timestamp created_at
    }

    LIKES {
        uuid user_id FK
        uuid post_id FK
        timestamp created_at
    }

    BOOKMARKS {
        uuid user_id FK
        uuid post_id FK
        timestamp created_at
    }

    REPOSTS {
        uuid user_id FK
        uuid post_id FK
        timestamp created_at
    }

    NOTIFICATIONS {
        uuid id PK
        uuid user_id FK
        uuid actor_id FK
        uuid post_id FK
        string type
        string message
        boolean is_read
        timestamp created_at
    }

    USERS ||--o{ SESSIONS : owns
    USERS ||--o{ POSTS : authors
    POSTS ||--o{ POSTS : replies
    USERS ||--o{ MEDIA : uploads
    POSTS ||--o{ MEDIA : contains
    USERS ||--o{ FOLLOWS : follower
    USERS ||--o{ FOLLOWS : following
    USERS ||--o{ LIKES : creates
    POSTS ||--o{ LIKES : receives
    USERS ||--o{ BOOKMARKS : saves
    POSTS ||--o{ BOOKMARKS : bookmarked
    USERS ||--o{ REPOSTS : creates
    POSTS ||--o{ REPOSTS : reposted
    USERS ||--o{ NOTIFICATIONS : receives
    USERS ||--o{ NOTIFICATIONS : triggers
    POSTS ||--o{ NOTIFICATIONS : references
```

## Core workflows

### Register and login

```mermaid
sequenceDiagram
    participant FE as Frontend
    participant GW as API Gateway
    participant US as User Service
    participant DB as PostgreSQL

    FE->>GW: POST /api/users/register or /api/users/login
    GW->>GW: Validate request body
    GW->>US: gRPC Register/Login
    US->>DB: Create/find user
    US->>US: Hash or verify password
    US->>US: Issue access token
    US->>DB: Store hashed refresh token session
    US-->>GW: User + access token + refresh token
    GW-->>FE: JSON auth response
```

### Refresh token and logout

```mermaid
sequenceDiagram
    participant FE as Frontend
    participant GW as API Gateway
    participant US as User Service
    participant DB as PostgreSQL

    FE->>GW: POST /api/users/refresh-token
    GW->>US: gRPC RefreshToken
    US->>DB: Validate hashed refresh token session
    US->>US: Issue new access token
    US-->>GW: New access token
    GW-->>FE: JSON access token

    FE->>GW: POST /api/users/logout + Bearer token
    GW->>GW: Validate JWT and extract user_id
    GW->>US: gRPC Logout(user_id, refresh_token)
    US->>DB: Revoke matching session
    US-->>GW: Logout success
    GW-->>FE: Success response
```

### Create post

```mermaid
sequenceDiagram
    participant FE as Frontend
    participant GW as API Gateway
    participant CS as Content Service
    participant DB as PostgreSQL
    participant MQ as RabbitMQ
    participant FS as Feed Service

    FE->>GW: POST /api/contents/posts + Bearer token
    GW->>GW: Validate JWT and request body
    GW->>CS: gRPC CreatePost
    CS->>DB: Insert post and media links
    CS->>MQ: Publish post.created
    MQ-->>FS: Deliver post.created
    FS->>FS: Invalidate affected feed cache
    CS-->>GW: Created post
    GW-->>FE: 201 Created
```

### Like and unlike

```mermaid
sequenceDiagram
    participant FE as Frontend
    participant GW as API Gateway
    participant CS as Content Service
    participant DB as PostgreSQL
    participant MQ as RabbitMQ
    participant NS as Notification Service

    FE->>GW: POST /api/contents/posts/{postID}/likes
    GW->>GW: Validate JWT
    GW->>CS: gRPC LikePost
    CS->>DB: Insert like if not exists
    CS->>MQ: Publish post.liked
    MQ-->>NS: Deliver post.liked
    NS->>DB: Create notification
    CS-->>GW: Like response
    GW-->>FE: Success

    FE->>GW: DELETE /api/contents/posts/{postID}/likes
    GW->>CS: gRPC UnlikePost
    CS->>DB: Delete like
    CS-->>GW: Unlike response
    GW-->>FE: Success
```

### Follow and unfollow

```mermaid
sequenceDiagram
    participant FE as Frontend
    participant GW as API Gateway
    participant CS as Content Service
    participant DB as PostgreSQL
    participant MQ as RabbitMQ
    participant NS as Notification Service
    participant FS as Feed Service

    FE->>GW: POST /api/contents/users/{userID}/follow
    GW->>GW: Validate JWT
    GW->>CS: gRPC FollowUser
    CS->>DB: Insert follow relationship
    CS->>MQ: Publish user.followed
    MQ-->>NS: Create follow notification
    MQ-->>FS: Invalidate follower feed cache
    CS-->>GW: Follow response
    GW-->>FE: Success

    FE->>GW: DELETE /api/contents/users/{userID}/follow
    GW->>CS: gRPC UnfollowUser
    CS->>DB: Delete follow relationship
    CS->>MQ: Publish user.unfollowed
    MQ-->>FS: Invalidate follower feed cache
    CS-->>GW: Unfollow response
    GW-->>FE: Success
```

### Home feed

```mermaid
sequenceDiagram
    participant FE as Frontend
    participant GW as API Gateway
    participant FS as Feed Service
    participant REDIS as Redis
    participant DB as PostgreSQL

    FE->>GW: GET /api/feed/home + Bearer token
    GW->>GW: Validate JWT
    GW->>FS: gRPC GetHomeFeed
    FS->>REDIS: Read feed cache
    alt Cache hit
        REDIS-->>FS: Cached feed
    else Cache miss
        FS->>DB: Query posts from self and followed users
        DB-->>FS: Feed rows
        FS->>REDIS: Cache feed response
    end
    FS-->>GW: Feed response
    GW-->>FE: JSON feed
```

### Notifications

```mermaid
sequenceDiagram
    participant CS as Content Service
    participant MQ as RabbitMQ
    participant NS as Notification Service
    participant DB as PostgreSQL
    participant GW as API Gateway
    participant FE as Frontend

    CS->>MQ: Publish social event
    MQ-->>NS: Consume event
    NS->>DB: Insert notification

    FE->>GW: GET /api/notifications + Bearer token
    GW->>GW: Validate JWT
    GW->>NS: gRPC GetNotifications
    NS->>DB: Query user notifications
    NS-->>GW: Notification list
    GW-->>FE: JSON response

    FE->>GW: PATCH /api/notifications/{id}/read
    GW->>NS: gRPC MarkAsRead
    NS->>DB: Update is_read
    NS-->>GW: Success
    GW-->>FE: Success
```

### Redis feed cache

```mermaid
flowchart TD
    REQUEST[Home feed request] --> CACHEKEY[Build Redis cache key by user and pagination]
    CACHEKEY --> GET[GET cached feed]
    GET --> HIT{Cache hit?}
    HIT -->|Yes| RETURN[Return cached feed]
    HIT -->|No| QUERY[Query PostgreSQL feed rows]
    QUERY --> SET[SET feed cache with TTL]
    SET --> RETURN

    EVENT[RabbitMQ social event] --> INVALIDATE[Invalidate affected user feed cache]
    INVALIDATE --> NEXT[Next request rebuilds cache]
```

## Repository structure

```mermaid
flowchart TD
    ROOT[threadly-backend]
    ROOT --> GITHUB[.github/workflows]
    ROOT --> DB[db]
    DB --> MIGRATIONS[migrations]
    DB --> MODELS[models]
    DB --> POSTGRES[postgres.go]

    ROOT --> DOCS[docs/diagrams]
    ROOT --> PKG[pkg]
    PKG --> CACHE[cache]
    PKG --> CONFIG[config]
    PKG --> LOGGER[logger]
    PKG --> MESSAGING[messaging]
    PKG --> MIDDLEWARE[middleware]

    ROOT --> PROTO[proto]
    PROTO --> USERPROTO[user]
    PROTO --> CONTENTPROTO[content]
    PROTO --> FEEDPROTO[feed]
    PROTO --> NOTIFPROTO[notification]
    PROTO --> STORAGEPROTO[storage]

    ROOT --> SERVICES[services]
    SERVICES --> GATEWAY[api-gateway]
    SERVICES --> USER[user-service]
    SERVICES --> CONTENT[content-service]
    SERVICES --> FEED[feed-service]
    SERVICES --> NOTIF[notification-service]
    SERVICES --> STORAGE[storage-service]

    ROOT --> COMPOSE[docker-compose.yml]
    ROOT --> MAKEFILE[Makefile]
```

```text
threadly-backend/
├── .github/workflows/               # CI and manual deployment workflows
├── db/
│   ├── migrations/                   # golang-migrate SQL migrations
│   ├── models/                       # shared database model definitions
│   └── postgres.go                   # PostgreSQL connection helper
├── docs/
│   └── diagrams/                     # architecture and workflow Mermaid diagrams
├── pkg/
│   ├── cache/                        # Redis helpers
│   ├── config/                       # environment config loader
│   ├── logger/                       # zerolog setup
│   ├── messaging/                    # RabbitMQ abstraction
│   └── middleware/                   # auth/role middleware
├── proto/                            # Protocol Buffer contracts and generated Go files
├── services/
│   ├── api-gateway/                  # Gin REST API gateway
│   ├── user-service/                 # auth/user/session service
│   ├── content-service/              # posts and social interactions
│   ├── feed-service/                 # home feed and Redis cache
│   ├── notification-service/         # notifications and event consumer
│   └── storage-service/              # media upload service
├── docker-compose.yml
├── Makefile
├── go.mod
└── README.md
```

## Environment files

Use separate env files for local and Docker modes.

| File | Purpose | Commit? |
|---|---|---|
| `.env.example` | Safe template for required variables | Yes |
| `.env.local` | Local Mac development using `localhost` | No |
| `.env.docker` | Docker Compose using service hostnames | No |
| `.env` | Optional local fallback loaded by `godotenv` | No |

Important Docker hostnames:

```env
REDIS_ADDR=redis:6379
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
USER_SERVICE_GRPC_ADDR=user-service:50051
CONTENT_SERVICE_GRPC_ADDR=content-service:50052
FEED_SERVICE_GRPC_ADDR=feed-service:50053
NOTIFICATION_SERVICE_GRPC_ADDR=notification-service:50054
STORAGE_SERVICE_GRPC_ADDR=storage-service:50055
```

For Supabase PostgreSQL, use your Supabase connection string in `DATABASE_URL`.

## Running with Docker Compose

### Prerequisites

Install:

- Go
- Docker Desktop
- `golang-migrate`
- `protoc` and Go protobuf plugins, only if regenerating proto files
- `swag`, only if regenerating Swagger docs

### Docker runtime flow

```mermaid
flowchart TD
    DEV[Developer machine] -->|docker compose --env-file .env.docker up| COMPOSE[Docker Compose]
    COMPOSE --> GW[api-gateway container]
    COMPOSE --> US[user-service container]
    COMPOSE --> CS[content-service container]
    COMPOSE --> FS[feed-service container]
    COMPOSE --> NS[notification-service container]
    COMPOSE --> SS[storage-service container]
    COMPOSE --> REDIS[redis container]
    COMPOSE --> MQ[rabbitmq container]

    GW -->|user-service:50051| US
    GW -->|content-service:50052| CS
    GW -->|feed-service:50053| FS
    GW -->|notification-service:50054| NS
    GW -->|storage-service:50055| SS

    US --> DB[(Supabase PostgreSQL)]
    CS --> DB
    FS --> DB
    NS --> DB
    SS --> DB
    FS --> REDIS
    CS --> MQ
    MQ --> FS
    MQ --> NS
    SS --> OBJ[(Supabase Storage)]
```

### 1. Create Docker env file

```bash
cp .env.example .env.docker
```

Update `.env.docker` with real values:

```env
DATABASE_URL=postgresql://postgres:<password>@<host>:5432/postgres?sslmode=require
JWT_SECRET=your_local_dev_secret
SUPABASE_S3_ENDPOINT=...
SUPABASE_S3_REGION=...
SUPABASE_S3_ACCESS_KEY_ID=...
SUPABASE_S3_SECRET_ACCESS_KEY=...
SUPABASE_PUBLIC_STORAGE_URL=...
```

### 2. Run migrations

```bash
make migrate-up DATABASE_URL='postgresql://postgres:<password>@<host>:5432/postgres?sslmode=require'
```

Check migration version:

```bash
make migrate-version DATABASE_URL='postgresql://postgres:<password>@<host>:5432/postgres?sslmode=require'
```

### 3. Build and start services

```bash
make docker-build
make docker-up
make docker-ps
```

### 4. Check API Gateway

```bash
curl http://localhost:8080/health
```

RabbitMQ dashboard:

```text
http://localhost:15672
username: guest
password: guest
```

## Running individual services locally

Create a local env file:

```bash
cp .env.example .env.local
```

Use `localhost` values for local dependencies:

```env
REDIS_ADDR=localhost:6379
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
USER_SERVICE_GRPC_ADDR=localhost:50051
CONTENT_SERVICE_GRPC_ADDR=localhost:50052
FEED_SERVICE_GRPC_ADDR=localhost:50053
NOTIFICATION_SERVICE_GRPC_ADDR=localhost:50054
STORAGE_SERVICE_GRPC_ADDR=localhost:50055
```

Run services:

```bash
make run-user
make run-content
make run-feed
make run-notification
make run-storage
make run-gateway
```

Or with Air hot reload:

```bash
make dev-user
make dev-content
make dev-feed
make dev-notification
make dev-storage
make dev-gateway
```

## Database migrations

Threadly uses `golang-migrate` with migration files stored in `db/migrations`.

Create a migration:

```bash
make migrate-create name=add_some_table
```

Run migrations:

```bash
make migrate-up DATABASE_URL='postgresql://postgres:<password>@<host>:5432/postgres?sslmode=require'
```

Rollback one migration:

```bash
make migrate-down DATABASE_URL='postgresql://postgres:<password>@<host>:5432/postgres?sslmode=require'
```

Check current migration version:

```bash
make migrate-version DATABASE_URL='postgresql://postgres:<password>@<host>:5432/postgres?sslmode=require'
```

## Testing

Run all backend tests:

```bash
make test
```

Run coverage:

```bash
make test-coverage
make coverage-html
```

Run service-specific tests:

```bash
make test-api-gateway
make test-user
make test-content
make test-feed
make test-notification
make test-storage
```

## CI/CD

Threadly uses GitHub Actions for CI/CD.

Current CI design:

- One required backend CI gate for branch protection
- Six path-filtered service CI workflows
- Six manual deployment workflows prepared for future deployment

The service CI workflows run only when their service folder or shared backend paths change.

Shared paths include:

```text
pkg/**
proto/**
db/**
go.mod
go.sum
docker-compose.yml
.github/workflows/**
```

Manual deployment workflows are intentionally `workflow_dispatch` only. They are placeholders for future Docker image build, registry push, deployment, and post-deploy health checks.

```mermaid
flowchart TD
    DEV[Feature branch] --> PR[Pull request to main]
    PR --> GATE[Backend Required CI]
    PR --> SVC[Path-filtered service CI]
    GATE --> REVIEW[Code review and branch protection]
    SVC --> REVIEW
    REVIEW --> MERGE[Merge to main]
    MERGE --> MAINCI[CI runs again on main]

    MANUAL[Manual workflow_dispatch] --> DEPLOY[Deploy selected service]
    DEPLOY --> TEST[Run service tests]
    TEST --> BUILD[Future: build Docker image]
    BUILD --> PUSH[Future: push image]
    PUSH --> RELEASE[Future: deploy and health check]
```

## API documentation

Generate Swagger/OpenAPI docs:

```bash
make swagger-gen
```

Then run the API Gateway and open:

```text
http://localhost:8080/swagger/index.html
```

## Security design

Implemented or planned security concerns:

- Passwords are hashed, not stored as plain text
- Access tokens are JWTs
- Refresh tokens are stored as hashed session records
- Protected routes validate bearer tokens in the API Gateway
- User-specific APIs should verify that token user ID matches requested user/account ownership
- Logout should be protected and revoke the authenticated user's session
- Role-based checks are available through middleware and should be used for admin-only APIs

## Portfolio highlights

This project demonstrates:

- Designing a backend from architecture to implementation
- Building a Go microservices monorepo
- REST-to-gRPC API Gateway pattern
- Shared database schema with service-level repository boundaries
- Authentication using JWT and refresh-token sessions
- Event-driven backend workflows with RabbitMQ
- Redis feed caching and cache invalidation
- Dockerised local development environment
- GitHub Actions CI and manual deployment workflow design
- Database migration workflow with `golang-migrate`
- API documentation through Swagger/OpenAPI
- High test coverage across service layers

## Roadmap

Planned improvements:

- Finalise auth hardening: admin-only registration, stricter role checks, and route ownership checks
- Add rate limiting at the API Gateway using Redis
- Complete full end-to-end integration tests
- Add deployment implementation to manual GitHub Actions workflows
- Add production-grade Docker image publishing
- Add observability: request IDs, metrics, traces, and structured audit logs
- Add frontend integration with React + Vite + TypeScript
