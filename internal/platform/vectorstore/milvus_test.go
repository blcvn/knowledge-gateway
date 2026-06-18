package vectorstore_test

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/milvus-io/milvus-proto/go-api/v2/commonpb"
	"github.com/milvus-io/milvus-proto/go-api/v2/milvuspb"
	"github.com/milvus-io/milvus-proto/go-api/v2/schemapb"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"github.com/milvus-io/milvus-sdk-go/v2/mocks"
	tmock "github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"kg-service/internal/platform/conformance"
	"kg-service/internal/platform/vectorstore"
)

const milvusTestCollection = "kg_vectors"

type milvusTestDoc struct {
	vector        []float64
	nodeID        string
	nodeType      string
	domainID      string
	ownerTenantID string
	ownerAppID    string
	aclVisibleTo  []string
	isDeleted     bool
	statusValue   string
	authority     int64
	syncVersion   int64
	domainProps   map[string]any
}

type milvusTestState struct {
	mu               sync.Mutex
	collectionExists bool
	schema           *entity.Schema
	docs             map[string]milvusTestDoc
}

func (s *milvusTestState) filteredDocs() []milvusTestDoc {
	out := make([]milvusTestDoc, 0, len(s.docs))
	for _, doc := range s.docs {
		out = append(out, doc)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].nodeID < out[j].nodeID
	})
	return out
}

