package write

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/session"
)

// DeleteByScopeWithVersion soft-deletes every live node and relationship matching the filter, as
// part of an open sync session.
//
// A graph version is mandatory. A scope-wide delete outside a version would leave no manifest of
// what was removed, which is precisely the audit trail that makes such a delete recoverable and
// reviewable.
//
// Relationships are removed before nodes. The application layer treats an edge as only meaningful
// while both endpoints exist, so removing edges first means no intermediate state is observable in
// which an edge points at a tombstoned node.
func (s *Service) DeleteByScopeWithVersion(ctx context.Context, actor access.Identity, req ScopeDeleteRequest) (ScopeDeleteResponse, error) {
	if err := s.ensureOwnerIdentityReady(actor); err != nil {
		return ScopeDeleteResponse{}, err
	}
	if err := validateScopeDeleteRequest(req); err != nil {
		return ScopeDeleteResponse{}, errors.Join(ErrValidation, err)
	}

	domain, err := s.ontology.GetVisibleDomain(actor, req.DomainID)
	if err != nil {
		if errors.Is(err, ontology.ErrForbidden) || errors.Is(err, ontology.ErrNotFound) {
			return ScopeDeleteResponse{}, ErrForbidden
		}
		return ScopeDeleteResponse{}, err
	}
	if err := s.ensureWritePermission(actor, domain, req.DomainID); err != nil {
		return ScopeDeleteResponse{}, err
	}

	log.Printf("write delete_by_scope start tenant=%s app=%s domain=%s graph_scope=%s levels=%d graph_version_id=%s",
		actor.TenantID, actor.AppID, req.DomainID, req.GraphScope, len(req.Levels), req.GraphVersionID)

	var response ScopeDeleteResponse
	_, err = s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		repo := s.repositoryForScope(scope)
		state, err := s.loadSyncSessionForWrite(ctx, repo, actor, req.GraphVersionID)
		if err != nil {
			return err
		}
		// The session already owns a lease on one graph scope. Deleting a different scope under
		// that lease would mutate a graph nothing is holding the write lock for.
		if req.GraphScope != state.identity.GraphScope {
			return errors.Join(ErrValidation, fmt.Errorf("graph scope mismatch: %s != %s", req.GraphScope, state.identity.GraphScope))
		}

		deletedRels, err := repo.SoftDeleteRelationshipsByScope(ctx, req.ScopeFilter)
		if err != nil {
			return err
		}
		deletedNodes, err := repo.SoftDeleteNodesByScope(ctx, req.ScopeFilter, s.now())
		if err != nil {
			return err
		}

		entities := make([]GraphVersionEntityRecord, 0, len(deletedRels)+len(deletedNodes))
		for _, rel := range deletedRels {
			response.RelationshipIDs = append(response.RelationshipIDs, rel.ID)
			entities = append(entities, GraphVersionEntityRecord{
				VersionID:  state.version.VersionID,
				EntityKind: "relationship",
				EntityID:   rel.ID,
				ChangeKind: "DELETE",
			})
		}
		for _, node := range deletedNodes {
			response.NodeIDs = append(response.NodeIDs, node.ID)
			entities = append(entities, GraphVersionEntityRecord{
				VersionID:  state.version.VersionID,
				EntityKind: "node",
				EntityID:   node.ID,
				ChangeKind: "DELETE",
			})
		}
		if len(entities) == 0 {
			return nil
		}
		return repo.AddGraphVersionEntities(ctx, state.version.VersionID, entities)
	})
	if err != nil {
		return ScopeDeleteResponse{}, err
	}

	response.Count = len(response.NodeIDs) + len(response.RelationshipIDs)
	s.recordWriteAudit(actor, actor.TenantID, actor.AppID, "kg.scope.delete", "kg_graph_scope", req.GraphScope, "allow", "", map[string]any{
		"domain_id":        req.DomainID,
		"graph_scope":      req.GraphScope,
		"graph_version_id": req.GraphVersionID,
		"nodes":            len(response.NodeIDs),
		"relationships":    len(response.RelationshipIDs),
	})
	log.Printf("write delete_by_scope ok tenant=%s app=%s graph_scope=%s nodes=%d relationships=%d graph_version_id=%s",
		actor.TenantID, actor.AppID, req.GraphScope, len(response.NodeIDs), len(response.RelationshipIDs), req.GraphVersionID)
	return response, nil
}

