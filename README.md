# Jobber - Job Application Tracking Platform

A comprehensive platform for tracking job applications, managing interview stages, and organizing your job search.

---

## 📁 Project Structure

```
/Jobber/
├── be/                      # Backend (Go modular monolith)
│   ├── cmd/                 # Entry points (api, seed)
│   ├── internal/            # Shared infrastructure (config, auth, db, redis, ai, s3)
│   ├── modules/             # 12 business domains
│   │   ├── auth/            # Authentication (register, login, JWT)
│   │   ├── users/           # User profiles
│   │   ├── applications/    # Core: application tracking
│   │   ├── jobs/            # Job postings + Kanban board
│   │   ├── companies/       # Company management + stats
│   │   ├── resumes/         # Resume versions + S3 storage
│   │   ├── comments/        # Notes on applications/stages
│   │   ├── analytics/       # Dashboard statistics
│   │   ├── calendar/        # Google Calendar integration
│   │   ├── jobimport/       # Import jobs by URL (JSON-LD + AI)
│   │   ├── matchscore/      # AI resume-to-job matching
│   │   └── subscriptions/   # Plans + Paddle webhooks
│   ├── migrations/          # Database migrations (golang-migrate)
│   ├── docs/                # Swagger/OpenAPI documentation
│   └── Makefile             # Backend build commands
│
├── fe/                      # Frontend (React + TypeScript)
│   ├── src/
│   │   ├── pages/           # Route-level pages (15 pages)
│   │   ├── features/        # Domain feature modules
│   │   ├── services/        # API client layer (ky)
│   │   ├── stores/          # Zustand state stores
│   │   ├── shared/          # UI components, i18n, utilities
│   │   └── entities/        # Domain entity types
│   ├── package.json         # Frontend dependencies
│   └── vite.config.ts       # Vite configuration
│
├── ext/                     # Chrome Extension (early stage)
├── terraform/               # Infrastructure as Code (Hetzner Cloud)
├── docker-compose.yml       # Production: PostgreSQL + Redis + Backend + Frontend + Caddy
├── Caddyfile                # Reverse proxy config (automatic HTTPS)
├── Makefile                 # Root-level commands (deploy, terraform, db)
└── SYSTEM_SPECIFICATION.md  # Complete system architecture reference
```

---

## 🚀 Quick Start

### Prerequisites

**For Local Development:**
- Go 1.25+
- Node.js 20+
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

**Infrastructure details:** See [terraform/README.md](./terraform/README.md)

---

## 📚 Documentation

- **[SYSTEM_SPECIFICATION.md](./SYSTEM_SPECIFICATION.md)** - Complete system architecture and feature documentation
- **[be/README.md](./be/README.md)** - Backend architecture, API endpoints, development guide
- **[terraform/README.md](./terraform/README.md)** - Terraform infrastructure documentation
- **[Makefile](./Makefile)** - All available deployment and database commands

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
- **Language:** Go 1.25
- **Framework:** Gin (HTTP router)
- **Database:** PostgreSQL 15
- **Cache:** Redis 7
- **Auth:** JWT tokens (access + refresh)
- **SQL Generator:** sqlc
- **Docs:** Swagger/OpenAPI
- **Storage:** S3 (Hetzner Object Storage)
- **AI:** Anthropic Claude (job import, match score)
- **Calendar:** Google Calendar v3 OAuth2

### Frontend
- **Language:** TypeScript (strict mode)
- **Framework:** React 19
- **Build:** Vite 7
- **Styling:** Tailwind CSS 3
- **State:** Zustand 5 (client) + TanStack React Query 5 (server)
- **HTTP:** ky
- **Drag & Drop:** @dnd-kit
- **Routing:** React Router v7
- **i18n:** i18next (EN, RU, UA)

---

## 🔐 Environment Variables

### Backend (`be/.env`)