func newMilvusTestAdapter(t *testing.T) *vectorstore.MilvusVectorAdapter {
	t.Helper()

	state := &milvusTestState{
		docs: map[string]milvusTestDoc{},
	}
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	mock := mocks.NewMilvusServiceServer(t)
	milvuspb.RegisterMilvusServiceServer(srv, mock)

	mock.EXPECT().Connect(tmock.Anything, tmock.Anything).RunAndReturn(func(_ context.Context, req *milvuspb.ConnectRequest) (*milvuspb.ConnectResponse, error) {
		_ = req
		return &milvuspb.ConnectResponse{Status: successStatus()}, nil
	})
	checkHealth := mock.EXPECT().CheckHealth(tmock.Anything, tmock.Anything)
	checkHealth.Call.Maybe()
	checkHealth.RunAndReturn(func(_ context.Context, req *milvuspb.CheckHealthRequest) (*milvuspb.CheckHealthResponse, error) {
		_ = req
		return &milvuspb.CheckHealthResponse{Status: successStatus(), IsHealthy: true}, nil
	})
	mock.EXPECT().HasCollection(tmock.Anything, tmock.Anything).RunAndReturn(func(_ context.Context, req *milvuspb.HasCollectionRequest) (*milvuspb.BoolResponse, error) {
		state.mu.Lock()
		defer state.mu.Unlock()
		return &milvuspb.BoolResponse{Status: successStatus(), Value: req.GetCollectionName() == milvusTestCollection && state.collectionExists}, nil
	})
	mock.EXPECT().DescribeCollection(tmock.Anything, tmock.Anything).RunAndReturn(func(_ context.Context, req *milvuspb.DescribeCollectionRequest) (*milvuspb.DescribeCollectionResponse, error) {
		_ = req
		return &milvuspb.DescribeCollectionResponse{
			Status: successStatus(),
			Schema: milvusTestSchema(milvusTestCollection).ProtoMessage(),
		}, nil
	})
	mock.EXPECT().CreateCollection(tmock.Anything, tmock.Anything).RunAndReturn(func(_ context.Context, req *milvuspb.CreateCollectionRequest) (*commonpb.Status, error) {
		state.mu.Lock()
		state.collectionExists = true
		state.schema = milvusTestSchema(req.GetCollectionName())
		state.mu.Unlock()
		return successStatus(), nil
	})
	mock.EXPECT().CreateIndex(tmock.Anything, tmock.Anything).RunAndReturn(func(_ context.Context, req *milvuspb.CreateIndexRequest) (*commonpb.Status, error) {
		_ = req
		return successStatus(), nil
	})
	mock.EXPECT().DescribeIndex(tmock.Anything, tmock.Anything).RunAndReturn(func(_ context.Context, req *milvuspb.DescribeIndexRequest) (*milvuspb.DescribeIndexResponse, error) {
		_ = req
		return &milvuspb.DescribeIndexResponse{
			Status: successStatus(),
			IndexDescriptions: []*milvuspb.IndexDescription{
				{
					IndexName: "embedding",
					FieldName: "embedding",
					State:     commonpb.IndexState_Finished,
				},
			},
		}, nil
	})
	mock.EXPECT().LoadCollection(tmock.Anything, tmock.Anything).RunAndReturn(func(_ context.Context, req *milvuspb.LoadCollectionRequest) (*commonpb.Status, error) {
		_ = req
		return successStatus(), nil
	})
	mock.EXPECT().GetLoadingProgress(tmock.Anything, tmock.Anything).RunAndReturn(func(_ context.Context, req *milvuspb.GetLoadingProgressRequest) (*milvuspb.GetLoadingProgressResponse, error) {
		_ = req
		return &milvuspb.GetLoadingProgressResponse{Status: successStatus(), Progress: 100}, nil
	})
	mock.EXPECT().Upsert(tmock.Anything, tmock.Anything).RunAndReturn(func(_ context.Context, req *milvuspb.UpsertRequest) (*milvuspb.MutationResult, error) {
		doc := milvusDocFromUpsertRequest(req)
		state.mu.Lock()
		state.docs[doc.nodeID] = doc
		state.mu.Unlock()
		return &milvuspb.MutationResult{
			Status: successStatus(),
			IDs: &schemapb.IDs{
				IdField: &schemapb.IDs_StrId{
					StrId: &schemapb.StringArray{Data: []string{doc.nodeID}},
				},
			},
			Acknowledged: true,
			UpsertCnt:    1,
			Timestamp:    uint64(doc.syncVersion),
		}, nil
	})
	mock.EXPECT().Flush(tmock.Anything, tmock.Anything).RunAndReturn(func(_ context.Context, req *milvuspb.FlushRequest) (*milvuspb.FlushResponse, error) {
		_ = req
		return &milvuspb.FlushResponse{Status: successStatus()}, nil
	})
	mock.EXPECT().Delete(tmock.Anything, tmock.Anything).RunAndReturn(func(_ context.Context, req *milvuspb.DeleteRequest) (*milvuspb.MutationResult, error) {
		nodeID := extractMilvusDeleteID(req.GetExpr())
		state.mu.Lock()
		delete(state.docs, nodeID)
		state.mu.Unlock()
		return &milvuspb.MutationResult{Status: successStatus(), DeleteCnt: 1, Acknowledged: true}, nil
	})
	mock.EXPECT().Search(tmock.Anything, tmock.Anything).RunAndReturn(func(_ context.Context, req *milvuspb.SearchRequest) (*milvuspb.SearchResults, error) {
		state.mu.Lock()
		docs := state.filteredDocs()
		state.mu.Unlock()
		queryVec := decodeMilvusQueryVector(req.GetPlaceholderGroup())
		results := scoreMilvusDocs(docs, queryVec)
		limit := int(req.GetNq())
		if limit <= 0 {
			limit = len(results)
		}
		if len(results) > limit {
			results = results[:limit]
		}
		return buildMilvusSearchResults(results, req.GetOutputFields()), nil
	})
	mock.EXPECT().Query(tmock.Anything, tmock.Anything).RunAndReturn(func(_ context.Context, req *milvuspb.QueryRequest) (*milvuspb.QueryResults, error) {
		state.mu.Lock()
		docs := state.filteredDocs()
		state.mu.Unlock()
		return buildMilvusQueryResults(filterMilvusDocs(docs, req.GetExpr()), req.GetOutputFields()), nil
	})

	go func() {
		_ = srv.Serve(lis)
	}()
	t.Cleanup(srv.Stop)

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	grpcClient, err := client.NewClient(context.Background(), client.Config{
		Address: "bufnet",
		DialOptions: []grpc.DialOption{
			grpc.WithContextDialer(dialer),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		},
	})
	if err != nil {
		t.Fatalf("create milvus client: %v", err)
	}
	return vectorstore.NewMilvusVectorAdapter(vectorstore.MilvusConfig{
		Collection: milvusTestCollection,
		Client:     grpcClient,
	})
}

