# Agent Analysis & Decomposition

## Phase 1: Requirement & Codebase Analysis

### From Requirements (PRD / User Story)
1. **Intent Extraction:** Identify all distinct goals, user journeys, and system behaviors described in the requirement.
2. **Domain Clustering:** Group related intents into logical domains (e.g., "UI Generation," "Code Review," "API Testing," "Data Analysis").
3. **Complexity Assessment:** Evaluate if a domain is simple enough for one agent or complex enough to warrant multiple specialized sub-agents.
4. **Input/Output Contract Definition:** For each identified domain, define precisely:
   - **Input:** What data does this agent receive? (e.g., user story text, a file path, a JSON schema)
   - **Output:** What does it produce? (e.g., a React component, a test plan, a bug report)

### From Existing Codebase
1. **Code Structure Scanning:** Read directory structures, file naming patterns, and module boundaries to understand the system's domain model.
2. **Responsibility Mapping:** Identify which parts of the codebase perform which category of work (data access, business logic, API handling, UI rendering).
3. **Gap & Pain Point Identification:** Find areas with high complexity, low test coverage, repetitive patterns, or frequent bug clusters — these are prime candidates for dedicated agents.
4. **Agent Opportunity Matrix:**

| Codebase Area | Identified Pain | Proposed Agent | Agent Responsibility |
|---|---|---|---|
| `handlers/` | Boilerplate HTTP code | `api-handler-agent` | Generate standard CRUD handlers from schema |
| `components/` | Inconsistent UI patterns | `ui-consistency-agent` | Audit and fix components against design system |
| `*_test.go` | Missing test coverage | `test-writer-agent` | Generate table-driven tests for uncovered functions |

## Phase 2: Agent System Design

### Agent Catalog Document
After analysis, produce an **Agent Catalog** document that maps the full multi-agent system:
```markdown
# Agent Catalog: [Project Name]

## Agent: [agent-name]
- **Role:** [One-sentence description]
- **Triggers:** [What event or condition activates this agent?]
- **Inputs:** [What does it consume?]
- **Outputs:** [What does it produce?]
- **Skills Required:** [List of skill file paths this agent reads]
- **Success Criteria:** [How do we know the agent succeeded?]
- **Collaborates With:** [Other agents it hands off to or receives from]
```
