package vectorstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	milvusclient "github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

type MilvusConfig struct {
	Endpoint   string
	Collection string
	Client     milvusclient.Client
}

type MilvusVectorAdapter struct {
	Client     milvusclient.Client
	Endpoint   string
	Collection string

	mu          sync.Mutex
	connectErr  error
	schemaReady bool
	dim         int
}

func NewMilvusVectorAdapter(cfg MilvusConfig) *MilvusVectorAdapter {
	return &MilvusVectorAdapter{
		Client:     cfg.Client,
		Endpoint:   cfg.Endpoint,
		Collection: cfg.Collection,
	}
}

func (a *MilvusVectorAdapter) Upsert(ctx context.Context, doc VectorDocument) error {
	client, err := a.ensureClient(ctx)
	if err != nil {
		return err
	}
	if len(doc.Embedding) == 0 {
		return errors.New("milvus vector adapter requires an embedding")
	}
	if err := a.ensureCollection(ctx, client, len(doc.Embedding)); err != nil {
		return err
	}

	authority := int64(0)
	if doc.AuthorityScore != nil {
		authority = int64(*doc.AuthorityScore)
	}
	aclJSON, _ := json.Marshal(doc.ACLVisibleTo)
	propsJSON, _ := json.Marshal(cloneMap(doc.DomainProps))
	vector := make([]float32, 0, len(doc.Embedding))
	for _, value := range doc.Embedding {
		vector = append(vector, float32(value))
	}

	_, err = client.Upsert(ctx, a.Collection, "",
		entity.NewColumnVarChar("node_id", []string{doc.NodeID}),
		entity.NewColumnVarChar("node_type", []string{doc.NodeType}),
		entity.NewColumnVarChar("domain_id", []string{doc.DomainID}),
		entity.NewColumnVarChar("owner_tenant_id", []string{doc.OwnerTenantID}),
		entity.NewColumnVarChar("owner_app_id", []string{doc.OwnerAppID}),
		entity.NewColumnJSONBytes("acl_visible_to", [][]byte{aclJSON}),
		entity.NewColumnBool("is_deleted", []bool{doc.IsDeleted}),
		entity.NewColumnVarChar("status_value", []string{doc.StatusValue}),
		entity.NewColumnInt64("authority_score", []int64{authority}),
		entity.NewColumnInt64("sync_version", []int64{doc.SyncVersion}),
		entity.NewColumnJSONBytes("domain_props", [][]byte{propsJSON}),
		entity.NewColumnFloatVector("embedding", len(vector), [][]float32{vector}),
	)
	if err != nil {
		return err
	}
	return client.Flush(ctx, a.Collection, false)
}

