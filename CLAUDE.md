# IndustryDB - Industry-Specific Business Data SaaS

## Overview

IndustryDB is a SaaS platform providing verified local business data by industry. Access leads for tattoo studios, beauty salons, gyms, restaurants, and more at affordable prices.

**Domain:** industrydb.io
**Tagline:** "Industry-specific business data. Verified. Affordable."

## Quick Start

```bash
# Start all services with Docker
make dev

# View logs
make logs

# Stop services
make stop

# Run tests
make test

# Data acquisition
make fetch-industry INDUSTRY=tattoo
make import-db
```

## Project Structure

```
industrydb/
├── CLAUDE.md              # This file - main project guide
├── TODO.md                # Legacy task list (deprecated - use Notion)
├── docker-compose.yml     # Service orchestration
├── Makefile              # Common commands
├── .env.example          # Environment variables template
│
├── .claude/              # Claude Code configuration
│   ├── settings.json     # Permissions & safety rules
│   ├── skills/           # Domain expertise documents
│   ├── commands/         # Slash commands (/command)
│   ├── agents/           # Specialized AI agents
│   └── hooks/            # Automation hooks
│
├── scripts/              # Data pipeline (Python)
│   ├── data-acquisition/ # OSM fetchers, normalizer
│   └── data-import/      # PostgreSQL import
│
├── backend/              # Go API (Echo + Ent)
│   ├── ent/schema/       # Database schemas
│   ├── pkg/              # Application code
│   └── cmd/              # Entry points
│
├── frontend/             # Next.js 14 Dashboard
│   └── src/
│       ├── app/          # App Router pages
│       ├── components/   # React components
│       └── lib/          # Utilities
│
└── data/                 # Data output directory
```

## 📋 Project Management with TaskMaster

**🚨 CRITICAL: TaskMaster es la ÚNICA herramienta de gestión de tareas 🚨**

**TaskMaster (Gestión Completa de Tareas):**
- ✅ **Backlog completo del proyecto** (287 tareas de TODO.md)
- ✅ **Tracking de progreso** con estados (pending, in_progress, completed)
- ✅ **Organización por categorías** (Backend, Frontend, Infrastructure, Legal, etc.)
- ✅ **Priorización** (Critical, High, Medium, Low)
- ✅ **Dependencias entre tareas**
- ✅ **Estimaciones de tiempo**
- ✅ **Fuente única de verdad** para gestión de tareas

**Notion (SOLO Documentación):**
- ✅ Documentación de features y API endpoints
- ✅ Guías de arquitectura y deployment
- ✅ Sales playbooks y business docs
- ✅ Референce documentation para desarrolladores
- ❌ **NO gestión de tareas**
- ❌ **NO backlog de tareas**
- ❌ **NO Kanban boards de tareas**
- ❌ **NO sprint planning**

**Workflow Correcto:**

1. **Consultar TaskMaster (SIEMPRE PRIMERO):**
   ```bash
   # Ver todas las tareas
   TaskList

   # Filtrar por estado
   TaskList | grep "pending"

   # Ver tarea específica
   TaskGet <task_id>
   ```

2. **Seleccionar tarea de alta prioridad:**
   - Prioridad: Critical o High
   - Estado: Pending
   - Sin dependencias bloqueantes

3. **Marcar como In Progress:**
   ```bash
   TaskUpdate <task_id> --status in_progress
   ```

4. **Ejecutar con TDD/DDD:**
   - Escribir tests (Red)
   - Implementar código mínimo (Green)
   - Refactorizar (Refactor)
   - Alcanzar 80% coverage

5. **Marcar como Completed:**
   ```bash
   TaskUpdate <task_id> --status completed
   ```

6. **Commit convencional:**
   ```bash
   git commit -m "feat: implement X

   - Implementation details
   - Tests added (85% coverage)

   TaskMaster: #<task_id>"
   ```

7. **Siguiente tarea:**
   - Volver a paso 1

**Migración desde TODO.md:**

TODO.md tiene **287 tareas** que deben crearse en TaskMaster:
- Backend: 80 tareas (56% complete)
- Frontend: 60 tareas (0% complete) 🔴 PRIORITY
- Infrastructure: 35 tareas (0% complete)
- Legal: 22 tareas (45% complete)
- Testing: 30 tareas (0% complete) 🔴 PRIORITY
- Documentation: 20 tareas (0% complete)

**Proceso de migración:**
1. Leer TODO.md sección por sección
2. Crear tareas en TaskMaster con TaskCreate
3. Incluir: nombre, descripción, prioridad, categoría, estimación
4. Establecer dependencias entre tareas
5. Marcar tareas ya completadas como "completed"
6. TODO.md se vuelve archivo de referencia histórica

**Workspace:** https://www.notion.so/IndustryDB-Business-Data-SaaS-Platform-2faae93b443d81b4aa83e5a01fe900d4

### Task Management Philosophy

**Before starting ANY task:**
1. ✅ **Open Notion workspace** and navigate to Project Management page
2. ✅ **Check backlog database** for next priority task (filter by Priority/Status)
3. ✅ **Read full task card** including acceptance criteria, dependencies, estimates
4. ✅ **Review related documentation** in Notion (API docs, architecture, security guidelines)
5. ✅ **Update task status in Notion** (Pending → In Progress)
6. ✅ **Create implementation plan** following TDD/DDD principles (document in task comments)
7. ✅ **Execute with tests** (unit tests + e2e tests, 80% coverage minimum)
8. ✅ **Update task status in Notion** (In Progress → Done)
9. ✅ **Add completion notes** in Notion (what was done, blockers encountered, decisions made)
10. ✅ **Commit with conventional commit** message linking to Notion task
11. ✅ **Move to next task** in Notion backlog

### Notion Documentation Structure

**Workspace:** https://www.notion.so/IndustryDB-Business-Data-SaaS-Platform-2faae93b443d81b4aa83e5a01fe900d4

**⚠️ IMPORTANTE: Notion es SOLO para documentación, NO para gestión de tareas**

#### 1. Executive Dashboard
**Page ID:** `2faae93b443d81df8f66cba6514b9fd4`
**Contenido:** Project overview, métricas de alto nivel, arquitectura
**NO incluye:** Gestión de tareas (usar TaskMaster)

#### 2. Product Department
**Page ID:** `2faae93b443d81f0b952df8fddc10014`
**Contenido:** Documentación de features, roadmap de producto, especificaciones
**NO incluye:** Backlog de tareas (usar TaskMaster)

#### 3. Engineering Department
**Page ID:** `2faae93b443d81428570e8314e920628`
**Contenido:** API Reference (65+ endpoints), database schema, guías técnicas
**NO incluye:** Tareas de desarrollo (usar TaskMaster)

#### 4. Operations/DevOps Department
**Page ID:** `2faae93b443d81129f70cab7f7b495cc`
**Contenido:** Deployment guides (AWS/GCP/Azure), infrastructure docs
**NO incluye:** Tareas de DevOps (usar TaskMaster)

#### 5. Business/Sales Department
**Page ID:** `2faae93b443d819a991fc8ebbe57c74b`
**Contenido:** Pricing strategy, sales playbooks, market analysis
**NO incluye:** Tareas de negocio (usar TaskMaster)

