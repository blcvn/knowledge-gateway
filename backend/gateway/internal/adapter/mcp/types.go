package mcp


type ToolRegistry struct{}
func (r *ToolRegistry) RegisterAll(tools ...Tool) {}
func (r *ToolRegistry) Register(tool Tool) {}
func (r *ToolRegistry) Count() int { return 0 }


type Schema struct {
    Type       string
    Properties map[string]Property
    Required   []string
}

type Property struct {
    Type        string
    Description string
    Enum        []string
    Items       *Property
    Default     any
}


type Dummy struct{}