func (a *MilvusVectorAdapter) Delete(ctx context.Context, nodeID string) error {
	client, err := a.ensureClient(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(a.Collection) == "" {
		return errors.New("milvus vector adapter is not configured")
	}
	return client.Delete(ctx, a.Collection, "", fmt.Sprintf(`node_id in ["%s"]`, escapeMilvusString(nodeID)))
}

func (a *MilvusVectorAdapter) ANN(ctx context.Context, query []float64, filter VectorFilter, opts ANNOptions) ([]VectorResult, error) {
	client, err := a.ensureClient(ctx)
	if err != nil {
		return nil, err
	}
	if err := a.loadCollection(ctx, client); err != nil {
		return nil, err
	}
	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	expr := milvusExpr(filter)
	searchParam, _ := entity.NewIndexFlatSearchParam()
	vector := float32Vector(query)
	results, err := client.Search(ctx, a.Collection, []string{}, expr,
		milvusOutputFields(),
		[]entity.Vector{entity.FloatVector(vector)},
		"embedding",
		entity.COSINE,
		opts.TopK,
		searchParam,
	)
	if err != nil {
		return nil, err
	}
	out := make([]VectorResult, 0, len(results))
	for _, result := range results {
		if result.Err != nil {
			return nil, result.Err
		}
		for i := 0; i < result.ResultCount; i++ {
			doc, err := milvusDocFromSearchResult(result, i)
			if err != nil {
				return nil, err
			}
			if !milvusACLAllowed(doc.ACLVisibleTo, filter.ACLVisibleTo) {
				continue
			}
			score := float64(result.Scores[i])
			out = append(out, VectorResult{Document: doc, Score: score})
		}
	}
	return out, nil
}

func (a *MilvusVectorAdapter) Snapshot(ctx context.Context) ([]VectorDocument, error) {
	client, err := a.ensureClient(ctx)
	if err != nil {
		return nil, err
	}
	if err := a.loadCollection(ctx, client); err != nil {
		if errors.Is(err, errMilvusCollectionMissing) {
			return []VectorDocument{}, nil
		}
		return nil, err
	}
	rs, err := client.Query(ctx, a.Collection, []string{}, "", milvusOutputFields())
	if err != nil {
		return nil, err
	}
	return milvusDocsFromResultSet(rs)
}

func (a *MilvusVectorAdapter) ReadSyncVersion(ctx context.Context, entityID string) (int64, error) {
	client, err := a.ensureClient(ctx)
	if err != nil {
		return 0, err
	}
	if err := a.loadCollection(ctx, client); err != nil {
		if errors.Is(err, errMilvusCollectionMissing) {
			return 0, nil
		}
		return 0, err
	}
	rs, err := client.QueryByPks(ctx, a.Collection, []string{}, entity.NewColumnVarChar("node_id", []string{entityID}), []string{"sync_version"})
	if err != nil {
		return 0, err
	}
	col := rs.GetColumn("sync_version")
	if col == nil || col.Len() == 0 {
		return 0, nil
	}
	return col.GetAsInt64(0)
}

func (a *MilvusVectorAdapter) ensureClient(ctx context.Context) (milvusclient.Client, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.Client != nil {
		return a.Client, nil
	}
	if a.connectErr != nil {
		return nil, a.connectErr
	}
	if strings.TrimSpace(a.Endpoint) == "" {
		a.connectErr = errors.New("milvus vector adapter is not configured")
		return nil, a.connectErr
	}
	addr := milvusAddress(a.Endpoint)
	c, err := milvusclient.NewDefaultGrpcClient(ctx, addr)
	if err != nil {
		a.connectErr = err
		return nil, err
	}
	a.Client = c
	return a.Client, nil
}

func (a *MilvusVectorAdapter) ensureCollection(ctx context.Context, client milvusclient.Client, dim int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.schemaReady {
		return nil
	}
	if a.Collection == "" {
		return errors.New("milvus vector adapter is not configured")
	}
	ok, err := client.HasCollection(ctx, a.Collection)
	if err != nil {
		return err
	}
	if !ok {
		schema := milvusSchema(a.Collection, dim)
		if err := client.CreateCollection(ctx, schema, 1); err != nil {
			return err
		}
		idx, err := entity.NewIndexFlat(entity.COSINE)
		if err != nil {
			return err
		}
		if err := client.CreateIndex(ctx, a.Collection, "embedding", idx, false); err != nil {
			return err
		}
	}
	if err := client.LoadCollection(ctx, a.Collection, false); err != nil {
		return err
	}
	a.schemaReady = true
	a.dim = dim
	return nil
}

func (a *MilvusVectorAdapter) loadCollection(ctx context.Context, client milvusclient.Client) error {
	if a.Collection == "" {
		return errMilvusCollectionMissing
	}
	ok, err := client.HasCollection(ctx, a.Collection)
	if err != nil {
		return err
	}
	if !ok {
		return errMilvusCollectionMissing
	}
	return client.LoadCollection(ctx, a.Collection, false)
}

func milvusSchema(collection string, dim int) *entity.Schema {
	schema := entity.NewSchema().WithName(collection).WithAutoID(false)
	schema.WithField(entity.NewField().WithName("node_id").WithDataType(entity.FieldTypeVarChar).WithIsPrimaryKey(true))
	schema.WithField(entity.NewField().WithName("node_type").WithDataType(entity.FieldTypeVarChar))
	schema.WithField(entity.NewField().WithName("domain_id").WithDataType(entity.FieldTypeVarChar))
	schema.WithField(entity.NewField().WithName("owner_tenant_id").WithDataType(entity.FieldTypeVarChar))
	schema.WithField(entity.NewField().WithName("owner_app_id").WithDataType(entity.FieldTypeVarChar))
	schema.WithField(entity.NewField().WithName("acl_visible_to").WithDataType(entity.FieldTypeJSON))
	schema.WithField(entity.NewField().WithName("is_deleted").WithDataType(entity.FieldTypeBool))
	schema.WithField(entity.NewField().WithName("status_value").WithDataType(entity.FieldTypeVarChar))
	schema.WithField(entity.NewField().WithName("authority_score").WithDataType(entity.FieldTypeInt64))
	schema.WithField(entity.NewField().WithName("sync_version").WithDataType(entity.FieldTypeInt64))
	schema.WithField(entity.NewField().WithName("domain_props").WithDataType(entity.FieldTypeJSON))
	schema.WithField(entity.NewField().WithName("embedding").WithDataType(entity.FieldTypeFloatVector).WithDim(int64(dim)))
	return schema
}

func milvusExpr(filter VectorFilter) string {
	parts := make([]string, 0, 4)
	if len(filter.DomainIDs) > 0 {
		parts = append(parts, milvusInExpr("domain_id", filter.DomainIDs))
	}
	if len(filter.NodeTypes) > 0 {
		parts = append(parts, milvusInExpr("node_type", filter.NodeTypes))
	}
	if len(filter.OwnerTenantIDs) > 0 {
		parts = append(parts, milvusInExpr("owner_tenant_id", filter.OwnerTenantIDs))
	}
	if len(filter.OwnerAppIDs) > 0 {
		parts = append(parts, milvusInExpr("owner_app_id", filter.OwnerAppIDs))
	}
	return strings.Join(parts, " and ")
}

func milvusInExpr(field string, values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf(`"%s"`, escapeMilvusString(value)))
	}
	return fmt.Sprintf("%s in [%s]", field, strings.Join(quoted, ", "))
}