func TestMilvusVectorAdapterConformance(t *testing.T) {
	conformance.AssertVectorAdapterConformance(t, newMilvusTestAdapter(t))
}

func successStatus() *commonpb.Status {
	return &commonpb.Status{ErrorCode: commonpb.ErrorCode_Success}
}

func milvusTestSchema(collection string) *entity.Schema {
	return entity.NewSchema().WithName(collection).WithAutoID(false).
		WithField(entity.NewField().WithName("node_id").WithDataType(entity.FieldTypeVarChar).WithIsPrimaryKey(true)).
		WithField(entity.NewField().WithName("node_type").WithDataType(entity.FieldTypeVarChar)).
		WithField(entity.NewField().WithName("domain_id").WithDataType(entity.FieldTypeVarChar)).
		WithField(entity.NewField().WithName("owner_tenant_id").WithDataType(entity.FieldTypeVarChar)).
		WithField(entity.NewField().WithName("owner_app_id").WithDataType(entity.FieldTypeVarChar)).
		WithField(entity.NewField().WithName("acl_visible_to").WithDataType(entity.FieldTypeJSON)).
		WithField(entity.NewField().WithName("is_deleted").WithDataType(entity.FieldTypeBool)).
		WithField(entity.NewField().WithName("status_value").WithDataType(entity.FieldTypeVarChar)).
		WithField(entity.NewField().WithName("authority_score").WithDataType(entity.FieldTypeInt64)).
		WithField(entity.NewField().WithName("sync_version").WithDataType(entity.FieldTypeInt64)).
		WithField(entity.NewField().WithName("domain_props").WithDataType(entity.FieldTypeJSON)).
		WithField(entity.NewField().WithName("embedding").WithDataType(entity.FieldTypeFloatVector).WithDim(2))
}

func milvusDocFromUpsertRequest(req *milvuspb.UpsertRequest) milvusTestDoc {
	doc := milvusTestDoc{domainProps: map[string]any{}}
	for _, field := range req.GetFieldsData() {
		switch field.GetFieldName() {
		case "node_id":
			doc.nodeID = firstString(field)
		case "node_type":
			doc.nodeType = firstString(field)
		case "domain_id":
			doc.domainID = firstString(field)
		case "owner_tenant_id":
			doc.ownerTenantID = firstString(field)
		case "owner_app_id":
			doc.ownerAppID = firstString(field)
		case "acl_visible_to":
			_ = json.Unmarshal(firstJSONBytes(field), &doc.aclVisibleTo)
		case "is_deleted":
			doc.isDeleted = firstBool(field)
		case "status_value":
			doc.statusValue = firstString(field)
		case "authority_score":
			doc.authority = firstInt64(field)
		case "sync_version":
			doc.syncVersion = firstInt64(field)
		case "domain_props":
			_ = json.Unmarshal(firstJSONBytes(field), &doc.domainProps)
		case "embedding":
			doc.vector = firstFloatVector(field)
		}
	}
	return doc
}

func buildMilvusQueryResults(docs []milvusTestDoc, outputFields []string) *milvuspb.QueryResults {
	result := &milvuspb.QueryResults{Status: successStatus()}
	result.FieldsData = buildMilvusFieldData(docs, outputFields)
	return result
}

func buildMilvusSearchResults(docs []scoredMilvusDoc, outputFields []string) *milvuspb.SearchResults {
	ids := make([]string, 0, len(docs))
	scores := make([]float32, 0, len(docs))
	plain := make([]milvusTestDoc, 0, len(docs))
	for _, doc := range docs {
		ids = append(ids, doc.doc.nodeID)
		scores = append(scores, float32(doc.score))
		plain = append(plain, doc.doc)
	}
	return &milvuspb.SearchResults{
		Status: successStatus(),
		Results: &schemapb.SearchResultData{
			NumQueries: 1,
			TopK:       int64(len(docs)),
			Topks:      []int64{int64(len(docs))},
			Ids: &schemapb.IDs{
				IdField: &schemapb.IDs_StrId{
					StrId: &schemapb.StringArray{Data: ids},
				},
			},
			Scores:     scores,
			FieldsData: buildMilvusFieldData(plain, outputFields),
		},
	}
}

