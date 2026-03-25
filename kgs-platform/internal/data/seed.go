package data

import (
	"context"

	"kgs-platform/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// KGNodeTypes — 19 node types across 4 layers (KnowledgeGraph_v4.md)
var KGNodeTypes = []biz.EntityType{
	// PRD/URD Layer
	{AppID: "system", Name: "Feature", Description: "Product feature from PRD"},
	{AppID: "system", Name: "UserStory", Description: "User story from PRD"},
	{AppID: "system", Name: "BusinessRule", Description: "Business rule"},
	{AppID: "system", Name: "Actor", Description: "System actor/user role"},
	{AppID: "system", Name: "UseCase", Description: "Use case definition"},
	{AppID: "system", Name: "DataEntity", Description: "Domain data entity"},
	{AppID: "system", Name: "Constraint", Description: "System constraint"},
	// SRS Layer
	{AppID: "system", Name: "SRSRequirement", Description: "Structured software requirement (v4.0)"},
	{AppID: "system", Name: "SystemInterface", Description: "System interface spec (v4.0)"},
	// UI Doc Layer
	{AppID: "system", Name: "UIScreen", Description: "UI screen (v4.0)"},
	{AppID: "system", Name: "UIComponent", Description: "UI component (v4.0)"},
	{AppID: "system", Name: "UIFlow", Description: "UI navigation flow (v4.0)"},
	{AppID: "system", Name: "UIValidationRule", Description: "UI field validation rule (v4.0)"},
	// Test Artifact Layer
	{AppID: "system", Name: "TestRequirement", Description: "Test requirement derived from features"},
	{AppID: "system", Name: "TestDesign", Description: "Test design document"},
	{AppID: "system", Name: "TestCase", Description: "Individual test case"},
	{AppID: "system", Name: "TestSuite", Description: "Test suite grouping"},
	{AppID: "system", Name: "TestScript", Description: "Automated test script"},
}

// KGEdgeTypes — key edge types from KnowledgeGraph_v4.md
var KGEdgeTypes = []biz.RelationType{
	// Cross-layer traceability
	{AppID: "system", Name: "REFINES", Description: "SRSRequirement refines Feature/UserStory"},
	{AppID: "system", Name: "DERIVES_FROM", Description: "TestRequirement derives from Feature"},
	{AppID: "system", Name: "TESTS", Description: "TestCase tests a Requirement"},
	{AppID: "system", Name: "IMPLEMENTS", Description: "TestScript implements TestCase"},
	{AppID: "system", Name: "GROUPS", Description: "TestSuite groups TestCases"},
	// SRS edges
	{AppID: "system", Name: "SPECIFIES_INTERFACE", Description: "SRSRequirement specifies SystemInterface"},
	// UI edges (v4.0)
	{AppID: "system", Name: "RENDERED_ON", Description: "UIComponent rendered on UIScreen"},
	{AppID: "system", Name: "CONTAINS_COMPONENT", Description: "UIScreen contains UIComponent"},
	{AppID: "system", Name: "NAVIGATES_TO", Description: "UIFlow navigates to UIScreen"},
	{AppID: "system", Name: "VALIDATES_FIELD", Description: "UIValidationRule validates UIComponent field"},
	{AppID: "system", Name: "TESTED_ON_SCREEN", Description: "TestCase tested on UIScreen"},
	// KG structure
	{AppID: "system", Name: "HAS_CHILD", Description: "Parent-child relationship"},
	{AppID: "system", Name: "DEPENDS_ON", Description: "Dependency relationship"},
	{AppID: "system", Name: "RELATED_TO", Description: "Generic association"},
	{AppID: "system", Name: "AUTOMATES", Description: "TestScript automates TestCase"},
	{AppID: "system", Name: "PART_OF", Description: "Component is part of system"},
}

// SeedOntology initializes the core Knowledge Graph Types if missing
func SeedOntology(ctx context.Context, db *gorm.DB, logger log.Logger) {
	helper := log.NewHelper(logger)
	countNodes := 0
	countEdges := 0

	for _, nt := range KGNodeTypes {
		var existing biz.EntityType
		if err := db.Where("app_id = ? AND name = ?", nt.AppID, nt.Name).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.WithContext(ctx).Create(&nt).Error; err != nil {
					helper.Errorf("failed to seed node type %s: %v", nt.Name, err)
				} else {
					countNodes++
				}
			}
		}
	}

	for _, et := range KGEdgeTypes {
		var existing biz.RelationType
		if err := db.Where("app_id = ? AND name = ?", et.AppID, et.Name).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.WithContext(ctx).Create(&et).Error; err != nil {
					helper.Errorf("failed to seed edge type %s: %v", et.Name, err)
				} else {
					countEdges++
				}
			}
		}
	}

	if countNodes > 0 || countEdges > 0 {
		helper.Infof("knowledge ontology seeded successfully: %d node types, %d edge types added", countNodes, countEdges)
	}
}
