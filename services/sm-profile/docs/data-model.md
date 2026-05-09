---
id: DOC-S04
service: sm-profile
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-profile — Data Model

> **Database**: PostgreSQL

## Tables

### organization_settings

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK | |
| org_id | VARCHAR(36) | UNIQUE, NOT NULL | Organization |
| should_llm_filter | BOOL | DEFAULT false | Enable LLM content filter |
| filter_prompt | TEXT | | Custom filter prompt |
| include_items | TEXT[] | | Whitelist items (max 20) |
| exclude_items | TEXT[] | | Blacklist items (max 20) |
| google_drive_custom_key_enabled | BOOL | DEFAULT false | Custom Google OAuth |
| google_drive_client_id | TEXT | | |
| google_drive_client_secret | TEXT | | Encrypted |
| notion_custom_key_enabled | BOOL | DEFAULT false | Custom Notion OAuth |
| notion_client_id | TEXT | | |
| notion_client_secret | TEXT | | Encrypted |
| onedrive_custom_key_enabled | BOOL | DEFAULT false | Custom OneDrive OAuth |
| onedrive_client_id | TEXT | | |
| onedrive_client_secret | TEXT | | Encrypted |
| updated_at | TIMESTAMPTZ | NOT NULL | |

### user_profiles

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| org_id | VARCHAR(36) | PK | |
| user_id | VARCHAR(36) | PK | Composite PK |
| static_preferences | JSONB | DEFAULT '{}' | User-set preferences |
| dynamic_traits | JSONB | DEFAULT '{}' | Learned from memories |
| trait_version | INT | DEFAULT 0 | Trait update counter |
| updated_at | TIMESTAMPTZ | NOT NULL | |

## Index Strategy

| Index | Columns | Purpose |
|-------|---------|---------|
| idx_settings_org | org_id (UNIQUE) | Org lookup |
| idx_profile_user | (org_id, user_id) | Profile lookup (composite PK) |
