# 💰 Wallet Ledger Service

[![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)](https://golang.org/)
[![API Version](https://img.shields.io/badge/API-v1.1.0-green.svg)](api/apispec.json)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](#license)

Backend API for a digital wallet and financial transaction system. Supports wallet management, fund transfers, double-entry ledger tracking, audit logging, and asynchronous event processing via RabbitMQ. Includes Admin endpoints for transaction monitoring and audit review.

---

## 📖 Table of Contents

- [Features](#-features)
- [Tech Stack](#-tech-stack)
- [Architecture](#-architecture)
- [Project Structure](#-project-structure)
- [Database Schema](#-database-schema)
- [Prerequisites](#-prerequisites)
- [Getting Started](#-getting-started)
  - [Option A: Docker Compose (Recommended)](#option-a-docker-compose-recommended)
  - [Option B: Local Development](#option-b-local-development)
- [Configuration](#-configuration)
- [Build & Run](#-build--run)
- [Testing](#-testing)
- [API Reference](#-api-reference)
- [Deployment](#-deployment)
- [Makefile Commands](#-makefile-commands)
- [License & Contact](#-license--contact)

---

## ✨ Features

- **User Authentication**: Secure user registration and login with roles (`USER` or `ADMIN`). Authentication tokens (JWT) are stored securely in `httpOnly` cookies.
- **Automatic Wallet Creation**: A wallet is automatically provisioned for each new user with `IDR` currency and zero balance.
- **Wallet Balances**: Retrieve the authenticated user's wallet details and real-time balance.
- **Top-up Transactions**: Simulate adding funds to the authenticated user's wallet.
- **P2P Transfer**: Transfer funds from the authenticated user's wallet to another user's wallet, with validations for sufficient balance and wallet existence.
- **Double-Entry Ledger**: Every successful financial transaction generates immutable debit/credit ledger entries for auditing and ledger integrity.
- **Transaction History**: Retrieve a paginated and type-filtered (e.g. `TOPUP`, `TRANSFER`) transaction history.
- **Transaction Detail**: View a transaction's comprehensive details along with its associated ledger entries.
- **Asynchronous Audit Logging**: All user activities (login, logout, transactions, etc.) are captured asynchronously using RabbitMQ to keep the request-response cycle fast.
- **Idempotency Control**: High-risk financial operations (`/wallets/topup`, `/transfers`) are protected against accidental double-submissions via the `Idempotency-Key` header and Redis storage.
- **Rate Limiting**: Protects API endpoints against DDoS attacks and brute forcing (Redis-based sliding window rate-limiter set to 100 requests per minute).
- **CORS Protection**: Safe cross-origin resource sharing configured.
- **Role-Based Access Control (RBAC)**: Distinct authorization paths for `USER` and `ADMIN` roles.
- **Admin Dashboard Endpoints**:
  - Retrieve all registered users.
  - Monitor all system-wide transactions.
  - Review all system-wide audit logs.
- **Automated Database Migrations**: Uses `golang-migrate` to manage database schema updates.
- **Database Seeder**: Pre-populate the database with a default Admin account for testing.
- **Asynchronous Workers**: Independent consumers processing queues for Audit, Notification, and Analytics via RabbitMQ.
- **Graceful Shutdown**: Stops the HTTP server and workers safely, closing database pool connections without interrupting ongoing requests.

---

## 🛠️ Tech Stack

| Category | Technology | Description |
|---|---|---|
| **Language** | Go 1.25 | Fast, compiled, statically typed language. |
| **HTTP Framework** | [Gin Gonic v1.12.0](https://github.com/gin-gonic/gin) | High-performance, lightweight HTTP router. |
| **Database** | PostgreSQL 17 | Relational database to ensure transactional integrity. |
| **Caching & Rate Limiting** | Redis 9 (v9.20.0 client) | In-memory store for idempotency keys and rate limit counters. |
| **Message Broker** | RabbitMQ 4 | Message queue for asynchronous background jobs. |
| **ORM / Query Generator** | [sqlc](https://sqlc.dev/) | Compiles raw SQL queries into type-safe Go code. |
| **Database Driver** | pgx/v5 | PostgreSQL driver and toolkit for Go. |
| **Migrations** | [golang-migrate v4](https://github.com/golang-migrate/migrate) | Database migrations CLI and library. |
| **Authentication** | JWT (v5.3.1) | Signed JSON Web Tokens for authentication. |
| **Config Management** | Viper & godotenv | Robust configuration and environment management. |
| **Precision Math** | shopspring/decimal | High-precision decimal arithmetic for currency. |
| **Testing** | testify & mockery | Mocking framework and assertion toolkit. |
| **Containerization** | Docker & Docker Compose | Multi-stage Docker builds and full-stack container orchestration. |

---

## 🏗️ Architecture

The project is designed using clean, layered, and idiomatic Go architecture. It separates concerns across distinct layers to maintain high testability and maintainability.

```
                  ┌──────────────────────┐
                  │        Client        │
                  └──────────┬───────────┘
                             │ (HTTP Requests)
                             ▼
┌────────────────────────────────────────────────────────┐
│                   Gin Router Middleware                │
│ (CORS → Recovery → Logger → RateLimit → JWTAuth ...)   │
└────────────────────────────┬───────────────────────────┘
                             │
                             ▼
                  ┌──────────────────────┐
                  │       Handlers       │
                  └──────────┬───────────┘
                             │ (DTOs)
                             ▼
                  ┌──────────────────────┐
                  │       Services       │  ◄────────┐
                  └──────────┬───────────┘           │
                             │                       │
           ┌─────────────────┴─────────────────┐     │ (AMQP Messages)
           │ (SQL Transactions)                │     │
           ▼                                   ▼     │
┌────────────────────┐               ┌────────────────────┐
│    Repositories    │               │  RabbitMQ Channel  │
└──────────┬─────────┘               └─────────┬──────────┘
           │                                   │
           ▼ (pgx pool)                        ▼
┌────────────────────┐               ┌────────────────────┐
│     PostgreSQL     │               │ Background Workers │
└────────────────────┘               │ (Audit, Notif, etc)│
                                     └────────────────────┘
```

### Components

1. **Handlers (Delivery Layer)**: Parses incoming JSON requests, validates headers and request bodies using DTO validators, calls services, and formats JSON responses.
2. **Services (Business Logic)**: Houses core application logic, handles transactions across database operations, and publishes asynchronous events to RabbitMQ.
3. **Repositories (Data Access)**: Executes type-safe database queries generated by `sqlc` to interact with PostgreSQL.
4. **Middleware**:
   - **Rate Limiting**: Checks request count in Redis using a sliding window.
   - **JWT Auth**: Verifies the `accessToken` cookie and extracts claims into the request context.
   - **Audit Logger**: Extracts action data and pushes it to RabbitMQ.
   - **Idempotency**: Inspects `Idempotency-Key` headers, storing successful response payloads in Redis to prevent duplicated operations.
5. **Double-Entry Ledger Pattern**: To ensure absolute financial integrity:
   - A **TOPUP** creates one transaction record (`COMPLETED`) and a single `CREDIT` ledger entry in the destination wallet.
   - A **TRANSFER** creates one transaction record (`COMPLETED`) and two ledger entries: a `DEBIT` on the source wallet and a `CREDIT` on the target wallet.
   - All balance updates and ledger creations are executed inside strict PostgreSQL database transactions (`SERIALIZABLE` or select `FOR UPDATE` locks) to prevent race conditions.

---

## 📁 Project Structure

```
wallet-ledger-service/
├── api/
│   └── apispec.json            # OpenAPI 3.1 schema specification
├── bin/                        # Compiled executable binaries (git-ignored)
├── cmd/
│   ├── seed/                   # Database seeder CLI (seeds Admin user)
│   │   └── main.go
│   └── server/                 # Main HTTP server entry point
│       └── main.go
├── database/
│   ├── migrations/             # SQL migration files (.up.sql / .down.sql)
│   └── queries/                # Raw SQL queries compiled by sqlc
├── internal/
│   ├── config/                 # Config loader and client builders (DB, Redis, MQ)
│   ├── dto/                    # Request/Response Data Transfer Objects & custom validators
│   ├── handler/                # HTTP controllers (Gin route handlers)
│   ├── middleware/             # CORS, JWT, rate-limiting, audit, idempotency middlewares
│   ├── mocks/                  # Auto-generated unit test mocks (git-ignored)
│   ├── model/                  # Domain entity models (User, Wallet, Transaction, etc.)
│   ├── repository/             # Data repository interfaces and sqlc wrappers
│   │   └── sqlc/               # Auto-generated code from sqlc schema and queries
│   ├── service/                # Business logic services
│   ├── utils/                  # Helper utilities (JWT, password, custom ID generators, responses)
│   └── worker/                 # RabbitMQ background consumers
├── .env.example                # Sample environment variables configuration template
├── Dockerfile                  # Multi-stage production deployment Dockerfile
├── docker-compose.yml          # Container orchestration manifest
├── go.mod                      # Dependency manifest
├── go.sum                      # Dependency lockfile
├── Makefile                    # Task runner shortcuts
└── sqlc.yaml                   # sqlc configuration file
```

---

## 🗄️ Database Schema

The service relies on a highly index-optimized relational schema in PostgreSQL.

```mermaid
erDiagram
    users {
        uuid id PK
        varchar email UK
        varchar password_hash
        varchar role "USER | ADMIN"
        timestamp_with_tz created_at
        timestamp_with_tz updated_at
    }

    wallets {
        varchar id PK "WLT-XXXXXXXXXX"
        uuid user_id FK, UK
        decimal balance "CHECK balance >= 0"
        varchar currency
        timestamp_with_tz created_at
        timestamp_with_tz updated_at
    }

    transactions {
        uuid id PK
        varchar reference_no UK "TXN-YYYYMMDD-XXXXXXXX"
        varchar type "TOPUP | TRANSFER"
        varchar status "PENDING | COMPLETED | FAILED"
        decimal amount "CHECK amount > 0"
        varchar source_wallet_id FK "nullable"
        varchar target_wallet_id FK
        timestamp_with_tz created_at
    }

    ledger_entries {
        varchar id PK "LED-XXXXXXXXXX"
        uuid transaction_id FK
        varchar wallet_id FK
        varchar entry_type "DEBIT | CREDIT"
        decimal amount "CHECK amount > 0"
        timestamp_with_tz created_at
    }

    audit_logs {
        varchar id PK "AUD-XXXXXXXXXX"
        uuid user_id FK
        varchar action
        varchar ip_address
        varchar endpoint
        timestamp_with_tz created_at
    }

    users ||--o| wallets : "has one"
    users ||--o{ audit_logs : "triggers"
    wallets ||--o{ transactions : "as source"
    wallets ||--o{ transactions : "as target"
    transactions ||--o{ ledger_entries : "generates"
    wallets ||--o{ ledger_entries : "records"
```

---

## 📋 Prerequisites

To run this application, make sure you have the following installed:

- **Go**: Version `1.25` or higher ([Download](https://golang.org/dl/))
- **Docker**: Version `20.10+` and **Docker Compose**
- **golang-migrate CLI** (optional, for manual database migrations): `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
- **sqlc** (optional, for rebuilding database queries): `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- **mockery** (optional, for generating mock files): `go install github.com/vektra/mockery/v2@latest`

---

## 🚀 Getting Started

### Option A: Docker Compose (Recommended)

Running the service via Docker Compose sets up the entire application stack including PostgreSQL, Redis, RabbitMQ, and the application itself.

1. **Clone the repository**:
   ```bash
   git clone https://github.com/marcceljanara/wallet-ledger-service.git
   cd wallet-ledger-service
   ```

2. **Configure the environment**:
   ```bash
   cp .env.example .env
   ```

3. **Start the containers**:
   ```bash
   docker compose up --build -d
   ```

4. **Verify the services**:
   Check if the API is running on `http://localhost:8080`. You can monitor logs via:
   ```bash
   docker compose logs -f app
   ```

5. **Seed the database (Optional)**:
   Add a default administrator account:
   ```bash
   docker compose exec app /server seed
   ```

---

### Option B: Local Development

If you prefer to run database and message brokers locally (or inside Docker) but compile the Go app on your local machine:

1. **Clone the repository**:
   ```bash
   git clone https://github.com/marcceljanara/wallet-ledger-service.git
   cd wallet-ledger-service
   ```

2. **Configure local environment variables**:
   Create a `.env` file from the example:
   ```bash
   cp .env.example .env
   ```
   Modify `.env` to point to your local PostgreSQL, Redis, and RabbitMQ:
   ```env
   DATABASE_URL=postgres://wallet_user:wallet_pass@localhost:5432/wallet_db?sslmode=disable
   REDIS_URL=redis://localhost:6379/0
   RABBITMQ_URL=amqp://guest:guest@localhost:5672/
   ```

3. **Install dependencies**:
   ```bash
   go mod download
   ```

4. **Run migrations**:
   Apply SQL schema migrations:
   ```bash
   make migrate-up
   ```

5. **Seed Admin user**:
   Create the default admin credentials (`admin@example.com` / `admin123`):
   ```bash
   make seed
   ```

6. **Start the server**:
   ```bash
   make run
   ```

---

## ⚙️ Configuration

The application is configured using environment variables (loaded via `.env` or system environment via Viper).

| Variable | Description | Default | Required |
|---|---|---|---|
| `SERVER_PORT` | Port the HTTP server will listen on. | `8080` | No |
| `DATABASE_URL` | PostgreSQL connection string (DSN). | — | **Yes** |
| `REDIS_URL` | Redis server address URL. | — | **Yes** |
| `RABBITMQ_URL` | RabbitMQ connection string URL. | — | **Yes** |
| `JWT_SECRET` | Secret key used to sign and verify JWT authentication tokens. | — | **Yes** |
| `JWT_EXPIRATION` | Validity duration for authentication tokens. | `24h` | No |
| `COOKIE_SECURE` | If `true`, the `accessToken` cookie will only be transmitted via HTTPS. | `true` | No |

---

## 📦 Build & Run

To build the Go binary locally:

```bash
# Compile and build the binary in the bin/ directory
make build

# Run the compiled binary
./bin/server
```

To build a standalone production Docker image:

```bash
docker build -t wallet-ledger-service:latest .
```

---

## 🧪 Testing

The codebase includes extensive unit tests with mocked repositories and services.

```bash
# Run all unit tests with verbosity and coverage report
make test
```

### Mock Generation

If interfaces in the repository or service layers are modified, regenerate mocks using:

```bash
make mockery
```

---

## 🔌 API Reference

Authentication is session-based. On successful login, the server returns an `httpOnly`, `SameSite=Strict` cookie named `accessToken` which must be included in subsequent requests.

### Public Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/auth/register` | Register a user (automatically spawns a wallet) |
| `POST` | `/api/v1/auth/login` | Log in and receive JWT `accessToken` in cookie |

### Protected Endpoints (User Role)

Requires a valid `accessToken` cookie.

| Method | Endpoint | Headers | Description |
|---|---|---|---|
| `POST` | `/api/v1/auth/logout` | — | Clears token cookie |
| `GET` | `/api/v1/wallets/me` | — | Retrieve current user's wallet and balance |
| `POST` | `/api/v1/wallets/topup` | `Idempotency-Key` | Simulated top-up of funds |
| `POST` | `/api/v1/transfers` | `Idempotency-Key` | Transfer money to another wallet |
| `GET` | `/api/v1/transactions` | — | Paginated transaction logs (query params: `page`, `limit`, `type`) |
| `GET` | `/api/v1/transactions/:transactionId` | — | View details of a specific transaction + ledger records |
| `GET` | `/api/v1/ledger/entries` | — | Paginated ledger history (query params: `page`, `limit`, `entry_type`) |
| `GET` | `/api/v1/audit/logs` | — | Paginated audit trails of user's own actions |

### Admin Endpoints

Requires valid `accessToken` with `role: ADMIN`.

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/admin/users` | List all registered users |
| `GET` | `/api/v1/admin/transactions` | Monitor all transactions across the entire system |
| `GET` | `/api/v1/admin/transactions/:transactionId` | Inspect detail of any transaction in the system |
| `GET` | `/api/v1/admin/audit-logs` | Retrieve all audit logs for all users |

For full details on request payloads, query parameters, headers, and responses, please check the complete OpenAPI 3.1 schema definition file at [api/apispec.json](api/apispec.json).

---

## 🐳 Deployment

The Docker Compose orchestration deploys four distinct containers:

1. **`app`**: Built from the Go application code using a multi-stage Dockerfile. On startup, it runs database migrations before listening on port `8080`.
2. **`postgres`**: Runs PostgreSQL 17 to host relational transaction tables. Utilizes docker-volumes (`pgdata`) for data persistence.
3. **`redis`**: Runs Redis 7. Required to store rate-limiting window counters and response hashes for API idempotency.
4. **`rabbitmq`**: Runs RabbitMQ 4 with Management Plugins. Available interfaces:
   - AMQP Port: `5672`
   - Management Web UI: `http://localhost:15672` (default credentials: `guest` / `guest`)

### Production Considerations

Before pushing this system to production, make sure to:
- Change the `JWT_SECRET` in `.env` to a secure, randomly generated hash.
- Keep `COOKIE_SECURE=true` so session tokens are only sent over HTTPS.
- Scale workers horizontally depending on RabbitMQ queue depth.
- Configure resource limits and logging metrics inside cloud environments.

---

## ⚙️ Makefile Commands

A `Makefile` is provided in the root directory for fast commands execution.

| Command | Action | Description |
|---|---|---|
| `make run` | `go run ./cmd/server` | Starts the Go server locally. |
| `make seed` | `go run ./cmd/seed` | Seeds database with a default Admin user. |
| `make build` | `go build -o bin/server ./cmd/server` | Compiles Go binary. |
| `make test` | `go test ./... -v -cover` | Executes tests with code coverage check. |
| `make sqlc` | `sqlc generate` | Generates Go database access queries from SQL. |
| `make mockery` | `mockery --all ...` | Generates mocks for testing. |
| `make migrate-up` | `migrate ... up` | Runs database migrations forward. |
| `make migrate-down`| `migrate ... down` | Rolls back database migrations. |

---

## 📄 License & Contact

### License
This project is licensed under the [MIT License](LICENSE) - see the LICENSE file for details.

### Contact
Project Creator & Maintainer: **Marccel Janara**
Repository Link: [https://github.com/marcceljanara/wallet-ledger-service](https://github.com/marcceljanara/wallet-ledger-service)
