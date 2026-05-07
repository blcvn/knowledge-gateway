---
skill_id: SKILL-016
version: 1.0.0
status: active
priority: existing
group: Quản lý & Governance
created_at: 2026-04-24
---

# SKILL-016 · AI Agent Architecture & Orchestration

## Mô tả

Phân tích yêu cầu để xác định agent cần có, viết skill files chuẩn, thiết kế orchestration patterns, điều khiển agent qua tài liệu.

## Agents sử dụng

Meta-skill — dùng để quản lý toàn bộ hệ thống agent.

---

## Năng lực cốt lõi

### 1. Agent Decomposition

```
Nguyên tắc decompose agents:
├── Single Responsibility: Mỗi agent có 1 rõ ràng domain
├── Skill-based: Agent = tập hợp skills cần thiết cho domain đó
├── Minimal coupling: Agents giao tiếp qua docs và artifacts
└── Observable: Mỗi agent output là document/artifact có thể review

Agents trong Knowledge Gateway:
├── requirement-parser-agent    → Parse PRD → structured requirements
├── semantic-extractor-agent    → Extract entities, relations từ requirements
├── knowledge-graph-agent       → Build và query Neo4j graph
├── ui-schema-generator-agent   → KG → JSON UI Schema
├── ui-renderer-agent           → JSON Schema → React components
├── traceability-validator-agent → Validate trace: UI ↔ Requirements
├── doc-consistency-agent       → Ensure code ↔ doc consistency
└── qa-pipeline-agent           → End-to-end quality validation
```

### 2. Skill File Authoring Standard

```yaml
# Cấu trúc chuẩn của một skill file
---
skill_id: SKILL-NNN
version: 1.0.0
status: active | draft | deprecated
priority: P0 | P1 | P2 | P3 | existing
group: {nhóm chức năng}
created_at: YYYY-MM-DD
---

# SKILL-NNN · {Tên skill}

## Mô tả
{Mô tả ngắn gọn (2-3 câu) về mục đích và phạm vi}

## Agents sử dụng
- `agent-name-1`
- `agent-name-2`

## Năng lực cốt lõi
{Phân chia thành các nhóm con có đánh số}
{Code examples cụ thể, không abstract}
{Patterns và anti-patterns}

## Checklist
{Danh sách kiểm tra trước khi apply skill}
```

### 3. Orchestration Patterns

#### Sequential Pipeline

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Parser     │────▶│  Extractor   │────▶│  KG Builder  │
│   Agent      │     │   Agent      │     │    Agent     │
└──────────────┘     └──────────────┘     └──────────────┘
       ▼                    ▼                     ▼
  requirements.json   entities.json         graph populated

Use when: Pipeline stages with strict data dependencies
```

#### Fan-Out Pattern

```
                    ┌──────────────┐
            ┌──────▶│  NLP Agent   │
            │       └──────────────┘
┌────────┐  │       ┌──────────────┐
│ Parser │──┼──────▶│  LLM Agent   │──▶ Merge Results
│ Agent  │  │       └──────────────┘
└────────┘  │       ┌──────────────┐
            └──────▶│ Rule Agent   │
                    └──────────────┘

Use when: Multiple independent analyses on same input
```

#### Critic Pattern

```
┌──────────────┐         ┌──────────────┐
│  Generator   │────────▶│    Critic    │
│   Agent      │◀────────│    Agent     │
└──────────────┘  revise └──────────────┘
        │           ▲           │
        │           └───────────┘
        │            iterate (max 3x)
        ▼ approved
   final output

Use when: Output quality needs validation before proceeding
```

#### Router Pattern

```
                    ┌──────────────────┐
                    │  Router Agent    │
                    │ (classify input) │
                    └──────────────────┘
                     /        |        \
                    ▼         ▼         ▼
            ┌──────────┐ ┌────────┐ ┌──────────┐
            │  Simple  │ │Complex │ │ Ambiguous│
            │  Parser  │ │  LLM   │ │  Human   │
            └──────────┘ └────────┘ └──────────┘

Use when: Different inputs need different processing strategies
```

### 4. Documentation-Driven Control

```
Agent Control Philosophy:
━━━━━━━━━━━━━━━━━━━━━━━━

Agents are controlled by DOCUMENTS, not code:
├── Skill files define WHAT the agent can do
├── Workflow docs define HOW agents are orchestrated
├── Input artifacts define WHAT to process
└── Output artifacts define WHAT was produced

Benefits:
- Agents can be updated without code changes
- Behavior is auditable (review skill files)
- Non-engineers can understand and modify agent behavior
- Easy to add/remove/replace agents

Control flow via artifacts:
requirements.md → [parser-agent] → structured-requirements.json
                                         ↓
                              [extractor-agent] → entities.json
                                                      ↓
                                          [kg-agent] → neo4j graph
```

### 5. Agent Skill Matrix

```
Skill × Agent Assignment Matrix:
(copied from skills-catalog.md for quick reference)

SKILL-001 Arch Design  → ALL agents (foundation)
SKILL-002 UI/UX        → ui-schema-gen, ui-renderer
SKILL-003 API Design   → kg-agent, ui-schema-gen, qa
SKILL-004 Golang       → parser, extractor, kg, ui-schema-gen
SKILL-005 Graph DB     → kg-agent, traceability-validator
SKILL-006 React+TS     → ui-renderer, ui-schema-gen
SKILL-007 JSON Schema  → ui-renderer
SKILL-008 LLM Eng      → parser, extractor
SKILL-009 NLP          → parser, extractor
SKILL-010 Data Pipeline→ parser, extractor, kg, ui-schema-gen
SKILL-011 UI Testing   → qa-pipeline
SKILL-012 Go Testing   → qa-pipeline, traceability-validator
SKILL-013 Security     → qa-pipeline, doc-consistency
SKILL-014 DevOps       → doc-consistency (CI), qa-pipeline
SKILL-015 Doc Mgmt     → doc-consistency
SKILL-016 AI Arch      → ALL agents (meta-skill)
```

### 6. Agent Onboarding Process

```
Adding a new agent:
1. Identify the domain (what problem does this agent solve?)
2. List required skills from SKILL catalog
3. Define input artifacts (what does it consume?)
4. Define output artifacts (what does it produce?)
5. Write agent profile file in .agents/agents/{name}/profile.md
6. Map agent to skill files in .agents/agents/{name}/skills.yaml
7. Update Skill × Agent matrix in skills-catalog.md
8. Add to workflow-guide.md orchestration section
9. Write test cases: given [input artifact] expect [output artifact]
```

---

## Agent Profile Template

```markdown
# {Agent Name}

## Vai trò
{Một câu mô tả nhiệm vụ cốt lõi}

## Kỹ năng sử dụng
- SKILL-NNN: {reason}
- SKILL-NNN: {reason}

## Input
- `{artifact-name}.{ext}`: {mô tả format và nội dung}

## Output
- `{artifact-name}.{ext}`: {mô tả format và nội dung}

## Điều kiện thành công (Acceptance Criteria)
- [ ] {measurable criterion 1}
- [ ] {measurable criterion 2}

## Escalation
Khi nào cần human review:
- {condition 1}
- {condition 2}
```

## Checklist

- [ ] Agent decomposition theo Single Responsibility
- [ ] Skill mapping đầy đủ cho mỗi agent
- [ ] Input/output artifacts được document rõ ràng
- [ ] Orchestration pattern phù hợp được chọn
- [ ] Skill × Agent matrix được cập nhật
- [ ] Agent profile file tồn tại tại `.agents/agents/{name}/profile.md`
- [ ] Test cases cho agent behavior được viết