func milvusOutputFields() []string {
	return []string{
		"node_id",
		"node_type",
		"domain_id",
		"owner_tenant_id",
		"owner_app_id",
		"acl_visible_to",
		"is_deleted",
		"status_value",
		"authority_score",
		"sync_version",
		"domain_props",
	}
}

func milvusDocFromSearchResult(result milvusclient.SearchResult, idx int) (VectorDocument, error) {
	doc := VectorDocument{}
	if id, err := result.IDs.GetAsString(idx); err == nil {
		doc.NodeID = id
	}
	if col := result.Fields.GetColumn("node_type"); col != nil {
		doc.NodeType, _ = col.GetAsString(idx)
	}
	if col := result.Fields.GetColumn("domain_id"); col != nil {
		doc.DomainID, _ = col.GetAsString(idx)
	}
	if col := result.Fields.GetColumn("owner_tenant_id"); col != nil {
		doc.OwnerTenantID, _ = col.GetAsString(idx)
	}
	if col := result.Fields.GetColumn("owner_app_id"); col != nil {
		doc.OwnerAppID, _ = col.GetAsString(idx)
	}
	if col := result.Fields.GetColumn("is_deleted"); col != nil {
		doc.IsDeleted, _ = col.GetAsBool(idx)
	}
	if col := result.Fields.GetColumn("status_value"); col != nil {
		doc.StatusValue, _ = col.GetAsString(idx)
	}
	if col := result.Fields.GetColumn("authority_score"); col != nil {
		if v, err := col.GetAsInt64(idx); err == nil {
			score := int(v)
			doc.AuthorityScore = &score
		}
	}
	if col := result.Fields.GetColumn("sync_version"); col != nil {
		doc.SyncVersion, _ = col.GetAsInt64(idx)
	}
	if col := result.Fields.GetColumn("acl_visible_to"); col != nil {
		if raw, err := col.Get(idx); err == nil {
			switch v := raw.(type) {
			case []byte:
				_ = json.Unmarshal(v, &doc.ACLVisibleTo)
			case string:
				_ = json.Unmarshal([]byte(v), &doc.ACLVisibleTo)
			}
		}
	}
	if col := result.Fields.GetColumn("domain_props"); col != nil {
		if raw, err := col.Get(idx); err == nil {
			switch v := raw.(type) {
			case []byte:
				_ = json.Unmarshal(v, &doc.DomainProps)
			case string:
				_ = json.Unmarshal([]byte(v), &doc.DomainProps)
			}
		}
	}
	if doc.DomainProps == nil {
		doc.DomainProps = map[string]any{}
	}
	return doc, nil
}

