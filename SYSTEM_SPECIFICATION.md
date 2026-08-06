# Jobber - System Specification

## Document Purpose

This document describes the complete architecture, features, and business logic of the Jobber system—a job application tracking platform. It serves as the canonical reference for understanding how the system works without reading code.

**Audience:** Engineers, product managers, architects, and stakeholders

**Last Updated:** August 3, 2026 (Permanent Architectural Merge: Jobs and Applications unified into single Job entity with pipeline status. Migration 000034 completed.)

---

## 1️⃣ System Overview

### Problem Statement

Job seekers apply to multiple positions across different companies and struggle to:
- Track application status across many job postings
- Remember which resume version was submitted
- Monitor progress through interview stages
- Maintain notes and reminders for follow-ups
- Visualize their application pipeline

### Solution

Jobber is a **job application tracking platform** that provides centralized management of:
- Job postings from multiple companies
- Multiple resume versions
- Applications linking jobs and resumes
- Customizable application stages
- Timeline of application progress
- Comments and notes
- Reminders for follow-ups

### Users

**Primary User:** Individual job seekers managing their own application process

**Use Cases:**
- Track 10-100+ concurrent job applications
- Manage multiple resume versions (general, technical, managerial)
- Follow applications through multi-stage interview processes
- Set reminders for follow-ups, deadlines, and interviews
- Review historical timeline of each application

### High-Level Product Goals

1. **Single Source of Truth:** All application data in one place
2. **Visibility:** Clear view of where each application stands
3. **Organization:** Structured data vs. spreadsheet chaos
4. **Reminders:** Never miss follow-ups or deadlines
5. **History:** Audit trail of what happened and when

### Core Principles

#### Backend-First Architecture
- Backend owns all business logic
- Frontend is a thin presentation layer
- No computed state in frontend
- Backend provides derived fields (e.g., `stage_name`, not just `stage_id`)

#### State vs History Separation
- Entities represent **current state** (e.g., `current_stage_id`)
- Stages/comments represent **historical record** (what happened)
- State is never derived from history
- Backend always knows "what is," frontend never computes it

#### Modular Monolith
- Clean domain boundaries (applications, jobs, companies, resumes)
- Hexagonal architecture (ports/adapters)
- Repository pattern for data access
- Service layer for business logic
- Handler layer for HTTP

#### Data Ownership
- All data scoped to `user_id`
- Multi-tenancy at application level
- Users see only their own data

---

## 2️⃣ Domain Map

### Core Domains

#### Authentication (`modules/auth`)
**Responsibility:** User registration, login, JWT token management

**Boundaries:**
- Owns user credentials
- Manages access/refresh tokens
- Handles token rotation and revocation

**What It Owns:**
- User accounts
- Password hashing
- JWT tokens
- Refresh token storage

---

#### Companies (`modules/companies`)
**Responsibility:** Company information for job postings with aggregated statistics

**Boundaries:**
- Independent entity (can exist without jobs)
- Referenced by jobs (optional)
- User-scoped (user can track companies before applying)
- Provides enriched DTOs with application statistics

**What It Owns:**
- Company name
- Company location
- Company notes

**What It Computes (Derived Fields):**
- Applications count (total and active)
- Last activity timestamp (from applications, stages, comments)
- Derived status (idle, active, interviewing)

---

#### Jobs (`modules/jobs`) **[CORE AGGREGATE]**
**Responsibility:** Job posting details with unified application pipeline management

**Boundaries:**
- Can reference a company (optional)
- Owns application stages (replaces `job_stages` table; renamed from `application_stages`)
- Owns stage templates (reusable definitions)
- Owns current state and application history

**What It Owns:**
- Job title (required)
- Job source (LinkedIn, Indeed, etc.)
- Job URL
- Job-specific notes
- Optional company association
- Pipeline status: `'saved'` (wishlist) | `'applied'` | `'on_hold'` | `'offer'` | `'rejected'` | `'archived'` (default: `'saved'`)
- Applied date (`applied_at`, null if saved; server-stamped when transitioning out of 'saved', cleared on return to 'saved')
- Resume association (one of `resume_id` XOR `resume_builder_id`, not both)
- Current stage pointer (`current_stage_id`, nullable)
- Favorite flag (`is_favorite`)

---

#### Resumes (`modules/resumes`)
**Responsibility:** Resume version management

**Boundaries:**
- Independent entity
- Referenced by applications
- Can have multiple versions per user
- Has `is_active` flag for version management

**What It Owns:**
- Resume title (e.g., "Software Engineer - Technical")
- File URL (link to PDF/document)
- Active status

---

#### Comments (`modules/comments`)
**Responsibility:** Notes and comments on jobs and stages

**Boundaries:**
- Scoped to jobs
- Can be job-level or stage-level
- Belongs to a specific job (always)
- Optionally linked to a job_stage

**What It Owns:**
- Comment content
- Job association (required)
- Stage association (optional)
- Timestamp

---

#### Reminders (`modules/reminders`)
**Responsibility:** Future notifications and follow-ups

**Boundaries:**
- Scoped to jobs
- Optionally linked to job_stages
- Has `is_done` flag for completion tracking
- Time-based trigger (`remind_at`)

**What It Owns:**
- Reminder message
- Reminder timestamp
- Completion status
- Application/stage association

---

#### Tags (`modules/tags`)
**Responsibility:** Categorization and filtering

**Boundaries:**
- User-defined labels
- Polymorphic relations (can tag applications, jobs, companies)
- Optional color coding

**What It Owns:**
- Tag name
- Tag color
- Tag relations to entities

---

## 3️⃣ Core Entities

### User

**Purpose:** Represents a registered user of the system

**Key Fields:**
- `id` (UUID)
- `email` (unique)
- `name`
- `password_hash`
- `locale` (for internationalization)

**Relationships:**
- Has many: companies, jobs, resumes, applications, stage templates, tags, comments, reminders

**Ownership Rules:**
- All data in the system is scoped to user
- Users cannot see other users' data
- User deletion cascades to all owned entities

---

### Company

**Purpose:** Represents a company that posts jobs

**Key Fields:**
- `id` (UUID)
- `user_id` (owner)
- `name` (required)
- `location` (optional)
- `notes` (optional)

**Relationships:**
- Belongs to: User
- Has many: Jobs (optional - company can exist without jobs)

**Ownership Rules:**
- User can create companies proactively (before applying)
- Jobs can reference company or be standalone
- Company deletion sets `job.company_id` to null

---

### Job

**Purpose:** Represents a job posting

**Key Fields:**
- `id` (UUID)
- `user_id` (owner)
- `company_id` (optional FK)
- `title` (required)
- `source` (e.g., "LinkedIn", "Company Website")
- `url` (link to posting)
- `notes`
- `board_column` (pipeline stage: `wishlist` | `applied` | `interview` | `offer` | `rejected`, default: `wishlist`)

**Relationships:**
- Belongs to: User
- Belongs to: Company (optional)
- Has many: Applications

**Ownership Rules:**
- Can exist without company
- Referenced by applications (required)
- Job deletion cascades to applications

---

### Resume

**Purpose:** Represents a version of user's resume

**Key Fields:**
- `id` (UUID)
- `user_id` (owner)
- `title` (e.g., "Technical Resume", "Managerial Resume")
- `file_url` (link to document)
- `is_active` (version control flag)

**Relationships:**
- Belongs to: User
- Has many: Applications

**Ownership Rules:**
- Users can have multiple resume versions
- Applications reference specific resume version
- Resume deletion restricted if referenced by applications

---

### Job **[CORE AGGREGATE ROOT]** (Merged: formerly "Job" + "Application")

**Purpose:** Unified entity representing both a saved job and a submitted application in a single pipeline

**Key Fields:**
- `id` (UUID)
- `user_id` (owner)
- `company_id` (optional FK)
- `title` (required)
- `source` (e.g., "LinkedIn")
- `url` (job posting link)
- `notes` (optional user notes)
- `is_favorite` (boolean)
- `status` ("saved" | "applied" | "on_hold" | "offer" | "rejected" | "archived", default: "saved")
- `applied_at` (nullable timestamp; null = saved/wishlist; IS NOT NULL ↔ "is an application")
- `resume_id` (nullable FK to resumes)
- `resume_builder_id` (nullable FK to resume_builders)
- `current_stage_id` (nullable FK to job_stages)

**Relationships:**
- Belongs to: User, Company (optional)
- Has many: Job stages (formerly `application_stages`), Comments, Reminders
- Has one: Current stage (via `current_stage_id`)

**Ownership Rules:**
- Uniqueness: One row per job saved/applied (no duplicates per user)
- Current stage is denormalized for performance (source of truth)
- Status drives UI/business logic (not derived from history)

**State Representation:**
- `status`: Current pipeline state (mutable)
- `applied_at`: Marker for "application submitted" (null = saved card)
- `current_stage_id`: Pointer to active interview stage
- Job stages + Comments: Historical record (append-only)

---

### Stage Template

**Purpose:** Reusable stage definitions (e.g., "Phone Screen", "Onsite Interview")

**Key Fields:**
- `id` (UUID)
- `user_id` (owner)
- `name` (e.g., "HR Screen", "Technical Interview")
- `order` (suggested sequence)

**Relationships:**
- Belongs to: User
- Has many: Application stages (via template reference)

**Ownership Rules:**
- User-specific (each user defines their own templates)
- Reusable across multiple applications
- Cannot delete if referenced by active application stages

---

### Job Stage (Renamed: formerly "Application Stage")

**Purpose:** Represents a stage instance in a job's interview pipeline (append-only)

