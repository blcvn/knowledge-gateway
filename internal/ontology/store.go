package ontology

import (
	"slices"
	"sync"
)

type DomainStore interface {
	CreateDomain(domain Domain) Domain
	GetDomain(id string) (Domain, bool)
	ListDomains() []Domain
	CreateVersion(version OntologyVersion)
	GetCurrentVersion(domainID string) (OntologyVersion, bool)
	CreateNodeType(schema NodeTypeSchema) NodeTypeSchema
	GetNodeType(domainID, nodeTypeName string) (NodeTypeSchema, bool)
	ListNodeTypes(domainID string) []NodeTypeSchema
	CreateRelType(schema RelTypeSchema) RelTypeSchema
	GetRelType(domainID, relTypeName, fromNodeType, toNodeType string) (RelTypeSchema, bool)
	ListRelTypes(domainID string) []RelTypeSchema
	CreateCrossDomainRule(rule CrossDomainRelRule) CrossDomainRelRule
	ListCrossDomainRules(fromDomainID string) []CrossDomainRelRule
	CreateQueryTemplate(template QueryTemplate) QueryTemplate
	GetQueryTemplate(domainID, templateName string) (QueryTemplate, bool)
	UpdateQueryTemplate(template QueryTemplate) (QueryTemplate, bool)
	ListQueryTemplates(domainID string) []QueryTemplate
	UpsertStatusFieldConfig(config StatusFieldConfig) StatusFieldConfig
	GetStatusFieldConfig(domainID string) (StatusFieldConfig, bool)
}

type MemoryStore struct {
	mu        sync.RWMutex
	domains   map[string]Domain
	versions  map[string]OntologyVersion
	nodeTypes map[string]NodeTypeSchema
	relTypes  map[string]RelTypeSchema
	rules     []CrossDomainRelRule
	templates map[string]QueryTemplate
	statuses  map[string]StatusFieldConfig
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		domains:   map[string]Domain{},
		versions:  map[string]OntologyVersion{},
		nodeTypes: map[string]NodeTypeSchema{},
		relTypes:  map[string]RelTypeSchema{},
		rules:     []CrossDomainRelRule{},
		templates: map[string]QueryTemplate{},
		statuses:  map[string]StatusFieldConfig{},
	}
}

func (s *MemoryStore) Seed(domains []Domain, versions []OntologyVersion, nodeTypes []NodeTypeSchema, relTypes []RelTypeSchema, rules []CrossDomainRelRule, templates []QueryTemplate, statuses []StatusFieldConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, domain := range domains {
		s.domains[domain.ID] = domain
	}
	for _, version := range versions {
		s.versions[version.DomainID] = version
	}
	for _, schema := range nodeTypes {
		s.nodeTypes[schema.ID] = schema
	}
	for _, schema := range relTypes {
		s.relTypes[schema.ID] = schema
	}
	s.rules = append([]CrossDomainRelRule(nil), rules...)
	for _, template := range templates {
		s.templates[template.ID] = template
	}
	for _, status := range statuses {
		s.statuses[status.DomainID] = status
	}
}

func (s *MemoryStore) CreateDomain(domain Domain) Domain {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.domains[domain.ID] = domain
	return domain
}

func (s *MemoryStore) GetDomain(id string) (Domain, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	domain, ok := s.domains[id]
	return domain, ok
}

func (s *MemoryStore) ListDomains() []Domain {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Domain, 0, len(s.domains))
	for _, domain := range s.domains {
		result = append(result, domain)
	}
	slices.SortFunc(result, func(a, b Domain) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return result
}

func (s *MemoryStore) CreateVersion(version OntologyVersion) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.versions[version.DomainID] = version
}

func (s *MemoryStore) GetCurrentVersion(domainID string) (OntologyVersion, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	version, ok := s.versions[domainID]
	return version, ok
}

func (s *MemoryStore) CreateNodeType(schema NodeTypeSchema) NodeTypeSchema {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodeTypes[schema.ID] = schema
	return schema
}

