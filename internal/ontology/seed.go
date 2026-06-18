package ontology

import (
	"time"

	"kg-service/internal/access"
)

func SeedDomains() []Domain {
	now := time.Date(2026, 6, 17, 10, 10, 0, 0, time.UTC)
	return []Domain{
		{
			ID:            "sample-registry",
			Name:          "Sample Registry",
			OwnerTenantID: access.PlatformTenantID,
			Status:        "active",
			Version:       1,
			Visibility:    "public",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "sample-policy",
			Name:          "Sample Policy",
			OwnerTenantID: access.PlatformTenantID,
			Status:        "active",
			Version:       1,
			Visibility:    "public",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "shared-domain",
			Name:          "Shared Domain",
			OwnerTenantID: "22222222-2222-2222-2222-222222222222",
			Status:        "active",
			Version:       1,
			Visibility:    "private",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
}

func SeedVersions() []OntologyVersion {
	now := time.Date(2026, 6, 17, 10, 10, 0, 0, time.UTC)
	return []OntologyVersion{
		{DomainID: "sample-registry", Version: 1, PublishedAt: now},
		{DomainID: "sample-policy", Version: 1, PublishedAt: now},
		{DomainID: "shared-domain", Version: 1, PublishedAt: now},
	}
}

func SeedNodeTypes() []NodeTypeSchema {
	now := time.Date(2026, 6, 17, 10, 15, 0, 0, time.UTC)
	return []NodeTypeSchema{
		{
			ID:           "shared-domain.SharedDocument",
			DomainID:     "shared-domain",
			NodeTypeName: "SharedDocument",
			GraphLabel:   "SharedDocument",
			RequiredProps: []PropertySchema{
				{Name: "title", Type: "string"},
			},
			OptionalProps: []PropertySchema{
				{Name: "summary", Type: "string"},
			},
			Version:   1,
			CreatedAt: now,
		},
		{
			ID:           "sample-policy.Topic",
			DomainID:     "sample-policy",
			NodeTypeName: "Topic",
			GraphLabel:   "Topic",
			RequiredProps: []PropertySchema{
				{Name: "topic_key", Type: "string"},
			},
			OptionalProps: []PropertySchema{
				{Name: "title", Type: "string"},
			},
			Version:   1,
			CreatedAt: now,
		},
		{
			ID:           "sample-policy.ActionGuide",
			DomainID:     "sample-policy",
			NodeTypeName: "ActionGuide",
			GraphLabel:   "ActionGuide",
			RequiredProps: []PropertySchema{
				{Name: "guide_key", Type: "string"},
			},
			OptionalProps: []PropertySchema{
				{Name: "title", Type: "string"},
			},
			Version:   1,
			CreatedAt: now,
		},
		{
			ID:           "sample-policy.ReferenceDoc",
			DomainID:     "sample-policy",
			NodeTypeName: "ReferenceDoc",
			GraphLabel:   "ReferenceDoc",
			RequiredProps: []PropertySchema{
				{Name: "reference_key", Type: "string"},
			},
			OptionalProps: []PropertySchema{
				{Name: "title", Type: "string"},
			},
			Version:   1,
			CreatedAt: now,
		},
		{
			ID:           "sample-policy.Obligation",
			DomainID:     "sample-policy",
			NodeTypeName: "Obligation",
			GraphLabel:   "Obligation",
			RequiredProps: []PropertySchema{
				{Name: "obligation_key", Type: "string"},
			},
			OptionalProps: []PropertySchema{
				{Name: "summary", Type: "string"},
			},
			Version:   1,
			CreatedAt: now,
		},
		{
			ID:           "sample-policy.Schedule",
			DomainID:     "sample-policy",
			NodeTypeName: "Schedule",
			GraphLabel:   "Schedule",
			RequiredProps: []PropertySchema{
				{Name: "schedule_key", Type: "string"},
			},
			OptionalProps: []PropertySchema{
				{Name: "effective_on", Type: "string"},
			},
			Version:   1,
			CreatedAt: now,
		},
		{
			ID:           "sample-policy.Record",
			DomainID:     "sample-policy",
			NodeTypeName: "Record",
			GraphLabel:   "Record",
			RequiredProps: []PropertySchema{
				{Name: "record_key", Type: "string"},
			},
			OptionalProps: []PropertySchema{
				{Name: "title", Type: "string"},
			},
			Version:   1,
			CreatedAt: now,
		},
		{
			ID:           "noi_bo_hop_dong.HopDongMau",
			DomainID:     "noi_bo_hop_dong",
			NodeTypeName: "HopDongMau",
			GraphLabel:   "HopDongMau",
			RequiredProps: []PropertySchema{
				{Name: "ten", Type: "string"},
			},
			OptionalProps: []PropertySchema{
				{Name: "ghi_chu", Type: "string"},
			},
			Version:   1,
			CreatedAt: now,
		},
		{
			ID:           "noi_bo_hop_dong.KhoanMau",
			DomainID:     "noi_bo_hop_dong",
			NodeTypeName: "KhoanMau",
			GraphLabel:   "KhoanMau",
			RequiredProps: []PropertySchema{
				{Name: "ten", Type: "string"},
			},
			Version:   1,
			CreatedAt: now,
		},
	}
}

func SeedRelTypes() []RelTypeSchema {
	return []RelTypeSchema{
		{
			ID:           relKey("shared-domain", "REFERENCES", "SharedDocument", "SharedDocument"),
			DomainID:     "shared-domain",
			RelTypeName:  "REFERENCES",
			FromNodeType: "SharedDocument",
			ToNodeType:   "SharedDocument",
			SameDomain:   true,
		},
		{
			ID:           relKey("sample-policy", "ROUTES_TO", "Topic", "ActionGuide"),
			DomainID:     "sample-policy",
			RelTypeName:  "ROUTES_TO",
			FromNodeType: "Topic",
			ToNodeType:   "ActionGuide",
			SameDomain:   true,
		},
		{
			ID:           relKey("sample-policy", "CITES", "Record", "ReferenceDoc"),
			DomainID:     "sample-policy",
			RelTypeName:  "CITES",
			FromNodeType: "Record",
			ToNodeType:   "ReferenceDoc",
			SameDomain:   true,
		},
		{
			ID:           relKey("sample-policy", "REQUIRES", "Topic", "Obligation"),
			DomainID:     "sample-policy",
			RelTypeName:  "REQUIRES",
			FromNodeType: "Topic",
			ToNodeType:   "Obligation",
			SameDomain:   true,
		},
		{
			ID:           relKey("sample-policy", "SCHEDULED_BY", "Record", "Schedule"),
			DomainID:     "sample-policy",
			RelTypeName:  "SCHEDULED_BY",
			FromNodeType: "Record",
			ToNodeType:   "Schedule",
			SameDomain:   true,
		},
		{
			ID:           relKey("noi_bo_hop_dong", "THAM_CHIEU", "HopDongMau", "KhoanMau"),
			DomainID:     "noi_bo_hop_dong",
			RelTypeName:  "THAM_CHIEU",
			FromNodeType: "HopDongMau",
			ToNodeType:   "KhoanMau",
			SameDomain:   true,
		},
		{
			ID:           relKey("noi_bo_hop_dong", "DINH_KEM", "HopDongMau", "PhuLucHopDong"),
			DomainID:     "noi_bo_hop_dong",
			RelTypeName:  "DINH_KEM",
			FromNodeType: "HopDongMau",
			ToNodeType:   "PhuLucHopDong",
			SameDomain:   true,
		},
	}
}

func SeedCrossDomainRules() []CrossDomainRelRule {
	return []CrossDomainRelRule{
		{
			ID:                "sample-policy.ATTACHES.ReferenceDoc",
			RelTypeName:       "ATTACHES",
			FromDomainID:      "sample-policy",
			ToDomainID:        "sample-policy",
			FromNodeTypes:     []string{"Topic"},
			ToNodeTypes:       []string{"ReferenceDoc"},
			Required:          true,
			BridgePropertyKey: "bridge_reference_ids",
		},
		{
			ID:                "noi_bo_hop_dong.DINH_KEM.PhuLucHopDong",
			RelTypeName:       "DINH_KEM",
			FromDomainID:      "noi_bo_hop_dong",
			ToDomainID:        "noi_bo_hop_dong",
			FromNodeTypes:     []string{"HopDongMau"},
			ToNodeTypes:       []string{"PhuLucHopDong"},
			Required:          true,
			BridgePropertyKey: "bridge_dinh_kem_ids",
		},
	}
}

func SeedQueryTemplates() []QueryTemplate {
	now := time.Date(2026, 6, 17, 10, 20, 0, 0, time.UTC)
	return []QueryTemplate{
		{
			ID:           "sample-policy.action-guide",
			DomainID:     "sample-policy",
			TemplateName: "action-guide",
			PatternSpec: map[string]any{
				"start": map[string]any{
					"node_type": "Topic",
					"match": map[string]any{
						"topic_key": "$topic_key",
					},
				},
				"hops": []any{
					map[string]any{
						"rel_type":     "ROUTES_TO",
						"to_node_type": "ActionGuide",
					},
				},
			},
			ParamSchema: []ParameterSchema{
				{Name: "topic_key", Type: "string", Required: true},
			},
			ReturnFields: []string{"ActionGuide.title"},
			Status:       "active",
			Version:      1,
			CreatedAt:    now,
		},
		{
			ID:           "sample-policy.topic-routing",
			DomainID:     "sample-policy",
			TemplateName: "topic-routing",
			PatternSpec: map[string]any{
				"start": map[string]any{
					"node_type": "Topic",
					"match": map[string]any{
						"topic_key": "$topic_key",
					},
				},
			},
			ParamSchema: []ParameterSchema{
				{Name: "topic_key", Type: "string", Required: true},
			},
			ReturnFields: []string{"Topic.topic_key", "Topic.title"},
			Status:       "active",
			Version:      1,
			CreatedAt:    now,
		},
		{
			ID:           "sample-policy.reference-check",
			DomainID:     "sample-policy",
			TemplateName: "reference-check",
			PatternSpec: map[string]any{
				"start": map[string]any{
					"node_type": "Record",
					"match": map[string]any{
						"record_key": "$record_key",
					},
				},
			},
			ParamSchema: []ParameterSchema{
				{Name: "record_key", Type: "string", Required: true},
			},
			ReturnFields: []string{"Record.record_key", "Record.status_value"},
			Status:       "active",
			Version:      1,
			CreatedAt:    now,
		},
		{
			ID:           "sample-policy.obligation-summary",
			DomainID:     "sample-policy",
			TemplateName: "obligation-summary",
			PatternSpec: map[string]any{
				"start": map[string]any{
					"node_type": "Obligation",
					"match": map[string]any{
						"obligation_key": "$obligation_key",
					},
				},
			},
			ParamSchema: []ParameterSchema{
				{Name: "obligation_key", Type: "string", Required: true},
			},
			ReturnFields: []string{"Obligation.obligation_key", "Obligation.summary"},
			Status:       "active",
			Version:      1,
			CreatedAt:    now,
		},
		{
			ID:           "sample-policy.schedule-trace",
			DomainID:     "sample-policy",
			TemplateName: "schedule-trace",
			PatternSpec: map[string]any{
				"start": map[string]any{
					"node_type": "Schedule",
					"match": map[string]any{
						"schedule_key": "$schedule_key",
					},
				},
			},
			ParamSchema: []ParameterSchema{
				{Name: "schedule_key", Type: "string", Required: true},
			},
			ReturnFields: []string{"Schedule.schedule_key", "Schedule.effective_on"},
			Status:       "active",
			Version:      1,
			CreatedAt:    now,
		},
	}
}

func SeedStatusFieldConfigs() []StatusFieldConfig {
	return []StatusFieldConfig{
		{
			DomainID:            "sample-policy",
			StatusFieldName:     "record_status",
			ValidStatusValues:   []string{"active"},
			WarningStatusValues: []string{"review"},
			AuthorityFieldName:  "document_class",
			AuthorityValuesMap: map[string]int{
				"policy":    4,
				"procedure": 3,
				"guide":     2,
				"memo":      1,
			},
		},
	}
}