#### 6. Legal/Compliance Department
**Page ID:** `2faae93b443d811bb1b3f97701addffb`
**Contenido:** GDPR compliance docs, legal policies, audit procedures
**NO incluye:** Tareas legales (usar TaskMaster)

#### 7. Reference Documentation
**Contenido:** Arquitectura, principios de código (SOLID, TDD, DDD), guías de estilo
**NO incluye:** Gestión de tareas o sprints (usar TaskMaster)

### Workflow: Starting a New Task

**⚠️ MANDATORY: All task management ONLY in TaskMaster**

```bash
# ========================================
# STEP 1: CONSULT TASKMASTER (ALWAYS FIRST)
# ========================================
# List all pending tasks
TaskList

# Filter by priority (Critical/High first)
# Look for:
# - Status: pending
# - Priority: critical or high
# - No blocking dependencies
# - Matches your expertise area

# ========================================
# STEP 2: SELECT AND START TASK
# ========================================
# Get full task details
TaskGet <task_id>

# Read ALL sections:
# - Task description
# - Acceptance criteria
# - Dependencies (verify all complete)
# - Estimate
# - Related documentation

# Mark as in progress
TaskUpdate <task_id> --status in_progress

# ========================================
# STEP 3: REVIEW DOCUMENTATION IN NOTION
# ========================================
# Open relevant Notion documentation page:
# - Backend task → Engineering Department (API Reference)
# - Frontend task → Product Department (Feature Specs)
# - Infrastructure → Operations (Deployment Guides)
# - Legal → Legal Department (Compliance Docs)

# Review:
# - API endpoints documentation (if backend)
# - Database schema reference (if data model)
# - Feature specifications (if frontend)
# - Security guidelines (ALWAYS)
# - Architecture patterns (follow established)

# ========================================
# STEP 4: CHECK CODEBASE CONTEXT
# ========================================
cd /home/jordanlanch/work/sideProjects/industrydb
make dev  # Start services if needed

# Step 5: Create implementation plan
# - Break down into subtasks
# - Identify files to modify
# - Design approach following SOLID/DDD
# - Plan TDD cycle (Red → Green → Refactor)

# Step 6: Execute with TDD
# Example for new feature:
# 1. Write failing test (Red)
go test ./pkg/myfeature/... -v -run TestNewFeature

# 2. Implement minimum code (Green)
# (code implementation)

# 3. Refactor (keep tests passing)
# (code refactoring)

# 4. Verify coverage
go test ./pkg/myfeature/... -v -cover
# Target: 80% coverage minimum

# ========================================
# STEP 7: MARK TASK AS COMPLETED
# ========================================
# Verify all acceptance criteria met
# Verify tests passing (80%+ coverage)
# Update TaskMaster
TaskUpdate <task_id> --status completed

# ========================================
# STEP 8: COMMIT WITH CONVENTIONAL COMMIT
# ========================================
git add -A
git commit -m "feat: implement JWT blacklist for logout

- Add Redis blacklist for revoked tokens
- Create logout endpoint (/api/v1/auth/logout)
- Add middleware to check blacklist on each request
- Implement TTL matching JWT expiration
- Add unit tests (92% coverage)
- Add integration tests for logout flow
- Update API documentation in Notion

TaskMaster: #<task_id>
"

# ========================================
# STEP 9: MOVE TO NEXT TASK
# ========================================
# Return to Step 1
# List pending tasks in TaskMaster
# Select next priority task
# Repeat workflow
```

### Task Status Management in TaskMaster

**Status Workflow:**
```
pending → in_progress → completed
```

**When to use each status:**
- **pending:** Task not started, waiting to be picked up
- **in_progress:** Currently working on it
- **completed:** Done with all acceptance criteria met + tests passing

**Rules:**
- ✅ ONLY change to "in_progress" when you actually start work
- ✅ MUST verify acceptance criteria before marking "completed"
- ✅ MUST have tests passing (80%+ coverage) before "completed"
- ✅ MUST link task ID in commit message
- ❌ NEVER mark "completed" without tests passing
- ❌ NEVER skip TaskUpdate when starting/completing work

### Development Principles (ALWAYS Follow)

#### 1. Clean Architecture
**Layers (dependency flows inward):**
```
External APIs/UI
       ↓
Interface Adapters (Handlers, Controllers)
       ↓
Business Logic (Use Cases, Services)
       ↓
Domain Models (Entities, Value Objects)
```

**Example:**
```go
// ✅ GOOD: Clean separation
// Domain (backend/pkg/leads/models.go)
type Lead struct {
    ID       string
    Name     string
    Industry string
}

// Use Case (backend/pkg/leads/service.go)
func (s *Service) SearchLeads(ctx context.Context, filters Filters) ([]Lead, error) {
    // Business logic here
}

// Handler (backend/pkg/api/handlers/leads.go)
func (h *Handler) SearchLeads(c echo.Context) error {
    filters := parseFilters(c)
    leads, err := h.service.SearchLeads(ctx, filters)
    // HTTP-specific handling
}
```

#### 2. SOLID Principles

**Single Responsibility:**
```go
// ❌ BAD: Handler does too much
func (h *Handler) CreateUser(c echo.Context) error {
    // Parsing, validation, DB access, email sending - TOO MUCH
}

// ✅ GOOD: Each layer has one responsibility
func (h *Handler) CreateUser(c echo.Context) error {
    req := h.parser.Parse(c)              // Parse
    if err := h.validator.Validate(req); // Validate
    user, err := h.service.Create(req)    // Business logic
    h.emailer.SendWelcome(user)           // Side effect
    return c.JSON(200, user)              // Response
}
```

**Open/Closed:**
```go
// ✅ GOOD: Open for extension, closed for modification
type ExportStrategy interface {
    Export(leads []Lead) ([]byte, error)
}

type CSVExporter struct{}
func (e *CSVExporter) Export(leads []Lead) ([]byte, error) { }

type ExcelExporter struct{}
func (e *ExcelExporter) Export(leads []Lead) ([]byte, error) { }

// Adding JSON export doesn't require modifying existing code
type JSONExporter struct{}
func (e *JSONExporter) Export(leads []Lead) ([]byte, error) { }
```

**Dependency Inversion:**
```go
// ❌ BAD: Depends on concrete implementation
type Service struct {
    db *ent.Client  // Tied to Ent
}

// ✅ GOOD: Depends on abstraction
type LeadRepository interface {
    Create(ctx context.Context, lead *Lead) error
    FindByID(ctx context.Context, id string) (*Lead, error)
}

type Service struct {
    repo LeadRepository  // Can swap implementations
}
```

#### 3. TDD Workflow (Red-Green-Refactor)

**Always write tests BEFORE implementation:**

