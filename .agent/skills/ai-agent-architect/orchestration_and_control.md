# Multi-Agent Orchestration & Documentation-Driven Control

## Orchestration Patterns

### 1. Sequential Pipeline
Agents execute in a fixed, ordered chain. The output of one agent is the input of the next.
```
[Requirement] → [Analyzer Agent] → [Design Agent] → [Code Agent] → [Test Agent] → [Done]
```
**Best for:** Deterministic workflows with clear stage gates (e.g., design → implement → test).

### 2. Parallel Fan-Out / Fan-In
A coordinator dispatches work to multiple specialized agents simultaneously and aggregates their results.
```
                    ┌─ [Frontend Agent] ─┐
[Coordinator] ──────┼─ [Backend Agent]  ─┼──── [Aggregator] → [Final Output]
                    └─ [Test Agent]     ─┘
```
**Best for:** Independent tasks that can be done concurrently (e.g., generating frontend and backend code in parallel).

### 3. Critic / Validator Pattern
An output from a generator agent is passed to a separate critic agent for quality review before delivery.
```
[Generator Agent] → [Output] → [Critic Agent] → [Approved / Rejected + Feedback]
                                                         ↓ (if rejected)
                                              [Generator Agent re-runs with feedback]
```
**Best for:** High-stakes outputs where correctness and quality are critical (e.g., security audits, API contract generation).

### 4. Router Pattern
A router agent reads the incoming request and dispatches it to the most appropriate specialist agent.
```
[Input] → [Router Agent] → [golang-expert] | [react-ts-expert] | [ui-ux-expert] | ...
```
**Best for:** General-purpose entry points where the type of work is not known upfront.

---

## Documentation-Driven Agent Control

### The Control Document Principle
An agent's behavior must be fully configurable through its documentation without any code changes. This is achieved by:

1. **Persona Documents** (`persona.md`): Control the agent's identity, philosophy, and default decision-making posture.
2. **Skill Documents** (`*.md`): Control the agent's domain knowledge and task execution procedures.
3. **Configuration Documents** (`config.md`): Control runtime parameters — output format, verbosity, scope limits, tooling constraints.
4. **Workflow Documents** (`.agent/workflows/*.md`): Control multi-step procedures the agent follows for specific triggered tasks.

### Agent Control Hierarchy
```
persona.md              ← WHO the agent is (values, constraints, defaults)
    └── core_skills.md  ← WHAT the agent knows
    └── [topic].md      ← HOW the agent handles specific domains
    └── config.md       ← HOW the agent behaves at runtime
        └── workflow.md ← WHAT STEPS the agent takes for a given task
```

### Adjusting Agent Behavior Without Code Changes
| Goal | Document to Modify |
|---|---|
| Change agent's role or focus | `persona.md` |
| Add domain knowledge | Add new `[topic].md` skill file |
| Change output format (JSON → Markdown) | `config.md` or skill's Output Specification section |
| Add a new multi-step task flow | Create new workflow in `.agent/workflows/` |
| Restrict agent scope | Add to `Anti-Patterns` in relevant skill file |
| Update decision logic | Update `Decision Rules` in relevant skill file |

### Workflow Document Standard Template
```markdown
# Workflow: [Workflow Name]
## Trigger
[When should an agent execute this workflow? e.g., "When a new component is requested"]

## Agents Involved
- [Agent 1] — [Role in this workflow]
- [Agent 2] — [Role in this workflow]

## Steps
1. [Agent 1] reads [input] and produces [output A]
2. [Agent 2] receives [output A] and produces [output B]
3. [Validation step]: [Agent/human] verifies [output B] against [success criteria]

## Success Criteria
[Measurable definition of a successful workflow completion]

## Failure Handling
[What happens if a step fails? Retry? Escalate? Use fallback?]
```