// DeleteRelationshipsByExternalRefWithVersion removes exactly the edges the caller names by its own
// references.
//
// This is the delta-delete a scoped snapshot needs: the caller upserts the full desired content of
// a scope, then removes the references that are no longer part of it. References that do not
// resolve are reported as untouched rather than as errors — the caller is asserting a desired end
// state, and an edge already being gone satisfies it.
func (s *Service) DeleteRelationshipsByExternalRefWithVersion(ctx context.Context, actor access.Identity, req RelationshipDeleteByExternalRefRequest) (RelationshipDeleteByExternalRefResponse, error) {
	if err := s.ensureOwnerIdentityReady(actor); err != nil {
		return RelationshipDeleteByExternalRefResponse{}, err
	}
	refs := trimmedRefs(req.ExternalRefs)
	if len(refs) == 0 {
		return RelationshipDeleteByExternalRefResponse{}, errors.Join(ErrValidation, errors.New("external_refs is required"))
	}

	log.Printf("write delete_relationships_by_external_ref start tenant=%s app=%s refs=%d graph_version_id=%s",
		actor.TenantID, actor.AppID, len(refs), strings.TrimSpace(req.GraphVersionID))

	var response RelationshipDeleteByExternalRefResponse
	_, err := s.sessionManager.Within(ctx, session.WriteIdentity{
		TenantID: actor.TenantID,
		AppID:    actor.AppID,
	}, func(scope session.SessionScope) error {
		repo := s.repositoryForScope(scope)

		var versionID string
		if strings.TrimSpace(req.GraphVersionID) != "" {
			state, err := s.loadSyncSessionForWrite(ctx, repo, actor, req.GraphVersionID)
			if err != nil {
				return err
			}
			versionID = state.version.VersionID
		}

		deleted, err := repo.SoftDeleteRelationshipsByExternalRefs(ctx, refs)
		if err != nil {
			return err
		}
		entities := make([]GraphVersionEntityRecord, 0, len(deleted))
		for _, rel := range deleted {
			response.RelationshipIDs = append(response.RelationshipIDs, rel.ID)
			response.ExternalRefs = append(response.ExternalRefs, rel.ExternalRef)
			if versionID != "" {
				entities = append(entities, GraphVersionEntityRecord{
					VersionID:  versionID,
					EntityKind: "relationship",
					EntityID:   rel.ID,
					ChangeKind: "DELETE",
				})
			}
		}
		if len(entities) == 0 {
			return nil
		}
		return repo.AddGraphVersionEntities(ctx, versionID, entities)
	})
	if err != nil {
		return RelationshipDeleteByExternalRefResponse{}, err
	}

	response.Count = len(response.RelationshipIDs)
	s.recordWriteAudit(actor, actor.TenantID, actor.AppID, "kg.relationship.delete_by_external_ref", "kg_relationship", "", "allow", "", map[string]any{
		"requested":        len(refs),
		"deleted":          response.Count,
		"graph_version_id": strings.TrimSpace(req.GraphVersionID),
	})
	log.Printf("write delete_relationships_by_external_ref ok tenant=%s app=%s requested=%d deleted=%d",
		actor.TenantID, actor.AppID, len(refs), response.Count)
	return response, nil
}

func validateScopeDeleteRequest(req ScopeDeleteRequest) error {
	if strings.TrimSpace(req.DomainID) == "" {
		return errors.New("domain_id is required")
	}
	if strings.TrimSpace(req.GraphScope) == "" {
		return errors.New("graph_scope is required")
	}
	if strings.TrimSpace(req.GraphVersionID) == "" {
		return errors.New("graph_version_id is required")
	}
	for i, level := range req.Levels {
		if strings.TrimSpace(level.Level) == "" {
			return fmt.Errorf("levels[%d].level is required", i)
		}
	}
	return nil
}

func trimmedRefs(refs []string) []string {
	out := make([]string, 0, len(refs))
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if _, dup := seen[ref]; dup {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	return out
}
