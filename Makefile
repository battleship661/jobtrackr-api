.PHONY: db-up db-down db-migrate db-psql run

db-up:
	docker compose up -d

db-down:
	docker compose down

db-migrate:
	docker exec -i jobtrackr_postgres psql -U jobtrackr -d jobtrackr < db/migrations/001_init.sql

db-psql:
	docker exec -it jobtrackr_postgres psql -U jobtrackr -d jobtrackr
db-tables:
	docker exec -it jobtrackr_postgres psql -U jobtrackr -d jobtrackr -c "\dt"

run:
	go run ./cmd/api

