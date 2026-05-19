# Ayaka Backend Template

Ayaka is a backend API template developed in the Go programming language, adhering to the principles of Clean Architecture. This system is engineered to provide a scalable, secure, and modular foundation for enterprise-level application development.

## Technical Specifications

* **Language:** [Golang](https://go.dev/)
* **Web Framework:** [Go Fiber v2](https://gofiber.io/)
* **Database:** PostgreSQL
* **ORM:** [GORM](https://gorm.io/)
* **Configuration:** Viper + Godotenv (with Custom Environment Interpolation)
* **Validation:** Go-Playground Validator v10 (with Custom DB Rules)
* **Containerization:** [Docker](https://www.docker.com/) & Docker Compose (Multi-stage Alpine Build)
* **Continuous Integration:** [GitHub Actions](https://github.com/features/actions) (Automated Testing & Linting via SQLite)

## Core Functional Features

1. **Strict Layered Architecture:** Implementation of clear boundaries between the `core` layer (Business Logic) and the `adapter` layer (Infrastructure) to maintain framework-agnostic business rules.
2. **Dependency Injection:** Utilization of the Builder Pattern to decouple Handlers, Services, and Repositories, consolidated at the composition root (`builder.go`) for enhanced testability.
3. **Observability and Logging:** Integration of a structured JSON Logger compatible with Datadog/Elasticsearch and automated Request ID tracking for distributed tracing.
4. **Health Monitoring:** A dedicated `/health` endpoint for real-time system status reporting, suitable for Kubernetes and Docker Swarm orchestration.
5. **Schema Management:** Automated database migrations via Go structs (Code-First approach).
6. **Advanced Validation:** Custom database-aware validation tags (`unique`, `incolumn`) designed to enforce referential integrity and mitigate injection risks.
7. **Process Management:** Implementation of graceful shutdown procedures to ensure data integrity during server termination.

## 📂 Project Structure
```text
be-ayaka/
├── .github/               # GitHub Actions workflows
│   └── workflows/
│       └── ci.yml         # Automated CI pipeline script
├── cmd/                   # Application entry point (CLI commands, root.go, server.go)
├── config/                # Configuration setup (Viper, Godotenv)
├── internal/              # Private application codebase
│   ├── adapter/           # Infrastructure layer (Database connections, 3rd party APIs)
│   │   ├── database/      # Database initialization & driver setup (PostgreSQL)
│   │   ├── email/         # SMTP implementation for system notifications
│   │   └── repository/    # GORM implementations (Fulfills core repository contracts)
│   ├── bootstrap/         # The Wiring (Dependency Injection, App Init, Routes)
│   ├── core/              # Core Business Logic (The Holy Grail - Framework Agnostic)
│   │   ├── customerrors/  # Standardized Domain & Application Custom Errors
│   │   ├── entity/        # Business Rules: Pure Data Structs
│   │   ├── port/          # Contracts: Inbound (Service) & Outbound (Repository) Interfaces
│   │   └── service/       # Application Business Rules: Use Cases / Orchestrator
│   ├── delivery/          # Transport Mechanism (The Receptionist)
│   │   └── http/          # HTTP Handlers (Fiber controllers: health, user, auth)
│   ├── middleware/        # HTTP Interceptors (JWT Auth, Role checking, Request ID)
│   └── testingutils/      # Test fixtures, mock generators, and test suites helpers
├── logs/                  # Application runtime and error log files
├── pkg/                   # Reusable, domain-agnostic utilities (Hash, JWT, Logger, Validator)
├── .env.sample            # Local environment variables template
├── Dockerfile             # Multi-stage production Docker build recipe
├── docker-compose.yml     # Local multi-container orchestration (App + Postgres)
├── config.yaml            # Configuration mapping with environment variable interpolation
├── TESTING.md             # Guide for running unit tests, integration tests, and generating coverage
└── main.go                # Application root executing the cmd command processor
```

## Development Protocol

Developers are required to follow this sequence when implementing new features:

1. **Domain Entity:** Define your pure data structure in `internal/core/entity/`.
2. **Repository Contract:** Define the database contract in `internal/core/port/`.
3. **Infrastructure Adapter:** Implement the repository interface in `internal/adapter/repository/`.
4. **Service Layer:** Implement the business logic and use cases in `internal/core/service/`.
5. **Delivery Layer:** Implement transport-specific logic in `internal/delivery/http/`.
6. **Composition:** Wire all dependencies (Adapter -> Service -> Handler) together in `internal/bootstrap/builder.go`.
7. **Routing:** Register the endpoint in `internal/bootstrap/routes.go`.

### Pre-Push Checklist
Before committing (`git commit`) and pushing code, developers are encouraged to run the verification suite locally. **However, these checks are also automatically enforced by the GitHub Actions CI pipeline on every push and pull request to `main`, `master`, or `dev` branches.**
```bash
# 1. Format code according to official Go standards
go fmt ./...

# 2. Run static analysis to detect subtle logical errors
go vet ./...

# 3. Ensure all unit tests pass perfectly
go test ./...

# 4. Verify the application compiles cleanly without build errors
go build
```

## Usage Examples

### How to Use Custom Validators
This template comes with powerful, database-aware custom validation tags. You can easily apply them to your Data Transfer Object (DTO) structs using the `validate` tag.

**Example Struct:**
```go
package dto

type RegisterUserRequest struct {
    Name     string `json:"name" validate:"required,min=3"`
    
    // unique=table_name->column_name
    // Checks if the email already exists in the 'users' table
    Email    string `json:"email" validate:"required,email,unique=users->email"`
    
    // incolumn=table_name->column_name
    // Validates if the provided role_id actually exists in the 'roles' table (Foreign Key check)
    RoleID   string `json:"role_id" validate:"required,incolumn=roles->id"`

    // username: Enforces uppercase, lowercase, number, hyphens, or dots
    Username string `json:"username" validate:"required,username,whitespace"`

    // complexpassword: Enforces uppercase, lowercase, number, and special char (8-12 chars)
    // whitespace: Rejects any space characters
    Password string `json:"password" validate:"required,complexpassword,whitespace"`
}
```

### Available Custom Validation Tags:

* **`unique=[table_name]->[column_name]`**: Returns an error if the value already exists in the specified database table. Perfect for validating unique emails or usernames during registration.
* **`incolumn=[table_name]->[column_name]`**: Returns an error if the value does not exist in the specified database table. Extremely useful for validating Foreign Keys before inserting data.
* **`complexpassword`**: Enforces high-security password policies.
* **`whitespace`**: Ensures the input string contains no spaces.
* **`username`**: Ensure the input string contains only uppercase, lowercase, number, hyphens, or dots.
(Note: You can edit or add new validator on `pkg/validator/validator.go`)

## Getting Started (Local Development)

### 1. Database Setup
Ensure PostgreSQL is installed and running on your local machine. Create a new, empty database (e.g., `ayaka_db`).

### 2. Environment Configuration (.env)
Clone the .env.sample file to .env in the root directory and configure your credentials:
```env
# server
SERVER_PORT='8000'

# database configuration
DATABASE_HOST='localhost'
DATABASE_PORT='5432'
DATABASE_USER='postgres'
DATABASE_PASS='password'
DATABASE_NAME='ayaka_db'

# jwt configuration
JWT_SECRET='jwt secret code'
JWT_EXPIRED='24'

JWT_REFRESH_SECRET='REFRESH_SECRET_KEY'
JWT_REFRESH_EXPIRED='10080' # minutes
```
(Note: Never commit your .env file to a public repository!)

### 3. Install Dependencies
Run the following command to download all required Go modules:
``` bash
go mod tidy
```

### 4. Run the Test Suite
This template includes comprehensive unit test suites for the Repository, Service, and Handler layers. Run the following command to verify system integrity:
``` bash
go test -v ./...
```

### 5. Run the Server
Use the following command to start the server and trigger the auto-migration process:
``` bash
go run main.go svc
```
If the setup is successful, you will see the success log indicating a stable connection to PostgreSQL, followed by the AYAKA ASCII Art in your terminal.

Check the system status by accessing:
```
GET http://localhost:8000/health
```

## Getting Started (Docker Compose - Recommended)

If you don't want to install PostgreSQL or Go manually on your local machine, you can spin up the entire infrastructure (Ayaka Backend + PostgreSQL Database) in an isolated network with a single command:

### 1. Environment Configuration
Ensure your `.env` file is ready (copied from `.env.sample`). Docker Compose will automatically inject these variables into the container.

### 2. Up and Build
Run the following command in your terminal:
```bash
docker compose up --build
```
Docker Compose will automatically:

1. Initialize a localized PostgreSQL 16 Alpine container.

2. Build the lightweight Ayaka Go binary inside an Alpine container.

3. Handle time-zone matching (`Asia/Jakarta`) and bind ports automatically.

4. Once the ASCII Art appears, the system is ready at `GET http://localhost:8000/health`.

## Credits & Acknowledgements

* **Hanashiro Yuriku**
* **Gemini** - *AI Pair Programmer*
  Served as a technical consultant and coding assistant, providing architectural insights and collaborative debugging throughout the development journey.

  **Disclaimer:** *The names Ayaka used in this architectural blueprint are inspired by characters from Genshin Impact. This project is open-source, purely for personal/educational purposes, and is not affiliated with, endorsed, or sponsored by HoYoverse/Cognosphere.*