```go
// Step 1: RED - Write failing test
func TestSearchLeads_WithFilters(t *testing.T) {
    service := setupTestService()

    filters := Filters{
        Industry: "tattoo",
        Country:  "US",
    }

    leads, err := service.SearchLeads(context.Background(), filters)

    assert.NoError(t, err)
    assert.Len(t, leads, 5)
    assert.Equal(t, "tattoo", leads[0].Industry)
}

// Run test → FAILS (method doesn't exist)
// go test ./pkg/leads/... -v -run TestSearchLeads_WithFilters

// Step 2: GREEN - Write minimum code to pass
func (s *Service) SearchLeads(ctx context.Context, filters Filters) ([]Lead, error) {
    // Simplest implementation that passes test
    return s.repo.Search(ctx, filters)
}

// Run test → PASSES
// go test ./pkg/leads/... -v -run TestSearchLeads_WithFilters

// Step 3: REFACTOR - Improve code quality
func (s *Service) SearchLeads(ctx context.Context, filters Filters) ([]Lead, error) {
    // Add caching
    cacheKey := filters.CacheKey()
    if cached, ok := s.cache.Get(cacheKey); ok {
        return cached.([]Lead), nil
    }

    // Add validation
    if err := filters.Validate(); err != nil {
        return nil, err
    }

    // Search with timeout
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    leads, err := s.repo.Search(ctx, filters)
    if err != nil {
        return nil, err
    }

    // Cache result
    s.cache.Set(cacheKey, leads, 5*time.Minute)

    return leads, nil
}

// Run test → STILL PASSES
```

#### 4. DDD (Domain-Driven Design)

**Ubiquitous Language:**
```go
// ✅ Use business terms in code
type Lead struct {          // Not "BusinessData"
    Industry    string      // Not "Category"
    QualityScore int        // Not "Rating"
}

// ✅ Method names match business operations
func (s *Service) ExportLeads(...)  // Not "GetData"
func (s *Service) SearchLeads(...)  // Not "Query"
```

**Bounded Contexts:**
```
industrydb/backend/pkg/
├── leads/          # Lead Management Context
│   ├── models.go   # Domain models
│   ├── service.go  # Business logic
│   └── repository.go
├── billing/        # Billing Context
│   ├── subscription.go
│   └── invoice.go
└── auth/           # Authentication Context
    ├── user.go
    └── jwt.go
```

**Aggregates:**
```go
// Lead is aggregate root
type Lead struct {
    ID       string
    Name     string
    Contacts []Contact  // Value objects
    Address  Address    // Value object
}

// Always access Contacts through Lead (preserve invariants)
func (l *Lead) AddContact(c Contact) error {
    if len(l.Contacts) >= 10 {
        return errors.New("max 10 contacts per lead")
    }
    l.Contacts = append(l.Contacts, c)
    return nil
}
```

### Code Quality Standards

**Every pull request must:**
- ✅ Pass all existing tests (no regressions)
- ✅ Add tests for new code (80% coverage minimum)
- ✅ Follow SOLID principles (reviewed in PR)
- ✅ Use Clean Architecture (layers respected)
- ✅ Include API documentation (if API changed)
- ✅ Update Notion task status (link in commit message)
- ✅ Pass linters (golangci-lint, ESLint)

**Code review checklist:**
```markdown
## Code Review Checklist

### Architecture
- [ ] Follows Clean Architecture (correct layer)
- [ ] Respects SOLID principles (especially SRP, DIP)
- [ ] Bounded context boundaries respected (DDD)

### Testing
- [ ] Unit tests cover happy path + edge cases
- [ ] Integration tests for external dependencies
- [ ] E2E tests for user workflows
- [ ] Coverage ≥ 80% for new code

### Security
- [ ] Input validation (all user input)
- [ ] Error handling (no sensitive info leaked)
- [ ] Context timeouts (all DB/external calls)
- [ ] Rate limiting (if API endpoint)

### Performance
- [ ] Database queries optimized (indexes used)
- [ ] Caching implemented (if appropriate)
- [ ] No N+1 queries
- [ ] Context cancellation handled

### Documentation
- [ ] API docs updated (if endpoints changed)
- [ ] Notion task updated (status + notes)
- [ ] Code comments for complex logic
- [ ] README updated (if setup changed)
```

### Task Backlog (287 Tasks Total)

**⚠️ CRITICAL: All tasks in TaskMaster ONLY**

**Source:** TODO.md (to be migrated to TaskMaster)

**Backlog Summary:**

**By Category:**
- Backend: 80 tasks (56% complete) - **45 done, 35 pending**
- Frontend: 60 tasks (0% complete) - **0 done, 60 pending** 🔴 **HIGH PRIORITY**
- Infrastructure: 35 tasks (0% complete) - **0 done, 35 pending**
- Legal: 22 tasks (45% complete) - **10 done, 12 pending**
- Testing: 30 tasks (0% complete) - **0 done, 30 pending** 🔴 **HIGH PRIORITY**
- Documentation: 20 tasks (0% complete) - **0 done, 20 pending**

**By Priority:**
- 🔴 Critical: 45 tasks (23 complete = 51%)
- 🟠 High: 78 tasks (27 complete = 35%)
- 🟡 Medium: 89 tasks (5 complete = 6%)
- 🟢 Low: 75 tasks (0 complete = 0%)

**By Status:**
- ✅ completed: 55 tasks (19%)
- 🟡 in_progress: 8 tasks (3%)
- 📋 pending: 224 tasks (78%)

**Current Phase (Phase 1 - Weeks 1-3):**
- ✅ Security backend: 45/45 tasks (100%) - **COMPLETE**
- ⏳ Legal compliance: 10/18 tasks (56%) - **IN PROGRESS**
- 🔴 **4 Critical Blockers** preventing production:
  1. Email service integration (SendGrid/AWS SES) - 4-8h
  2. Database SSL certificates - 2-4h
  3. JWT secret rotation (Secrets Manager) - 4-6h
  4. Stripe production keys - 2-3h

**Next Phase (Phase 2 - Weeks 4-6):**
- UI/UX polish: 22 tasks
- Admin dashboard: 15 tasks
- Error boundaries: 8 tasks
- Form validation: 12 tasks
- Accessibility (WCAG): 18 tasks

**Migration Status:**
- ⏳ IN PROGRESS: Creating 287 tasks in TaskMaster from TODO.md
- 📝 Current: 16 tasks created (translation work)
- 📋 Remaining: 271 tasks to create
- 🎯 Goal: Complete TaskMaster migration by end of week

**How to check tasks:**
```bash
# List all tasks
TaskList

# Count tasks by status
TaskList | grep "pending" | wc -l
TaskList | grep "completed" | wc -l

# Find high priority tasks
TaskList | grep "priority.*high"
```

## Tech Stack

| Component | Technology | Version |
|-----------|------------|---------|
| Backend | Go + Echo + Ent | 1.22+ |
| Frontend | Next.js (App Router) | 14.x |
| Database | PostgreSQL + PostGIS | 15 |
| Cache | Redis | 7 |
| Payments | Stripe | Latest |
| Container | Docker Compose | 3.8 |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     DATA PIPELINE                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  [OpenStreetMap API] ──▶ [Python Fetchers] ──▶ [Normalizer]│
│                                │                            │
│                                ▼                            │
│  [PostgreSQL + PostGIS] ◀── [Import Script]                │
│         │                                                   │
│         ▼                                                   │
│  [Go Backend (Echo)] ◀── [Redis Cache]                     │
│         │                                                   │
│         ▼                                                   │
│  [Next.js Dashboard] ◀── [Stripe Billing]                  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Internationalization (i18n)

**Implemented:** 2026-01-29

IndustryDB supports multiple languages using **next-intl** with URL-based locale routing.

