import os
import re

SERVICES = [
    'zep-admin',
    'zep-core',
    'zep-graph',
    'zep-memory',
    'zep-search',
    'zep-thread',
    'zep-user'
]

def generate_tasks(service):
    tdd_path = f"services/{service}/specs/tdd.md"
    if not os.path.exists(tdd_path):
        print(f"Skipping {service}, no TDD found.")
        return
        
    with open(tdd_path, 'r') as f:
        content = f.read()
        
    tasks_dir = f"services/{service}/specs/tasks"
    os.makedirs(tasks_dir, exist_ok=True)
    
    prefix = service.split('-')[1][:3].upper()
    
    # 1. Domain Task
    task_1 = f"""---
id: TASK-{prefix}-001
title: Domain Models & Core Algorithms
service: {service}
status: Todo
priority: P0
created: 2026-05-11
---

# Domain Models & Core Algorithms

## Objective
Implement the core domain entities, value objects, and algorithms.

## Specs Mapping
Please refer to the following content from `specs/tdd.md` to implement this task:

```markdown
{content}
```

## Acceptance Criteria
- [ ] Domain models compile and have no external dependencies.
- [ ] Core algorithms are fully implemented and unit tested.
"""

    # 2. Usecase Task
    task_2 = f"""---
id: TASK-{prefix}-002
title: Usecases & Orchestration
service: {service}
status: Todo
priority: P0
created: 2026-05-11
---

# Usecases & Orchestration

## Objective
Implement the business logic orchestration and usecases.

## Specs Mapping
Please refer to the following content from `specs/tdd.md` to implement this task:

```markdown
{content}
```

## Acceptance Criteria
- [ ] Usecases implemented fulfilling the service overview.
- [ ] Usecases correctly coordinate domain logic and ports.
"""

    # 3. Repository Task
    task_3 = f"""---
id: TASK-{prefix}-003
title: Data Models & Repositories
service: {service}
status: Todo
priority: P0
created: 2026-05-11
---

# Data Models & Repositories

## Objective
Implement the storage and persistence adapters.

## Specs Mapping
Please refer to the following content from `specs/tdd.md` to implement this task:

```markdown
{content}
```

## Acceptance Criteria
- [ ] Database schema / migrations created.
- [ ] Repository implementations accurately query the data models.
"""

    # 4. gRPC and NATS Task
    task_4 = f"""---
id: TASK-{prefix}-004
title: gRPC Handlers & Events
service: {service}
status: Todo
priority: P0
created: 2026-05-11
---

# gRPC Handlers & Events

## Objective
Implement the external communication interfaces via gRPC and NATS.

## Specs Mapping
Please refer to the following content from `specs/tdd.md` to implement this task:

```markdown
{content}
```

## Acceptance Criteria
- [ ] Proto files defined/matched and gRPC handlers implemented.
- [ ] NATS publishers and subscribers correctly configured.
"""

    # 5. Infra & Observability
    task_5 = f"""---
id: TASK-{prefix}-005
title: Infrastructure & Observability
service: {service}
status: Todo
priority: P1
created: 2026-05-11
---

# Infrastructure & Observability

## Objective
Wire dependencies, configure the service, and setup telemetry.

## Specs Mapping
Please refer to the following content from `specs/tdd.md` to implement this task:

```markdown
{content}
```

## Acceptance Criteria
- [ ] Google Wire configured.
- [ ] Telemetry (OTel/Prometheus) and structured logging setup.
- [ ] Service compiles into a runnable binary.
"""

    tasks = [task_1, task_2, task_3, task_4, task_5]
    
    for i, t_content in enumerate(tasks):
        with open(f"{tasks_dir}/TASK-{prefix}-{i+1:03d}.md", 'w') as f:
            f.write(t_content)
            
    print(f"Generated 5 mapped tasks from TDD for {service}")

for s in SERVICES:
    generate_tasks(s)
