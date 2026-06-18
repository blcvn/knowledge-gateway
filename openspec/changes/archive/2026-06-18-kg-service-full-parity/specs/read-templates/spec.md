# read-templates

## ADDED Requirements

### Requirement: Graph-backed template execution
The system SHALL execute registered query templates through the selected graph backend adapter instead of iterating the write projection store directly.

#### Scenario: Execute a registered template through the graph adapter
- WHEN an actor executes an active template for a visible domain
- THEN the service SHALL compile the stored DSL into a graph query plan
- AND SHALL send the execution to the graph adapter
- AND SHALL return only results that satisfy ACL and lifecycle constraints.

### Requirement: Backend-enforced query safeguards
The system SHALL enforce ACL predicates, hop filtering, lifecycle rules, timeout handling, and row limits during graph-backed template execution.

#### Scenario: A template traverses multiple hops
- WHEN the template contains traversal hops
- THEN the graph execution SHALL apply ACL predicates at the start node and every hop
- AND SHALL omit nodes that fail lifecycle filtering
- AND SHALL stop when the configured row cap is reached
- AND SHALL return a timeout error when the configured deadline is exceeded.
