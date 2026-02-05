# JobTrackr API

A production-style backend API for tracking job applications, built with **Go**, **PostgreSQL**, and **Docker**.  
Includes database migrations, health checks, and full CRUD endpoints.

---

## Tech Stack
- Go (Golang)
- PostgreSQL
- Docker & Docker Compose

---

## Features
- API health + DB health endpoints
- Database migrations using SQL
- CRUD for job applications
- Filtering and partial updates (PATCH)
- Local dev workflow with Makefile

---

## Getting Started

### 1) Clone the repo
```bash
git clone https://github.com/battleship661/jobtrackr-api.git
cd jobtrackr-api

---

## Run Locally

### Start the database and run migrations
```bash
make db-up
make db-migrate
make db-tables

---

## API Endpoints

### Health
- `GET /health`
- `GET /health/db`

---

### Applications
- `POST /applications`
- `GET /applications`
- `GET /applications/{id}`
- `PATCH /applications/{id}`
- `DELETE /applications/{id}`

---

## Example Requests

```bash
curl http://localhost:8080/applications \
  -H "X-User-Id: <USER_UUID>"


---
