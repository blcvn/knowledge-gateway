package neo4j

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"vnp-memory/services/graphiti-store/internal/usecase/port"
)

type Neo4jDriver struct {
	driver neo4j.DriverWithContext
	config Neo4jConfig

	// Cached repository instances (created once)
	entityNodeRepo    *entityNodeRepo
	entityEdgeRepo    *entityEdgeRepo
	episodeNodeRepo   *episodeNodeRepo
	communityNodeRepo *communityNodeRepo
	sagaNodeRepo      *sagaNodeRepo
	episodicEdgeRepo  *episodicEdgeRepo
	communityEdgeRepo *communityEdgeRepo
	hasEpisodeRepo    *hasEpisodeEdgeRepo
	nextEpisodeRepo   *nextEpisodeEdgeRepo
	searchRepo        *searchRepo
	maintenanceRepo   *maintenanceRepo
	bulkRepo          *bulkRepo
}

type Neo4jConfig struct {
	URI      string
	Username string
	Password string
	Database string
}

// NewNeo4jDriver creates and verifies a Neo4j driver connection
func NewNeo4jDriver(ctx context.Context, cfg Neo4jConfig) (*Neo4jDriver, error) {
	drv, err := neo4j.NewDriverWithContext(
		cfg.URI,
		neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
		func(c *neo4j.Config) {
			c.MaxConnectionPoolSize = 50
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create neo4j driver: %w", err)
	}

	if err := drv.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("neo4j connectivity: %w", err)
	}

	d := &Neo4jDriver{driver: drv, config: cfg}

	// Init all repositories with shared driver reference
	d.entityNodeRepo = &entityNodeRepo{driver: d}
	d.entityEdgeRepo = &entityEdgeRepo{driver: d}
	d.episodeNodeRepo = &episodeNodeRepo{driver: d}
	d.communityNodeRepo = &communityNodeRepo{driver: d}
	d.sagaNodeRepo = &sagaNodeRepo{driver: d}
	d.episodicEdgeRepo = &episodicEdgeRepo{driver: d}
	d.communityEdgeRepo = &communityEdgeRepo{driver: d}
	d.hasEpisodeRepo = &hasEpisodeEdgeRepo{driver: d}
	d.nextEpisodeRepo = &nextEpisodeEdgeRepo{driver: d}
	d.searchRepo = &searchRepo{driver: d}
	d.maintenanceRepo = &maintenanceRepo{driver: d}
	d.bulkRepo = &bulkRepo{
		driver:           d,
		entityNodes:      d.entityNodeRepo,
		entityEdges:      d.entityEdgeRepo,
		episodeNodes:     d.episodeNodeRepo,
		sagaNodes:        d.sagaNodeRepo,
		episodicEdges:    d.episodicEdgeRepo,
		hasEpisodeEdges:  d.hasEpisodeRepo,
		nextEpisodeEdges: d.nextEpisodeRepo,
	}
	return d, nil
}

func (d *Neo4jDriver) Close(ctx context.Context) error {
	return d.driver.Close(ctx)
}

func (d *Neo4jDriver) Ping(ctx context.Context) error {
	return d.driver.VerifyConnectivity(ctx)
}

func (d *Neo4jDriver) Provider() port.GraphProvider {
	return port.ProviderNeo4j
}

func (d *Neo4jDriver) ExecuteQuery(ctx context.Context, query string, params map[string]any) ([]port.Record, error) {
	db := d.config.Database
	if db == "" {
		db = "neo4j"
	}

	result, err := neo4j.ExecuteQuery(ctx, d.driver, query, params,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(db),
	)
	if err != nil {
		return nil, fmt.Errorf("execute query: %w", err)
	}

	records := make([]port.Record, len(result.Records))
	for i, rec := range result.Records {
		records[i] = port.Record{
			Keys:   rec.Keys,
			Values: rec.Values,
		}
	}
	return records, nil
}

func (d *Neo4jDriver) BeginTransaction(ctx context.Context) (port.Transaction, error) {
	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: d.config.Database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	tx, err := session.BeginTransaction(ctx)
	if err != nil {
		session.Close(ctx)
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	return &neo4jTransaction{tx: tx, session: session}, nil
}

// Repository accessors
func (d *Neo4jDriver) EntityNodes() port.EntityNodeRepository          { return d.entityNodeRepo }
func (d *Neo4jDriver) EpisodeNodes() port.EpisodeNodeRepository        { return d.episodeNodeRepo }
func (d *Neo4jDriver) CommunityNodes() port.CommunityNodeRepository    { return d.communityNodeRepo }
func (d *Neo4jDriver) SagaNodes() port.SagaNodeRepository              { return d.sagaNodeRepo }
func (d *Neo4jDriver) EntityEdges() port.EntityEdgeRepository          { return d.entityEdgeRepo }
func (d *Neo4jDriver) EpisodicEdges() port.EpisodicEdgeRepository      { return d.episodicEdgeRepo }
func (d *Neo4jDriver) CommunityEdges() port.CommunityEdgeRepository    { return d.communityEdgeRepo }
func (d *Neo4jDriver) HasEpisodeEdges() port.HasEpisodeEdgeRepository  { return d.hasEpisodeRepo }
func (d *Neo4jDriver) NextEpisodeEdges() port.NextEpisodeEdgeRepository { return d.nextEpisodeRepo }
func (d *Neo4jDriver) Search() port.SearchRepository                    { return d.searchRepo }
func (d *Neo4jDriver) Maintenance() port.MaintenanceRepository          { return d.maintenanceRepo }
func (d *Neo4jDriver) Bulk() port.BulkRepository                        { return d.bulkRepo }

// neo4jTransaction wraps neo4j transaction to implement port.Transaction
type neo4jTransaction struct {
	tx      neo4j.ExplicitTransaction
	session neo4j.SessionWithContext
}

func (t *neo4jTransaction) Run(ctx context.Context, query string, params map[string]any) ([]port.Record, error) {
	result, err := t.tx.Run(ctx, query, params)
	if err != nil {
		return nil, err
	}
	rawRecords, err := result.Collect(ctx)
	if err != nil {
		return nil, err
	}
	records := make([]port.Record, len(rawRecords))
	for i, rec := range rawRecords {
		records[i] = port.Record{Keys: rec.Keys, Values: rec.Values}
	}
	return records, nil
}

func (t *neo4jTransaction) Commit(ctx context.Context) error {
	defer t.session.Close(ctx)
	return t.tx.Commit(ctx)
}

func (t *neo4jTransaction) Rollback(ctx context.Context) error {
	defer t.session.Close(ctx)
	return t.tx.Rollback(ctx)
}