func milvusDocsFromResultSet(rs milvusclient.ResultSet) ([]VectorDocument, error) {
	if rs.Len() == 0 {
		return []VectorDocument{}, nil
	}
	result := make([]VectorDocument, 0, rs.Len())
	for i := 0; i < rs.Len(); i++ {
		doc := VectorDocument{}
		if col := rs.GetColumn("node_id"); col != nil {
			doc.NodeID, _ = col.GetAsString(i)
		}
		if col := rs.GetColumn("node_type"); col != nil {
			doc.NodeType, _ = col.GetAsString(i)
		}
		if col := rs.GetColumn("domain_id"); col != nil {
			doc.DomainID, _ = col.GetAsString(i)
		}
		if col := rs.GetColumn("owner_tenant_id"); col != nil {
			doc.OwnerTenantID, _ = col.GetAsString(i)
		}
		if col := rs.GetColumn("owner_app_id"); col != nil {
			doc.OwnerAppID, _ = col.GetAsString(i)
		}
		if col := rs.GetColumn("is_deleted"); col != nil {
			doc.IsDeleted, _ = col.GetAsBool(i)
		}
		if col := rs.GetColumn("status_value"); col != nil {
			doc.StatusValue, _ = col.GetAsString(i)
		}
		if col := rs.GetColumn("authority_score"); col != nil {
			if v, err := col.GetAsInt64(i); err == nil {
				score := int(v)
				doc.AuthorityScore = &score
			}
		}
		if col := rs.GetColumn("sync_version"); col != nil {
			doc.SyncVersion, _ = col.GetAsInt64(i)
		}
		if col := rs.GetColumn("acl_visible_to"); col != nil {
			if raw, err := col.Get(i); err == nil {
				switch v := raw.(type) {
				case []byte:
					_ = json.Unmarshal(v, &doc.ACLVisibleTo)
				case string:
					_ = json.Unmarshal([]byte(v), &doc.ACLVisibleTo)
				}
			}
		}
		if col := rs.GetColumn("domain_props"); col != nil {
			if raw, err := col.Get(i); err == nil {
				switch v := raw.(type) {
				case []byte:
					_ = json.Unmarshal(v, &doc.DomainProps)
				case string:
					_ = json.Unmarshal([]byte(v), &doc.DomainProps)
				}
			}
		}
		if doc.DomainProps == nil {
			doc.DomainProps = map[string]any{}
		}
		result = append(result, doc)
	}
	return result, nil
}

func milvusACLAllowed(docACL, filterACL []string) bool {
	if len(filterACL) == 0 {
		return true
	}
	for _, allowed := range filterACL {
		for _, candidate := range docACL {
			if allowed == candidate {
				return true
			}
		}
	}
	return false
}

func float32Vector(values []float64) []float32 {
	result := make([]float32, 0, len(values))
	for _, value := range values {
		result = append(result, float32(value))
	}
	return result
}

func escapeMilvusString(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}

func milvusAddress(endpoint string) string {
	if strings.Contains(endpoint, "://") {
		if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
			return u.Host
		}
	}
	return strings.TrimSpace(endpoint)
}

var errMilvusCollectionMissing = errors.New("milvus collection is not configured")