### Supported Languages
- 🇺🇸 **English (en)** - Default
- 🇪🇸 **Spanish (es)**
- 🇫🇷 **French (fr)**

### URL Structure
```
/en/dashboard          # English
/es/dashboard          # Spanish
/fr/dashboard          # French
/dashboard             # Redirects to /en/dashboard (default)
```

### Features
- **URL-based routing:** Clean URLs with locale prefix (`/es/dashboard`)
- **Language switcher:** Component in footer, preserves path when switching
- **351+ translation keys** per language
- **12 pages translated:** All auth + dashboard pages
- **Dynamic pluralization:** ICU MessageFormat support
- **Accessibility:** All ARIA labels translated

### Architecture
```
frontend/
├── middleware.ts              # Locale routing
├── i18n.ts                   # i18n config
├── messages/
│   ├── en.json               # English (351 keys)
│   ├── es.json               # Spanish (351 keys)
│   └── fr.json               # French (351 keys)
└── src/app/
    ├── layout.tsx            # Root layout
    └── [locale]/             # Locale-based routing
        ├── (auth)/           # Login, register, etc.
        └── dashboard/        # All dashboard pages
```

### Usage in Components
```tsx
import { useTranslations } from 'next-intl'

function MyComponent() {
  const t = useTranslations('namespace')

  return (
    <>
      <h1>{t('title')}</h1>
      <p>{t('found', { count: results.length })}</p>
    </>
  )
}
```

### Adding New Languages
1. Create `messages/{locale}.json`
2. Add locale to `i18n.ts` locales array
3. Add to language switcher component
4. Translate all keys in new file

### Pages Translated
**Authentication (5):** login, register, forgot-password, verify-email, reset-password
**Dashboard (7):** home, leads, exports, analytics, api-keys, saved-searches, settings

**Documentation:** See [I18N_IMPLEMENTATION_REPORT.md](I18N_IMPLEMENTATION_REPORT.md) for detailed information.

## Development Workflow

### Available Slash Commands

```bash
# Workflows (complete processes)
/workflows:full-setup      # Complete project setup
/workflows:feature-dev     # TDD feature development
/workflows:tdd-cycle       # Red-green-refactor cycle
/workflows:data-acquisition # Fetch and import data
/workflows:deploy          # Deploy to production

# Development tools
/dev:build                 # Build all services
/dev:test                  # Run tests with coverage
/dev:lint                  # Run linters
/dev:debug                 # Debug assistance

# Data operations
/data:fetch                # Fetch industry data
/data:import               # Import to database
/data:export               # Export data
/data:validate             # Validate data quality

# Git operations
/git:commit                # Create conventional commit
/git:pr-create             # Create pull request
```

### Autonomous Mode

Run Claude Code in fully autonomous mode:

```bash
# Execute full setup
claude --dangerously-skip-permissions -p "Execute /workflows:full-setup"

# Complete next TODO task
claude --dangerously-skip-permissions -p "Read TODO.md, find next pending task, complete it, commit, repeat."

# Feature development
claude --dangerously-skip-permissions -p "Execute /workflows:feature-dev 'Add user registration endpoint'"
```

## API Endpoints

### Authentication
```
POST /api/v1/auth/register    # Create account
POST /api/v1/auth/login       # Login, get JWT
GET  /api/v1/auth/me          # Get current user
```

### Leads
```
GET  /api/v1/leads            # Search leads (with filters)
POST /api/v1/leads/export     # Export to CSV/Excel
GET  /api/v1/leads/:id        # Get single lead
```

### User & Billing
```
GET  /api/v1/user/usage       # Usage statistics
POST /api/v1/billing/checkout # Create Stripe checkout
POST /api/v1/billing/webhook  # Stripe webhook
GET  /api/v1/user/data-export # Export personal data (GDPR)
DELETE /api/v1/user/account   # Delete account (GDPR)
```

### Admin (Requires admin or superadmin role)
```
GET    /api/v1/admin/stats       # Platform statistics
GET    /api/v1/admin/users       # List users (paginated, filterable)
GET    /api/v1/admin/users/:id   # Get user details
PATCH  /api/v1/admin/users/:id   # Update user (tier, role, limits)
DELETE /api/v1/admin/users/:id   # Suspend user account
```

### Analytics
```
GET /api/v1/user/analytics/daily      # Daily usage statistics
GET /api/v1/user/analytics/summary    # Aggregated usage summary
GET /api/v1/user/analytics/breakdown  # Usage breakdown by action type
```

**Query Parameters:**
- `days` (optional) - Number of days to analyze (default: 30, max: 365)

**Example:**
```bash
# Get last 30 days of daily usage
GET /api/v1/user/analytics/daily?days=30

# Get 90-day summary
GET /api/v1/user/analytics/summary?days=90
```

### API Keys (Business Tier Feature)
**Implemented:** 2026-01-27

API keys provide programmatic access to the IndustryDB API. Available only for Business tier subscribers.

```
POST   /api/v1/api-keys          # Create new API key
GET    /api/v1/api-keys          # List all user's API keys
GET    /api/v1/api-keys/:id      # Get single API key details
PATCH  /api/v1/api-keys/:id      # Update API key name
POST   /api/v1/api-keys/:id/revoke  # Revoke API key (soft delete)
DELETE /api/v1/api-keys/:id      # Delete API key (hard delete)
GET    /api/v1/api-keys/stats    # Get API key usage statistics
```

**Security Features:**
- Keys are SHA256 hashed before storage (never store plain text)
- Plain key shown only once on creation
- Keys have format: `idb_[64 hex characters]`
- Optional expiration dates
- Revocation system (separate from deletion)
- Async usage tracking (non-blocking)

**Usage:**
```bash
# Create API key
POST /api/v1/api-keys
{
  "name": "Production API Key",
  "expires_at": "2027-01-01T00:00:00Z"  # Optional
}

# Authenticate with API key
curl -H "X-API-Key: idb_abc123..." https://api.industrydb.io/api/v1/leads
```

**Implementation:**
- Service: `backend/pkg/apikey/service.go`
- Handler: `backend/pkg/api/handlers/apikey.go`
- Schema: `backend/ent/schema/apikey.go`

### Query Parameters for /api/v1/leads
```
?industry=tattoo|beauty|barber|gym|restaurant
&country=US|GB|ES|DE|...
&city=New York
&has_email=true
&has_phone=true
&page=1
&limit=50
```

## Security & Rate Limiting

### CORS Configuration
**Implemented:** 2026-01-26

CORS is configured with strict origin restrictions:
- **Development:** `http://localhost:5678`
- **Production:** `https://industrydb.io`, `https://www.industrydb.io`

Allowed methods: GET, POST, PUT, PATCH, DELETE
Credentials: Enabled

**Implementation:** `backend/cmd/api/main.go` lines 63-80

### Rate Limiting
**Implemented:** 2026-01-26

Rate limiting protects against brute force, DoS attacks, and excessive API usage:

| Endpoint | Limit | Purpose |
|----------|-------|---------|
| `POST /auth/register` | 3 per hour per IP | Prevent account spam |
| `POST /auth/login` | 5 per minute per IP | Prevent brute force attacks |
| `POST /webhook/stripe` | 100 per minute | Handle Stripe webhook bursts |
| All other endpoints | 60 per minute per IP | General API protection |

