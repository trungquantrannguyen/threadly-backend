# Threadly Backend

A production-style backend for a Twitter/X-style social media platform built with Go, Gin, PostgreSQL, Redis, RabbitMQ, and a pragmatic microservice architecture.

## Tech Stack

- Go
- Gin
- Supabase PostgreSQL
- Redis
- RabbitMQ
- Docker Compose
- zerolog
- sqlc later
- gRPC later
- GitHub Actions later

## Architecture

The backend uses an API Gateway with multiple backend services:

- API Gateway
- User Service
- Content Service
- Feed Service
- Storage Service
- Notification Service

The project uses a shared PostgreSQL database to preserve relational integrity, joins, and foreign keys while keeping business logic separated by service.

## Run locally

```bash
cp .env.example .env
make docker-up
make run-gateway
```