func buildMilvusFieldData(docs []milvusTestDoc, outputFields []string) []*schemapb.FieldData {
	if len(docs) == 0 {
		return []*schemapb.FieldData{}
	}
	include := func(name string) bool {
		if len(outputFields) == 0 {
			return true
		}
		for _, candidate := range outputFields {
			if candidate == name {
				return true
			}
		}
		return false
	}
	fields := make([]*schemapb.FieldData, 0, 8)
	if include("node_id") {
		fields = append(fields, entity.NewColumnVarChar("node_id", docStrings(docs, func(doc milvusTestDoc) string { return doc.nodeID })).FieldData())
	}
	if include("node_type") {
		fields = append(fields, entity.NewColumnVarChar("node_type", docStrings(docs, func(doc milvusTestDoc) string { return doc.nodeType })).FieldData())
	}
	if include("domain_id") {
		fields = append(fields, entity.NewColumnVarChar("domain_id", docStrings(docs, func(doc milvusTestDoc) string { return doc.domainID })).FieldData())
	}
	if include("owner_tenant_id") {
		fields = append(fields, entity.NewColumnVarChar("owner_tenant_id", docStrings(docs, func(doc milvusTestDoc) string { return doc.ownerTenantID })).FieldData())
	}
	if include("owner_app_id") {
		fields = append(fields, entity.NewColumnVarChar("owner_app_id", docStrings(docs, func(doc milvusTestDoc) string { return doc.ownerAppID })).FieldData())
	}
	if include("acl_visible_to") {
		fields = append(fields, entity.NewColumnJSONBytes("acl_visible_to", docJSONStrings(docs, func(doc milvusTestDoc) any { return doc.aclVisibleTo })).FieldData())
	}
	if include("is_deleted") {
		fields = append(fields, entity.NewColumnBool("is_deleted", docBools(docs, func(doc milvusTestDoc) bool { return doc.isDeleted })).FieldData())
	}
	if include("status_value") {
		fields = append(fields, entity.NewColumnVarChar("status_value", docStrings(docs, func(doc milvusTestDoc) string { return doc.statusValue })).FieldData())
	}
	if include("authority_score") {
		fields = append(fields, entity.NewColumnInt64("authority_score", docInt64s(docs, func(doc milvusTestDoc) int64 { return doc.authority })).FieldData())
	}
	if include("sync_version") {
		fields = append(fields, entity.NewColumnInt64("sync_version", docInt64s(docs, func(doc milvusTestDoc) int64 { return doc.syncVersion })).FieldData())
	}
	if include("domain_props") {
		fields = append(fields, entity.NewColumnJSONBytes("domain_props", docJSONStrings(docs, func(doc milvusTestDoc) any { return doc.domainProps })).FieldData())
	}
	return fields
}

func docStrings(docs []milvusTestDoc, fn func(milvusTestDoc) string) []string {
	result := make([]string, 0, len(docs))
	for _, doc := range docs {
		result = append(result, fn(doc))
	}
	return result
}

func docInt64s(docs []milvusTestDoc, fn func(milvusTestDoc) int64) []int64 {
	result := make([]int64, 0, len(docs))
	for _, doc := range docs {
		result = append(result, fn(doc))
	}
	return result
}

func docBools(docs []milvusTestDoc, fn func(milvusTestDoc) bool) []bool {
	result := make([]bool, 0, len(docs))
	for _, doc := range docs {
		result = append(result, fn(doc))
	}
	return result
}

func docJSONStrings(docs []milvusTestDoc, fn func(milvusTestDoc) any) [][]byte {
	result := make([][]byte, 0, len(docs))
	for _, doc := range docs {
		raw, _ := json.Marshal(fn(doc))
		result = append(result, raw)
	}
	return result
}

type scoredMilvusDoc struct {
	doc   milvusTestDoc
	score float64
}

func scoreMilvusDocs(docs []milvusTestDoc, query []float64) []scoredMilvusDoc {
	result := make([]scoredMilvusDoc, 0, len(docs))
	for _, doc := range docs {
		result = append(result, scoredMilvusDoc{doc: doc, score: milvusCosine(query, doc.vector)})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].score > result[j].score {
			return true
		}
		if result[i].score < result[j].score {
			return false
		}
		return result[i].doc.nodeID < result[j].doc.nodeID
	})
	return result
}