func (s *MemoryStore) GetNodeType(domainID, nodeTypeName string) (NodeTypeSchema, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	schema, ok := s.nodeTypes[domainID+"."+nodeTypeName]
	return schema, ok
}

func (s *MemoryStore) ListNodeTypes(domainID string) []NodeTypeSchema {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []NodeTypeSchema
	for _, schema := range s.nodeTypes {
		if schema.DomainID == domainID {
			result = append(result, schema)
		}
	}
	slices.SortFunc(result, func(a, b NodeTypeSchema) int {
		if a.NodeTypeName < b.NodeTypeName {
			return -1
		}
		if a.NodeTypeName > b.NodeTypeName {
			return 1
		}
		return 0
	})
	return result
}

func relKey(domainID, relTypeName, fromNodeType, toNodeType string) string {
	return domainID + "." + relTypeName + "." + fromNodeType + "." + toNodeType
}

func (s *MemoryStore) CreateRelType(schema RelTypeSchema) RelTypeSchema {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.relTypes[schema.ID] = schema
	return schema
}

func (s *MemoryStore) GetRelType(domainID, relTypeName, fromNodeType, toNodeType string) (RelTypeSchema, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	schema, ok := s.relTypes[relKey(domainID, relTypeName, fromNodeType, toNodeType)]
	return schema, ok
}

func (s *MemoryStore) ListRelTypes(domainID string) []RelTypeSchema {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []RelTypeSchema
	for _, schema := range s.relTypes {
		if schema.DomainID == domainID {
			result = append(result, schema)
		}
	}
	slices.SortFunc(result, func(a, b RelTypeSchema) int {
		if a.RelTypeName < b.RelTypeName {
			return -1
		}
		if a.RelTypeName > b.RelTypeName {
			return 1
		}
		if a.FromNodeType < b.FromNodeType {
			return -1
		}
		if a.FromNodeType > b.FromNodeType {
			return 1
		}
		if a.ToNodeType < b.ToNodeType {
			return -1
		}
		if a.ToNodeType > b.ToNodeType {
			return 1
		}
		return 0
	})
	return result
}

func (s *MemoryStore) CreateCrossDomainRule(rule CrossDomainRelRule) CrossDomainRelRule {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = append(s.rules, rule)
	return rule
}

func (s *MemoryStore) ListCrossDomainRules(fromDomainID string) []CrossDomainRelRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []CrossDomainRelRule
	for _, rule := range s.rules {
		if rule.FromDomainID == fromDomainID {
			result = append(result, rule)
		}
	}
	return result
}

func (s *MemoryStore) CreateQueryTemplate(template QueryTemplate) QueryTemplate {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.templates[template.ID] = template
	return template
}

func (s *MemoryStore) GetQueryTemplate(domainID, templateName string) (QueryTemplate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	template, ok := s.templates[domainID+"."+templateName]
	return template, ok
}

func (s *MemoryStore) UpdateQueryTemplate(template QueryTemplate) (QueryTemplate, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.templates[template.ID]; !ok {
		return QueryTemplate{}, false
	}
	s.templates[template.ID] = template
	return template, true
}

func (s *MemoryStore) ListQueryTemplates(domainID string) []QueryTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []QueryTemplate
	for _, template := range s.templates {
		if template.DomainID == domainID {
			result = append(result, template)
		}
	}
	slices.SortFunc(result, func(a, b QueryTemplate) int {
		if a.TemplateName < b.TemplateName {
			return -1
		}
		if a.TemplateName > b.TemplateName {
			return 1
		}
		return 0
	})
	return result
}

func (s *MemoryStore) UpsertStatusFieldConfig(config StatusFieldConfig) StatusFieldConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses[config.DomainID] = config
	return config
}

func (s *MemoryStore) GetStatusFieldConfig(domainID string) (StatusFieldConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	config, ok := s.statuses[domainID]
	return config, ok
}