**Configuration:**
```env
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_BURST=10
```

**Implementation:**
- Middleware: `backend/pkg/middleware/rate_limiter.go`
- Uses `golang.org/x/time/rate` for token bucket algorithm
- Per-IP tracking with automatic cleanup
- Configurable per-endpoint limits

**Testing:**
```bash
go test -v ./pkg/middleware/... -run TestRateLimiter
```

### Return URL Validation (Open Redirect Protection)
**Implemented:** 2026-01-26

Stripe billing portal return URLs are validated to prevent open redirect attacks:

**Security Checks:**
1. **Protocol whitelist:** Only `http` and `https` (blocks javascript:, data:, ftp:, file:)
2. **No userinfo:** Rejects URLs with username/password (prevents phishing: `https://attacker@industrydb.io`)
3. **Host whitelist:**
   - Development: `localhost:5678`
   - Production: `industrydb.io`, `www.industrydb.io`
4. **Safe fallback:** Returns `https://industrydb.io/dashboard/settings/billing` if validation fails

**Implementation:**
- Function: `validateReturnURL()` in `backend/pkg/api/handlers/billing.go`
- Applied in: `CreatePortalSession` handler
- Follows: **SOLID principles** (SRP - single responsibility), **DDD** (bounded context)

**Test Coverage:** 13 test cases including:
- Malicious external URLs
- Subdomain attacks (`industrydb.io.evil.com`)
- Protocol attacks (`javascript:`, `data:`, `ftp:`)
- Phishing attempts (`https://attacker@industrydb.io`)
- Invalid formats

**Testing:**
```bash
go test -v ./pkg/api/handlers/billing_test.go -run TestValidateReturnURL
```

All tests pass ✅ (TDD: Red → Green → Refactor cycle)

## Legal Compliance

### Terms of Service Acceptance
**Implemented:** 2026-01-26

Users must accept Terms of Service and Privacy Policy during registration. The acceptance timestamp is tracked in the database.

**Implementation:**
- **Field:** `accepted_terms_at` in `backend/ent/schema/user.go` (lines 53-56)
- **Handler:** Automatically set during registration in `backend/pkg/api/handlers/auth.go` (line 87)
- **Frontend:** Checkbox in registration form (required)

**Database Schema:**
```go
field.Time("accepted_terms_at").
    Optional().
    Nillable().
    Comment("When user accepted Terms of Service and Privacy Policy")
```

**How it works:**
1. User registers and checks "I agree to Terms and Privacy Policy"
2. Backend saves current timestamp in `accepted_terms_at` field
3. Timestamp is stored in database for compliance tracking
4. Can be queried later for legal compliance reporting

### Cookie Consent Banner (GDPR)
**Implemented:** 2026-01-26

Cookie consent banner complies with GDPR requirements for European users. Users can accept or decline cookies, and their preference is stored for 365 days.

**Implementation:**
- **Component:** `frontend/src/components/cookie-consent.tsx`
- **Package:** `react-cookie-consent` v9.0.0
- **Cookie Name:** `industrydb_cookie_consent`
- **Expiration:** 365 days

**Features:**
- Accept/Decline buttons
- Link to Privacy Policy
- Google Analytics consent mode integration
- Cookie Settings button in footer (allows users to change preferences)

**User Flow:**
1. Banner appears at bottom of screen on first visit
2. User can click "Accept All Cookies" or "Decline"
3. Preference is stored in cookie for 365 days
4. Banner does not reappear until cookie expires or is cleared
5. User can change preferences via "Cookie Settings" in footer

**Google Analytics Integration:**
```javascript
// On Accept
gtag('consent', 'update', {
  'analytics_storage': 'granted'
});

// On Decline
gtag('consent', 'update', {
  'analytics_storage': 'denied'
});
```

**Files Modified:**
- `frontend/src/app/layout.tsx` - Added CookieBanner component
- `frontend/src/components/footer.tsx` - Added Cookie Settings button
- `frontend/package.json` - Added react-cookie-consent dependency

**Installation:**
```bash
cd frontend
npm install  # Installs react-cookie-consent and other dependencies
```

**Note:** Ensure `frontend/node_modules` has correct ownership before running npm install:
```bash
sudo chown -R $USER:$USER frontend/node_modules
```

### GDPR Data Export
**Implemented:** 2026-01-26

Users can download all their personal data in JSON format, complying with GDPR Article 15 (Right of Access).

**Implementation:**

**Backend:**
- **Endpoint:** `GET /api/v1/user/data-export`
- **Handler:** `ExportPersonalData()` in `backend/pkg/api/handlers/user.go`
- **Authentication:** Requires valid JWT token
- **Timeout:** 10 seconds context timeout

**Data Included:**
1. **User Profile:**
   - ID, email, name
   - Subscription tier
   - Email verification status
   - Stripe customer ID
   - Account creation/update timestamps
   - Last login timestamp
   - Terms acceptance timestamp

2. **Usage Statistics:**
   - Current usage count
   - Usage limit
   - Remaining leads
   - Last reset timestamp

3. **Subscription History:**
   - All past and current subscriptions
   - Stripe subscription IDs
   - Billing periods
   - Status (active, canceled, etc.)

4. **Export History:**
   - All data exports created
   - Formats (CSV, Excel)
   - Filters applied
   - Lead counts
   - Status and URLs
   - Expiration dates

5. **Metadata:**
   - Export timestamp
   - Format version
   - Data structure version

**Frontend:**
- **Service:** `userService.exportPersonalData()` in `frontend/src/services/user.service.ts`
- **UI:** Button in Settings page under "Privacy & Data" section
- **Download:** Automatic JSON file download with timestamp
- **Filename:** `industrydb-personal-data-{timestamp}.json`

**User Flow:**
1. User navigates to Dashboard → Settings
2. Scrolls to "Privacy & Data" section
3. Clicks "Download My Data (GDPR)" button
4. Backend generates complete data export (10s timeout)
5. JSON file downloads automatically to user's device
6. User can view/store data as needed

**Security:**
- Requires authentication (JWT)
- User can only export their own data
- Context timeout prevents hanging requests
- Data sanitized before export

**Testing:**
```bash
# Backend
cd backend
go test ./pkg/api/handlers/user_test.go -v -run TestExportPersonalData

# Manual test
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:7890/api/v1/user/data-export \
  -o my-data.json
```

**Files Modified/Created:**
- `backend/pkg/api/handlers/user.go` - Added ExportPersonalData handler
- `backend/cmd/api/main.go` - Registered `/user/data-export` route
- `frontend/src/services/user.service.ts` - Created user service with export function
- `frontend/src/app/dashboard/settings/page.tsx` - Added Privacy & Data card with download button

### GDPR Data Deletion (Right to be Forgotten)
**Implemented:** 2026-01-26

Users can permanently delete their account and all associated data, complying with GDPR Article 17 (Right to Erasure).

**Implementation:**

**Backend:**
- **Endpoint:** `DELETE /api/v1/user/account`
- **Handler:** `DeleteAccount()` in `backend/pkg/api/handlers/user.go`
- **Authentication:** Requires valid JWT token + password verification
- **Method:** Soft delete (anonymization)

