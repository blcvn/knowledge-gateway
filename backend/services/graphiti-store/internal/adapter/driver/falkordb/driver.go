// Package falkordb provides a stub GraphDriver for FalkorDB.
// FalkorDB uses one graph per group_id (natural multi-tenancy).
// Full implementation is deferred to Wave 2+ roadmap.
package falkordb

import (
	"context"
	"fmt"
	"time"

	"vnp-memory/pkg/graph"
	"vnp-memory/services/graphiti-store/internal/usecase/port"
)

// FalkorDBDriver — stub implementation.
// Returns ErrNotImplemented for all operations except Provider() and Ping().
type FalkorDBDriver struct{}

var ErrNotImplemented = fmt.Errorf("falkordb: operation not yet implemented")

func (d *FalkorDBDriver) Provider() port.GraphProvider    { return port.ProviderFalkorDB }
func (d *FalkorDBDriver) Ping(ctx context.Context) error  { return nil }
func (d *FalkorDBDriver) Close(ctx context.Context) error { return nil }

func (d *FalkorDBDriver) ExecuteQuery(ctx context.Context, query string, params map[string]any) ([]port.Record, error) {
	return nil, ErrNotImplemented
}

func (d *FalkorDBDriver) BeginTransaction(ctx context.Context) (port.Transaction, error) {
	// FalkorDB does not support ACID transactions
	// Return a no-op transaction (best-effort)
	return &noopTx{}, nil
}

// All repository accessors return a stub that returns ErrNotImplemented
func (d *FalkorDBDriver) EntityNodes() port.EntityNodeRepository          { return &stubEntityNodeRepo{} }
func (d *FalkorDBDriver) EpisodeNodes() port.EpisodeNodeRepository        { return &stubEpisodeNodeRepo{} }
func (d *FalkorDBDriver) CommunityNodes() port.CommunityNodeRepository    { return &stubCommunityNodeRepo{} }
func (d *FalkorDBDriver) SagaNodes() port.SagaNodeRepository              { return &stubSagaNodeRepo{} }
func (d *FalkorDBDriver) EntityEdges() port.EntityEdgeRepository          { return &stubEntityEdgeRepo{} }
func (d *FalkorDBDriver) EpisodicEdges() port.EpisodicEdgeRepository      { return &stubEpisodicEdgeRepo{} }
func (d *FalkorDBDriver) CommunityEdges() port.CommunityEdgeRepository    { return &stubCommunityEdgeRepo{} }
func (d *FalkorDBDriver) HasEpisodeEdges() port.HasEpisodeEdgeRepository  { return &stubHasEpisodeRepo{} }
func (d *FalkorDBDriver) NextEpisodeEdges() port.NextEpisodeEdgeRepository { return &stubNextEpisodeRepo{} }
func (d *FalkorDBDriver) Search() port.SearchRepository                    { return &stubSearchRepo{} }
func (d *FalkorDBDriver) Maintenance() port.MaintenanceRepository          { return &stubMaintenanceRepo{} }
func (d *FalkorDBDriver) Bulk() port.BulkRepository                        { return &stubBulkRepo{} }

type noopTx struct{}

func (t *noopTx) Run(ctx context.Context, q string, p map[string]any) ([]port.Record, error) {
	return nil, nil
}
func (t *noopTx) Commit(ctx context.Context) error   { return nil }
func (t *noopTx) Rollback(ctx context.Context) error { return nil }

// ─── Stub implementations ─────────────────────────────────────────────────────

type stubEntityNodeRepo struct{}

func (s *stubEntityNodeRepo) Save(ctx context.Context, node graph.EntityNode, tx port.Transaction) error {
	return ErrNotImplemented
}
func (s *stubEntityNodeRepo) SaveBulk(ctx context.Context, nodes []graph.EntityNode, tx port.Transaction, batchSize int) error {
	return ErrNotImplemented
}
func (s *stubEntityNodeRepo) GetByUUID(ctx context.Context, uuid string) (*graph.EntityNode, error) {
	return nil, ErrNotImplemented
}
func (s *stubEntityNodeRepo) GetByUUIDs(ctx context.Context, uuids []string) ([]*graph.EntityNode, error) {
	return nil, ErrNotImplemented
}
func (s *stubEntityNodeRepo) Delete(ctx context.Context, uuid string, tx port.Transaction) error {
	return ErrNotImplemented
}
func (s *stubEntityNodeRepo) DeleteByGroupID(ctx context.Context, groupID string, tx port.Transaction, batchSize int) error {
	return ErrNotImplemented
}

