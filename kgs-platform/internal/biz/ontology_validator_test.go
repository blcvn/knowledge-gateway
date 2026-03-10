package biz

import (
	"context"
	"errors"
	"io"
	"testing"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/datatypes"
)

type stubOntologyRepo struct {
	entityType    *EntityType
	entityErr     error
	relationType  *RelationType
	relationErr   error
	entityCalls   int
	relationCalls int
}

func (s *stubOntologyRepo) GetEntityType(context.Context, string, string) (*EntityType, error) {
	s.entityCalls++
	return s.entityType, s.entityErr
}

func (s *stubOntologyRepo) GetRelationType(context.Context, string, string) (*RelationType, error) {
	s.relationCalls++
	return s.relationType, s.relationErr
}

type stubGraphRepoForOntology struct {
	nodes map[string]map[string]any
}

func (s *stubGraphRepoForOntology) CreateNode(context.Context, string, string, string, map[string]any) (map[string]any, error) {
	return nil, nil
}

func (s *stubGraphRepoForOntology) GetNode(_ context.Context, _, _, nodeID string) (map[string]any, error) {
	if s.nodes == nil {
		return nil, errors.New("node not found")
	}
	node, ok := s.nodes[nodeID]
	if !ok {
		return nil, errors.New("node not found")
	}
	return node, nil
}

func (s *stubGraphRepoForOntology) CreateEdge(context.Context, string, string, string, string, string, map[string]any) (map[string]any, error) {
	return nil, nil
}

func (s *stubGraphRepoForOntology) ExecuteQuery(context.Context, string, map[string]any) (map[string]any, error) {
	return nil, nil
}

func (s *stubGraphRepoForOntology) GetFullGraph(context.Context, string, string, int, int) (*FullGraphResult, error) {
	return nil, nil
}

func newOntologyValidatorForTest(repo OntologyRepo, graph GraphRepo, cfg OntologyValidatorConfig) *OntologyValidator {
	return NewOntologyValidator(repo, graph, cfg, log.NewStdLogger(io.Discard))
}

func strictConfig() OntologyValidatorConfig {
	return OntologyValidatorConfig{
		Enabled:             true,
		StrictMode:          true,
		SchemaValidation:    false,
		EdgeConstraintCheck: true,
	}
}

