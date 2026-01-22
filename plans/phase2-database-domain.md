# Phase 2: Database & Domain Layer

## Status: ✅ COMPLETED

This phase implements the database schemas and domain types that define the core data model.

## Overview

The database and domain layer consists of:
- **Database Schemas** (`internal/platform/db/ent/schema/`) - Ent ORM schema definitions
- **Domain Types** (`internal/domain/`) - Shared business logic types and constants

## Database Schemas

All schemas are defined using Ent ORM and located in `internal/platform/db/ent/schema/`.

### Core Entities

#### 1. Users (`user.go`)

User accounts with quota management.

**Fields:**
- `id` (UUID, PK)
- `email` (string, unique)
- `display_name` (string)
- `avatar_url` (string)
- `last_requested_at` (timestamp) - Quota anchor for rate limiting
- `created_at` (timestamp)

**Relationships:**
- One-to-many: `USER_IDENTITIES`
- One-to-many: `TUTORIALS`

---

#### 2. User Identities (`user_identity.go`)

OAuth provider mappings for multi-provider authentication.

**Fields:**
- `id` (UUID, PK)
- `user_id` (UUID, FK → USERS)
- `provider` (string) - "github" or "google"
- `provider_id` (string, unique) - External provider ID

**Relationships:**
- Many-to-one: `USERS`

---

#### 3. Projects (`project.go`)

GitHub repositories tracked by IssueSight.

**Fields:**
- `id` (UUID, PK)
- `gh_repo_id` (bigint, unique) - GitHub repository ID
- `owner_handle` (string)
- `repo_name` (string)
- `full_name` (string, unique) - "owner/repo"
- `language` (string)
- `created_at` (timestamp)

**Relationships:**
- One-to-many: `GITHUB_ISSUES`
- Many-to-many: `CONCEPTS` (via `PROJECT_CONCEPTS`)

---

#### 4. GitHub Issues (`github_issue.go`)

Issues fetched from GitHub, linked to projects.

**Fields:**
- `id` (UUID, PK)
- `project_id` (UUID, FK → PROJECTS)
- `issue_number` (int)
- `gh_issue_id` (bigint, unique) - GitHub issue ID
- `raw_data` (JSONB) - Cached GitHub JSON response
- `last_synced_at` (timestamp)

**Relationships:**
- Many-to-one: `PROJECTS`
- One-to-one: `TUTORIAL_CONTENTS` (unique constraint)

---

#### 5. Tutorial Contents (`tutorial_content.go`)

AI-generated context bridges (one per issue).

**Fields:**
- `id` (UUID, PK)
- `issue_id` (UUID, FK → GITHUB_ISSUES, unique)
- `title` (string)
- `markdown_body` (text) - The AI output
- `status` (string) - "PENDING", "COMPLETED", "FAILED"
- `created_at` (timestamp)
- `updated_at` (timestamp)

**Relationships:**
- One-to-one: `GITHUB_ISSUES`
- One-to-many: `TUTORIALS`
- Many-to-many: `CONCEPTS` (via `TUTORIAL_CONCEPTS`)

---

#### 6. Tutorials (`tutorial.go`)

Junction table tracking which users have unlocked which tutorial contents.

**Fields:**
- `id` (UUID, PK)
- `user_id` (UUID, FK → USERS)
- `content_id` (UUID, FK → TUTORIAL_CONTENTS)
- `is_original_requester` (boolean)
- `created_at` (timestamp)

**Relationships:**
- Many-to-one: `USERS`
- Many-to-one: `TUTORIAL_CONTENTS`

---

#### 7. Concepts (`concept.go`)

Reusable concept definitions for tagging and categorization.

**Fields:**
- `id` (UUID, PK)
- `slug` (string, unique) - e.g., "message-queues"
- `name` (string)
- `description` (text)

**Relationships:**
- Many-to-many: `PROJECTS` (via `PROJECT_CONCEPTS`)
- Many-to-many: `TUTORIAL_CONTENTS` (via `TUTORIAL_CONCEPTS`)
- Self-referential: `CONCEPT_RELATIONSHIPS` (parent/child)

---

#### 8. Concept Relationships (`concept_relationship.go`)

Hierarchical concept relationships.

**Fields:**
- `parent_id` (UUID, FK → CONCEPTS)
- `child_id` (UUID, FK → CONCEPTS)
- `rel_type` (string) - e.g., "subconcept_of"

**Relationships:**
- Many-to-one: `CONCEPTS` (as parent)
- Many-to-one: `CONCEPTS` (as child)

---

#### 9. Project Concepts (`project_concept.go`)

Junction table linking projects to concepts.

**Fields:**
- `project_id` (UUID, FK → PROJECTS)
- `concept_id` (UUID, FK → CONCEPTS)

---

#### 10. Tutorial Concepts (`tutorial_concept.go`)

Junction table linking tutorial contents to concepts.

**Fields:**
- `content_id` (UUID, FK → TUTORIAL_CONTENTS)
- `concept_id` (UUID, FK → CONCEPTS)

---

## Domain Types

Domain types are located in `internal/domain/` and provide shared business logic types.

### Files

- `user.go` - User domain types
- `repository.go` - Repository/project domain types
- `github.go` - GitHub-specific domain types
- `tutorial.go` - Tutorial domain types
- `constants.go` - Shared constants
- `errors.go` - Domain-specific errors

---

## Key Relationships Summary

### One-to-Many
- `USERS` → `USER_IDENTITIES`
- `USERS` → `TUTORIALS`
- `PROJECTS` → `GITHUB_ISSUES`
- `TUTORIAL_CONTENTS` → `TUTORIALS`
- `CONCEPTS` → `PROJECT_CONCEPTS`
- `CONCEPTS` → `TUTORIAL_CONCEPTS`
- `CONCEPTS` → `CONCEPT_RELATIONSHIPS`

### One-to-One
- `GITHUB_ISSUES` → `TUTORIAL_CONTENTS` (unique `issue_id` constraint)

### Many-to-Many
- `PROJECTS` ↔ `CONCEPTS` (via `PROJECT_CONCEPTS`)
- `TUTORIAL_CONTENTS` ↔ `CONCEPTS` (via `TUTORIAL_CONCEPTS`)
- `CONCEPTS` ↔ `CONCEPTS` (via `CONCEPT_RELATIONSHIPS`)

---

## Design Decisions

### Hybrid Schema (Structured + JSONB)

GitHub's API response is large and volatile. Instead of strictly normalizing every field:
- **Structured Columns:** `id`, `status`, `issue_number` (indexed for fast lookups)
- **JSONB:** `raw_data` (stored as-is for future flexibility without schema migrations)

### Unique Constraints

- `USERS.email` - One account per email
- `USER_IDENTITIES.provider_id` - One identity per provider ID
- `PROJECTS.full_name` - One project per "owner/repo"
- `PROJECTS.gh_repo_id` - One project per GitHub repo ID
- `GITHUB_ISSUES.gh_issue_id` - One issue per GitHub issue ID
- `TUTORIAL_CONTENTS.issue_id` - One tutorial per issue (unique constraint)
- `CONCEPTS.slug` - One concept per slug

### Quota Management

User quota tracking via `USERS.last_requested_at` timestamp for rate limiting without separate quota table.

---

## Migration

Ent handles migrations automatically. To generate migrations:

```bash
go generate ./internal/platform/db/ent
```

---

## Next Steps

With the database and domain layer complete, services can now:
1. Use type-safe database queries via Ent
2. Share domain types across services
3. Maintain data integrity through proper constraints

**See:** [Phase 3: Collector Service](./phase3-collector-service.md) for first service implementation.
