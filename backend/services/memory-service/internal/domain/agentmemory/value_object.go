package agentmemory

type MemoryType string

const (
    MemTypePattern      MemoryType = "pattern"
    MemTypePreference   MemoryType = "preference"
    MemTypeArchitecture MemoryType = "architecture"
    MemTypeBug          MemoryType = "bug"
    MemTypeWorkflow     MemoryType = "workflow"
    MemTypeFact         MemoryType = "fact"
)

func IsValidType(t string) bool {
    switch MemoryType(t) {
    case MemTypePattern, MemTypePreference, MemTypeArchitecture, MemTypeBug, MemTypeWorkflow, MemTypeFact:
        return true
    }
    return false
}
