# Jobber - Job Application Tracking Platform

A comprehensive platform for tracking job applications, managing interview stages, and organizing your job search.

---

## 📁 Project Structure

```
/Jobber/
├── be/                      # Backend (Go)
│   ├── cmd/                 # Application entry points
│   ├── internal/            # Internal packages (config, platform)
│   ├── modules/             # Business domains (applications, jobs, etc.)
│   ├── migrations/          # Database migrations
│   ├── docs/                # Swagger documentation
│   ├── go.mod               # Go dependencies
│   ├── Makefile             # Backend build commands
│   └── docker-compose.yml   # Infrastructure (PostgreSQL, Redis)
│
├── fe/                      # Frontend (React + TypeScript)
│   ├── src/                 # Source code
│   │   ├── pages/           # Page components
│   │   ├── features/        # Feature modules
│   │   ├── services/        # API services
│   │   └── shared/          # Shared utilities
│   ├── package.json         # Frontend dependencies
│   └── vite.config.ts       # Vite configuration
│
└── *.md                     # Project documentation
```

---

## 🚀 Quick Start

### Prerequisites

**For Local Development:**
- Go 1.21+
- Node.js 18+
- Docker & Docker Compose

**For Production Deployment:**
- Hetzner Cloud account
- Terraform >= 1.0
- SSH key pair

### 🖥️ Local Development

#### Backend Setup

```bash
cd be/

# Start infrastructure (PostgreSQL + Redis)
docker-compose up -d

# Install dependencies
go mod download

# Run migrations (automatic on first start)
# Or manually: make migrate-up

# Start backend server
make run

# Or with hot reload:
make dev
```

Backend runs at: `http://localhost:8080`

Swagger UI at: `http://localhost:8080/swagger/index.html`

#### Frontend Setup

```bash
cd fe/

# Install dependencies
npm install

# Start dev server
npm run dev
```

Frontend runs at: `http://localhost:5173`

### 🚀 Production Deployment (Hetzner Cloud)

Deploy the entire application to a Hetzner Cloud server with one command:

```bash
# 1. Setup infrastructure with Terraform
make terraform-init
make terraform-apply

# 2. Configure environment
make setup-env
# Edit .env with your production values

# 3. Deploy application
make deploy

# Your app is now live at http://<server-ip>
```

**Detailed deployment guide:** See [DEPLOYMENT.md](./DEPLOYMENT.md)

**Infrastructure details:** See [terraform/README.md](./terraform/README.md)

---

## 📚 Documentation

### Getting Started
- **[DEPLOYMENT.md](./DEPLOYMENT.md)** - 🚀 **Production deployment guide (Hetzner Cloud + Docker)**
- **[SYSTEM_SPECIFICATION.md](./SYSTEM_SPECIFICATION.md)** - Complete system architecture and feature documentation
- **[be/START_HERE.md](./be/START_HERE.md)** - Backend quick start guide
- **[be/SETUP.md](./be/SETUP.md)** - Detailed backend setup instructions

### Infrastructure
- **[terraform/README.md](./terraform/README.md)** - Terraform infrastructure documentation
- **[Makefile](./Makefile)** - All available commands and shortcuts

### Backend Guides
- **[be/SWAGGER_GUIDE.md](./be/SWAGGER_GUIDE.md)** - API documentation with Swagger
- **[be/PAGINATION_GUIDE.md](./be/PAGINATION_GUIDE.md)** - Pagination usage
- **[be/MIGRATIONS_GUIDE.md](./be/MIGRATIONS_GUIDE.md)** - Database migrations

### Architecture
- **[ARCHITECTURE_DECISIONS.md](./ARCHITECTURE_DECISIONS.md)** - Architecture Decision Records (ADRs)
- **[TECHNICAL_ROADMAP.md](./TECHNICAL_ROADMAP.md)** - Future evolution path
- **[PR_REVIEW_RESPONSE.md](./PR_REVIEW_RESPONSE.md)** - Architectural discussion

---

## 🏗️ Architecture

### Backend (Go + PostgreSQL + Redis)

**Architecture Style:** Modular Monolith with Hexagonal Architecture

**Structure:**
- `cmd/` - Application entry points
- `internal/` - Shared infrastructure (auth, database, HTTP, logging)
- `modules/` - Business domains with clean boundaries
  - Each module: `handler/` → `service/` → `repository/` → `model/`

**Principles:**
- Backend-first (business logic in backend)
- State vs History separation
- User-scoped data (multi-tenancy)
- Paginated responses

### Frontend (React + TypeScript + Vite)

**Architecture Style:** Feature-Sliced Design

**Structure:**
- `pages/` - Route components
- `features/` - Domain features (applications, jobs, etc.)
- `services/` - API client layer
- `shared/` - Reusable UI components and utilities

