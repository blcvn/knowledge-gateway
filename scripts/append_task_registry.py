import os

SERVICES = ['sm-analytics', 'sm-auth', 'sm-connector', 'sm-project', 'sm-search']

for s in SERVICES:
    tdd_path = f"services/{s}/specs/tdd.md"
    if os.path.exists(tdd_path):
        with open(tdd_path, 'r') as f:
            content = f.read()
        
        if "Task Specs Registry" not in content:
            appendix = f"""
## Task Specs Registry

_To be populated during implementation._

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| TASK-{s.split('-')[1][:3].upper()}-001 | Implement Domain Models | Pending | P0 |
| TASK-{s.split('-')[1][:3].upper()}-002 | Implement Usecases | Pending | P0 |
| TASK-{s.split('-')[1][:3].upper()}-003 | Implement Adapters and Repositories | Pending | P0 |
| TASK-{s.split('-')[1][:3].upper()}-004 | Infrastructure and Telemetry setup | Pending | P1 |
"""
            with open(tdd_path, 'a') as f:
                f.write(appendix)
                print(f"Appended to {tdd_path}")
        else:
            print(f"{tdd_path} already has Task Specs Registry")