func TestOntologyValidatorValidateEntity_Disabled(t *testing.T) {
	repo := &stubOntologyRepo{}
	v := newOntologyValidatorForTest(repo, nil, OntologyValidatorConfig{Enabled: false})

	if err := v.ValidateEntity(context.Background(), "app-1", "Requirement", map[string]any{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if repo.entityCalls != 0 {
		t.Fatalf("expected repo not called when validator disabled")
	}
}

func TestOntologyValidatorValidateEntity_RepoNil(t *testing.T) {
	v := newOntologyValidatorForTest(nil, nil, strictConfig())

	if err := v.ValidateEntity(context.Background(), "app-1", "Requirement", map[string]any{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestOntologyValidatorValidateEntity_EntityTypeExists(t *testing.T) {
	repo := &stubOntologyRepo{
		entityType: &EntityType{Name: "Requirement"},
	}
	v := newOntologyValidatorForTest(repo, nil, strictConfig())

	if err := v.ValidateEntity(context.Background(), "app-1", "Requirement", map[string]any{"priority": "HIGH"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestOntologyValidatorValidateEntity_UnknownTypeStrict(t *testing.T) {
	repo := &stubOntologyRepo{
		entityType: nil,
	}
	v := newOntologyValidatorForTest(repo, nil, strictConfig())

	err := v.ValidateEntity(context.Background(), "app-1", "Unknown", map[string]any{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	kerr := kerrors.FromError(err)
	if kerr == nil {
		t.Fatalf("expected kratos error, got %T", err)
	}
	if kerr.Reason != "ERR_SCHEMA_INVALID" {
		t.Fatalf("expected reason ERR_SCHEMA_INVALID, got %s", kerr.Reason)
	}
}

func TestOntologyValidatorValidateEntity_UnknownTypeSoft(t *testing.T) {
	repo := &stubOntologyRepo{
		entityType: nil,
	}
	cfg := strictConfig()
	cfg.StrictMode = false
	v := newOntologyValidatorForTest(repo, nil, cfg)

	if err := v.ValidateEntity(context.Background(), "app-1", "Unknown", map[string]any{}); err != nil {
		t.Fatalf("expected nil error in soft mode, got %v", err)
	}
}

func TestOntologyValidatorValidateEntity_RepoLookupFail(t *testing.T) {
	repo := &stubOntologyRepo{
		entityErr: errors.New("db unavailable"),
	}
	v := newOntologyValidatorForTest(repo, nil, strictConfig())

	if err := v.ValidateEntity(context.Background(), "app-1", "Requirement", map[string]any{}); err != nil {
		t.Fatalf("expected nil error when lookup fails, got %v", err)
	}
}

func TestOntologyValidatorValidateEdge_RelationExistsAndValidTypes(t *testing.T) {
	repo := &stubOntologyRepo{
		relationType: &RelationType{
			Name:        "DEPENDS_ON",
			SourceTypes: datatypes.JSON([]byte(`["Requirement"]`)),
			TargetTypes: datatypes.JSON([]byte(`["UseCase"]`)),
		},
	}
	graph := &stubGraphRepoForOntology{
		nodes: map[string]map[string]any{
			"n1": {"label": "Requirement"},
			"n2": {"label": "UseCase"},
		},
	}
	v := newOntologyValidatorForTest(repo, graph, strictConfig())

	if err := v.ValidateEdge(context.Background(), "app-1", "tenant-1", "DEPENDS_ON", "n1", "n2"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestOntologyValidatorValidateEdge_UnknownRelationStrict(t *testing.T) {
	repo := &stubOntologyRepo{relationType: nil}
	v := newOntologyValidatorForTest(repo, &stubGraphRepoForOntology{}, strictConfig())

	err := v.ValidateEdge(context.Background(), "app-1", "tenant-1", "UNKNOWN", "n1", "n2")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	kerr := kerrors.FromError(err)
	if kerr == nil || kerr.Reason != "ERR_SCHEMA_INVALID" {
		t.Fatalf("expected ERR_SCHEMA_INVALID, got %#v", kerr)
	}
}

func TestOntologyValidatorValidateEdge_InvalidSourceTypeStrict(t *testing.T) {
	repo := &stubOntologyRepo{
		relationType: &RelationType{
			Name:        "DEPENDS_ON",
			SourceTypes: datatypes.JSON([]byte(`["UseCase"]`)),
			TargetTypes: datatypes.JSON([]byte(`["Requirement"]`)),
		},
	}
	graph := &stubGraphRepoForOntology{
		nodes: map[string]map[string]any{
			"n1": {"label": "Requirement"},
			"n2": {"label": "Requirement"},
		},
	}
	v := newOntologyValidatorForTest(repo, graph, strictConfig())

	err := v.ValidateEdge(context.Background(), "app-1", "tenant-1", "DEPENDS_ON", "n1", "n2")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestOntologyValidatorValidateEdge_InvalidTargetTypeStrict(t *testing.T) {
	repo := &stubOntologyRepo{
		relationType: &RelationType{
			Name:        "DEPENDS_ON",
			SourceTypes: datatypes.JSON([]byte(`["Requirement"]`)),
			TargetTypes: datatypes.JSON([]byte(`["UseCase"]`)),
		},
	}
	graph := &stubGraphRepoForOntology{
		nodes: map[string]map[string]any{
			"n1": {"label": "Requirement"},
			"n2": {"label": "DataModel"},
		},
	}
	v := newOntologyValidatorForTest(repo, graph, strictConfig())

	err := v.ValidateEdge(context.Background(), "app-1", "tenant-1", "DEPENDS_ON", "n1", "n2")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestOntologyValidatorValidateEdge_EmptyConstraints(t *testing.T) {
	repo := &stubOntologyRepo{
		relationType: &RelationType{
			Name:        "DEPENDS_ON",
			SourceTypes: datatypes.JSON([]byte(`[]`)),
			TargetTypes: datatypes.JSON([]byte(`[]`)),
		},
	}
	v := newOntologyValidatorForTest(repo, &stubGraphRepoForOntology{}, strictConfig())

	if err := v.ValidateEdge(context.Background(), "app-1", "tenant-1", "DEPENDS_ON", "n1", "n2"); err != nil {
		t.Fatalf("expected nil error with empty constraints, got %v", err)
	}
}

func TestOntologyValidatorValidateEdge_EdgeConstraintDisabled(t *testing.T) {
	repo := &stubOntologyRepo{
		relationType: &RelationType{
			Name:        "DEPENDS_ON",
			SourceTypes: datatypes.JSON([]byte(`["Requirement"]`)),
			TargetTypes: datatypes.JSON([]byte(`["UseCase"]`)),
		},
	}
	cfg := strictConfig()
	cfg.EdgeConstraintCheck = false
	v := newOntologyValidatorForTest(repo, nil, cfg)

	if err := v.ValidateEdge(context.Background(), "app-1", "tenant-1", "DEPENDS_ON", "n1", "n2"); err != nil {
		t.Fatalf("expected nil error when edge constraints disabled, got %v", err)
	}
}