**Key Fields:**
- `id` (UUID)
- `job_id` (FK, formerly `application_id`)
- `stage_template_id` (FK)
- `status` ("pending" | "completed")
- `order` (sequence within job's pipeline)
- `started_at` (timestamp)
- `completed_at` (nullable timestamp)

**Relationships:**
- Belongs to: Job (formerly Application)
- References: Stage template

**Ownership Rules:**
- Append-only (stages are never updated after creation, only marked complete)
- Sequential order maintained via `order` field
- Current stage referenced by `job.current_stage_id`

**Business Logic:**
- New stage automatically becomes current
- Completing a stage updates `completed_at`
- Previous stage can be marked complete when new stage added

---

### Comment

**Purpose:** Notes attached to jobs or stages

**Key Fields:**
- `id` (UUID)
- `user_id` (owner)
- `job_id` (required FK, formerly `application_id`)
- `stage_id` (nullable FK to job_stages)
- `content` (text)

**Relationships:**
- Belongs to: User, Job
- Optionally belongs to: Job Stage

**Ownership Rules:**
- Always linked to job
- If `stage_id` is null: job-level comment
- If `stage_id` is set: stage-specific comment

**Scope Logic:**
- Job comments: `stage_id === null`
- Stage comments: `stage_id !== null`

---

### Reminder

**Purpose:** Future notifications for follow-ups

**Key Fields:**
- `id` (UUID)
- `user_id` (owner)
- `job_id` (required FK, formerly `application_id`)
- `stage_id` (nullable FK to job_stages)
- `remind_at` (future timestamp)
- `message`
- `is_done` (completion flag)

**Relationships:**
- Belongs to: User, Job
- Optionally belongs to: Job Stage

**Ownership Rules:**
- Always linked to job
- Can be stage-specific or job-wide
- Marked done manually by user

---

### Tag

**Purpose:** User-defined labels for categorization

**Key Fields:**
- `id` (UUID)
- `user_id` (owner)
- `name` (e.g., "urgent", "dream-job")
- `color` (hex code, optional)

**Relationships:**
- Belongs to: User
- Has many: Tag relations (polymorphic)

**Ownership Rules:**
- User-specific tags
- Can tag applications, jobs, companies via `tag_relations` table

---

### Tag Relation (Polymorphic Association)

**Purpose:** Links tags to jobs and companies

**Key Fields:**
- `id` (UUID)
- `tag_id` (FK)
- `entity_type` ("job" | "company")
- `entity_id` (UUID)

**Relationships:**
- Belongs to: Tag
- References: Job or Company (polymorphic; "application" type removed in migration 000034)

**Ownership Rules:**
- One tag can be applied to multiple entities
- One entity can have multiple tags

---

## 4️⃣ Feature Catalog

### 🔹 Feature: User Authentication

#### Purpose
Allow users to register, log in, and maintain secure sessions via JWT tokens.

#### User Flow
1. User registers with email, name, and password
2. System creates account and returns access + refresh tokens
3. User logs in with email + password
4. System validates credentials and returns tokens
5. User includes access token in API requests (Bearer token)
6. When access token expires, user refreshes using refresh token
7. User logs out, system revokes refresh tokens

#### Business Rules

**Allowed:**
- Register with unique email
- Log in with valid credentials
- Refresh tokens before expiry
- Log out (revokes all refresh tokens)

**Forbidden:**
- Register with duplicate email
- Log in with invalid credentials
- Use expired tokens
- Access others' data

**Edge Cases:**
- Access token expires: use refresh token
- Refresh token expires: re-login required
- Logout revokes all user's refresh tokens
- Password hashing uses bcrypt

#### API Endpoints

**POST /api/v1/auth/register**
- **What:** Create new user account
- **State Read:** None
- **State Modified:** Creates user, generates tokens, stores refresh token hash

**POST /api/v1/auth/login**
- **What:** Authenticate user
- **State Read:** User credentials
- **State Modified:** Generates new tokens, stores refresh token hash

**POST /api/v1/auth/refresh**
- **What:** Get new access token using refresh token
- **State Read:** Refresh token validity
- **State Modified:** Generates new token pair, rotates refresh token

**POST /api/v1/auth/logout**
- **What:** Revoke all user's refresh tokens
- **State Read:** User ID from access token
- **State Modified:** Marks all user's refresh tokens as revoked

#### Backend Logic

**Registration:**
1. Validate email format and uniqueness
2. Validate password strength (min length)
3. Hash password using bcrypt
4. Create user record
5. Generate JWT access + refresh tokens
6. Store hashed refresh token in database
7. Return user DTO + tokens

**Login:**
1. Lookup user by email
2. Compare password hash using bcrypt
3. If valid, generate JWT tokens
4. Store refresh token hash
5. Return user DTO + tokens

**Token Refresh:**
1. Verify refresh token signature
2. Check token not revoked in database
3. Generate new access + refresh tokens
4. Rotate refresh token (invalidate old, store new)
5. Return new tokens

**Logout:**
1. Extract user ID from access token
2. Mark all user's refresh tokens as revoked
3. Return success

#### Frontend Responsibilities

**What Frontend Renders:**
- Registration form (email, name, password)
- Login form (email, password)
- Error messages for invalid credentials

**What Frontend Triggers:**
- Registration API call
- Login API call
- Automatic token refresh when 401 received
- Logout action

**What Frontend Must NOT Compute:**
- Password validation rules (backend enforces)
- Token expiration logic (backend provides)
- User permissions (backend controls)

---

### 🔹 Feature: Company Management

#### Purpose
Track companies user is interested in or applying to, store company-specific notes.

#### User Flow
1. User creates company record (name, location, notes)
2. System saves company associated with user
3. User can list all their companies (paginated)
4. User can update company details
5. User can delete company (if not referenced by jobs)

#### Business Rules

**Allowed:**
- Create company with name (required)
- Add optional location and notes
- Update company details
- Delete company if no jobs reference it
- View own companies only

**Forbidden:**
- Create company without name
- View other users' companies
- Delete company referenced by jobs (FK constraint)

**Edge Cases:**
- Company name can duplicate (user may track "Apple" in different contexts)
- Deleting company with jobs sets `job.company_id = null`
- Company can exist without jobs (proactive tracking)

#### API Endpoints

**POST /api/v1/companies**
- **What:** Create new company with enriched response
- **State Read:** None
- **State Modified:** Creates company record
- **Returns:** CompanyDTO with aggregated statistics

**GET /api/v1/companies**
- **What:** List companies (paginated, sortable, enriched)
- **Query Params:** `limit`, `offset`, `sort_by` (name|last_activity|applications_count), `sort_dir` (asc|desc)
- **State Read:** User's companies with application aggregations
- **State Modified:** None
- **Returns:** Enriched CompanyDTO array with statistics

**GET /api/v1/companies/{id}**
- **What:** Get company details with enriched fields
- **State Read:** Specific company with statistics
- **State Modified:** None
- **Returns:** Enriched CompanyDTO

**GET /api/v1/companies/{id}/related-counts**
- **What:** Get counts of related jobs and applications (for delete warnings)
- **State Read:** Company's related data counts
- **State Modified:** None
- **Returns:** `{ jobs_count: int, applications_count: int }`

**PATCH /api/v1/companies/{id}**
- **What:** Update company with enriched response
- **State Read:** Company ownership check
- **State Modified:** Updates company fields
- **Returns:** Enriched CompanyDTO

**DELETE /api/v1/companies/{id}**
- **What:** Delete company (jobs will have company_id set to NULL)
- **State Read:** Company ownership check
- **State Modified:** Deletes company, orphans related jobs
- **Note:** Frontend warns users about related data using related-counts endpoint

#### Backend Logic

**Create:**
1. Validate user authenticated
2. Validate name not empty
3. Trim whitespace from name
4. Create company with user_id
5. Return company DTO

**List:**
1. Validate user authenticated
2. Parse pagination params (limit, offset)
3. Parse sorting params (sort_by, sort_dir)
4. Execute enriched query with:
   - LEFT JOIN jobs and applications
   - Aggregate applications_count and active_applications_count
   - Calculate last_activity_at from applications, stages, and comments
   - Derive company status based on application data
5. Apply sorting and pagination
6. Return enriched CompanyDTO array

**Update:**
1. Validate user owns company
2. Update only provided fields
3. Trim whitespace from name if updated
4. Return updated company DTO

**Delete:**
1. Validate user owns company
2. Check no jobs reference company (FK constraint)
3. Delete company record

#### Frontend Responsibilities

**What Frontend Renders:**
- Company list with enriched statistics cards
- Sorting controls (name, last activity, applications count)
- Context menu (⋮) on each card with Edit/Delete actions
- Status badges (idle, active, interviewing)
- Applications count and last activity timestamps
- Quick navigation to filtered applications
- Company creation/edit modal (shared component)
- Context-aware delete confirmation dialog
- Empty states for no companies or no applications

**What Frontend Triggers:**
- Create company API call
- Fetch enriched companies on page load with sorting params
- Update company on form submit
- Delete company after fetching related counts
- Navigate to filtered applications view

**What Frontend Must NOT Compute:**
- Application counts (backend aggregates)
- Company status (backend derives)
- Last activity timestamp (backend calculates)
- Whether company can be deleted (backend validates)
- Any statistics or derived fields

#### Companies Page UI Features (Implemented February 2026)

**Company Card Layout:**
- Company name (title)
- Location (if provided)
- Status badge (idle/active/interviewing) with color coding
- Context menu (⋮) in top-right corner
- Statistics section:
  - Total applications count
  - Active applications count
  - Last activity timestamp (relative)
- Notes preview (line-clamped)
- "View Applications" button (only if applications > 0)

**Sorting Controls:**
- By name (A-Z, Z-A)
- By last activity (newest first, oldest first)
- By applications count (most to least, least to most)
- Visual indicators for active sort and direction

**Context Menu Actions:**
- **Edit**: Opens modal pre-filled with company data
- **Delete**: Opens confirmation dialog with related data warnings

**Delete Confirmation Dialog:**
- Fetches related jobs and applications count
- Shows amber warning box if related data exists
- Specific messaging: "X jobs will be affected, Y applications will lose company reference"
- Safe delete message when no related data
- Prevents accidental deletions with context

**Empty States:**
- No companies: Friendly message with "Create your first company" CTA
- Company with no applications: Shows "No applications yet" in stats area

**Navigation:**
- Clicking "View Applications (N)" navigates to `/app/jobs?company_id={id}` (filtered view of company's jobs/applications)
- Provides filtered view of company's applications
- Companies page is no longer a dead-end

---

### 🔹 Feature: Job Management (Merged: formerly "Job" + "Application" features)

#### Purpose
Track job postings and applications in unified pipeline. Manage saved wishlist cards and submitted applications with interview stage tracking.

#### User Flow - Saving a Job
1. User saves job from browser extension or manually (creates job with `status='saved'`, `applied_at=NULL`)
2. User optionally links job to company
3. User adds job URL, source (LinkedIn, Indeed, etc.), notes
4. System saves job as wishlist card
5. Job appears on board in "Saved" column

#### User Flow - Applying to Job
1. User opens saved job card
2. User submits application: selects resume, optionally fills/confirms date
3. System transitions job: `status='applied'`, `applied_at=NOW()` (server-stamped)
4. Job moves from "Saved" to "Applied" column
5. User adds interview stages as they progress

#### User Flow - Managing Application
1. User views job on board (status shows in column)
2. User drags job to new column → updates `status` (on_hold/offer/rejected/archived)
3. When leaving `saved` → `applied_at` auto-set if not already set
4. When returning to `saved` → `applied_at` cleared to NULL
5. User adds stages, comments, reminders to track progress
6. User can archive or delete job

#### Kanban Board (Single Board: `/app/jobs`)
- **Columns:** Saved | Applied | On Hold | Offer | Rejected | Archived
- **Card Content:** Job title, company name, last activity, match score (if available), favorite flag
- **Drag-and-drop:** Moving card between columns updates `status` via `PATCH /jobs/:id`
- **Optimistic updates:** UI updates instantly, reverts on API failure
- **View toggle:** Board / List, persisted in localStorage

#### Business Rules

**Allowed:**
- Create job with title (required), default `status='saved'`, `applied_at=NULL`
- Optionally link to company
- Add source, URL, notes, description
- Set/update `status` to one of: `saved`, `applied`, `on_hold`, `offer`, `rejected`, `archived`
- Mark job as favorite (`is_favorite=true`)
- Apply job (transition out of `saved` → auto-stamp `applied_at`)
- Return job to saved (transition to `saved` → clear `applied_at`)
- Update job details (title, company, notes, etc.)
- Associate resume (one of `resume_id` XOR `resume_builder_id`)
- Delete job (cascades to stages, comments, reminders)
- View own jobs only

**Forbidden:**
- Create job without title
- Link to other users' companies
- View other users' jobs
- Set status to invalid value
- Have both `resume_id` and `resume_builder_id` set (constraint enforced)

**Edge Cases:**
- Job can exist without company
- Job can exist without resume (saved state)
- Job without resume can be applied (resume optional)
- Deleting company sets `job.company_id = null`
- Job URL can be any string
- Job source is free text
- Default `status='saved'`, `applied_at=NULL`
- "Is an application" predicate = `applied_at IS NOT NULL`

#### API Endpoints

**POST /api/v1/jobs**
- **What:** Create new job (saved card, no application)
- **Request Body:** `{ title, company_id?, source?, url?, notes?, description?, resume_id?, resume_builder_id? }`
- **Default:** `status='saved'`, `applied_at=NULL`
- **Returns:** JobDTO with enriched fields

**GET /api/v1/jobs**
- **What:** List jobs (paginated, filterable by status, sortable)
- **Query Params:**
  - `limit`, `offset` (pagination)
  - `status` (filter: empty/'active'=not archived, 'all', or exact status like 'applied')
  - `sort=field:dir` (field: created_at|title|company_name|last_activity|status|applied_at; dir: asc|desc)
- **Returns:** JobDTOs array with enriched fields (company_name, current_stage_id/name, last_activity_at, resume {id,name,type})
- **Backward Compat:** Empty status or status='active' → status != 'archived' (compatible with Chrome extension)

**GET /api/v1/jobs/{id}**
- **What:** Get job details with timeline and comments
- **Returns:** JobDTO + job_comments array + job_stages array (optionally split by job_comments/stage_comments)

**PATCH /api/v1/jobs/{id}**
- **What:** Update job fields or transition status
- **Request Body:** `{ title?, company_id?, source?, url?, notes?, status?, resume_id?, resume_builder_id?, is_favorite? }`
- **Logic:**
  - Leaving `saved` → if applied_at is NULL, set applied_at=NOW()
  - Entering `saved` → clear applied_at=NULL
- **Returns:** Updated JobDTO

**POST /api/v1/jobs/{id}/favorite**
- **What:** Toggle favorite flag
- **Returns:** Updated JobDTO with is_favorite toggled

**DELETE /api/v1/jobs/{id}**
- **What:** Delete job and cascade to stages, comments, reminders
- **Returns:** 204 No Content

#### Job Stages (formerly Application Stages)

**POST /api/v1/jobs/{id}/stages**
- **What:** Add interview stage to job
- **Request Body:** `{ stage_template_id, comment? }`
- **Logic:** Creates job_stage, updates job.current_stage_id, optionally creates inline comment
- **Returns:** JobStageDTO

**GET /api/v1/jobs/{id}/stages**
- **What:** List all stages for job (append-only history)
- **Returns:** Array of JobStageDTO

**PATCH /api/v1/jobs/{id}/stages/{stageId}**
- **What:** Mark stage complete
- **Request Body:** `{ status: 'completed' }`
- **Returns:** Updated JobStageDTO

**DELETE /api/v1/jobs/{id}/stages/{stageId}**
- **What:** Delete stage (rare, typically only pending stages)
- **Returns:** 204 No Content

#### Subscriptions

**Single "jobs" Limit (Tracked Cards, excluding archived):**
- Free: 25 tracked jobs
- Pro: 100 tracked jobs
- Enterprise: Unlimited

**Removed:** Separate applications limit/usage tracking

#### Analytics

**Computed over:** jobs WHERE applied_at IS NOT NULL (only applied jobs count as "applications")
- `total_applications`: COUNT WHERE applied_at IS NOT NULL
- `active_applications`: COUNT WHERE status IN ('applied', 'on_hold')
- `closed_applications`: COUNT WHERE status IN ('rejected', 'offer', 'archived') AND applied_at IS NOT NULL

**Funnel (GET /analytics/funnel):**
- First bucket "Applied" (stage_order 1) is derived directly from jobs (applied_at IS NOT NULL). It does NOT require stage templates or job_stages — a user with applications but no tracked stages still sees this bucket. Omitted when the user has zero applications.
- Subsequent buckets come from the user's stage_templates with `order > 1` (the same "got a response" convention as response_rate/sources), counting DISTINCT applied jobs that have a job_stage for that template.
- Stage templates with `order = 1` (e.g. a user-created "Applied" template) are NOT shown as a separate bucket — the jobs-derived bucket replaces them.
- Conversion/drop-off rates are computed between adjacent buckets (LAG over stage_order).

#### Backend Logic

**Create:**
1. Validate user authenticated, title not empty
2. If company_id provided, verify company exists and belongs to user
3. Default: status='saved', applied_at=NULL, is_favorite=false
4. Validate XOR resume: not both resume_id and resume_builder_id set
5. Create job record
6. Return enriched JobDTO

**List:**
1. Validate user authenticated
2. Parse pagination (limit, offset), status filter, sort params
3. Backward compat: empty/`active` status → status != 'archived'
4. Query jobs with enriched data (company_name, current_stage_name, last_activity_at)
5. Apply sort whitelist, pagination
6. Return paginated enriched DTOs

**Update:**
1. Validate user owns job
2. If status changing:
   - Leaving `saved` → apply_at = NOW() if NULL
   - Entering `saved` → applied_at = NULL
3. If company_id changed, verify new company belongs to user
4. Validate XOR resume
5. Return updated JobDTO

**Delete:**
1. Validate user owns job
2. Cascade delete: job_stages, comments, reminders
3. Return 204

#### Frontend Responsibilities

**What Frontend Renders:**
- Single board (`/app/jobs` with nav label "Applications")
- Kanban columns: Saved | Applied | On Hold | Offer | Rejected | Archived
- List view with pagination, filtering, sorting
- Job card: title, company, match score, last activity, favorite toggle
- Job detail page: full job info, stages timeline, comments, resume
- Create job modal (initializes as saved)
- Update job/status modal
- Apply confirmation dialog (resume selection)

**What Frontend Triggers:**
- Create job (saved)
- Fetch jobs list with status filtering
- Drag-drop to change status (PATCH)
- Update job details
- Mark favorite
- Delete job
- Add stages with inline comments
- Add/delete comments

**What Frontend Must NOT Compute:**
- Whether job can be deleted (backend enforces)
- Company validation (backend verifies)
- Last activity timestamp (backend calculates)
- Applied date auto-stamping (backend controls)

---

### 🔹 Feature: Resume Management

#### Purpose
Manage multiple resume versions, track which resume was used for each application.

#### User Flow
1. User creates resume record (title + file URL)
2. User marks resume as active/inactive for version control
3. System saves resume associated with user
4. User can list resumes (paginated)
5. User can update resume details
6. User can delete resume (if not referenced by applications)

#### Business Rules

**Allowed:**
- Create resume with title and file URL
- Toggle `is_active` flag
- Have multiple active resumes
- Update resume details
- Delete resume if no applications reference it
- View own resumes only

**Forbidden:**
- Create resume without title or URL
- View other users' resumes
- Delete resume referenced by applications (FK restrict)

**Edge Cases:**
- Multiple resumes can be active simultaneously
- File URL is not validated (any string accepted)
- Resume title is free text

#### API Endpoints

**POST /api/v1/resumes**
- **What:** Create new resume
- **State Read:** None
- **State Modified:** Creates resume record

**GET /api/v1/resumes**
- **What:** List resumes (paginated)
- **State Read:** User's resumes
- **State Modified:** None

**GET /api/v1/resumes/{id}**
- **What:** Get resume details
- **State Read:** Specific resume
- **State Modified:** None

**PATCH /api/v1/resumes/{id}**
- **What:** Update resume
- **State Read:** Resume ownership check
- **State Modified:** Updates resume fields

**DELETE /api/v1/resumes/{id}**
- **What:** Delete resume
- **State Read:** Resume ownership check
- **State Modified:** Deletes resume (if not referenced)

#### Backend Logic

**Create:**
1. Validate user authenticated
2. Validate title and file_url not empty
3. Default `is_active = true`
4. Create resume with user_id
5. Return resume DTO

**List:**
1. Validate user authenticated
2. Parse pagination params (limit, offset)
3. Query resumes where `user_id = current_user`
4. Return paginated results with total count

**Update:**
1. Validate user owns resume
2. Update only provided fields
3. Return updated resume DTO

**Delete:**
1. Validate user owns resume
2. Check no applications reference resume (FK restrict)
3. Delete resume record

#### Frontend Responsibilities

**What Frontend Renders:**
- Resume list with active/inactive badges
- Resume creation form
- Resume edit form
- Delete confirmation dialog

**What Frontend Triggers:**
- Create resume API call
- Fetch resumes on page load
- Update resume on form submit
- Toggle active status
- Delete resume on confirmation

**What Frontend Must NOT Compute:**
- Whether resume can be deleted (backend enforces)
- Default active status (backend provides)

---


---

### 🔹 Feature: Stage Templates

#### Purpose
Define reusable stage definitions that can be applied to multiple applications.

#### User Flow
1. User creates stage templates (e.g., "Phone Screen", "Technical Interview")
2. User assigns order numbers to suggest sequence (e.g., 0, 1, 2, 3...)
3. System saves templates for user
4. User can list all templates (paginated, sorted by order)
5. User can update template name or order
6. User can delete template (if not used by applications)
7. When creating application stage, user selects from templates dropdown
8. Templates displayed with their order number (e.g., "0. Applied", "1. Phone Screen")

#### Business Rules

**Allowed:**
- Create template with name and order
- Update template name or order
- Delete template if not referenced by application stages
- View own templates only
- Have duplicate names (user choice)

**Forbidden:**
- Create template without name
- Template name longer than 255 chars
- View other users' templates
- Delete template used by application stages (FK restrict)

**Edge Cases:**
- Order is a suggestion, not enforced in workflow
- Order is displayed directly in UI (no offset or transformation)
- Multiple templates can have same order (user responsibility)
- Template name uniqueness not enforced (user can duplicate)
- Updating template name/order doesn't affect existing application stages (they reference template ID)
- Frontend displays: `{order}. {name}` (e.g., "1. Phone Screen")

#### API Endpoints

**POST /api/v1/stage-templates**
- **What:** Create new stage template
- **State Read:** None
- **State Modified:** Creates template record

**GET /api/v1/stage-templates**
- **What:** List stage templates (paginated)
- **State Read:** User's templates
- **State Modified:** None

**PATCH /api/v1/stage-templates/{templateId}**
- **What:** Update stage template
- **State Read:** Template ownership check
- **State Modified:** Updates template fields

**DELETE /api/v1/stage-templates/{templateId}**
- **What:** Delete stage template
- **State Read:** Template ownership check
- **State Modified:** Deletes template (if not referenced)

#### Backend Logic

**Create:**
1. Validate user authenticated
2. Validate name not empty (trim whitespace)
3. Create template with user_id, name, order
4. Return template DTO

**List:**
1. Validate user authenticated
2. Parse pagination params (limit, offset)
3. Query templates where `user_id = current_user`
4. Order by `order` field, then created_at
5. Return paginated results with total count

**Update:**
1. Validate user owns template
2. Update name (trim whitespace) or order
3. Return updated template DTO

**Delete:**
1. Validate user owns template
2. Check no application stages reference template (FK restrict)
3. Delete template record

#### Frontend Responsibilities

**What Frontend Renders:**
- Stage template list ordered by `order` field ascending
- Visual order indicator (numbered badge showing `order` value)
- Template creation form with name and order inputs
- Template edit form (placeholder for future)
- Delete confirmation dialog
- Stage selection dropdown showing `{order}. {name}` format

**What Frontend Triggers:**
- Create template API call
- Fetch templates on page load
- Update template on form submit
- Delete template on confirmation

**What Frontend Must NOT Compute:**
- Whether template can be deleted (backend enforces FK)
- Default order (user provides)

---

### 🔹 Feature: Job Stages (Formerly "Application Stages")

#### Purpose
Track job progress through interview stages, maintain timeline of stage history.

#### User Flow
1. User opens job detail page
2. User clicks "Add new stage"
3. User selects stage template from dropdown
4. User optionally adds comment about the stage
5. System creates new stage with status "pending"
6. System updates job.current_stage_id to new stage
7. User can view timeline of all stages
8. User can mark stage as completed
9. System updates stage.completed_at timestamp

#### Business Rules

**Allowed:**
- Add stage to job using stage template
- Add optional comment when creating stage
- View all stages for a job
- Complete a stage (sets completed_at timestamp)
- Multiple stages can be pending simultaneously
- View own job stages only

**Forbidden:**
- Add stage without template
- Modify stage after creation (except completion)
- Delete stages (append-only)
- View other users' stages

**Edge Cases:**
- Adding stage updates `job.current_stage_id` to new stage
- Stages are append-only (never modified or deleted)
- Order is automatically assigned (count of existing stages)
- Completing stage does NOT automatically advance to next stage
- Multiple pending stages allowed (user may schedule interviews ahead)

#### API Endpoints

**POST /api/v1/jobs/{id}/stages**
- **What:** Add new stage to job
- **State Read:** Job, stage template, existing stages count
- **State Modified:** Creates stage, updates job.current_stage_id, optionally creates comment

**GET /api/v1/jobs/{id}/stages**
- **What:** List all stages for job
- **State Read:** Job's stages
- **State Modified:** None

**PATCH /api/v1/jobs/{id}/stages/{stageId}**
- **What:** Mark stage as completed
- **State Read:** Stage ownership via job
- **State Modified:** Updates stage.status to "completed", sets completed_at

**DELETE /api/v1/jobs/{id}/stages/{stageId}**
- **What:** Delete stage (rarely used, for cleanup)
- **State Read:** Stage ownership via job
- **State Modified:** Deletes stage, orphans associated comments

#### Backend Logic

**Add Stage:**
1. Validate user owns job
2. Validate stage template exists and belongs to user
3. Query existing stages to determine order (count)
4. Create job_stage with:
   - job_id
   - stage_template_id
   - status = "pending"
   - order = count of existing stages
   - started_at = now
5. Update job.current_stage_id to new stage ID
6. If comment provided, create comment linked to stage
7. Return JobStageDTO with stage name from template

**List Stages:**
1. Validate user owns job
2. Query job_stages for job
3. Join with stage_templates to get stage names
4. Order by `order` field ascending
5. Return JobStageDTO array with enriched stage_name

**Complete Stage:**
1. Validate user owns job
2. Validate stage belongs to job
3. Update stage.status = "completed"
4. Set stage.completed_at = now (or custom timestamp if provided)
5. Return updated JobStageDTO

#### Frontend Responsibilities

**What Frontend Renders:**
- Timeline showing stages in chronological order
- "Add new stage" button
- Stage selection modal with templates
- Optional comment input in stage modal
- Stage status badges (pending/completed)
- Complete button for pending stages

**What Frontend Triggers:**
- Add stage API call with template ID and optional comment
- Fetch stages on job detail page load
- Complete stage on button click

**What Frontend Must NOT Compute:**
- Current stage (backend provides via job.current_stage_id)
- Stage order (backend assigns)
- Stage name (backend provides via template)
- Whether stage can be completed (all stages can be marked complete)

---

### 🔹 Feature: Comments

#### Purpose
Add notes and context to jobs and stages, maintain conversation history.

#### User Flow
1. User views job detail page
2. User sees two comment areas:
   - Job comments (general notes)
   - Stage comments (in timeline)
3. User adds job-level comment (stage_id = null)
4. System saves comment linked to job
5. User adds stage-specific comment when creating stage
6. System saves comment linked to job + stage
7. User can delete comments

#### Business Rules

**Allowed:**
- Create comment on job (stage_id = null)
- Create comment on stage (stage_id set)
- Add comment when creating new stage (inline)
- Delete own comments
- View comments on own jobs
- Content can be any non-empty text (multi-line supported)
- Line breaks preserved in comment display

**Forbidden:**
- Create comment without content
- Comment on other users' jobs
- Edit existing comments (delete + recreate instead)
- View other users' comments

**Edge Cases:**
- Comment with stage_id = null: job-level
- Comment with stage_id set: stage-specific
- Backend embeds job-level comments in Job DTO (no separate fetch needed)
- Frontend uses embedded comments for job-level, fetches stage comments separately
- Deleting job cascades to comments
- Deleting stage sets comment.stage_id = null (orphans comment to job level)
- Multi-line content preserved with whitespace-pre-wrap CSS

#### API Endpoints

**POST /api/v1/comments**
- **What:** Create comment on job or stage
- **Request Body:** `{ job_id, stage_id?, content }`
- **State Read:** Job existence check
- **State Modified:** Creates comment record
- **Status:** ✅ IMPLEMENTED

**GET /api/v1/jobs/{id}/comments**
- **What:** List all comments for job (both job-level and stage-level)
- **State Read:** Job's comments
- **State Modified:** None
- **Status:** ✅ IMPLEMENTED (Note: Job-level comments also embedded in GET /api/v1/jobs/{id} response)

**DELETE /api/v1/comments/{id}**
- **What:** Delete comment
- **State Read:** Comment ownership check
- **State Modified:** Deletes comment record
- **Status:** ✅ IMPLEMENTED

#### Backend Logic

**Create Comment:**
1. Validate user authenticated
2. Validate content not empty (trim whitespace)
3. Validate job_id provided
4. If stage_id provided, verify stage exists and belongs to job
5. Create comment with user_id, job_id, stage_id (nullable), content
6. Return comment DTO

**List Comments:**
1. Validate job ID provided
2. Query all comments where job_id matches
3. Return all comments (both stage_id null and not null)
4. Frontend filters for display

**Delete Comment:**
1. Validate user owns comment
2. Delete comment record

**Inline Comment on Stage Creation:**
- When adding stage, if comment provided in request body:
  1. Create stage as normal
  2. Create comment with job_id and new stage_id
  3. Return success (comment creation errors are logged but don't block stage creation)
- **Status:** ✅ IMPLEMENTED via POST /api/v1/jobs/{id}/stages with optional "comment" field

#### Frontend Responsibilities

**What Frontend Renders:**
- Job comments section (uses embedded `job_comments` from job DTO)
- Stage comments in timeline (fetched separately or embedded with stages)
- Multi-line textarea (rows=3) for comment input in job details
- Optional single-line comment input in "Add Stage" modal
- Delete button per comment
- Comments displayed with preserved line breaks using `whitespace-pre-wrap` CSS class
- Real-time updates via React Query cache invalidation

**What Frontend Triggers:**
- Create comment API call (POST /api/v1/comments)
- Comments automatically loaded with job (embedded in DTO)
- Delete comment on confirmation (DELETE /api/v1/comments/{id})
- Add comment when creating stage (inline, included in stage creation request)

**What Frontend Must NOT Compute:**
- Comment ownership (backend provides via user_id)
- Backend enforces user can only delete own comments
- Comment scoping (backend provides pre-split job_comments in DTO)

**Backend Optimization:**
- Job-level comments embedded in Job DTO as `job_comments` field
- Eliminates separate API call for job comments
- Stage comments fetched separately or embedded with stages
- Frontend uses embedded data directly, no client-side filtering needed for job comments

---

### 🔹 Feature: Reminders

#### Purpose
Schedule future notifications for follow-ups, deadlines, or interviews.

#### User Flow
1. User views job
2. User creates reminder with:
   - Future timestamp (remind_at)
   - Message
   - Optional stage association
3. System saves reminder
4. User can mark reminder as done
5. User can view/filter reminders by done status

#### Business Rules

**Allowed:**
- Create reminder for job
- Optionally link reminder to stage
- Set future timestamp
- Mark reminder as done (is_done = true)
- View own reminders only

**Forbidden:**
- Create reminder without message or timestamp
- Set past timestamp (not enforced but illogical)
- Reminder on other users' jobs

**Edge Cases:**
- Reminder with stage_id = null: job-wide
- Reminder with stage_id set: stage-specific
- No automatic notifications (requires background worker)
- Marking done doesn't delete reminder (soft complete)
- Deleting job cascades to reminders
- Deleting stage orphans reminder (sets stage_id = null)

#### API Endpoints

**Note:** Reminder endpoints are defined in models but **not implemented in handlers/routes yet**.

**Expected Endpoints (if implemented):**
- POST /api/v1/reminders
- GET /api/v1/reminders
- PATCH /api/v1/reminders/{id}
- DELETE /api/v1/reminders/{id}

#### Backend Logic

**Create Reminder:**
1. Validate user authenticated
2. Validate message not empty
3. Validate remind_at timestamp provided
4. Validate job exists and belongs to user
5. If stage_id provided, verify stage belongs to job
6. Create reminder with is_done = false
7. Return reminder DTO

**List Reminders:**
1. Query reminders for user
2. Optional filter by is_done status
3. Optional filter by job_id
4. Order by remind_at ascending
5. Return reminder DTOs

**Mark Done:**
1. Validate user owns reminder
2. Update is_done = true
3. Return updated reminder DTO

**Delete:**
1. Validate user owns reminder
2. Delete reminder record

#### Frontend Responsibilities

**What Frontend Renders:**
- Reminder list (global or per-job)
- Reminder creation form
- Mark done checkbox
- Delete button

**What Frontend Triggers:**
- Create reminder API call
- Fetch reminders on page load
- Mark done on checkbox
- Delete reminder on confirmation

**What Frontend Must NOT Compute:**
- Reminder firing logic (requires backend background worker)
- Timezone conversion (store UTC, display in user locale)

**Current State:**
- Models and repository exist
- Handler/routes not yet implemented
- Feature planned but not exposed via API

---

### 🔹 Feature: Tags

#### Purpose
Categorize and filter applications, jobs, and companies with user-defined labels.

#### User Flow
1. User creates tag (e.g., "urgent", "remote", "dream-job")
2. User optionally assigns color
3. User applies tag to entities (applications, jobs, companies)
4. User filters by tags
5. User removes tags from entities
6. User deletes unused tags

#### Business Rules

**Allowed:**
- Create tag with name
- Optional color (hex code)
- Tag multiple entity types (polymorphic)
- Multiple tags per entity
- Same tag on multiple entities
- View own tags only

**Forbidden:**
- Create tag without name
- Tag name longer than 100 chars
- Tag other users' entities
- View other users' tags

**Edge Cases:**
- Tag name uniqueness per user (enforced by DB constraint)
- Color validation not enforced (any string accepted)
- Deleting tag cascades to tag_relations
- Deleting entity cascades to its tag_relations

#### API Endpoints

**Note:** Tag endpoints are defined in models but **not fully implemented in handlers/routes yet** (based on code inspection).

**Expected Endpoints (if implemented):**
- POST /api/v1/tags
- GET /api/v1/tags
- PATCH /api/v1/tags/{id}
- DELETE /api/v1/tags/{id}
- POST /api/v1/tags/{id}/apply (apply to entity)
- DELETE /api/v1/tag-relations/{id} (remove from entity)

#### Backend Logic

**Create Tag:**
1. Validate user authenticated
2. Validate name not empty (trim whitespace)
3. Check name uniqueness for user (DB constraint)
4. Create tag with optional color
5. Return tag DTO

**Apply Tag:**
1. Validate user owns tag
2. Validate entity exists and belongs to user
3. Validate entity_type is "job" or "company" (removed "application" in migration 000034)
4. Create tag_relation with entity_type and entity_id
5. Return success

**Remove Tag:**
1. Validate user owns tag_relation (via tag)
2. Delete tag_relation record

**Delete Tag:**
1. Validate user owns tag
2. Delete tag (cascades to tag_relations)

#### Frontend Responsibilities

**What Frontend Renders:**
- Tag list with color chips
- Tag creation form
- Tag multi-select on entity forms
- Tag filters on list pages
- Remove tag button per entity

**What Frontend Triggers:**
- Create tag API call
- Apply tag to entity
- Remove tag from entity
- Delete tag on confirmation

**What Frontend Must NOT Compute:**
- Which entities can be tagged (backend defines entity_type enum)
- Tag name uniqueness (backend enforces)

**Current State:**
- Models and repository exist
- Handler/routes not yet implemented
- Feature planned but not exposed via API

---

### 🔹 Feature: Stats Sharing (`modules/sharing`)

#### Purpose
Let users publish a read-only snapshot of their job search statistics (overview incl. rejected count + funnel) via a public link — a growth loop targeting job-seeker audiences on LinkedIn/Twitter.

Note: `rejected_applications` was added to the analytics overview (API + UI card) together with this feature — it is a subset of `closed_applications` (closed = rejected + offer + archived).

#### User Flow
1. User opens Analytics page and clicks "Share"
2. User sees a preview of exactly what will be published
3. User confirms → backend freezes current aggregates into a snapshot and returns a share link `/s/{token}`
4. User copies the link and posts it anywhere
5. Anyone opening the link sees the frozen stats (no auth required) with a "Track your job search with Jobber" CTA
6. User can list and revoke share links; a revoked link stops working immediately

#### Business Rules

**Allowed:**
- Share on any plan (free included — growth feature, deliberately not plan-gated)
- Up to 20 active share links per user (`MaxActiveShares`)
- Multiple snapshots at different points in time

**Forbidden:**
- Personal identifiers in the public payload: no company names, resume titles, sources, user id, or email — aggregates only
- Guessable share URLs (token = 32 random bytes, base64url, 256-bit entropy)
- Search engine indexing of share pages (`noindex` meta + `X-Robots-Tag`)

**Edge Cases:**
- Snapshot is frozen at creation: later changes to jobs/stages do NOT update published shares
- Snapshot carries `schema_version` so future format changes can render old payloads
- Deleting a user cascades to their shares (FK `ON DELETE CASCADE`)
- Concurrent creates use single-statement check-and-insert against the cap
- Custom stage names ARE included in the funnel — the pre-publish preview is the user's consent step

#### API Endpoints

**Authenticated:**
- POST /api/v1/shares → freeze current stats, returns `ShareDTO {id, token, snapshot, created_at}` (409 `SHARE_LIMIT_REACHED` at cap)
- GET /api/v1/shares → list own shares
- DELETE /api/v1/shares/{id} → revoke (404 `SHARE_NOT_FOUND` if foreign/missing)

**Public (no auth, IP rate limit 30 req/min, key prefix `public_share`):**
- GET /api/v1/public/shares/{token} → `PublicShareDTO {snapshot, created_at}` (owner/id stripped)
- GET /api/v1/public/shares/{token}/preview-html → minimal HTML with Open Graph meta for social crawlers; humans hitting it directly are meta-refreshed to the SPA page `/s/{token}`

#### Backend Logic

**Create Share:**
1. Load overview + funnel from analytics repository (`StatsProvider` port, satisfied structurally)
2. Map into `StatsSnapshot` (schema_version=1, generated_at)
3. Generate token (crypto/rand 32 bytes → base64url, 43 chars)
4. Atomic insert while user is under `MaxActiveShares`, else `SHARE_LIMIT_REACHED`

**Storage:** `shared_stats` table (migration 000035): id UUID PK, user_id FK→users CASCADE, token TEXT UNIQUE, snapshot JSONB, created_at.

**OG preview:** nginx routes social-crawler user agents hitting `/s/{token}` to `preview-html`; og:title/description carry the headline numbers, og:image is the static `/og-image.png`.

#### Frontend Responsibilities

**What Frontend Renders:**
- Share button + modal on Analytics page (preview, create, copy link, list/revoke)
- Public page `/s/{token}` reusing Analytics presentational components, with CTA footer
- "Link revoked or not found" state for dead tokens

**What Frontend Must NOT Compute:**
- Snapshot contents (backend freezes them — frontend renders the returned snapshot verbatim)
- Token generation

---

## 5️⃣ State vs History (UNCHANGED PRINCIPLE - Updated Terminology)

### Current State Entities

**Purpose:** Represent "what is" right now

**Entities:**
- `Job.status` → Current pipeline status (saved|applied|on_hold|offer|rejected|archived)
- `Job.applied_at` → Was this job applied to? (null = saved, NOT null = application)
- `Job.current_stage_id` → Which interview stage is current?
- `JobStage.status` → Is stage pending or completed?
- `Reminder.is_done` → Is reminder complete?

**Characteristics:**
- Mutable (can be updated)
- Denormalized for performance
- Source of truth for queries like "show applied jobs in pipeline"
- Backend always provides current state, frontend never derives it

**Example:**
```json
{
  "id": "job-123",
  "status": "applied",
  "applied_at": "2024-01-15T10:00:00Z",
  "current_stage_id": "stage-456"
}
```
Frontend renders: "This job is in Applied stage, currently at 'Technical Interview' stage"

---

### Historical Record Entities

**Purpose:** Represent "what happened" over time

**Entities:**
- `JobStage` (renamed from ApplicationStage) → Timeline of interview stages (append-only)
- `Comment` → Timeline of notes (now job_id instead of application_id)
- `Reminder` → Scheduled future events (now job_id instead of application_id)

**Characteristics:**
- Append-only (stages never updated, only marked complete)
- Immutable timeline
- Never used to compute current state
- Used for display, audit trail, and user reference

**Example:**
```json
[
  { "stage": "Applied", "started_at": "2024-01-15", "completed_at": "2024-01-16" },
  { "stage": "Phone Screen", "started_at": "2024-01-20", "completed_at": "2024-01-20" },
  { "stage": "Technical Interview", "started_at": "2024-01-25", "completed_at": null }
]
```
Frontend renders: Timeline showing progression

---

### How State Transitions Are Handled

**Adding a Stage:**
1. Create new JobStage record (history)
2. Update Job.current_stage_id (state)
3. Both operations in same transaction

**Completing a Stage:**
1. Update JobStage.status = "completed" (history)
2. Set JobStage.completed_at (history)
3. Job.current_stage_id unchanged (still points to this stage)

**Transitioning Job Status:**
1. Update Job.status (e.g., saved → applied) (state)
2. If leaving 'saved': auto-stamp applied_at = NOW() if NULL
3. If returning to 'saved': clear applied_at = NULL
4. Job.current_stage_id unchanged (preserves last stage)
5. Historical stages remain intact

---

### Why History Is NOT Source of Truth

**Anti-Pattern (Avoided):**
```typescript
// BAD: Computing state from history
const currentStage = stages
  .filter(s => s.status === 'pending')
  .sort((a, b) => b.started_at - a.started_at)[0];
```

**Correct Pattern:**
```typescript
// GOOD: State provided by backend
const currentStageId = job.current_stage_id;
const currentStage = stages.find(s => s.id === currentStageId);
```

**Rationale:**
- State is denormalized for performance (no joins needed)
- Business logic in backend only (frontend just renders)
- Clear contract: backend decides "what is," frontend displays
- Avoids ambiguity: "pending stages" could be interpreted multiple ways

---

## 6️⃣ Backend-First Contracts

### What Backend Guarantees

#### 1. Computed Fields

Backend provides derived fields so frontend doesn't compute them.

**Example: Stage Names**
```json
{
  "id": "stage-123",
  "stage_template_id": "template-456",
  "stage_name": "Technical Interview"
}
```
Frontend receives `stage_name`, not just `stage_template_id`.

**Example: Current Stage**
```json
{
  "id": "app-123",
  "current_stage_id": "stage-789"
}
```
Frontend uses `current_stage_id` directly, doesn't compute from stages.

---

#### 2. Referential Integrity

Backend enforces data relationships.

**Foreign Key Constraints:**
- `Job.resume_id` → `Resume.id` (SET NULL on delete)
- `Job.resume_builder_id` → `ResumeBuilder.id` (SET NULL on delete)
- `Job.current_stage_id` → `JobStage.id` (SET NULL on delete)
- `JobStage.stage_template_id` → `StageTemplate.id` (RESTRICT on delete)
- `Comment.job_id` → `Job.id` (CASCADE on delete)
- `Reminder.job_id` → `Job.id` (CASCADE on delete)

**Business Rules:**
- Cannot delete resume if used by active jobs
- Cannot delete stage template if used by job stages
- Deleting job cascades to job_stages, comments, reminders

---

#### 3. Validation Rules

Backend validates all inputs.

**Examples:**
- Email format validation (auth)
- Password strength (auth)
- Name not empty (companies, jobs, resumes, stage templates)
- Status enum validation (application.status must be "active" or "closed")
- Stage status enum validation (must be "pending" or "completed")

---

#### 4. Pagination

All list endpoints return paginated results.

**Format:**
```json
{
  "items": [...],
  "pagination": {
    "limit": 20,
    "offset": 0,
    "total": 45
  }
}
```

**Defaults:**
- limit: 20
- offset: 0
- max limit: 100

---

### What Frontend Relies On

#### 1. DTOs Include All Display Data

Frontend never fetches additional data to display entities.

**Example: Application DTO includes job and resume titles:**
```json
{
  "id": "app-123",
  "job_id": "job-456",
  "job_title": "Senior Engineer",
  "resume_id": "resume-789",
  "resume_title": "Technical Resume"
}
```
Frontend displays "Senior Engineer" without fetching job.

---

#### 2. Derived Flags

Backend provides boolean flags for UI decisions.

**Current Implementation:**
- `stage.status` → "pending" | "completed" (frontend renders badge)
- `application.status` → "active" | "closed" (frontend renders indicator)

**Potential Future Additions:**
- `can_delete` → Boolean indicating if user can delete
- `is_current_stage` → Boolean indicating if stage is current
- `has_comments` → Boolean indicating if entity has comments

---

#### 3. Error Codes

Backend returns structured error codes, not just messages.

**Format:**
```json
{
  "error_code": "APPLICATION_NOT_FOUND",
  "error_message": "Application not found"
}
```

**Error Codes:**
- `VALIDATION_ERROR` (400)
- `UNAUTHORIZED` (401)
- `{ENTITY}_NOT_FOUND` (404)
- `INTERNAL_ERROR` (500)

---

### Examples of Derived Fields

**Job List Response (enriched):**
```json
{
  "items": [
    {
      "id": "job-123",
      "title": "Senior Backend Engineer",
      "company_id": "company-456",
      "company_name": "TechCorp",
      "resume_id": "resume-789",
      "resume": {
        "id": "resume-789",
        "title": "Technical Resume",
        "type": "pdf"
      },
      "current_stage_id": "stage-111",
      "current_stage_name": "Technical Interview",
      "status": "applied",
      "applied_at": "2024-01-15T10:00:00Z",
      "last_activity_at": "2024-01-25T14:30:00Z",
      "is_favorite": true,
      "job_comments": [
        {
          "id": "comment-1",
          "content": "Great company culture",
          "created_at": "2024-01-15T10:05:00Z"
        }
      ]
    }
  ],
  "pagination": { "limit": 20, "offset": 0, "total": 45 }
}
```

**What Frontend Does NOT Do:**
- Fetch company to get name (backend provides `company_name`)
- Fetch resume to get details (backend provides nested `resume` object)
- Fetch stage to get name (backend provides `current_stage_name`)
- Compute current stage (backend provides `current_stage_id`)
- Compute last activity (backend provides `last_activity_at`)
- Fetch job comments separately (backend provides `job_comments` embedded)

---

### How Ambiguity Is Avoided

#### 1. Explicit State Fields

No implicit state derivation.

**Good:**
```json
{
  "current_stage_id": "stage-123",
  "status": "active"
}
```

**Bad (Avoided):**
```json
{
  "stages": [...]
}
// Frontend computes: "Latest stage with status=pending is current"
```

---

#### 2. Backend Validates Enums

Status values are validated server-side.

**Allowed:**
- `application.status`: "active" | "closed"
- `stage.status`: "pending" | "completed"
- `tag_relation.entity_type`: "application" | "job" | "company"

**Forbidden:**
- Any other string values
- Null where not allowed

---

#### 3. DTOs Are Explicit

Backend returns full DTO, not raw database models.

**DTO Transformation (Job Entity):**
```go
// Domain model (merged Job + Application)
type Job struct {
    ID string
    UserID string
    Title string
    CompanyID *string
    ResumeID *string
    ResumeBuilderId *string
    CurrentStageID *string
    Status string // saved|applied|on_hold|offer|rejected|archived
    AppliedAt *time.Time
    IsFavorite bool
}

// Enriched DTO (what API returns)
type JobDTO struct {
    ID string `json:"id"`
    Title string `json:"title"`
    CompanyID *string `json:"company_id,omitempty"`
    CompanyName string `json:"company_name"`
    Resume *ResumeDTO `json:"resume,omitempty"`
    CurrentStageID *string `json:"current_stage_id,omitempty"`
    CurrentStageName string `json:"current_stage_name,omitempty"`
    Status string `json:"status"`
    AppliedAt *time.Time `json:"applied_at,omitempty"`
    LastActivityAt *time.Time `json:"last_activity_at"`
    IsFavorite bool `json:"is_favorite"`
    JobComments []CommentDTO `json:"job_comments,omitempty"`
}
```

---

## 7️⃣ Cross-Feature Interactions

### Job → Stages → Comments

**Flow:**
1. User creates job (optionally links resume)
2. User transitions job to 'applied' → auto-stamps applied_at
3. User adds stage to job
   - Creates `JobStage` record
   - Updates `Job.current_stage_id`
4. User optionally adds comment when creating stage
   - Creates `Comment` with job_id + stage_id
5. User adds job-level comment later
   - Creates `Comment` with job_id, stage_id = null

**Dependencies:**
- Stages belong to jobs (FK cascade)
- Comments belong to jobs (FK cascade)
- Comments optionally link to stages (FK set null on delete)

**Data Consistency:**
- Deleting job → deletes job_stages and comments
- Deleting job_stage → orphans stage comments (sets stage_id = null)
- Comments persist even if stage deleted (job-scoped)

---

### Job → Company

**Flow:**
1. User creates company (optional, can exist standalone)
2. User creates job, optionally links to company
3. User can apply to job, adding resume and stages

**Dependencies:**
- Job can exist without company
- Job can exist without resume (in saved state)
- Applying job requires resume (enforced at application time)

**Data Consistency:**
- Deleting company → sets `job.company_id = null` (jobs remain, orphaned)
- Deleting job → cascades to job_stages, comments, reminders
- Jobs always have valid user_id

**Company Statistics (Enriched DTOs):**
- Companies aggregate job data via SQL JOINs
- Query chain: `company → jobs` (where jobs have applied_at IS NOT NULL for "applications")
- Aggregations computed at query time:
  - `applications_count`: COUNT WHERE applied_at IS NOT NULL
  - `active_applications_count`: COUNT WHERE status IN ('applied', 'on_hold')
  - `last_activity_at`: MAX of job updates, job_stages, comments (where applied_at IS NOT NULL)
- Status derived from aggregated data:
  - `idle`: No applications (applied_at IS NULL jobs don't count)
  - `active`: Has active applications (status IN ('applied', 'on_hold'))
  - `interviewing`: Applied jobs with multiple stages (> 1 stage)

---

### Stage Templates → Application Stages

**Flow:**
1. User creates reusable stage templates
2. User adds stage to job, selects template
3. Job stage references template via `stage_template_id`
4. Template provides stage name, order suggestion

**Dependencies:**
- Job stages reference templates (FK restrict)
- Cannot delete template if used by job_stages

**Data Consistency:**
- Updating template name doesn't affect existing job_stages
- Stage records are immutable snapshots

---

### Reminders → Jobs/Stages

**Flow:**
1. User creates or applies to job
2. User adds reminder for future follow-up
3. Reminder links to job (required)
4. Reminder optionally links to stage (e.g., "Prepare for technical interview")

**Dependencies:**
- Reminders belong to jobs (FK cascade)
- Reminders optionally link to job_stages (FK set null on delete)

**Data Consistency:**
- Deleting job → deletes reminders
- Deleting job_stage → orphans stage reminders (sets stage_id = null)
- Reminders persist at job level

---

### Tags → Multiple Entities (Polymorphic)

**Flow:**
1. User creates tag (e.g., "urgent")
2. User applies same tag to job
3. User applies same tag to company

**Dependencies:**
- Tags polymorphically link to entities via `tag_relations`
- One tag can link to multiple entity types (job | company)
- "application" entity_type removed in migration 000034 (merged into jobs)

**Data Consistency:**
- Deleting tag → cascades to all tag_relations
- Deleting entity → cascades to its tag_relations
- Tags are independent across entity types

---

### Pagination Affects All List Operations

**All list endpoints return paginated results:**
- GET /api/v1/jobs (unified, supports status filtering)
- GET /api/v1/companies
- GET /api/v1/resumes
- GET /api/v1/stage-templates

**Format:**
```
GET /api/v1/jobs?limit=20&offset=40&status=applied
```

**Backend Logic:**
- Default limit: 20
- Max limit: 100
- Offset: defaults to 0
- Returns total count for pagination controls

**Frontend Interaction:**
- Fetch page 1 on load (offset=0)
- User clicks page 2 → fetch with offset=20
- Display total pages: Math.ceil(total / limit)

---

## 8️⃣ Permissions & Constraints

### Authentication & Authorization

**Authentication:**
- JWT-based (access token + refresh token)
- Access token: short-lived (configurable, default 15min)
- Refresh token: long-lived (configurable, default 7 days)
- Bearer token in Authorization header

**Authorization:**
- All endpoints (except auth) require authentication
- User ID extracted from JWT access token
- All data scoped to user_id
- No cross-user data access
- No admin/roles (single-tenant per user)

---

### Data Ownership Rules

**Who Can Do What:**

| Action | Allowed | Forbidden |
|--------|---------|-----------|
| Create entity | Own user only | For other users |
| Read entity | Own data only | Other users' data |
| Update entity | Own data only | Other users' data |
| Delete entity | Own data only | Other users' data |

**Enforcement:**
- All service methods accept `user_id` parameter
- Repositories filter by `user_id`
- Foreign keys enforce referential integrity

---

### Implicit Assumptions

1. **Single User Mode:**
   - No sharing/collaboration features
   - No team workspaces
   - No permissions beyond user-level

2. **English Only (for now):**
   - User has `locale` field (prepared for i18n)
   - Frontend not localized yet
   - Backend not localized yet

3. **No File Storage:**
   - Resume file_url is string (external URL)
   - No file upload feature
   - User stores files elsewhere (Google Drive, Dropbox, etc.)

4. **No Email/Notifications:**
   - Reminders stored but not sent
   - No email integration
   - No push notifications
   - User checks app manually

5. **No Background Jobs:**
   - Reminder firing requires background worker (not implemented)
   - No scheduled tasks
   - All operations synchronous

---

### Current Limitations

#### 1. ✅ Comments Fully Implemented (Updated)

**Status:**
- ✅ Models exist (Comment with job_id, stage_id)
- ✅ Repositories exist (CRUD operations)
- ✅ Handlers registered in main.go
- ✅ API routes exposed
- ✅ Full CRUD available
- ✅ Embedded job comments in Job DTO

**Available Endpoints:**
- POST /api/v1/comments (Create comment on job or stage)
- GET /api/v1/jobs/{id}/comments (List all comments for job)
- DELETE /api/v1/comments/{id} (Delete comment)
- Inline comment support in POST /api/v1/jobs/{id}/stages

#### 2. Reminders Handlers Not Implemented

**Status:**
- Models exist (Reminder)
- Repositories exist (CRUD operations)
- Handlers NOT created
- API routes NOT exposed

**Future Work:**
- Create reminder handler
- Register in main.go
- Add routes to API
- Implement notification system

---

#### 3. Tags Not Implemented

**Status:**
- Models exist (Tag, TagRelation)
- Repositories exist
- Handlers NOT created
- API routes NOT exposed

**Future Work:**
- Create tag handler
- Create tag relation handler
- Register in main.go
- Add routes to API

---

#### 4. No Filtering/Search

**Status:**
- List endpoints return all user's data (paginated)
- No query params for filtering
- No search functionality

**Future Work:**
- Add query params (e.g., `?status=active`)
- Add search (e.g., `?search=engineer`)
- Add sorting (e.g., `?sort=applied_at:desc`)

---

#### 5. No Aggregations/Statistics

**Status:**
- No dashboard endpoint
- No summary statistics (total applications, active, closed)
- No timeline aggregations

**Future Work:**
- Add dashboard endpoint
- Add statistics (counts, averages, trends)
- Add timeline summary

---

#### 6. Limited Validation

**Status:**
- Basic validation (required fields, length limits)
- No advanced validation (URL format, date ranges)
- No cross-field validation

**Future Work:**
- Validate URL format for job.url, resume.file_url
- Validate date ranges (applied_at not in future)
- Validate stage order (must be sequential)

---

#### 7. No Bulk Operations

**Status:**
- All operations single-item only
- No batch create/update/delete

**Future Work:**
- Bulk stage creation (apply template sequence)
- Bulk application updates (close multiple)
- Bulk tag application

---

## 9️⃣ Known Trade-offs & Technical Debt

### MVP Shortcuts

#### 1. Entity-Based Timeline (Not Event-Based)

**Current:**
- Timeline derived from entities (stages + comments)
- No explicit `events` table
- No event types, actors, metadata

**Why:**
- Simpler to implement
- Sufficient for manual user actions
- No integrations yet (no system events needed)

**Risks:**
- Not truly immutable (stages can be deleted)
- No audit trail for entity changes
- Cannot track automated actions

**Migration Path:**
- See ADR-001 in ARCHITECTURE_DECISIONS.md
- Phase 1 (v1.5): Add events table, dual-write pattern
- Phase 2 (v2.0): System events for integrations

**Trigger:**
- When adding integrations (email, calendar, ATS)
- When compliance/audit trail required

---

#### 2. Application-Scoped Comments (Not Generic)

**Current:**
- Comments have `application_id` + nullable `stage_id`
- Not polymorphic (entity_type + entity_id)

**Why:**
- Strong referential integrity (FK to applications)
- Better performance (indexed FK)
- Simpler queries
- Only 2 entity types need comments (application, stage)

**Risks:**
- Schema change needed per new entity type
- Doesn't scale beyond 3-5 entity types

**Migration Path:**
- See ADR-002 in ARCHITECTURE_DECISIONS.md
- Option A: Add explicit FK columns (job_id, company_id)
- Option B: Migrate to generic model (entity_type + entity_id)

**Trigger:**
- When ≥4 entity types need comments
- When building "universal notes" feature

---

#### 3. ✅ Backend-Split Comments (Implemented)

**Current Implementation:**
- ✅ Backend embeds application-level comments in Application DTO
- ✅ Frontend receives `application_comments` array directly
- ✅ No client-side filtering needed for application comments
- ✅ Stage comments can be fetched separately or embedded with timeline

**Benefits:**
- Eliminates client-side filtering for application comments
- Single API call for application + comments
- Better performance (no separate fetch)
- Fully "backend-first" approach
- Clear separation of concerns

**Remaining:**
- Stage comments still fetched separately (acceptable for performance)
- Could embed stage comments with stages in future if needed

**Status:** Migration to backend-split comments completed as of Feb 1, 2026

---

### Deferred Architectural Decisions

#### 1. Event Sourcing

**Deferred:**
- Full event-sourced architecture
- Events as source of truth
- Projections from events

**Why Deferred:**
- Significant complexity
- Requires event versioning strategy
- Requires team training
- Current approach sufficient

**When to Revisit:**
- Need time travel (replay to any point)
- Need what-if analysis (replay with changes)
- Need perfect audit trail (compliance)
- Phase 5 (v4.0) in roadmap

---

#### 2. Notification System

**Deferred:**
- Email notifications
- Push notifications
- Reminder firing

**Why Deferred:**
- Requires background workers
- Requires email service integration
- Requires frontend notification UI
- MVP doesn't require it

**When to Revisit:**
- User feedback requests notifications
- Reminder feature needs to be actionable

---

#### 3. Real-Time Updates

**Deferred:**
- WebSocket connections
- Real-time timeline updates
- Collaborative editing

**Why Deferred:**
- Single-user system (no collaboration)
- Polling sufficient for now
- Additional infrastructure complexity

**When to Revisit:**
- Adding team/workspace features
- Need instant updates across devices

---

#### 4. Advanced Search & Filtering

**Deferred:**
- Full-text search
- Faceted filtering
- Advanced query builder

**Why Deferred:**
- Pagination sufficient for MVP
- User data volume low (<100 applications typically)
- Basic filtering can be added incrementally

**When to Revisit:**
- Users have 100+ applications
- User feedback requests search
- Need to filter by multiple criteria

---

### Areas Intentionally Simplified

#### 1. No Soft Deletes

**Current:**
- Hard deletes (records removed from database)
- No `deleted_at` timestamp
- No "undo delete" feature

**Why:**
- Simpler queries (no `WHERE deleted_at IS NULL` everywhere)
- Smaller database (no zombie records)
- GDPR-friendly (data actually deleted)

**Trade-off:**
- Cannot restore deleted data
- No audit trail of deletions

**Future:**
- Could add soft deletes if undo feature requested
- Could add delete events to events table (Phase 1)

---

#### 2. No Optimistic Locking

**Current:**
- No `version` field on entities
- No concurrent edit detection
- Last write wins

**Why:**
- Single-user system (no concurrent editors)
- Low risk of conflicts

**Trade-off:**
- If two devices open, second save overwrites first

**Future:**
- Add `version` field if concurrent editing becomes issue

---

#### 3. No Database Migrations Rollback

**Current:**
- Migrations run on startup (automatic)
- No rollback mechanism
- No versioned migration history

**Why:**
- Simpler deployment
- Forward-only database changes
- Rollback rarely needed in practice

**Trade-off:**
- Cannot easily rollback schema changes

**Future:**
- Could use migration tool (golang-migrate, goose)
- Could add versioned migrations with rollback

---

#### 4. No Rate Limiting

**Current:**
- No API rate limits
- No throttling
- Open to abuse

**Why:**
- Single-user deployment context
- No public API
- Low abuse risk

**Trade-off:**
- Could be overwhelmed by accidental loops or malicious use

**Future:**
- Add rate limiting middleware if deployed publicly

---

### Risks If System Grows

#### 1. Timeline Query Performance

**Risk:**
- Fetching all stages + comments for application
- N+1 queries if not optimized
- Performance degrades with 50+ stages

**Mitigation:**
- Currently: Queries are simple (indexed FK)
- Future: Add eager loading, query optimization
- Future: Pagination for timeline if needed

---

#### 2. Comment Filtering at Scale

**Risk:**
- Fetching all comments and filtering client-side
- Performance degrades with 100+ comments

**Mitigation:**
- Currently: Acceptable for <50 comments per application
- Future: Backend pagination/filtering (Phase 4)

---

#### 3. No Caching

**Risk:**
- Every request hits database
- No caching layer (Redis used for refresh tokens only)

**Mitigation:**
- Currently: Queries are fast (indexed, user-scoped)
- Future: Add caching for read-heavy endpoints

---

#### 4. No Background Jobs

**Risk:**
- Reminders stored but not fired
- No scheduled tasks for email, notifications

**Mitigation:**
- Currently: Feature not active (users check manually)
- Future: Add background worker (cron, celery, etc.)

---

#### 5. Frontend Coupling to API Structure

**Risk:**
- Frontend tightly coupled to API DTOs
- API changes require frontend updates

**Mitigation:**
- Currently: Acceptable for monolith
- Future: Introduce API versioning (v2, v3)
- Future: GraphQL for flexible queries

---

## 🔟 Future Extension Points

### How Current Design Supports Future Features

The current architecture is designed to support evolution without rewrites. Below are planned extensions with clear implementation paths.

---

### 1. Event-Based Timeline

**Current State:**
- Entity-based timeline (stages + comments)
- No events table

**Extension Path:**

**Phase 1: Add Events Table**
```sql
CREATE TABLE application_events (
    id UUID PRIMARY KEY,
    application_id UUID NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    actor_id UUID,
    actor_type VARCHAR(20) DEFAULT 'user',
    payload JSONB,
    created_at TIMESTAMP NOT NULL
);
```

**Phase 2: Dual Write**
- Keep current entity updates (stages, comments)
- **Also** write corresponding events
- Timeline can read from either source

**Example:**
```go
func (s *JobService) AddStage(...) {
    // 1. Create stage (as before)
    stage := &JobStage{...}
    s.stageRepo.Create(ctx, stage)
    
    // 2. NEW: Log event
    event := &Event{
        JobID: jobID,
        EventType: "STAGE_ADDED",
        ActorID: userID,
        ActorType: "user",
        Payload: map[string]interface{}{
            "stage_id": stage.ID,
            "stage_name": template.Name,
        },
    }
    s.eventRepo.Create(ctx, event)
}
```

**Benefits:**
- Immutable audit trail
- Track system-generated events
- No breaking changes (additive only)

**Effort:** 2-3 days backend, 1 day frontend

**Trigger:** When integrations, compliance, or audit trail needed

---

### 2. System Events & Notifications

**Extension Path:**

**Phase 1: Expand Event Types**
```typescript
type EventType =
  | 'STAGE_ADDED' | 'COMMENT_ADDED'  // user actions
  | 'REMINDER_SENT' | 'EMAIL_OPENED'  // system actions
  | 'ATS_SYNC_COMPLETED' | 'CALENDAR_EVENT_CREATED'; // integrations
```

**Phase 2: Background Workers**
```go
// Cron job: Check reminders every minute
func (w *ReminderWorker) FireReminders(ctx context.Context) {
    reminders := w.repo.FindDue(ctx)
    for _, r := range reminders {
        // Send notification (email, push, etc.)
        // Log event
        w.eventRepo.Create(ctx, &Event{
            Type: "REMINDER_SENT",
            ActorType: "system",
            Payload: map[string]interface{}{
                "reminder_id": r.ID,
                "message": r.Message,
            },
        })
        // Mark reminder as done
        r.IsDone = true
        w.repo.Update(ctx, r)
    }
}
```

**Benefits:**
- Automated reminder notifications
- Email integrations
- System-generated timeline events

**Effort:** 3-4 days backend, 2-3 days frontend

**Trigger:** When automation or integrations added

---

### 3. Richer Collaboration

**Extension Path:**

**Phase 1: Add User Roles**
```sql
ALTER TABLE users ADD COLUMN role VARCHAR(20) DEFAULT 'user';
-- roles: 'user', 'coach', 'admin'
```

**Phase 2: Add Workspace/Team Concept**
```sql
CREATE TABLE workspaces (
    id UUID PRIMARY KEY,
    name VARCHAR(255),
    created_at TIMESTAMP
);

CREATE TABLE workspace_members (
    workspace_id UUID REFERENCES workspaces(id),
    user_id UUID REFERENCES users(id),
    role VARCHAR(20),
    PRIMARY KEY (workspace_id, user_id)
);

-- Scope entities to workspace
ALTER TABLE applications ADD COLUMN workspace_id UUID REFERENCES workspaces(id);
```

**Phase 3: Shared Comments**
```sql
ALTER TABLE comments ADD COLUMN visibility VARCHAR(20) DEFAULT 'private';
-- visibility: 'private', 'shared', 'public'
```

**Benefits:**
- Career coaches can view/comment on applications
- Team workspaces for job clubs
- Mentorship features

**Effort:** 2-3 weeks (major feature)

**Trigger:** User feedback requests collaboration

---

### 4. Advanced Filtering & Search

**Extension Path:**

**Phase 1: Query Parameters**
```
GET /api/v1/jobs?status=applied&company=TechCorp&stage=Interview
```

**Backend:**
```go
func (r *ApplicationRepository) List(ctx context.Context, userID string, filters Filters, limit, offset int) {
    query := "SELECT * FROM applications WHERE user_id = $1"
    args := []interface{}{userID}
    
    if filters.Status != "" {
        query += " AND status = $2"
        args = append(args, filters.Status)
    }
    // ... more filters
}
```

**Phase 2: Full-Text Search**
```sql
-- Add tsvector column
ALTER TABLE applications ADD COLUMN search_vector tsvector;

-- Create GIN index
CREATE INDEX idx_applications_search ON applications USING GIN(search_vector);

-- Update trigger
CREATE FUNCTION update_search_vector() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := to_tsvector('english', 
        COALESCE(NEW.notes, '') || ' ' || 
        COALESCE((SELECT title FROM jobs WHERE id = NEW.job_id), '')
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

**Benefits:**
- Find applications by keyword
- Filter by multiple criteria
- Advanced search UX

**Effort:** 1 week

**Trigger:** Users have 100+ applications, need to find specific ones

---

### 5. Integrations

**Extension Path:**

**Phase 1: Email Tracking**
```go
// Webhook from email service (SendGrid, Mailgun)
func (h *IntegrationHandler) EmailWebhook(c *gin.Context) {
    var webhook EmailWebhook
    c.BindJSON(&webhook)
    
    // Log event
    s.eventRepo.Create(ctx, &Event{
        ApplicationID: webhook.ApplicationID,
        EventType: "EMAIL_OPENED",
        ActorType: "integration",
        Payload: webhook.Data,
    })
}
```

**Phase 2: Calendar Integration**
```go
// Google Calendar API
func (s *ReminderService) CreateCalendarEvent(ctx context.Context, reminder *Reminder) error {
    event := &calendar.Event{
        Summary: reminder.Message,
        Start: &calendar.EventDateTime{DateTime: reminder.RemindAt},
    }
    _, err := s.calendarClient.Events.Insert("primary", event).Do()
    // Log event
    s.eventRepo.Create(ctx, &Event{
        EventType: "CALENDAR_EVENT_CREATED",
        ActorType: "integration",
    })
    return err
}
```

**Phase 3: ATS Integration (Greenhouse, Lever)**
```go
// Sync applications from ATS
func (s *ATSService) SyncApplications(ctx context.Context, userID string) error {
    atsApps := s.atsClient.GetApplications(userID)
    for _, atsApp := range atsApps {
        // Create or update local application
        // Log sync event
    }
}
```

**Benefits:**
- Email tracking in timeline
- Calendar sync for interviews
- ATS bi-directional sync

**Effort:** 1-2 weeks per integration

**Trigger:** User requests integration

---

### 6. Analytics & Dashboard

**Extension Path:**

**Phase 1: Add Dashboard Endpoint**
```go
type DashboardStats struct {
    TotalApplications int `json:"total_applications"`
    ActiveApplications int `json:"active_applications"`
    ClosedApplications int `json:"closed_applications"`
    
    AverageStages float64 `json:"average_stages"`
    AverageDaysToHire float64 `json:"average_days_to_hire"`
    
    TopCompanies []CompanyStat `json:"top_companies"`
    StageDistribution map[string]int `json:"stage_distribution"`
}

func (s *ApplicationService) GetDashboard(ctx context.Context, userID string) (*DashboardStats, error) {
    // Aggregate queries
}
```

**Phase 2: Timeline Aggregation**
```sql
-- Most active application periods
SELECT DATE_TRUNC('week', applied_at), COUNT(*)
FROM applications
WHERE user_id = $1
GROUP BY DATE_TRUNC('week', applied_at)
ORDER BY DATE_TRUNC('week', applied_at);
```

**Phase 3: Insights**
- "You're most successful with companies in X location"
- "Your average time to offer is Y days"
- "Your response rate is Z%"

**Benefits:**
- Visual dashboard
- Progress tracking
- Insights and recommendations

**Effort:** 1-2 weeks

**Trigger:** User wants analytics and trends

---

### 7. Mobile App

**Extension Path:**

**Phase 1: API Already Ready**
- REST API is device-agnostic
- JWT tokens work on mobile
- Pagination optimized for mobile

**Phase 2: Push Notifications**
```go
// FCM (Firebase Cloud Messaging)
func (s *NotificationService) SendPush(userID string, message string) error {
    tokens := s.getUserDeviceTokens(userID)
    for _, token := range tokens {
        s.fcmClient.Send(&messaging.Message{
            Token: token,
            Notification: &messaging.Notification{
                Title: "Reminder",
                Body: message,
            },
        })
    }
}
```

**Phase 3: Offline Support**
- Local storage on mobile
- Sync when online
- Optimistic UI updates

**Benefits:**
- Mobile access
- Push notifications
- Offline capability

**Effort:** 4-6 weeks (full mobile app)

**Trigger:** User requests mobile app

---

### Summary: Extension Readiness

| Feature | Current Support | Effort | Trigger |
|---------|----------------|--------|---------|
| Event-based timeline | Models ready | 2-3 days | Integrations, audit trail |
| System events | Event types expandable | 3-4 days | Automation |
| Notifications | Reminder model ready | 1-2 weeks | User requests |
| Collaboration | User model ready | 2-3 weeks | Career coaches |
| Search/filtering | Query patterns ready | 1 week | Large data volumes |
| Integrations | Webhook patterns ready | 1-2 weeks per | User requests |
| Analytics | Aggregation queries ready | 1-2 weeks | User wants insights |
| Mobile app | API device-agnostic | 4-6 weeks | User requests |

**Key Principle:** Build incrementally based on actual needs, not theoretical features.

---

## 📋 Appendix H: Recent Implementation Updates (February 2026)

### Companies Page Refactor

**Date:** February 1, 2026

**Overview:** Complete refactor of Companies List page with backend-first architecture, adding enriched DTOs, sorting, statistics, and improved UX.

#### Backend Enhancements

**CompanyDTO Enriched Fields:**
```go
type CompanyDTO struct {
    ID                      string     `json:"id"`
    Name                    string     `json:"name"`
    Location                *string    `json:"location,omitempty"`
    Notes                   *string    `json:"notes,omitempty"`
    CreatedAt               time.Time  `json:"created_at"`
    UpdatedAt               time.Time  `json:"updated_at"`
    ApplicationsCount       int        `json:"applications_count"`        // NEW
    ActiveApplicationsCount int        `json:"active_applications_count"` // NEW
    DerivedStatus           string     `json:"derived_status"`            // NEW
    LastActivityAt          *time.Time `json:"last_activity_at,omitempty"` // NEW
}
```

**Status Derivation Algorithm:**
```go
if applications_count == 0 → "idle"
else if max_stages > 1 → "interviewing"
else if active_applications_count > 0 → "active"
else → "idle"
```

**SQL Aggregation Pattern:**
- Uses CTE (Common Table Expression) to pre-compute application metrics
- LEFT JOINs: companies → jobs → applications
- Aggregates: COUNT(DISTINCT a.id), COUNT with FILTER
- Last activity: MAX(GREATEST()) across application updates, stages, comments
- Validates relationship chain: `a.user_id = j.user_id` (not `c.user_id`)

**New Endpoints:**
- `GET /api/v1/companies/:id/related-counts` - Returns jobs and applications counts for delete warnings

**Sorting Support:**
- Fields: `name`, `last_activity`, `applications_count`
- Directions: `asc`, `desc`
- Query params: `sort_by`, `sort_dir`

#### Frontend Features

**Company Cards Display:**
- Company name and location
- Status badge with color coding (idle=gray, active=green, interviewing=blue)
- Statistics: Total applications, Active applications
- Last activity timestamp (relative, e.g., "3 days ago")
- Notes preview (line-clamped to 2 lines)
- Quick action: "View Applications (N)" button

**Context Menu (⋮):**
- Edit action → Opens modal pre-filled with company data
- Delete action → Opens context-aware confirmation dialog

**Delete Confirmation:**
- Fetches related counts before showing dialog
- Shows warning if jobs or applications exist
- Specific messaging: "X jobs will lose company reference, Y applications affected"
- Prevents accidental data loss

**Sorting Controls:**
- Visual indicators for active sort
- Toggle between ASC/DESC on same field
- Persists sort state in component

**Empty States:**
- No companies: "Create your first company" with CTA
- Company with no applications: "No applications yet" message

**Navigation:**
- Clicking "View Applications" navigates to `/app/applications?company_id={id}`
- Provides filtered view of company's applications

#### Technical Details

**SQL Performance:**
- Single query returns all enriched data
- No N+1 queries
- Efficient aggregation with LEFT JOINs
- CTE pattern for complex calculations

**Frontend Architecture:**
- Zustand for state management
- React Query for data fetching
- No frontend computation of statistics
- Backend-first principle maintained

---

### Auth Persistence Implementation

**Date:** February 1, 2026

**Overview:** Implemented persistent authentication state across page refreshes using Zustand persist middleware and localStorage.

#### Problem Solved

Before: Users redirected to login on page refresh because auth state was stored in memory only.

After: Auth state persists across refreshes, with automatic token refresh on app load.

#### Implementation

**Auth Store with Persistence:**
```typescript
export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({ /* state */ }),
    { name: 'jobber-auth' } // localStorage key
  )
);
```

**localStorage Structure:**
```json
{
  "state": {
    "accessToken": "eyJhbGc...",
    "refreshToken": "eyJhbGc...",
    "user": { "id": "...", "email": "...", "name": "...", "locale": "en" },
    "isAuthenticated": true
  },
  "version": 0
}
```

**AuthProvider Component:**
- Wraps entire app (inside QueryClientProvider)
- Runs on app startup before rendering routes
- If a user is persisted, verifies the session via GET /api/v1/session (authenticated; the apiClient 401 interceptor transparently refreshes an expired access token and retries)
- Clears the session ONLY on an explicit 401/403 from the server; network failures, 404 and 5xx keep the user logged in (prevents logout on page reload during transient errors)
- Shows loading spinner during initialization
- Redirects to login if refresh fails

**Token Refresh Flow:**
1. App loads, AuthProvider checks localStorage
2. If refreshToken exists but accessToken expired → refresh
3. POST /api/v1/auth/refresh with refresh_token
4. Update localStorage with new tokens
5. Continue to app seamlessly

**API Client Enhancement:**
- Stores both access and refresh tokens
- On 401 response → attempts token refresh
- Retries original request with new token
- Clears auth and redirects to login if refresh fails

#### Benefits

✅ Users stay logged in across page refreshes  
✅ Automatic token refresh when expired  
✅ Seamless UX with loading states  
✅ Secure token rotation maintained  
✅ Multi-tab support via shared localStorage  

#### Security Considerations

- Tokens stored in localStorage (acceptable for this app's threat model)
- Refresh token rotation prevents replay attacks
- Backend still validates all tokens
- Users can manually logout to clear all tokens

---

## 📋 Appendix A: Complete API Endpoint Reference

### Authentication Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /api/v1/auth/register | No | Create new user account |
| POST | /api/v1/auth/login | No | Authenticate and get tokens |
| POST | /api/v1/auth/refresh | No | Refresh access token |
| POST | /api/v1/auth/logout | Yes | Revoke refresh tokens |

---

### Company Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /api/v1/companies | Yes | Create company (returns enriched DTO) |
| GET | /api/v1/companies | Yes | List companies (paginated, sortable, enriched) |
| GET | /api/v1/companies/{id} | Yes | Get company details (enriched) |
| GET | /api/v1/companies/{id}/related-counts | Yes | Get counts of related jobs/applications |
| PATCH | /api/v1/companies/{id} | Yes | Update company (returns enriched DTO) |
| DELETE | /api/v1/companies/{id} | Yes | Delete company (orphans related jobs) |

**Enriched CompanyDTO Fields:**
- `applications_count`: Total applied jobs (WHERE applied_at IS NOT NULL) across all company jobs
- `active_applications_count`: Applied jobs with status IN ('applied', 'on_hold')
- `derived_status`: "idle" | "active" | "interviewing"
- `last_activity_at`: Latest activity from applied jobs/job_stages/comments

**Sorting Support:**
- Query params: `sort_by` (name|last_activity|applications_count), `sort_dir` (asc|desc)
- Default: name ASC

---

### Job Endpoints (Unified: Job + Application)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /api/v1/jobs | Yes | Create job (saved card) |
| GET | /api/v1/jobs | Yes | List jobs (paginated, filterable by status, sortable) |
| GET | /api/v1/jobs/{id} | Yes | Get job details with timeline and comments |
| PATCH | /api/v1/jobs/{id} | Yes | Update job (auto-stamps applied_at on status transition) |
| DELETE | /api/v1/jobs/{id} | Yes | Delete job (cascades to stages, comments, reminders) |
| POST | /api/v1/jobs/{id}/favorite | Yes | Toggle favorite flag |

**Status Filter Backward Compatibility:**
- Empty status or `status=active` → `status != 'archived'` (Chrome extension compatibility)
- `status=all` → no filter
- `status={exact}` → exact match (saved|applied|on_hold|offer|rejected|archived)

**Sort Parameter:**
- Format: `sort=field:dir` (e.g., `sort=created_at:desc`)
- Fields: created_at, title, company_name, last_activity, status, applied_at
- Directions: asc, desc

---

### Resume Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /api/v1/resumes | Yes | Create resume |
| GET | /api/v1/resumes | Yes | List resumes (paginated) |
| GET | /api/v1/resumes/{id} | Yes | Get resume details |
| PATCH | /api/v1/resumes/{id} | Yes | Update resume |
| DELETE | /api/v1/resumes/{id} | Yes | Delete resume |

---

### Job Stage Endpoints (Renamed: formerly Application Stage)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /api/v1/jobs/{id}/stages | Yes | Add interview stage to job |
| GET | /api/v1/jobs/{id}/stages | Yes | List stages for job (append-only history) |
| PATCH | /api/v1/jobs/{id}/stages/{stageId} | Yes | Mark stage completed |
| DELETE | /api/v1/jobs/{id}/stages/{stageId} | Yes | Delete stage (rarely used for cleanup) |

---

### Stage Template Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /api/v1/stage-templates | Yes | Create stage template |
| GET | /api/v1/stage-templates | Yes | List stage templates (paginated) |
| PATCH | /api/v1/stage-templates/{templateId} | Yes | Update stage template |
| DELETE | /api/v1/stage-templates/{templateId} | Yes | Delete stage template |

---

### Comment Endpoints

| Method | Path | Auth | Description | Status |
|--------|------|------|-------------|--------|
| POST | /api/v1/comments | Yes | Create comment on job or stage | ✅ IMPLEMENTED |
| GET | /api/v1/jobs/{id}/comments | Yes | List comments for job | ✅ IMPLEMENTED |
| DELETE | /api/v1/comments/{id} | Yes | Delete comment | ✅ IMPLEMENTED |

**Note:** Job-level comments are also embedded in GET /api/v1/jobs/{id} response as `job_comments` field.

---

### System Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /health | No | Health check (Postgres, Redis) |
| GET | /ping | No | Simple ping (returns pong) |
| GET | /api/v1/session | Yes | Session check for the SPA; 401 triggers the client-side token refresh flow. Deliberately not under /auth/* (the frontend 401 interceptor skips refresh for auth endpoints) |
| GET | /swagger/index.html | No | Swagger UI (dev only) |

---

### Planned Endpoints (Not Implemented Yet)

**Reminders:**
- POST /api/v1/reminders (link to job, not applications)
- GET /api/v1/reminders
- PATCH /api/v1/reminders/{id}
- DELETE /api/v1/reminders/{id}

**Tags:**
- POST /api/v1/tags
- GET /api/v1/tags
- PATCH /api/v1/tags/{id}
- DELETE /api/v1/tags/{id}
- POST /api/v1/tags/{id}/apply (job|company only; applications removed in 000034)
- DELETE /api/v1/tag-relations/{id}

**Subscriptions:**
- GET /api/v1/subscriptions (unified "jobs" limit; applications limit removed)

**Dashboard:**
- GET /api/v1/dashboard

---

## 📋 Appendix B: Database Schema Summary

### Core Tables

**users**
- id, email (unique), name, password_hash, locale
- Owns: all other entities

**companies**
- id, user_id, name, location, notes
- Relationships: jobs (optional)

**jobs** (CORE AGGREGATE - Merged; formerly "Job" + "Application")
- id, user_id, company_id (nullable), title, source, url, notes, description, is_favorite
- status ('saved'|'applied'|'on_hold'|'offer'|'rejected'|'archived', default 'saved')
- applied_at (nullable; NOT NULL ↔ "is an application"), resume_id (nullable), resume_builder_id (nullable)
- current_stage_id (nullable FK to job_stages)
- Relationships: company (optional), job_stages (many, cascade), comments (many, cascade), reminders (many, cascade)

**resumes**
- id, user_id, title, file_url, is_active
- Relationships: jobs (many, set null on delete)

**stage_templates**
- id, user_id, name, order
- Relationships: job_stages (many, restrict delete)

**job_stages** (Renamed: formerly "application_stages", append-only)
- id, job_id, stage_template_id, status, order, started_at, completed_at
- Relationships: job (required, cascade), template (required, restrict delete)

**comments**
- id, user_id, job_id, stage_id (nullable to job_stages), content
- Relationships: job (required, cascade), job_stage (optional, set null on delete)

**reminders**
- id, user_id, job_id, stage_id (nullable to job_stages), remind_at, message, is_done
- Relationships: job (required, cascade), job_stage (optional, set null on delete)

**tags**
- id, user_id, name, color
- Relationships: tag_relations (polymorphic)

**tag_relations** (polymorphic; entity_type removed 'application' in 000034)
- id, tag_id, entity_type ('job'|'company'), entity_id
- Relationships: tag (required, cascade), job/company (required, cascade)

**refresh_tokens** (auth)
- id, user_id, token_hash, expires_at, revoked_at
- Relationships: user

---

### Foreign Key Cascade Rules

**Cascade Delete:**
- user → companies, jobs, resumes, stage_templates, tags, comments, reminders
- job → job_stages, comments, reminders
- tag → tag_relations

**Restrict Delete:**
- resume → jobs (cannot delete if used; enforce resume XOR resume_builder_id at application time)
- stage_template → job_stages (cannot delete if used)

**Set Null on Delete:**
- company → job.company_id
- resume → job.resume_id (SET NULL, was RESTRICT on applications)
- resume_builder → job.resume_builder_id (SET NULL)
- job_stage → comment.stage_id, reminder.stage_id
- job_stage → job.current_stage_id

---

## 📋 Appendix C: Error Code Reference

### HTTP Status Codes

| Code | Meaning | Usage |
|------|---------|-------|
| 200 | OK | Successful GET, PATCH, DELETE |
| 201 | Created | Successful POST |
| 400 | Bad Request | Validation error, invalid input |
| 401 | Unauthorized | Missing or invalid token |
| 404 | Not Found | Entity not found |
| 409 | Conflict | Duplicate entity (e.g., email exists) |
| 500 | Internal Server Error | Unexpected server error |

---

### Error Code Strings

**Authentication:**
- `UNAUTHORIZED` (401)
- `INVALID_CREDENTIALS` (401)
- `INVALID_EMAIL` (400)
- `INVALID_PASSWORD` (400)
- `USER_ALREADY_EXISTS` (409)

**Validation:**
- `VALIDATION_ERROR` (400)
- `INVALID_PAGINATION_PARAMS` (400)

**Entities Not Found:**
- `JOB_NOT_FOUND` (404)
- `COMPANY_NOT_FOUND` (404)
- `RESUME_NOT_FOUND` (404)
- `COMMENT_NOT_FOUND` (404)
- `REMINDER_NOT_FOUND` (404)
- `STAGE_TEMPLATE_NOT_FOUND` (404)
- `JOB_STAGE_NOT_FOUND` (404)

**Business Logic Errors:**
- `COMPANY_NAME_REQUIRED` (400)
- `JOB_TITLE_REQUIRED` (400)
- `RESUME_TITLE_REQUIRED` (400)
- `CONTENT_REQUIRED` (400)
- `STAGE_NAME_REQUIRED` (400)
- `INVALID_STATUS` (400)

**Generic:**
- `INTERNAL_ERROR` (500)

---

## 📋 Appendix D: Pagination Format

### Request

```
GET /api/v1/{resource}?limit={limit}&offset={offset}
```

**Query Parameters:**
- `limit` (optional, default: 20, max: 100)
- `offset` (optional, default: 0)

---

### Response

```json
{
  "items": [...],
  "pagination": {
    "limit": 20,
    "offset": 0,
    "total": 45
  }
}
```

**Fields:**
- `items`: Array of DTOs
- `pagination.limit`: Items per page (as requested)
- `pagination.offset`: Starting index (as requested)
- `pagination.total`: Total count of items (all pages)

---

### Frontend Calculation

```typescript
const totalPages = Math.ceil(pagination.total / pagination.limit);
const currentPage = Math.floor(pagination.offset / pagination.limit) + 1;
const hasNextPage = pagination.offset + pagination.limit < pagination.total;
const hasPrevPage = pagination.offset > 0;
```

---

## 📋 Appendix E: Authentication Flow

### Registration Flow

```
1. User submits: email, name, password
2. Backend validates:
   - Email format
   - Email uniqueness
   - Password strength
3. Backend hashes password (bcrypt)
4. Backend creates user record
5. Backend generates JWT tokens:
   - Access token (short-lived)
   - Refresh token (long-lived)
6. Backend stores refresh token hash in database
7. Backend returns:
   {
     "user": { id, email, name },
     "tokens": { access_token, refresh_token }
   }
8. Frontend stores tokens in localStorage via Zustand persist:
   - Key: "jobber-auth"
   - Persists: accessToken, refreshToken, user, isAuthenticated
```

---

### Login Flow

```
1. User submits: email, password
2. Backend looks up user by email
3. Backend compares password hash (bcrypt)
4. If valid:
   - Generate new JWT tokens
   - Store refresh token hash
   - Return user + tokens
5. If invalid:
   - Return 401 INVALID_CREDENTIALS
```

---

### Token Refresh Flow

```
1. Frontend detects access token expired (401 response)
2. Frontend retrieves refresh token from localStorage
3. Frontend sends refresh token to /api/v1/auth/refresh
4. Backend validates refresh token:
   - Signature valid
   - Not revoked in database
   - Not expired
5. If valid:
   - Generate new access token
   - Generate new refresh token (rotation)
   - Revoke old refresh token
   - Return new tokens
6. Frontend updates localStorage with new tokens
7. Frontend retries original request with new access token
8. If invalid:
   - Clear localStorage
   - Redirect to /login
```

**App Initialization:**
- On app load, AuthProvider checks localStorage for tokens
- If refresh token exists, attempts to refresh access token
- Shows loading spinner during token validation
- Seamlessly continues to app if valid, redirects to login if invalid

---

### Logout Flow

```
1. User clicks logout
2. Frontend clears localStorage (tokens and user data)
3. Frontend sends access token to /api/v1/auth/logout
3. Backend extracts user_id from token
4. Backend marks all user's refresh tokens as revoked
5. Backend returns success
6. Frontend clears tokens from storage
```

---

## 📋 Appendix F: Frontend-Backend Contract Examples

### Example 1: Job List (with status filter)

**Frontend Request:**
```
GET /api/v1/jobs?limit=10&offset=0&status=applied&sort=created_at:desc
Authorization: Bearer {access_token}
```

**Backend Response:**
```json
{
  "items": [
    {
      "id": "job-123",
      "title": "Senior Backend Engineer",
      "company_id": "company-456",
      "company_name": "TechCorp",
      "resume": {"id": "resume-789", "title": "Technical Resume", "type": "pdf"},
      "current_stage_id": "stage-111",
      "current_stage_name": "Technical Interview",
      "status": "applied",
      "applied_at": "2024-01-15T10:00:00Z",
      "last_activity_at": "2024-01-25T14:30:00Z",
      "is_favorite": false,
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-25T14:30:00Z"
    }
  ],
  "pagination": {
    "limit": 10,
    "offset": 0,
    "total": 1
  }
}
```

**Frontend Display:**
- Renders job list in kanban or table view
- Shows status badge in column (applied, on_hold, offer, etc.)
- Shows enriched data (company_name, resume details, last_activity)
- Pagination controls (1 of 1 pages)

---

### Example 2: Job Detail with Timeline and Comments

**Frontend Request:**
```
GET /api/v1/jobs/job-123
GET /api/v1/jobs/job-123/stages
GET /api/v1/jobs/job-123/comments
```

**Backend Responses:**

**Job (with embedded job_comments):**
```json
{
  "id": "job-123",
  "title": "Senior Backend Engineer",
  "company_id": "company-456",
  "company_name": "TechCorp",
  "status": "applied",
  "applied_at": "2024-01-15T10:00:00Z",
  "resume": {"id": "resume-789", "title": "Technical Resume", "type": "pdf"},
  "current_stage_id": "stage-111",
  "current_stage_name": "Technical Interview",
  "last_activity_at": "2024-01-25T14:30:00Z",
  "is_favorite": true,
  "job_comments": [
    {
      "id": "comment-1",
      "stage_id": null,
      "content": "Great company culture",
      "created_at": "2024-01-15T10:05:00Z"
    }
  ]
}
```

**Job Stages:**
```json
[
  {
    "id": "stage-100",
    "stage_name": "Applied",
    "status": "completed",
    "started_at": "2024-01-15T10:00:00Z",
    "completed_at": "2024-01-16T09:00:00Z"
  },
  {
    "id": "stage-111",
    "stage_name": "Technical Interview",
    "status": "pending",
    "started_at": "2024-01-20T14:30:00Z",
    "completed_at": null
  }
]
```

**Comments (all):**
```json
[
  {
    "id": "comment-1",
    "stage_id": null,
    "content": "Great company culture",
    "created_at": "2024-01-15T10:05:00Z"
  },
  {
    "id": "comment-2",
    "stage_id": "stage-111",
    "content": "Prepare LeetCode problems",
    "created_at": "2024-01-20T14:35:00Z"
  }
]
```

**Frontend Display:**
- Details block: Shows job-level comments from embedded `job_comments` array
- Timeline: Shows job_stages with stage-specific comments
- Current stage indicator (uses current_stage_id = stage-111)
- No need to filter comments (backend provides pre-split data)

---

### Example 3: Add Job Stage with Inline Comment

**Frontend Request:**
```
POST /api/v1/jobs/job-123/stages
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "stage_template_id": "template-456",
  "comment": "Scheduled for next Tuesday"
}
```

**Backend Logic:**
1. Create JobStage
2. Update Job.current_stage_id
3. Create Comment linked to job_stage (if comment provided)
4. Return stage DTO

**Backend Response:**
```json
{
  "id": "stage-222",
  "job_id": "job-123",
  "stage_template_id": "template-456",
  "stage_name": "Onsite Interview",
  "status": "pending",
  "order": 2,
  "started_at": "2024-01-25T10:00:00Z",
  "completed_at": null,
  "created_at": "2024-01-25T10:00:00Z"
}
```

**Frontend Actions:**
- Invalidates job_stages query (refetch)
- Invalidates comments query (refetch)
- Shows success toast
- Timeline updates automatically

---

## 📋 Appendix G: Architecture Principles Summary

### 1. Backend-First
- Backend owns all business logic
- Frontend is thin presentation layer
- No computed state in frontend

### 2. State vs History
- State: current_stage_id, status (mutable)
- History: stages, comments (append-only)
- State never derived from history

### 3. Modular Monolith
- Clean domain boundaries
- Hexagonal architecture (ports/adapters)
- Repository pattern for data access

### 4. User Data Ownership
- All data scoped to user_id
- Multi-tenancy at application level
- No cross-user access

### 5. Explicit Over Implicit
- Backend provides derived fields (stage_name)
- No frontend computation of business logic
- Clear contracts (DTOs, error codes)

### 6. Incremental Evolution
- Build what's needed now
- Clear extension points
- Documented migration paths

---

## 📄 Document Changelog

| Date | Version | Changes |
|------|---------|---------|
| 2026-08-03 | 2.0 | **PERMANENT ARCHITECTURAL MERGE:** Jobs and Applications unified into single Job entity. Migration 000034 completed. Status enum: saved\|applied\|on_hold\|offer\|rejected\|archived. applied_at markers "is an application" state. Tables renamed: application_stages→job_stages, comments/reminders application_id→job_id. Tag relations: removed 'application' entity_type (job\|company only). Subscriptions: unified "jobs" limit (free 25, pro 100). All endpoint paths updated: /api/v1/applications/* removed, /api/v1/jobs/* unified. Backend-first enriched DTOs now include company_name, current_stage_name, last_activity_at. State vs History principle preserved with updated terminology. |
| 2026-02-01 | 1.2 | Comments API fully implemented and documented, embedded application comments in DTO, stage numbering fix |
| 2026-02-01 | 1.1 | Added status change modal UI pattern, multi-line comment support with textarea |
| 2026-02-01 | 1.0 | Initial comprehensive specification |

---

**End of System Specification**

This document represents the complete canonical reference for the Jobber system as of August 3, 2026 (merged Job + Application architecture). All features, business logic, and architectural decisions are documented here for current and future team members.