**Deletion Process:**
1. **Password Verification:** User must provide current password
2. **Data Anonymization:**
   - Email changed to `deleted_{user_id}@deleted.local`
   - Name changed to "Deleted User"
   - Password hash cleared
   - Email verification flag reset
   - Stripe customer ID cleared
   - Login timestamps cleared
   - Terms acceptance cleared

3. **Related Data:**
   - All exports marked as "expired"
   - Subscription data preserved for accounting/legal (anonymized)
   - Future: Stripe subscription cancellation

4. **Account Status:** Account still exists but completely anonymized (soft delete)

**Why Soft Delete?**
- Maintains referential integrity in database
- Preserves billing/legal records (required by law)
- Prevents immediate ID reuse
- Allows for data recovery within grace period (optional feature)
- Future: Can implement hard delete after 30-90 days

**Frontend:**
- **Service:** `userService.deleteAccount(password)` in `user.service.ts`
- **UI:** "Delete Account" button in Settings → Privacy & Data
- **Confirmation Dialog:**
  - Password input field
  - Warning about permanence
  - List of what will be deleted
  - Cancel/Confirm buttons

**User Flow:**
1. User navigates to Settings → Privacy & Data
2. Clicks "Delete Account" button
3. Dialog appears with:
   - Warning about permanent deletion
   - Password confirmation field
   - List of data to be deleted
4. User enters password and clicks "Delete My Account"
5. Backend verifies password
6. Account is anonymized
7. User is logged out and redirected to home page

**Security:**
- Password verification required (prevents unauthorized deletion)
- JWT authentication (ensures user owns account)
- Context timeout (10 seconds)
- Soft delete prevents accidental data loss

**Testing:**
```bash
# Backend test
curl -X DELETE http://localhost:7890/api/v1/user/account \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"password":"your_password"}'

# Expected response:
# {"message":"Account deleted successfully"}

# Verify anonymization
psql -h localhost -U industrydb -d industrydb \
  -c "SELECT id, email, name FROM users WHERE id = USER_ID;"
# Should show: deleted_{id}@deleted.local, "Deleted User"
```

**Files Modified:**
- `backend/pkg/api/handlers/user.go` - Added DeleteAccount handler
- `backend/cmd/api/main.go` - Registered DELETE `/user/account` route
- `frontend/src/services/user.service.ts` - Added deleteAccount function
- `frontend/src/app/dashboard/settings/page.tsx` - Added delete dialog with password confirmation

**Future Enhancements:**
- Stripe subscription cancellation integration
- Grace period (30 days to recover account)
- Hard delete after grace period
- Email notification before deletion
- Download data automatically before deletion

### Audit Logs
**Implemented:** 2026-01-26

Complete audit logging system for tracking all important user actions and system events. Essential for compliance, security monitoring, and debugging.

**Implementation:**

**Backend Schema:**
- **Table:** `audit_logs`
- **Fields:**
  - `user_id` - User who performed action (nullable for system actions)
  - `action` - Action type (enum: user_login, user_logout, user_register, data_export, account_delete, etc.)
  - `resource_type` - Type of resource affected (user, lead, export, subscription)
  - `resource_id` - ID of affected resource
  - `ip_address` - IP address of user
  - `user_agent` - Browser/client user agent
  - `metadata` - Additional context (JSON)
  - `severity` - Event importance (info, warning, error, critical)
  - `description` - Human-readable description
  - `created_at` - Event timestamp

**Tracked Events:**
1. **Authentication:**
   - User registration
   - User login
   - User logout
   - Password changes
   - Email verification

2. **GDPR Compliance:**
   - Data exports (Article 15)
   - Account deletions (Article 17)

3. **Data Access:**
   - Lead searches
   - Lead views
   - Export creation
   - Export downloads

4. **Subscription:**
   - Subscription creation
   - Subscription updates
   - Subscription cancellations
   - Payment success/failures

5. **API Keys:**
   - API key creation
   - API key deletion

**Service Layer:**
- **Package:** `backend/pkg/audit`
- **Service:** `audit.Service`
- **Methods:**
  - `Log()` - Generic logging
  - `LogUserLogin()` - Login events
  - `LogUserLogout()` - Logout events
  - `LogUserRegister()` - Registration events
  - `LogAccountDelete()` - Deletion events (critical severity)
  - `LogDataExport()` - GDPR export events
  - `LogExportCreate()` - Export creation
  - `LogLeadSearch()` - Search events
  - `GetUserLogs()` - Retrieve user's logs
  - `GetRecentLogs()` - Admin: recent logs
  - `GetCriticalLogs()` - Admin: critical events

**Integration:**
- Automatic logging in handlers (non-blocking goroutines)
- IP address and User-Agent extraction from requests
- Context timeouts (5s) to prevent hanging
- Indexes on user_id, action, resource, created_at for fast queries

**Endpoints:**
```
GET /api/v1/user/audit-logs?limit=50
```

**Query Parameters:**
- `limit` - Number of logs to return (default: 50, max: 100)

**Response Format:**
```json
{
  "logs": [
    {
      "id": 1,
      "user_id": 123,
      "action": "user_login",
      "resource_type": null,
      "resource_id": null,
      "ip_address": "192.168.1.1",
      "user_agent": "Mozilla/5.0...",
      "metadata": {},
      "severity": "info",
      "description": "User logged in successfully",
      "created_at": "2026-01-26T15:30:00Z"
    },
    {
      "id": 2,
      "user_id": 123,
      "action": "data_export",
      "resource_type": null,
      "resource_id": null,
      "ip_address": "192.168.1.1",
      "user_agent": "Mozilla/5.0...",
      "metadata": {},
      "severity": "info",
      "description": "User exported personal data (GDPR)",
      "created_at": "2026-01-26T15:35:00Z"
    }
  ],
  "count": 2
}
```

**Severity Levels:**
- `info` - Normal operations (login, search, export)
- `warning` - Suspicious activity (multiple failed logins)
- `error` - Failed operations (payment failures)
- `critical` - Security events (account deletion, data export, suspicious access)

