# Jobber - Job Application Tracking Backend

A modular monolith backend for tracking job applications, built with Go.

## Architecture

This is a **modular monolith** designed with clear module boundaries for future microservice migration. Each module is self-contained with its own:
- Domain models
- Repository layer
- Service layer (use cases)
- HTTP handlers
- Port interfaces for inter-module communication

## Technology Stack

- **Language**: Go 1.25
- **HTTP Framework**: Gin
- **Database**: PostgreSQL 15 (pgx driver)
- **SQL Generator**: sqlc
- **Migrations**: golang-migrate
- **Cache/Queue**: Redis 7
- **Auth**: JWT (access + refresh tokens)
- **Logging**: zap (structured JSON)
- **Storage**: S3-compatible (Hetzner Object Storage)
- **AI**: Anthropic Claude SDK (job import parsing, match score)
- **Calendar**: Google Calendar v3 OAuth2
- **Payments**: Paddle (subscription webhooks)

## Project Structure

```
jobber/
├── cmd/
│   ├── api/              # Application entry point
│   └── seed/             # Database seeding for development
├── internal/
│   ├── config/           # Configuration management
│   └── platform/         # Shared infrastructure
│       ├── logger/       # Structured logging (zap)
│       ├── postgres/     # Database client & connection pool
│       ├── redis/        # Redis client
│       ├── auth/         # JWT utilities
│       ├── http/         # HTTP utilities & middleware
│       ├── ai/           # Anthropic Claude SDK wrapper
│       └── storage/      # S3 file storage client
├── modules/
│   ├── auth/             # Authentication (register, login, JWT)
│   ├── users/            # User profile management
│   ├── applications/     # Core: Application aggregate
│   ├── jobs/             # Job postings + Kanban board columns
│   ├── companies/        # Company management + derived stats
│   ├── resumes/          # Resume versions + S3 file storage
│   ├── comments/         # Comments on applications/stages
│   ├── analytics/        # Dashboard statistics
│   ├── calendar/         # Google Calendar OAuth2 integration
│   ├── jobimport/        # Import jobs by URL (JSON-LD + Claude AI)
│   ├── matchscore/       # AI resume-to-job matching + cache
│   ├── subscriptions/    # Plans, billing, Paddle webhooks
│   ├── reminders/        # Reminder system (model + repository)
│   └── tags/             # Tagging system (model + repository)
├── migrations/           # Database schema (golang-migrate)
└── docs/                 # Swagger/OpenAPI spec
```

## Getting Started

### Prerequisites

- Go 1.25+
- PostgreSQL 15+
- Redis 7+
- golang-migrate CLI
- sqlc CLI

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd jobber

# Copy environment file
cp .env.example .env

# Edit .env with your configuration
# Update JWT secrets and database credentials

# Install dependencies
go mod download

# Start infrastructure (PostgreSQL + Redis)
docker-compose up -d

# Start the server (migrations run automatically!)
make run
```

**🎉 New Feature:** Database migrations now run automatically on server startup!

### Development

```bash
# Run in development mode
make dev

# Run tests
make test

# Run linter
make lint

# Generate sqlc code
make sqlc

# Create a new migration
make migrate-create name=migration_name

# Apply migrations
make migrate-up

# Rollback migrations
make migrate-down
```

## API Endpoints

Base URL: `http://localhost:8080/api/v1`

**📖 Interactive API Documentation:** Swagger UI available at `http://localhost:8080/swagger/index.html`

**📄 Pagination:** All list endpoints support pagination with `limit` (max 500) and `offset` parameters.

### Health Check
- `GET /health` - Health status of all services

### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login user
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/logout` - Logout user

### Applications (Core)
- `POST /api/v1/applications` - Create application
- `GET /api/v1/applications` - List applications
- `GET /api/v1/applications/:id` - Get application details
- `PATCH /api/v1/applications/:id` - Update application
- `DELETE /api/v1/applications/:id` - Delete application
- `GET /api/v1/applications/:id/timeline` - Get application timeline

### More endpoints documented in API documentation...

## Error Handling

All errors follow a standard format:

```json
{
  "error_code": "APPLICATION_NOT_FOUND",
  "error_message": "Application not found"
}
```

Error codes are stable and machine-readable, suitable for client-side logic.

## Domain Model

### Core Entities

- **User**: Platform user
- **Application**: Core aggregate - represents a job application
- **Job**: Job posting details (with board_column for Kanban pipeline stages)
- **Company**: Company information
- **Resume**: User's resume versions
- **ApplicationStage**: Stages in application lifecycle (append-only)
- **StageTemplate**: Reusable stage definitions
- **Comment**: Notes on applications/stages
- **Reminder**: Scheduled reminders
- **Tag**: Flexible tagging system

### Key Principles

1. **Application is the central aggregate** - all other entities support it
2. **User-scoped data** - all data belongs to a user
3. **Append-only stages** - full history preservation
4. **Explicit state management** - no hidden state transitions
5. **Timeline is a projection** - built from events, not stored

## Development Principles

- **Explicit over implicit**: No magic, predictable behavior
- **Module independence**: Modules communicate via ports
- **Transaction boundaries**: One use case = one transaction
- **Context propagation**: Context flows through all layers
- **Structured logging**: All requests logged with request_id
- **No sensitive data in logs**: Never log passwords, tokens, or PII

## Testing

```bash
# Run all tests
make test

# Run unit tests
make test-unit

# Run integration tests
make test-integration

# Run with coverage
make test-coverage
```

## Deployment

See the root [Makefile](../Makefile) and [terraform/](../terraform/) for production deployment.

## License

MIT
