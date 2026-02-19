APP_NAME=app

# PID files for process management
API_PID=.api.pid
WORKER_PID=.worker.pid

# ---------- run ----------
api:
	go run ./cmd/api

worker:
	go run ./cmd/worker
# ---------- database ----------
migrate-up:
	migrate -path ./migrations -database $$DATABASE_URL up

migrate-down:
	migrate -path ./migrations -database $$DATABASE_URL down 1

migrate-create:
	migrate create -ext sql -dir ./migrations -seq $(name)

# ---------- sqlc ----------
sqlc:
	sqlc generate

# ---------- dev ----------
dev:
	docker compose up -d

dev-down:
	docker compose down
tidy:
	go mod tidy