type stubEntityEdgeRepo struct{}

func (s *stubEntityEdgeRepo) Save(ctx context.Context, edge graph.EntityEdge, tx port.Transaction) error {
	return ErrNotImplemented
}
func (s *stubEntityEdgeRepo) SaveBulk(ctx context.Context, edges []graph.EntityEdge, tx port.Transaction, batchSize int) error {
	return ErrNotImplemented
}
func (s *stubEntityEdgeRepo) GetByUUID(ctx context.Context, uuid string) (*graph.EntityEdge, error) {
	return nil, ErrNotImplemented
}
func (s *stubEntityEdgeRepo) GetBetweenNodes(ctx context.Context, srcUUID, tgtUUID string) ([]*graph.EntityEdge, error) {
	return nil, ErrNotImplemented
}
func (s *stubEntityEdgeRepo) GetByNodeUUID(ctx context.Context, nodeUUID string) ([]*graph.EntityEdge, error) {
	return nil, ErrNotImplemented
}
func (s *stubEntityEdgeRepo) Invalidate(ctx context.Context, uuid string, invalidAt time.Time, tx port.Transaction) error {
	return ErrNotImplemented
}
func (s *stubEntityEdgeRepo) Delete(ctx context.Context, uuid string, tx port.Transaction) error {
	return ErrNotImplemented
}

type stubEpisodeNodeRepo struct{}

func (s *stubEpisodeNodeRepo) Save(ctx context.Context, node graph.EpisodicNode, tx port.Transaction) error {
	return ErrNotImplemented
}
func (s *stubEpisodeNodeRepo) GetByUUID(ctx context.Context, uuid string) (*graph.EpisodicNode, error) {
	return nil, ErrNotImplemented
}
func (s *stubEpisodeNodeRepo) GetByEntityNodeUUID(ctx context.Context, entityNodeUUID string) ([]*graph.EpisodicNode, error) {
	return nil, ErrNotImplemented
}
func (s *stubEpisodeNodeRepo) RetrieveEpisodes(ctx context.Context, req port.RetrieveEpisodesReq) ([]*graph.EpisodicNode, error) {
	return nil, ErrNotImplemented
}
func (s *stubEpisodeNodeRepo) Delete(ctx context.Context, uuid string, tx port.Transaction) error {
	return ErrNotImplemented
}
func (s *stubEpisodeNodeRepo) DeleteByGroupID(ctx context.Context, groupID string, tx port.Transaction, batchSize int) error {
	return ErrNotImplemented
}

type stubCommunityNodeRepo struct{}

func (s *stubCommunityNodeRepo) Save(ctx context.Context, node graph.CommunityNode, tx port.Transaction) error {
	return ErrNotImplemented
}
func (s *stubCommunityNodeRepo) GetByUUID(ctx context.Context, uuid string) (*graph.CommunityNode, error) {
	return nil, ErrNotImplemented
}
func (s *stubCommunityNodeRepo) DeleteByGroupID(ctx context.Context, groupID string, tx port.Transaction) error {
	return ErrNotImplemented
}

type stubSagaNodeRepo struct{}

func (s *stubSagaNodeRepo) Save(ctx context.Context, node graph.SagaNode, tx port.Transaction) error {
	return ErrNotImplemented
}
func (s *stubSagaNodeRepo) GetByUUID(ctx context.Context, uuid, groupID string) (*graph.SagaNode, error) {
	return nil, ErrNotImplemented
}
func (s *stubSagaNodeRepo) GetByGroupID(ctx context.Context, groupID string) ([]*graph.SagaNode, error) {
	return nil, ErrNotImplemented
}

type stubEpisodicEdgeRepo struct{}

func (s *stubEpisodicEdgeRepo) Save(ctx context.Context, edge graph.EpisodicEdge, tx port.Transaction) error {
	return ErrNotImplemented
}
func (s *stubEpisodicEdgeRepo) SaveBulk(ctx context.Context, edges []graph.EpisodicEdge, tx port.Transaction) error {
	return ErrNotImplemented
}
func (s *stubEpisodicEdgeRepo) DeleteByEpisodeUUID(ctx context.Context, episodeUUID string, tx port.Transaction) error {
	return ErrNotImplemented
}