**Principles:**
- Thin presentation layer
- No business logic computation
- State management with Zustand
- Type-safe API calls

---

## 🛠️ Common Commands

### Development & Deployment (from project root)

```bash
# Local development
make dev              # Start all services locally (with hot reload)
make up               # Start production containers locally
make down             # Stop all services
make logs             # Follow logs
make ps               # Show running containers

# Terraform (infrastructure)
make terraform-init   # Initialize Terraform
make terraform-apply  # Create/update server
make terraform-destroy # Destroy server

# Deployment
make deploy           # Deploy to server (after terraform-apply)
make ssh              # SSH to server
make status           # Show system status

# Database
make db-shell         # Connect to PostgreSQL
make db-backup        # Backup database
make db-restore       # Restore database

# Utilities
make setup-env        # Create .env from example
make generate-secrets # Generate JWT secrets
make help             # Show all available commands
```

### Backend (run from `be/` directory)

```bash
make run          # Start server
make dev          # Start with hot reload
make build        # Build binary
make test         # Run tests
make swagger      # Generate Swagger docs
make docker-up    # Start PostgreSQL + Redis
make docker-down  # Stop infrastructure
```

### Frontend (run from `fe/` directory)

```bash
npm run dev       # Start dev server
npm run build     # Build for production
npm run preview   # Preview production build
npm run lint      # Run ESLint
```

---

## 📦 Tech Stack

### Backend
- **Language:** Go 1.21
- **Framework:** Gin (HTTP router)
- **Database:** PostgreSQL 15
- **Cache:** Redis 7
- **Auth:** JWT tokens
- **Docs:** Swagger/OpenAPI

### Frontend
- **Language:** TypeScript
- **Framework:** React 18
- **Build:** Vite
- **Styling:** Tailwind CSS
- **State:** Zustand
- **HTTP:** Axios
- **Routing:** React Router

---

## 🔐 Environment Variables

### Backend (`be/.env`)

```env
# Server
SERVER_PORT=8080
SERVER_ENV=development

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=jobber
DB_PASSWORD=jobber
DB_NAME=jobber

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT
JWT_ACCESS_SECRET=your-access-secret-here
JWT_REFRESH_SECRET=your-refresh-secret-here
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=7d
```

### Frontend (`fe/.env`)

```env
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

---

## 🧪 Testing

### Backend Tests

```bash
cd be/

# Run all tests
make test

# Run with coverage
make test-coverage
```

### Frontend Tests

```bash
cd fe/

# Run tests (when implemented)
npm run test
```

---

## 📊 API Documentation

**Interactive API Documentation (Swagger UI):**

1. Start backend: `cd be/ && make run`
2. Open browser: `http://localhost:8080/swagger/index.html`
3. Try endpoints directly in browser

**Complete API Reference:**
- See [SYSTEM_SPECIFICATION.md](./SYSTEM_SPECIFICATION.md#-appendix-a-complete-api-endpoint-reference)

---

## 🗂️ Key Features

- **Authentication** - JWT-based user registration and login
- **Company Management** - Track companies you're interested in
- **Job Management** - Save job postings
- **Resume Management** - Multiple resume versions
- **Application Tracking** - Core feature for tracking job applications
- **Stage Management** - Customizable interview stages
- **Comments** - Notes on applications and stages
- **Timeline** - Visual history of application progress
- **Reminders** - Schedule follow-ups (model ready, API pending)
- **Tags** - Categorize entities (model ready, API pending)

---

## 🚦 Development Workflow

### 1. Backend Development

```bash
cd be/

# Start infrastructure
docker-compose up -d

# Start backend with hot reload
make dev

# In another terminal: generate Swagger docs after changes
make swagger
```

### 2. Frontend Development

```bash
cd fe/

# Start frontend dev server
npm run dev

# Backend must be running at localhost:8080
```

### 3. Database Changes

```bash
cd be/

# Create new migration
make migrate-create name=add_new_feature

# Edit migration files in migrations/
# Then run:
make migrate-up
```

---

## 📝 Contributing

1. Create feature branch
2. Make changes
3. Update documentation if needed
4. Test locally
5. Create PR

---

## 📄 License

MIT License - See LICENSE file for details

---

## 🔗 Links

- **System Specification:** [SYSTEM_SPECIFICATION.md](./SYSTEM_SPECIFICATION.md)
- **Backend Docs:** [be/START_HERE.md](./be/START_HERE.md)
- **Architecture Decisions:** [ARCHITECTURE_DECISIONS.md](./ARCHITECTURE_DECISIONS.md)
- **Technical Roadmap:** [TECHNICAL_ROADMAP.md](./TECHNICAL_ROADMAP.md)

---

**Built with ❤️ for job seekers**
