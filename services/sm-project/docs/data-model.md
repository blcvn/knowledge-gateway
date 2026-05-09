---
id: DOC-S04
service: sm-project
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-project — Data Model

> **Database**: PostgreSQL

## Tables

### spaces

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | VARCHAR(26) | PK | NanoID |
| name | VARCHAR(255) | | Space name |
| description | TEXT | | |
| org_id | VARCHAR(36) | NOT NULL, INDEX | Organization |
| owner_id | VARCHAR(36) | NOT NULL | Creator |
| container_tag | VARCHAR(100) | UNIQUE(org_id, container_tag) | Scope tag |
| visibility | VARCHAR(10) | DEFAULT 'private' | public/private/unlisted |
| is_experimental | BOOL | DEFAULT false | Feature flag |
| content_text_index | JSONB | DEFAULT '{}' | KnowledgeBase index |
| index_size | INT | | Index entry count |
| metadata | JSONB | | |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |

### spaces_to_members

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| space_id | VARCHAR(26) | FK → spaces, PK | |
| user_id | VARCHAR(36) | PK | Composite PK |
| role | VARCHAR(10) | DEFAULT 'viewer' | owner/admin/editor/viewer |
| metadata | JSONB | | |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |

## Index Strategy

| Index | Columns | Purpose |
|-------|---------|---------|
| idx_space_org | org_id | List by org |
| idx_space_tag | (org_id, container_tag) UNIQUE | Tag lookup |
| idx_member_user | user_id | User's spaces |
