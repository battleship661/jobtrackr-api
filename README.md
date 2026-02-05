# JobTrackr API
A job applications tracker API (Go + Postgres + Docker).
# jobtrackr-api
# JobTrackr API

A production-style backend REST API to track job applications with secure authentication,
filtering, caching, and background jobs.

## Tech Stack
- Go (Golang)
- PostgreSQL
- Redis
- Docker & Docker Compose
- JWT Authentication
- GitHub Actions (CI)

## Features
- User registration & login (JWT)
- Create, update, and track job applications
- Filter by status, company, date
- Background jobs for follow-up reminders
- Redis caching for faster reads
- Containerized local development

## Getting Started
```bash
git clone https://github.com/battleship661/jobtrackr-api.git
cd jobtrackr-api
docker compose up -d
- Database health check endpoint (`GET /health/db`)
### Health Checks
```bash
curl http://localhost:8080/health
curl http://localhost:8080/health/db
## Database Schema
Tables:
- users
- applications
- application_events

### Run migrations
```bash
make db-up
make db-migrate
## Endpoints

### Health
- `GET /health`
- `GET /health/db`

### Applications (requires `X-User-Id` header for now)
- `POST /applications`
- `GET /applications?status=&company=`
- `GET /applications/{id}`
- `PATCH /applications/{id}`
- `DELETE /applications/{id}`

## Run locally

### 1) Start database
```bash
make db-up
make db-migrate
make db-tables
