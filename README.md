## Packages Used (for `README.md`)

### Core Backend

* **`github.com/go-chi/chi/v5`**
  Lightweight HTTP router for building REST APIs using Go’s standard library.

* **`github.com/go-chi/chi/v5/middleware`**
  Built-in middleware (request ID, panic recovery, etc.).

---

### Database

* **`github.com/jackc/pgx/v5`**
  High-performance PostgreSQL driver for Go.

* **`github.com/jackc/pgx/v5/pgxpool`**
  PostgreSQL connection pooling for efficient DB usage.

* **`github.com/sqlc-dev/sqlc`**
  Generates type-safe Go code from SQL queries.

* **`github.com/golang-migrate/migrate/v4`**
  Database migration tool for managing schema changes.

---

### Background Jobs / Queue

* **`github.com/hibiken/asynq`**
  Redis-based background job processing system.

* **`github.com/redis/go-redis/v9`**
  Redis client used for caching and job queue backend.

---

### Configuration

* **`github.com/caarlos0/env/v11`**
  Loads environment variables into Go structs.

* **`github.com/joho/godotenv`**
  Loads `.env` files for local development.

---

### Logging

* **`github.com/rs/zerolog`**
  Fast structured JSON logging library.

---

### Utilities

* **`github.com/google/uuid`**
  Generates unique IDs (used for request tracking).

---

## Simple Project Explanation (for README)

### What This Project Is

A production-ready Go backend boilerplate built with:

* clean architecture
* PostgreSQL database
* Redis integration
* background job processing
* structured logging
* Docker-based local environment
* graceful shutdown handling

It provides a solid starting point for building scalable backend services.

---

### Key Features

* REST API using Go + Chi router
* PostgreSQL with connection pooling
* Redis integration for caching and background jobs
* SQL-first database access using sqlc
* Structured request logging with request tracing
* Environment-based configuration
* Health check endpoint
* Graceful server shutdown
* Docker setup for local development

---

### Architecture Overview

```
HTTP Request
   → Middleware (request id + logging)
   → Handler
   → Service (business logic)
   → Repository (database layer)
   → PostgreSQL / Redis
```

---

### Purpose

This project serves as:

* a backend learning template
