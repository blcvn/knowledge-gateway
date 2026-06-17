package ontology

import (
	"time"

	"kg-service/internal/access"
)

func SeedDomains() []Domain {
	now := time.Date(2026, 6, 17, 10, 10, 0, 0, time.UTC)
	return []Domain{
		{
			ID:            "van_ban_phap_luat",
			Name:          "Van Ban Phap Luat",
			OwnerTenantID: access.PlatformTenantID,
			Status:        "active",
			Version:       1,
			Visibility:    "public",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "luat_thue_hkd",
			Name:          "Luat Thue HKD",
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
		{DomainID: "van_ban_phap_luat", Version: 1, PublishedAt: now},
		{DomainID: "luat_thue_hkd", Version: 1, PublishedAt: now},
		{DomainID: "shared-domain", Version: 1, PublishedAt: now},
	}
}

func SeedNodeTypes() []NodeTypeSchema {
	now := time.Date(2026, 6, 17, 10, 15, 0, 0, time.UTC)
	return []NodeTypeSchema{
		{
			ID:           "shared-domain.HopDongMau",
			DomainID:     "shared-domain",
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
			ID:           relKey("shared-domain", "THAM_CHIEU", "HopDongMau", "KhoanMau"),
			DomainID:     "shared-domain",
			RelTypeName:  "THAM_CHIEU",
			FromNodeType: "HopDongMau",
			ToNodeType:   "KhoanMau",
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
			ID:           "luat_thue_hkd.calculator",
			DomainID:     "luat_thue_hkd",
			TemplateName: "calculator",
			PatternSpec: map[string]any{
				"start": map[string]any{
					"node_type": "NhomDoanhThu",
					"match": map[string]any{
						"ma_nhom": "$ma_nhom",
					},
				},
				"hops": []any{
					map[string]any{
						"rel_type":     "CO_TY_LE",
						"to_node_type": "TyLeThue",
					},
				},
			},
			ParamSchema: []ParameterSchema{
				{Name: "ma_nhom", Type: "string", Required: true},
			},
			ReturnFields: []string{"TyLeThue.loai_thue"},
			Status:       "active",
			Version:      1,
			CreatedAt:    now,
		},
	}
}

func SeedStatusFieldConfigs() []StatusFieldConfig {
	return []StatusFieldConfig{
		{
			DomainID:            "luat_thue_hkd",
			StatusFieldName:     "tinh_trang",
			ValidStatusValues:   []string{"con_hieu_luc"},
			WarningStatusValues: []string{"bi_sua_doi"},
			AuthorityFieldName:  "loai_van_ban",
			AuthorityValuesMap: map[string]int{
				"Luat":     4,
				"NghiDinh": 3,
				"ThongTu":  2,
				"CongVan":  1,
			},
		},
	}
}