func filterMilvusDocs(docs []milvusTestDoc, expr string) []milvusTestDoc {
	if strings.TrimSpace(expr) == "" {
		return append([]milvusTestDoc(nil), docs...)
	}
	allowed := extractMilvusIDsFromExpr(expr)
	if len(allowed) == 0 {
		return append([]milvusTestDoc(nil), docs...)
	}
	return filterMilvusDocsByIDs(docs, allowed)
}

func filterMilvusDocsByIDs(docs []milvusTestDoc, ids []string) []milvusTestDoc {
	if len(ids) == 0 {
		return append([]milvusTestDoc(nil), docs...)
	}
	wanted := map[string]struct{}{}
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	out := make([]milvusTestDoc, 0, len(ids))
	for _, doc := range docs {
		if _, ok := wanted[doc.nodeID]; ok {
			out = append(out, doc)
		}
	}
	return out
}

func extractMilvusDeleteID(expr string) string {
	ids := extractMilvusIDsFromExpr(expr)
	if len(ids) > 0 {
		return ids[0]
	}
	return strings.TrimSpace(expr)
}

func extractMilvusPKs(expr string) []string {
	return extractMilvusIDsFromExpr(expr)
}

func extractMilvusIDsFromExpr(expr string) []string {
	start := strings.Index(expr, "[")
	end := strings.LastIndex(expr, "]")
	if start < 0 || end <= start {
		return nil
	}
	raw := expr[start+1 : end]
	parts := strings.Split(raw, ",")
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"`)
		if part != "" {
			ids = append(ids, strings.ReplaceAll(part, `\"`, `"`))
		}
	}
	return ids
}

func decodeMilvusQueryVector(phg []byte) []float64 {
	if len(phg) == 0 {
		return nil
	}
	var group commonpb.PlaceholderGroup
	if err := proto.Unmarshal(phg, &group); err != nil {
		return nil
	}
	if len(group.Placeholders) == 0 || len(group.Placeholders[0].Values) == 0 {
		return nil
	}
	raw := group.Placeholders[0].Values[0]
	const width = 4
	if len(raw)%width != 0 {
		return nil
	}
	values := make([]float64, 0, len(raw)/width)
	for i := 0; i < len(raw); i += width {
		bits := uint32(raw[i]) | uint32(raw[i+1])<<8 | uint32(raw[i+2])<<16 | uint32(raw[i+3])<<24
		values = append(values, float64(math.Float32frombits(bits)))
	}
	return values
}

func firstString(field *schemapb.FieldData) string {
	if field == nil || field.GetScalars() == nil || field.GetScalars().GetStringData() == nil || len(field.GetScalars().GetStringData().Data) == 0 {
		return ""
	}
	return field.GetScalars().GetStringData().Data[0]
}

func firstJSONBytes(field *schemapb.FieldData) []byte {
	if field == nil || field.GetScalars() == nil || field.GetScalars().GetJsonData() == nil || len(field.GetScalars().GetJsonData().Data) == 0 {
		return nil
	}
	return bytes.Clone(field.GetScalars().GetJsonData().Data[0])
}

func firstBool(field *schemapb.FieldData) bool {
	if field == nil || field.GetScalars() == nil || field.GetScalars().GetBoolData() == nil || len(field.GetScalars().GetBoolData().Data) == 0 {
		return false
	}
	return field.GetScalars().GetBoolData().Data[0]
}

func firstInt64(field *schemapb.FieldData) int64 {
	if field == nil || field.GetScalars() == nil || field.GetScalars().GetLongData() == nil || len(field.GetScalars().GetLongData().Data) == 0 {
		return 0
	}
	return field.GetScalars().GetLongData().Data[0]
}

func firstFloatVector(field *schemapb.FieldData) []float64 {
	if field == nil || field.GetVectors() == nil || field.GetVectors().GetFloatVector() == nil {
		return nil
	}
	values := field.GetVectors().GetFloatVector().GetData()
	out := make([]float64, 0, len(values))
	for _, value := range values {
		out = append(out, float64(value))
	}
	return out
}

func milvusCosine(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var dot, normA, normB float64
	for i := 0; i < n; i++ {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / math.Sqrt(normA*normB)
}
