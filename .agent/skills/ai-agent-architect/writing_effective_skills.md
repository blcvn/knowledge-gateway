# Writing Effective Agent Skills

## What Makes a Skill File Effective?

A skill file is the agent's operating manual. It must be:
- **Precise:** Unambiguous language. The agent should have exactly one valid interpretation of every instruction.
- **Scoped:** Covers exactly what the agent needs to know — no more, no less. Cross-domain knowledge belongs in a shared library, not a specialized skill.
- **Actionable:** Every section must result in a concrete action or decision the agent can take, not vague guidance.
- **Verifiable:** Include success criteria or output formats so output quality can be assessed.

## Skill File Standard Structure

```markdown
# [Skill Name]

## Purpose
One paragraph describing what this skill enables the agent to do and why.

## Context & Prerequisites
- What the agent must know or have access to before applying this skill.
- Dependencies on other skills, tools, or data sources.

## Core Competencies
### [Competency Area 1]
- Detailed, actionable knowledge points
- Include code examples, templates, or checklists where applicable

### [Competency Area 2]
- ...

## Decision Rules
A set of explicit IF/THEN rules the agent applies:
- IF [condition] THEN [action]
- IF [edge case] THEN [fallback behavior]

## Output Specification
- **Format:** [JSON / Markdown / TypeScript / etc.]
- **Required Fields:** [List required output properties]
- **Quality Gates:** [Conditions the output must satisfy before delivery]

## Anti-Patterns (What NOT to do)
- Explicit list of behaviors the agent must avoid
```

## Skill Library Design Principles

### 1. Layered Skill Architecture
Organize skills in layers so agents can compose them:
```
skills/
├── [agent-name]/
│   ├── persona.md          # Identity, philosophy, operating principles
│   ├── core_skills.md      # Primary domain knowledge
│   ├── [domain_topic].md   # Specific deep-dive skill
│   └── [domain_topic].md   # Another specific deep-dive skill
└── shared/                 # Cross-agent reusable knowledge
    ├── code_conventions.md
    └── output_formats.md
```

### 2. Versioning Skills
- Skills are living documents. Use clear section headers and change notes.
- When a skill changes behavior significantly, update the "Decision Rules" section and annotate with `[Updated: YYYY-MM-DD]`.

### 3. Testing Skill Effectiveness
A skill is effective if:
- An agent given only that skill file can complete its task correctly without prompting for clarification.
- The output consistently meets the **Output Specification** defined in the skill.
- Edge cases defined in **Decision Rules** are handled correctly.
