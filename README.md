# 🚀 IndustryDB

[![Backend CI](https://github.com/jordanlanch/industrydb-back/actions/workflows/ci.yml/badge.svg)](https://github.com/jordanlanch/industrydb-back/actions/workflows/ci.yml)

> Industry-specific business data. Verified. Affordable.

**IndustryDB** is a SaaS platform providing verified local business data by industry. Access leads for tattoo studios, beauty salons, gyms, restaurants, and more at affordable prices.

**Domain:** [industrydb.io](https://industrydb.io)

---

## 📋 Table of Contents

- [Quick Start](#-quick-start)
- [Prerequisites](#-prerequisites)
- [Development Setup](#-development-setup)
- [Architecture](#-architecture)
- [Available Commands](#-available-commands)
- [Project Structure](#-project-structure)
- [Tech Stack](#-tech-stack)
- [Environment Variables](#-environment-variables)
- [Contributing](#-contributing)

---

## ⚡ Quick Start

Get up and running in 3 simple steps:

```bash
# 1. Clone the repository
git clone https://github.com/jordanlanch/industrydb.git
cd industrydb

# 2. Copy environment variables
cp .env.example .env

# 3. Start all services with Docker
make dev
```

That's it! The application will be running at:
- **Frontend:** http://localhost:5678
- **Backend API:** http://localhost:7890

---

## 📦 Prerequisites

The only requirement is **Docker** and **Docker Compose**. Everything runs in containers with hot reload enabled.

- [Docker](https://docs.docker.com/get-docker/) (20.10+)
- [Docker Compose](https://docs.docker.com/compose/install/) (2.0+)

**No need to install:**
- ❌ Go
- ❌ Node.js
- ❌ PostgreSQL
- ❌ Redis
- ❌ Python

All dependencies are containerized!

---

## 🛠️ Development Setup

### 1. Environment Configuration

Copy the example environment file and customize if needed:

```bash
cp .env.example .env
```

The default configuration works out of the box for development.

### 2. Start Development Environment

```bash
make dev
```

This will:
1. Build all Docker images
2. Start PostgreSQL with PostGIS
3. Start Redis cache
4. Start backend API (Go + Echo) with hot reload
5. Start frontend (Next.js 14) with hot reload

### 3. View Logs

```bash
# All services
make logs

# Specific service
make logs-backend
make logs-frontend
make logs-db
```

### 4. Stop Services

```bash
make stop
```

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   DEVELOPMENT SETUP                     │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  [PostgreSQL + PostGIS] ◄──┐                           │
│  (Internal network only)   │                           │
│                             │                           │
│  [Redis Cache] ◄────────────┤                           │
│  (Internal network only)   │                           │
│                             │                           │
│  [Go Backend] ──────────────┘                           │
│  Port 7890 (Hot Reload)    │                           │
│                             │                           │
│  [Next.js Frontend] ────────┘                           │
│  Port 5678 (Hot Reload)                                │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**Key Features:**
- 🔄 **Hot Reload:** Changes to code automatically refresh
- 🐳 **Containerized:** No local dependencies needed
- 🔒 **Secure:** DB and Redis only accessible internally
- 📊 **Logs:** Access all logs via `make logs`

---

## 📝 Available Commands

Run `make help` to see all available commands:

### Development
```bash
make dev           # Start all services
make stop          # Stop all services
make restart       # Restart all services
make logs          # View logs from all services
make build         # Build all Docker images
```

### Testing
```bash
make test          # Run all tests
make test-backend  # Run backend tests only
make test-frontend # Run frontend tests only
```

### Data Pipeline
```bash
make fetch-industry INDUSTRY=tattoo COUNTRY=US CITY="New York"
make normalize-data
make import-db
make validate-data
```

### Database
```bash
make db-shell      # Open PostgreSQL shell
make redis-shell   # Open Redis CLI
make migrate-up    # Run database migrations
```

### Utilities
```bash
make clean         # Clean build artifacts
make clean-all     # Clean everything including volumes
make ps            # Show running containers
make stats         # Show container resource usage
```

---

## 📁 Project Structure

```
industrydb/
├── CLAUDE.md              # Project guide for Claude Code
├── TODO.md                # Master task list
├── PROJECT_STATUS_AND_PLAN.md  # Implementation plan
├── README.md              # This file
├── Makefile               # Common commands
├── docker-compose.yml     # Service orchestration
├── .env.example           # Environment variables template
├── .gitignore             # Git ignore rules
│
├── backend/               # Go API (Echo + Ent)
│   ├── cmd/api/           # Entry point
│   ├── config/            # Configuration
│   ├── ent/schema/        # Database schemas
│   ├── pkg/               # Application code
│   ├── Dockerfile         # Backend container
│   ├── .air.toml          # Hot reload config
│   ├── go.mod             # Go dependencies
│   └── go.sum             # Go checksums
│
├── frontend/              # Next.js 14 Dashboard
│   ├── src/app/           # App Router pages
│   ├── Dockerfile         # Frontend container
│   ├── package.json       # Node dependencies
│   ├── tsconfig.json      # TypeScript config
│   └── next.config.js     # Next.js config
│
├── scripts/               # Data pipeline
│   ├── data-acquisition/  # OSM fetchers
│   ├── data-import/       # PostgreSQL import
│   └── init-db.sh         # DB initialization
│
└── data/                  # Data output directory
    ├── output/            # Fetched data
    └── exports/           # Generated exports
```

---

## 🔧 Tech Stack

| Component | Technology | Version |
|-----------|------------|---------|
| **Backend** | Go + Echo + Ent | 1.24+ |
| **Frontend** | Next.js (App Router) | 14.x |
| **Database** | PostgreSQL + PostGIS | 15 |
| **Cache** | Redis | 7 |
| **Payments** | Stripe | Latest |
| **Container** | Docker Compose | 3.8 |

---

## 🌍 Environment Variables

Key environment variables (see `.env.example` for full list):

```env
# API
API_PORT=7890

# Frontend
FRONTEND_URL=http://localhost:5678
NEXT_PUBLIC_API_URL=http://localhost:7890

# Database (internal)
DATABASE_URL=postgres://industrydb:localdev@db:5432/industrydb?sslmode=disable

# Redis (internal)
REDIS_URL=redis://redis:6379

# JWT
JWT_SECRET=change-this-in-production

# Stripe
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
```

---

## 🔍 Hot Reload

Both frontend and backend have hot reload enabled:

### Backend (Go with Air)
When you edit `.go` files, Air automatically:
1. Detects changes
2. Recompiles the binary
3. Restarts the server

### Frontend (Next.js)
When you edit `.tsx`, `.ts`, `.jsx`, `.js` files:
1. Next.js detects changes
2. Rebuilds the page
3. Refreshes browser automatically

---

## 🐛 Troubleshooting

### Port Already in Use

If ports 7890 or 5678 are already in use, you can change them in `.env`:

```env
API_PORT=8080  # Change the mapped port in docker-compose.yml
```

And update `docker-compose.yml` ports section (lines 67 and 96).

### Container Won't Start

Check logs:
```bash
make logs
```

Rebuild containers:
```bash
make build
make restart
```

### Database Connection Issues

Ensure PostgreSQL is healthy:
```bash
docker-compose ps
```

Check if `industrydb-postgres` shows "(healthy)" status.

### Clear Everything and Start Fresh

```bash
make clean-all
make dev
```

---

## 📚 Documentation

- [CLAUDE.md](./CLAUDE.md) - Complete project guide
- [TODO.md](./TODO.md) - Task tracking
- [PROJECT_STATUS_AND_PLAN.md](./PROJECT_STATUS_AND_PLAN.md) - Implementation plan
- [API Documentation](http://localhost:7890/docs) - Swagger UI

---

## 🤝 Contributing

1. Create a feature branch
2. Make your changes (hot reload will show them immediately)
3. Run tests: `make test`
4. Commit with conventional commits:
   ```bash
   feat: add user registration endpoint
   fix: resolve authentication bug
   docs: update README
   ```
5. Push and create a PR

---

## 📜 License

MIT License - see [LICENSE](./LICENSE) file for details

---

## 🔗 Links

- **Website:** [industrydb.io](https://industrydb.io)
- **GitHub:** [github.com/jordanlanch/industrydb](https://github.com/jordanlanch/industrydb)
- **Issues:** [github.com/jordanlanch/industrydb/issues](https://github.com/jordanlanch/industrydb/issues)

---

**Made with ❤️ by Jordan Lanch**

*IndustryDB - Your source for verified business data*