**Performance:**
- Non-blocking: Logs written in goroutines (doesn't slow down requests)
- Indexed: Fast queries on common filters
- Batching: Can be enhanced with batch inserts for high volume

**Security:**
- IP tracking for fraud detection
- User-Agent tracking for suspicious clients
- Critical events flagged (account deletion, data export)
- Audit trail for compliance investigations

**Use Cases:**
1. **GDPR Compliance:** Prove user consent, track data access
2. **Security:** Detect brute force, suspicious activity
3. **Debugging:** Track user actions leading to issues
4. **Analytics:** Understand user behavior
5. **Support:** Help users with account issues

**Files Created:**
- `backend/ent/schema/auditlog.go` - Schema definition
- `backend/pkg/audit/service.go` - Audit service
- `backend/pkg/audit/helpers.go` - IP/User-Agent extraction
- `backend/pkg/api/handlers/audit.go` - HTTP handlers

**Files Modified:**
- `backend/ent/schema/user.go` - Added audit_logs edge
- `backend/pkg/api/handlers/auth.go` - Added logging for login/logout/register
- `backend/pkg/api/handlers/user.go` - Added logging for data export/deletion
- `backend/cmd/api/main.go` - Initialized audit service, registered routes

**Testing:**
```bash
# View your own audit logs
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:7890/api/v1/user/audit-logs?limit=10

# Check database directly
psql -h localhost -U industrydb -d industrydb \
  -c "SELECT * FROM audit_logs WHERE user_id = YOUR_USER_ID ORDER BY created_at DESC LIMIT 10;"
```

**Future Enhancements:**
- Admin dashboard for viewing all logs
- Real-time alerts for critical events
- Log retention policy (auto-delete old logs)
- Export audit logs for compliance reporting
- Anomaly detection (unusual patterns)
- Integration with SIEM systems

### Email Verification
**Implemented:** 2026-01-26

Complete email verification system to prevent fake accounts and ensure valid email addresses.

**Implementation:**

**Backend Schema:**
- **Fields Added to Users:**
  - `email_verification_token` - Unique token for verification (sensitive, 64 chars hex)
  - `email_verification_token_expires_at` - Token expiration (24 hours from creation)
  - `email_verified_at` - Timestamp when email was verified

**Flow:**
1. **Registration:**
   - User registers with email/password
   - Verification token generated (32 bytes random → hex encoded)
   - Token stored in database with 24-hour expiration
   - Verification email sent automatically
   - User can still login but with limited access

2. **Verification:**
   - User clicks link in email: `/verify-email/{token}`
   - Backend validates token and expiration
   - Marks email as verified
   - Clears verification token from database
   - Sends welcome email
   - Full access granted

3. **Resend Email:**
   - User can request new verification email
   - New token generated (invalidates old one)
   - New email sent with 24-hour expiration

**Email Service:**
- **Package:** `backend/pkg/email`
- **Current:** Logs emails to console (development)
- **Production:** Replace with SendGrid, AWS SES, or SMTP

**Endpoints:**
```
GET  /api/v1/auth/verify-email/:token      # Verify email with token
POST /api/v1/auth/resend-verification      # Resend verification email
```

**Middleware:**
- **Package:** `backend/pkg/middleware/email_verified.go`
- **Function:** `RequireEmailVerified(db)`
- **Usage:** Apply to endpoints requiring verified email

**Example Usage:**
```go
// Require email verification for sensitive operations
protected.GET("/sensitive-data", handler.GetSensitiveData,
    custommiddleware.RequireEmailVerified(db.Ent))
```

**Frontend:**

**Verification Page:**
- **Route:** `/verify-email/[token]`
- **File:** `frontend/src/app/(auth)/verify-email/[token]/page.tsx`
- **States:** Loading, Success, Error
- **Auto-redirect:** Dashboard after 3 seconds on success

**Verification Banner:**
- **Component:** `EmailVerificationBanner`
- **Location:** Dashboard layout (appears at top)
- **Features:**
  - Shows user's email
  - "Resend Email" button
  - Dismissible
  - Success/error feedback

**User Experience:**
1. User registers → Email sent
2. User logs in → Dashboard shows banner
3. User clicks "Resend Email" if needed
4. User checks email → Clicks verification link
5. Redirected to verification page → Success
6. Redirected to dashboard → Banner disappears
7. Full access granted

**Email Templates:**

**Verification Email:**
```
Subject: Verify your IndustryDB account

Hi [Name],

Welcome to IndustryDB! Please verify your email address by clicking the link below:

[Verification Link]

This link will expire in 24 hours.

If you didn't create an account, you can safely ignore this email.

Thanks,
The IndustryDB Team
```

**Welcome Email (after verification):**
```
Subject: Welcome to IndustryDB!

Hi [Name],

Your email has been verified! You now have full access to IndustryDB.

Get started:
- Search for leads in your industry
- Export data in CSV or Excel format
- Upgrade for more features

Visit your dashboard: [Dashboard Link]

Thanks,
The IndustryDB Team
```

**Security:**
- Random 32-byte token (cryptographically secure)
- 24-hour expiration
- Token cleared after successful verification
- One-time use (invalidated after verification)
- Sensitive field (not exposed in API responses)

**Error Handling:**
- Invalid token → "Invalid or expired verification token"
- Expired token → "Verification token has expired"
- Already verified → "Email already verified" (200 OK)

**Testing:**
```bash
# 1. Register new user
curl -X POST http://localhost:7890/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","email":"test@example.com","password":"password123"}'

# 2. Check console logs for verification link
# Look for: "Verification URL: http://localhost:5678/verify-email/{token}"

# 3. Verify email
curl http://localhost:7890/api/v1/auth/verify-email/{token}
# Response: {"message":"Email verified successfully"}

# 4. Resend verification (if needed)
curl -X POST http://localhost:7890/api/v1/auth/resend-verification \
  -H "Authorization: Bearer YOUR_JWT"
```

**Files Created:**
- `backend/pkg/email/service.go` - Email service
- `backend/pkg/middleware/email_verified.go` - Email verification middleware
- `frontend/src/app/(auth)/verify-email/[token]/page.tsx` - Verification page
- `frontend/src/components/email-verification-banner.tsx` - Notification banner

**Files Modified:**
- `backend/ent/schema/user.go` - Added verification fields
- `backend/pkg/api/handlers/auth.go` - Added verification endpoints
- `backend/cmd/api/main.go` - Initialized email service, registered routes
- `frontend/src/app/dashboard/layout.tsx` - Added verification banner

**Production Integration:**

**SendGrid Example:**
```go
// pkg/email/sendgrid.go
import "github.com/sendgrid/sendgrid-go"

func (s *Service) SendVerificationEmail(to, name, token string) error {
    from := mail.NewEmail(s.fromName, s.fromEmail)
    subject := "Verify your IndustryDB account"
    toEmail := mail.NewEmail(name, to)

    content := mail.NewContent("text/html", s.buildVerificationHTML(name, token))
    m := mail.NewV3MailInit(from, subject, toEmail, content)

    client := sendgrid.NewSendClient(s.sendGridAPIKey)
    _, err := client.Send(m)
    return err
}
```

**Environment Variables:**
```env
# Email Service
EMAIL_FROM=noreply@industrydb.io
EMAIL_FROM_NAME=IndustryDB
SENDGRID_API_KEY=SG.xxx
# or
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
```

**Future Enhancements:**
- HTML email templates with branding
- Email customization per user language
- Bulk verification reminder emails
- Admin: Force verify/unverify users
- Verification expiry reminders

## Environment Variables

Copy `.env.example` to `.env` and configure:

```env
# Database
DATABASE_URL=postgres://industrydb:localdev@db:5432/industrydb?sslmode=disable

# Redis
REDIS_URL=redis://redis:6379

# JWT
JWT_SECRET=your-secret-key-change-in-production

# Stripe
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...

# API
API_PORT=8080
FRONTEND_URL=http://localhost:5678
```

## Testing

```bash
# Backend tests
cd backend && go test ./... -v -cover

# Frontend tests
cd frontend && npm test

# Full test suite
make test

# Coverage target: 80% minimum
```

## Git Workflow

1. Use conventional commits:
   - `feat:` new feature
   - `fix:` bug fix
   - `docs:` documentation
   - `refactor:` code change
   - `test:` adding tests
   - `chore:` maintenance

2. Commit after each completed task
3. Push after significant milestones

Example:
```bash
git add -A
git commit -m "feat: add user registration endpoint

- Add User schema with Ent ORM
- Create register handler with JWT
- Add validation middleware
- Include unit tests"
```

## Safety Rules

**IMPORTANT:** When running autonomously, follow these rules:

### Never Execute Without Analysis
- `rm -rf` - Always list files first with `ls -la`
- `mv` - Verify source and destination exist
- `chmod/chown` - Understand impact first

### Protected Paths (Do NOT modify)
- `/etc/`, `/var/`, `/usr/`, `/boot/`
- `~/.ssh/`, `~/.gnupg/`
- `*.env`, `*.pem`, `*.key`

### Before Any Deletion
1. `pwd` - Confirm current directory
2. `ls -la [target]` - See what will be affected
3. Backup if important
4. Use `rm -v` (verbose) instead of `rm -f`

## Pricing Tiers

| Tier | Price | Leads/Month | Features |
|------|-------|-------------|----------|
| Free | $0 | 50 | Basic data |
| Starter | $49 | 500 | + Phone, Address |
| Pro | $149 | 2,000 | + Email, Social |
| Business | $349 | 10,000 | + API access |

## Industries Supported

**20 Industries** with comprehensive global coverage:

**Personal Care Services:**
- Tattoo Studios (`tattoo`)
- Beauty Salons (`beauty`)
- Barber Shops (`barber`)
- Nail Salons (`nail_salon`)
- Spas (`spa`)
- Massage Therapy (`massage`)

**Health & Fitness:**
- Gyms/Fitness Centers (`gym`)
- Dentists (`dentist`)
- Pharmacies (`pharmacy`)

**Food & Beverage:**
- Restaurants (`restaurant`)
- Cafes (`cafe`)
- Bars (`bar`)
- Bakeries (`bakery`)

**Automotive:**
- Car Repair (`car_repair`)
- Car Wash (`car_wash`)
- Car Dealers (`car_dealer`)

**Professional Services:**
- Lawyers (`lawyer`)
- Accountants (`accountant`)

**Retail:**
- Clothing Stores (`clothing`)
- Convenience Stores (`convenience`)

## Data Coverage
**Implemented:** 2026-01-29

IndustryDB contains **82,740 verified business leads** from **184 countries** across **20 industries**, sourced from OpenStreetMap.

### Global Statistics
- **Total Leads:** 82,740
- **Countries:** 184 (near-global coverage)
- **Industries:** 20 comprehensive categories
- **Average Quality Score:** 49.9/100
- **With Contact Info:** 68.5% (phone, email, or website)

### Top 10 Countries by Lead Count
1. 🇳🇱 **Netherlands:** 6,033 leads
2. 🇦🇹 **Austria:** 3,091 leads
3. 🇩🇪 **Germany:** 2,970 leads
4. 🇹🇼 **Taiwan:** 2,786 leads
5. 🇹🇷 **Turkey:** 2,574 leads
6. 🇭🇺 **Hungary:** 2,291 leads
7. 🇨🇱 **Chile:** 2,281 leads
8. 🇮🇪 **Ireland:** 2,259 leads
9. 🇨🇭 **Switzerland:** 2,254 leads
10. 🇦🇷 **Argentina:** 2,030 leads

### Regional Distribution
- **Europe:** 55.1% (45,600+ leads)
- **Asia:** 19.1% (15,800+ leads)
- **Americas:** 14.9% (12,300+ leads)
- **Oceania:** 6.8% (5,600+ leads)
- **Africa:** 4.1% (3,400+ leads)

### Colombia Market Data
- **Total Leads:** 922
- **Cities:** 150+
- **Top Industries:** car_repair (219), cafe (118), dentist (107), nail_salon (105), gym (93)
- **Top Cities:** Bogotá (372), Medellín (206), Cali (137)

### Data Quality
- **With Email:** 9.4% (7,810 leads)
- **With Phone:** 34.4% (28,453 leads)
- **With Website:** 28.0% (23,178 leads)
- **Complete Address:** 84.5% (70,000+ leads)
- **Geolocation:** 100% (latitude/longitude for all leads)

**For detailed import statistics, see [DATA_IMPORT_REPORT.md](/DATA_IMPORT_REPORT.md)**

## Performance & Caching
**Implemented:** 2026-01-28

Multi-layer caching and query optimization ensure fast response times with 100K+ leads.

### Backend Caching
**Redis Caching** with strategic TTL values:
- Industry list: 1 hour (rarely changes)
- Sub-niche counts: 15 minutes (lead counts change frequently)
- Lead search results: 5 minutes (balance freshness vs performance)

**Implementation:**
- `backend/pkg/cache/redis.go` - Enhanced cache client with pattern deletion, pipelines
- `backend/pkg/industries/service.go` - Industry data caching
- `backend/pkg/leads/service.go` - Lead search caching

**Key Features:**
- Pattern-based cache invalidation: `DeletePattern(ctx, "industries:*")`
- Pipeline operations: `GetMulti()`, `SetMulti()` for batch operations
- Automatic cache key generation from search parameters
- Context-aware timeouts

### Frontend Optimizations

**Performance Hooks** (`frontend/src/hooks/useVirtualization.ts`):
1. **Virtual Scrolling**: Render only visible items (90% reduction in DOM size)
2. **Infinite Scroll**: Load more results as user scrolls
3. **Debounced Search**: Delay execution until user stops typing (90% fewer API calls)
4. **Memoized Filtering**: Cache filtered results to avoid re-computation

**Usage Example:**
```typescript
import { useVirtualization, useDebouncedValue } from '@/hooks/useVirtualization';

function LeadList({ leads }) {
  const { containerRef, visibleItems, totalHeight, offsetY } = useVirtualization({
    items: leads,
    itemHeight: 120,
    bufferSize: 5,
  });

  return (
    <div ref={containerRef} className="h-screen overflow-auto">
      <div style={{ height: totalHeight, position: 'relative' }}>
        <div style={{ transform: `translateY(${offsetY}px)` }}>
          {visibleItems.map(lead => <LeadCard key={lead.id} lead={lead} />)}
        </div>
      </div>
    </div>
  );
}
```

### Database Optimization

**Comprehensive Indexes** on all searchable fields:
- Primary: `(industry, country)`, `(industry, country, city)`
- Sub-niche: `(industry, sub_niche)`, `(industry, country, sub_niche)`
- Filters: `email`, `phone`, `verified`, `quality_score`
- Industry-specific: `cuisine_type`, `sport_type`, `tattoo_style`

### Performance Targets
- API Response: <200ms (cached), <1s (uncached)
- Search Results: <1s for 10K+ results
- Industry List: <100ms (cached)
- Frontend Load: <3s Time to Interactive
- Cache Hit Rate: 80%+

**Documentation:** See [PERFORMANCE.md](/PERFORMANCE.md) for detailed benchmarks and optimization guide.

## Resources

- [Plan Document](/.claude/plans/keen-noodling-bird.md)
- [Anthropic Claude Code Best Practices](https://www.anthropic.com/engineering/claude-code-best-practices)
- [wshobson/commands](https://github.com/wshobson/commands) - Slash command examples

---

*Created: 2026-01-21*
*Project: IndustryDB (industrydb.io)*