| Variable | Required | Description | Default |
|----------|----------|-------------|---------|
| `SERVER_PORT` | No | HTTP server port | `8080` |
| `SERVER_ENV` | No | Environment (`development` / `production`) | `development` |
| `DB_HOST` | Yes | PostgreSQL host | `localhost` |
| `DB_PORT` | Yes | PostgreSQL port | `5432` |
| `DB_USER` | Yes | PostgreSQL user | `jobber` |
| `DB_PASSWORD` | Yes | PostgreSQL password | `jobber` |
| `DB_NAME` | Yes | PostgreSQL database name | `jobber` |
| `DB_SSL_MODE` | No | PostgreSQL SSL mode | `disable` |
| `DB_MAX_CONNS` | No | Max open connections | `25` |
| `DB_MAX_IDLE_CONNS` | No | Max idle connections | `5` |
| `DB_CONN_MAX_LIFETIME` | No | Connection max lifetime | `5m` |
| `REDIS_HOST` | Yes | Redis host | `localhost` |
| `REDIS_PORT` | Yes | Redis port | `6379` |
| `REDIS_PASSWORD` | No | Redis password | _(empty)_ |
| `REDIS_DB` | No | Redis database number | `0` |
| `JWT_ACCESS_SECRET` | **Yes** | JWT access token signing key | — |
| `JWT_REFRESH_SECRET` | **Yes** | JWT refresh token signing key | — |
| `JWT_ACCESS_EXPIRY` | No | Access token TTL | `15m` |
| `JWT_REFRESH_EXPIRY` | No | Refresh token TTL | `168h` |
| `ALLOWED_ORIGINS` | No | CORS origins (comma-separated, `*` in dev) | `*` |
| `LOG_LEVEL` | No | Log level (`debug`, `info`, `warn`, `error`) | `debug` |
| `LOG_FORMAT` | No | Log format (`json` / `text`) | `json` |
| `S3_ENDPOINT` | **Yes** | S3-compatible storage endpoint | — |
| `S3_BUCKET` | **Yes** | S3 bucket name | — |
| `S3_REGION` | No | S3 region | `eu-central` |
| `S3_ACCESS_KEY` | **Yes** | S3 access key | — |
| `S3_SECRET_KEY` | **Yes** | S3 secret key | — |
| `ANTHROPIC_API_KEY` | No | Anthropic API key (enables AI features) | _(empty)_ |
| `GOOGLE_CALENDAR_CLIENT_ID` | No | Google OAuth client ID | _(empty)_ |
| `GOOGLE_CALENDAR_CLIENT_SECRET` | No | Google OAuth client secret | _(empty)_ |
| `GOOGLE_CALENDAR_REDIRECT_URL` | No | Google OAuth redirect URL | _(empty)_ |
| `GOOGLE_CALENDAR_FRONTEND_URL` | No | Frontend URL for Calendar callback | _(empty)_ |
| `GOOGLE_CALENDAR_TOKEN_ENCRYPTION_KEY` | No | Encryption key for stored OAuth tokens | _(empty)_ |

### Frontend (`fe/.env`)

| Variable | Required | Description | Default |
|----------|----------|-------------|---------|
| `VITE_API_BASE_URL` | Yes | Backend API URL | `http://localhost:8080/api/v1` |

---

## 🧪 Testing

### Backend Tests