type stubCommunityEdgeRepo struct{}

func (s *stubCommunityEdgeRepo) Save(ctx context.Context, edge graph.CommunityEdge, tx port.Transaction) error {
	return ErrNotImplemented
}
func (s *stubCommunityEdgeRepo) DeleteByCommunityUUID(ctx context.Context, communityUUID string, tx port.Transaction) error {
	return ErrNotImplemented
}

type stubHasEpisodeRepo struct{}

func (s *stubHasEpisodeRepo) Save(ctx context.Context, edge graph.HasEpisodeEdge, tx port.Transaction) error {
	return ErrNotImplemented
}

type stubNextEpisodeRepo struct{}

func (s *stubNextEpisodeRepo) Save(ctx context.Context, edge graph.NextEpisodeEdge, tx port.Transaction) error {
	return ErrNotImplemented
}

type stubBulkRepo struct{}

func (s *stubBulkRepo) SaveBulk(ctx context.Context, req port.SaveBulkReq) error {
	return ErrNotImplemented
}

type stubMaintenanceRepo struct{}

func (s *stubMaintenanceRepo) ClearData(ctx context.Context, groupIDs []string) error {
	return ErrNotImplemented
}
func (s *stubMaintenanceRepo) BuildIndicesAndConstraints(ctx context.Context, deleteExisting bool) error {
	return ErrNotImplemented
}
func (s *stubMaintenanceRepo) DeleteAllIndexes(ctx context.Context) error { return ErrNotImplemented }
func (s *stubMaintenanceRepo) GetCommunityClusters(ctx context.Context, groupIDs []string) ([][]string, error) {
	return nil, ErrNotImplemented
}
func (s *stubMaintenanceRepo) RemoveCommunities(ctx context.Context, groupID string) error {
	return ErrNotImplemented
}
func (s *stubMaintenanceRepo) GetGroupStats(ctx context.Context, groupID string) (*port.GroupStats, error) {
	return nil, ErrNotImplemented
}
func (s *stubMaintenanceRepo) GetMentionedNodes(ctx context.Context, episodeUUIDs []string) ([]*graph.EntityNode, error) {
	return nil, ErrNotImplemented
}

type stubSearchRepo struct{}

func (s *stubSearchRepo) NodeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int, labels []string) ([]*graph.EntityNode, error) {
	return nil, ErrNotImplemented
}
func (s *stubSearchRepo) NodeSimilaritySearch(ctx context.Context, vector []float32, groupIDs []string, limit int, minScore float64) ([]*graph.EntityNode, error) {
	return nil, ErrNotImplemented
}
func (s *stubSearchRepo) NodeBFSSearch(ctx context.Context, originUUIDs []string, maxDepth int, groupIDs []string, limit int) ([]*graph.EntityNode, error) {
	return nil, ErrNotImplemented
}
func (s *stubSearchRepo) EdgeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int, filters port.EdgeSearchFilters) ([]*graph.EntityEdge, error) {
	return nil, ErrNotImplemented
}
func (s *stubSearchRepo) EdgeSimilaritySearch(ctx context.Context, req port.EdgeSimilarityReq) ([]*graph.EntityEdge, error) {
	return nil, ErrNotImplemented
}
func (s *stubSearchRepo) EdgeBFSSearch(ctx context.Context, originUUIDs []string, maxDepth int, groupIDs []string, limit int) ([]*graph.EntityEdge, error) {
	return nil, ErrNotImplemented
}
func (s *stubSearchRepo) EpisodeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int) ([]*graph.EpisodicNode, error) {
	return nil, ErrNotImplemented
}
func (s *stubSearchRepo) CommunityFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int) ([]*graph.CommunityNode, error) {
	return nil, ErrNotImplemented
}
func (s *stubSearchRepo) CommunitySimilaritySearch(ctx context.Context, vector []float32, groupIDs []string, limit int, minScore float64) ([]*graph.CommunityNode, error) {
	return nil, ErrNotImplemented
}
func (s *stubSearchRepo) NodeDistanceReranker(ctx context.Context, nodeUUIDs []string, centerUUID string) (map[string]float64, error) {
	return nil, ErrNotImplemented
}
func (s *stubSearchRepo) EpisodeMentionsReranker(ctx context.Context, nodeUUIDs []string) (map[string]int, error) {
	return nil, ErrNotImplemented
}