```bash
cd be/

# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests only
make test-integration

# Run with coverage report
make test-coverage
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

- **Authentication** - JWT-based registration/login with access + refresh tokens
- **Company Management** - Track companies with derived stats (active apps, last activity)
- **Job Management** - Save job postings with Kanban board (drag-and-drop) and grid views
- **Resume Management** - Multiple resume versions with S3 file storage
- **Application Tracking** - Core feature: link job + resume, track through interview pipeline
- **Stage Templates** - Customizable interview stage definitions
- **Comments** - Append-only notes on applications and stages
- **Timeline** - Visual history of application progress
- **Analytics** - Dashboard with pipeline stats and conversion metrics
- **Job Import** - Import jobs from LinkedIn, Indeed, DOU by URL (JSON-LD + Claude AI fallback)
- **AI Match Score** - Resume-to-job matching with Claude Haiku (score, categories, missing keywords)
- **Subscriptions** - Free/Pro/Enterprise plans with Paddle integration
- **Google Calendar** - Schedule interviews from app (hidden behind feature flag, pending Google verification)
- **i18n** - Full localization: English, Russian, Ukrainian
- **Reminders** - Schedule follow-ups (model + repository ready, API pending)
- **Tags** - Categorize entities (model + repository ready, API pending)

---

## Roadmap

Competitors: [Huntr](https://huntr.co) ($40/mo), [Teal](https://tealhq.com) (freemium), [JobHero](https://gojobhero.com) ($9/week)

| # | Feature | Details | Why |
|---|---------|---------|-----|
| 1 | **Chrome Extension** | Save jobs in 1 click from any job board, auto-fill application forms, integration with 40+ boards | Killer feature. Without it, switching from competitors is unlikely |
| 2 | **AI Resume Tailor** (Claude API) | AI Resume Builder, AI Cover Letter, Resume Tailor, Job Match Score (%), Keyword Extraction | Primary paid feature of competitors. Cost via Claude API is minimal, sell for $10-15/mo |
| ~~3~~ | ~~**Kanban Board**~~ | ~~Drag-and-drop board for visualizing jobs by pipeline stage~~ | ✅ **Done** — Grid/Board toggle, drag-and-drop with @dnd-kit, optimistic updates |
| 4 | **Contacts / CRM** | Track recruiters & hiring managers, link to jobs/companies, email templates for follow-ups | Complements tracking, low complexity |
| 5 | **Reminders & Tasks** | Email/push notifications, task checklists per application, follow-up reminders | Basic productivity feature |
| 6 | **AI Interview Practice** | Practice interviews with AI, generate questions from job description, feedback on answers | Differentiator from competitors |
| 7 | **Document Management** | Store & tag documents (resumes, cover letters), version per job | All documents in one place |
| 8 | **Employer Map** | Visualize job locations on a map | Nice-to-have, available in Huntr |

**Monetization:** Free tier (current features) + Pro tier ($10-15/mo: AI features, Chrome extension, unlimited imports). 3-4x cheaper than competitors.

### Job Parsing: How Competitors Do It

Three main approaches exist in the industry:

**1. Chrome Extension + Per-site Content Scripts (Huntr, Teal)**
- Extension has separate parsers for each supported job board (Teal — 50+, Huntr — 40+)
- On supported sites: auto-extracts title, company, location, salary, description via DOM selectors
- On unsupported sites: user manually copies data or fields remain empty
- Pros: Fast, no API costs, works offline
- Cons: UI changes on job boards constantly break parsers; LinkedIn especially problematic

**2. JSON-LD schema.org/JobPosting (Industry Standard) — Jobber uses this**
- Most job boards embed `<script type="application/ld+json">` with structured data for Google Jobs
- LinkedIn, Indeed, Glassdoor all use this format
- Pros: Reliable, standardized, doesn't depend on DOM
- Cons: Not all sites support it (DOU doesn't), data may be incomplete

**3. User-guided DOM + LLM (HuntingPad — newest approach)**
- User highlights job posting text on page
- Extension finds common DOM ancestor, prunes HTML (~80% token reduction)
- Optimized HTML sent to LLM for structured extraction
- Pros: Works on any site, no per-site parsers, doesn't break on UI changes
- Cons: Requires LLM API (but ~$0.001/parse with Claude Haiku)

**Recommended Strategy for Jobber (layered):**

| Layer | When | Coverage | Cost |
|-------|------|----------|------|
| JSON-LD | Always try first | LinkedIn, Indeed, Glassdoor (~70%) | Free |
| Per-site DOM | Top 10-15 boards without JSON-LD | DOU, HH.ru, Djinni | Free |
| Claude Haiku LLM | Any unsupported site | Remaining 100% | ~$0.001/parse |

This layered approach gives 99% coverage with minimal maintenance. The LLM fallback integrates naturally with the AI Resume Tailor feature (Claude API already connected).

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
- **Backend Docs:** [be/README.md](./be/README.md)

---

---

## 🚧 Hidden Features (implemented but not visible to users)

### Google Calendar Integration
**Flag:** `fe/src/shared/lib/features.ts` → `FEATURES.GOOGLE_CALENDAR = false`

Fully implemented — users can connect Google Calendar in Settings and schedule interviews directly from application stages. Hidden because the OAuth app hasn't passed Google's verification process (unverified apps show "Access blocked" to all users).

**To re-enable:**
1. Google Cloud Console → OAuth consent screen → **Publish App**
2. Set `FEATURES.GOOGLE_CALENDAR = true` in `fe/src/shared/lib/features.ts`
3. Deploy

---

**Built with ❤️ for job seekers**
