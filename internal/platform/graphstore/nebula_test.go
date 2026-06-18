//go:build nebula

package graphstore

import (
	"context"
	"errors"
	"strings"
	"testing"

	nebtypes "github.com/vesoft-inc/nebula-go/v5/pkg/types"
)

func TestNebulaGraphAdapterExecuteQueryAndReads(t *testing.T) {
	fake := &fakeNebulaClient{
		execute: func(stmt string) (nebtypes.Result, error) {
			switch {
			case strings.Contains(stmt, "MATCH (n0:Doc)") && strings.Contains(stmt, "RETURN id"):
				return &fakeNebulaResult{
					columns: []string{"id"},
					rows:    [][]nebtypes.Value{{fakeNebulaString("doc-a")}},
				}, nil
			case strings.Contains(stmt, "MATCH (n)") && strings.Contains(stmt, "labels(n)[0] AS node_type"):
				return &fakeNebulaResult{
					columns: []string{
						"id",
						"node_type",
						"domain_id",
						"owner_tenant_id",
						"owner_app_id",
						"visibility",
						"status_value",
						"is_deleted",
						"sync_version",
						"properties",
					},
					rows: [][]nebtypes.Value{{
						fakeNebulaString("doc-a"),
						fakeNebulaString("Doc"),
						fakeNebulaString("d"),
						fakeNebulaString("tenant"),
						fakeNebulaString("app"),
						fakeNebulaString("public"),
						fakeNebulaString("active"),
						fakeNebulaBool(false),
						fakeNebulaInt64(7),
						fakeNebulaString(`{"title":"alpha"}`),
					}},
				}, nil
			case strings.Contains(stmt, "RETURN coalesce(n._kg_sync_version, 0) AS sync_version"):
				return &fakeNebulaResult{
					columns: []string{"sync_version"},
					rows:    [][]nebtypes.Value{{fakeNebulaInt64(7)}},
				}, nil
			case strings.Contains(stmt, "INSERT NODE Doc"):
				return &fakeNebulaResult{columns: []string{}, rows: [][]nebtypes.Value{}}, nil
			default:
				return &fakeNebulaResult{}, errors.New("unexpected statement: " + stmt)
			}
		},
	}

	adapter := &nebulaRealAdapter{client: fake, graph: "kg"}
	results, err := adapter.ExecuteQuery(context.Background(), GraphQuery{
		StartNodeType: "Doc",
		ReturnFields:  []string{"id"},
	}, map[string]any{"acl_tokens": []string{"tenant:app"}})
	if err != nil {
		t.Fatalf("ExecuteQuery() error = %v", err)
	}
	if got := len(results); got != 1 {
		t.Fatalf("ExecuteQuery() len = %d, want 1", got)
	}
	if got := results[0]["id"]; got != "doc-a" {
		t.Fatalf("ExecuteQuery() id = %#v, want doc-a", got)
	}
	if got := fake.lastStatement(); !strings.Contains(got, "USE kg") {
		t.Fatalf("ExecuteQuery() stmt = %q, want graph selection", got)
	}

	nodes, err := adapter.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("ListNodes() len = %d, want 1", len(nodes))
	}
	if nodes[0].ID != "doc-a" || nodes[0].SyncVersion != 7 {
		t.Fatalf("ListNodes() node = %#v, want doc-a version 7", nodes[0])
	}

	version, err := adapter.ReadSyncVersion(context.Background(), "doc-a")
	if err != nil {
		t.Fatalf("ReadSyncVersion() error = %v", err)
	}
	if version != 7 {
		t.Fatalf("ReadSyncVersion() = %d, want 7", version)
	}

	if err := adapter.UpsertNode(context.Background(), GraphNode{
		ID:          "doc-b",
		NodeType:    "Doc",
		SyncVersion: 9,
		Properties:  map[string]any{"title": "beta"},
	}); err != nil {
		t.Fatalf("UpsertNode() error = %v", err)
	}
	if got := fake.lastStatement(); !strings.Contains(got, `INSERT NODE Doc`) || !strings.Contains(got, `"doc-b"`) {
		t.Fatalf("UpsertNode() stmt = %q, want node insert", got)
	}
}

type fakeNebulaClient struct {
	execute func(stmt string) (nebtypes.Result, error)
	last    string
}

func (c *fakeNebulaClient) Execute(stmt string) (nebtypes.Result, error) {
	return c.ExecuteContext(context.Background(), stmt)
}

func (c *fakeNebulaClient) ExecuteContext(_ context.Context, stmt string) (nebtypes.Result, error) {
	c.last = stmt
	if c.execute == nil {
		return &fakeNebulaResult{}, nil
	}
	return c.execute(stmt)
}

func (c *fakeNebulaClient) Ping() error                       { return nil }
func (c *fakeNebulaClient) PingContext(context.Context) error { return nil }
func (c *fakeNebulaClient) IsClosed() bool                    { return false }
func (c *fakeNebulaClient) Close() error                      { return nil }
func (c *fakeNebulaClient) GetSessionId() (int64, error)      { return 0, nil }
func (c *fakeNebulaClient) GetVersion() (string, error)       { return "fake", nil }

func (c *fakeNebulaClient) lastStatement() string {
	return c.last
}

type fakeNebulaResult struct {
	columns []string
	rows    [][]nebtypes.Value
	index   int
}

func (r fakeNebulaResult) Summary() nebtypes.Summary { return nil }
func (r fakeNebulaResult) Cursor() []byte            { return nil }
func (r fakeNebulaResult) RowSize() int              { return len(r.rows) }
func (r fakeNebulaResult) HasNext() bool             { return r.index < len(r.rows) }
func (r *fakeNebulaResult) Next() (nebtypes.Row, error) {
	if r.index >= len(r.rows) {
		return nil, errors.New("no more rows")
	}
	row := fakeNebulaRow{values: r.rows[r.index]}
	r.index++
	return row, nil
}
func (r fakeNebulaResult) Scan(...any) error                  { return nil }
func (r fakeNebulaResult) Columns() []string                  { return append([]string(nil), r.columns...) }
func (r fakeNebulaResult) ColumnTypes() []nebtypes.ColumnType { return nil }

type fakeNebulaRow struct {
	values []nebtypes.Value
}

func (r fakeNebulaRow) Values() []nebtypes.Value { return append([]nebtypes.Value(nil), r.values...) }
func (r fakeNebulaRow) GetValueByName(string) (nebtypes.Value, error) {
	return nil, errors.New("not implemented")
}
func (r fakeNebulaRow) GetValueByIndex(index int) (nebtypes.Value, error) {
	if index < 0 || index >= len(r.values) {
		return nil, errors.New("index out of range")
	}
	return r.values[index], nil
}

type fakeNebulaValue struct {
	kind nebtypes.ValueType
	str  string
	i64  int64
	b    bool
}

func fakeNebulaString(v string) nebtypes.Value {
	return fakeNebulaValue{kind: nebtypes.ValueTypeString, str: v}
}
func fakeNebulaInt64(v int64) nebtypes.Value {
	return fakeNebulaValue{kind: nebtypes.ValueTypeInt64, i64: v}
}
func fakeNebulaBool(v bool) nebtypes.Value {
	return fakeNebulaValue{kind: nebtypes.ValueTypeBool, b: v}
}

func (v fakeNebulaValue) String() string                     { return v.str }
func (v fakeNebulaValue) GetType() nebtypes.ValueType        { return v.kind }
func (v fakeNebulaValue) IsNull() bool                       { return false }
func (v fakeNebulaValue) AsBool() (nebtypes.Bool, error)     { return nebtypes.Bool(v.b), nil }
func (v fakeNebulaValue) AsInt8() (nebtypes.Int8, error)     { return nebtypes.Int8(v.i64), nil }
func (v fakeNebulaValue) AsInt16() (nebtypes.Int16, error)   { return nebtypes.Int16(v.i64), nil }
func (v fakeNebulaValue) AsInt32() (nebtypes.Int32, error)   { return nebtypes.Int32(v.i64), nil }
func (v fakeNebulaValue) AsInt64() (nebtypes.Int64, error)   { return nebtypes.Int64(v.i64), nil }
func (v fakeNebulaValue) AsUInt8() (nebtypes.UInt8, error)   { return nebtypes.UInt8(v.i64), nil }
func (v fakeNebulaValue) AsUInt16() (nebtypes.UInt16, error) { return nebtypes.UInt16(v.i64), nil }
func (v fakeNebulaValue) AsUInt32() (nebtypes.UInt32, error) { return nebtypes.UInt32(v.i64), nil }
func (v fakeNebulaValue) AsUInt64() (nebtypes.UInt64, error) { return nebtypes.UInt64(v.i64), nil }
func (v fakeNebulaValue) AsFloat() (nebtypes.Float, error) {
	return nebtypes.Float(float32(v.i64)), nil
}
func (v fakeNebulaValue) AsDouble() (nebtypes.Double, error) {
	return nebtypes.Double(float64(v.i64)), nil
}
func (v fakeNebulaValue) AsString() (nebtypes.String, error) { return nebtypes.String(v.str), nil }
func (v fakeNebulaValue) AsList() (nebtypes.List, error)     { return nil, errors.New("not implemented") }
func (v fakeNebulaValue) AsRecord() (nebtypes.Record, error) {
	return nil, errors.New("not implemented")
}
func (v fakeNebulaValue) AsDuration() (nebtypes.Duration, error) {
	return nil, errors.New("not implemented")
}
func (v fakeNebulaValue) AsLocalTime() (nebtypes.LocalTime, error) {
	return nil, errors.New("not implemented")
}
func (v fakeNebulaValue) AsLocalDatetime() (nebtypes.LocalDatetime, error) {
	return nil, errors.New("not implemented")
}
func (v fakeNebulaValue) AsDate() (nebtypes.Date, error) { return nil, errors.New("not implemented") }
func (v fakeNebulaValue) AsZonedDatetime() (nebtypes.ZonedDatetime, error) {
	return nil, errors.New("not implemented")
}
func (v fakeNebulaValue) AsZonedTime() (nebtypes.ZonedTime, error) {
	return nil, errors.New("not implemented")
}
func (v fakeNebulaValue) AsNode() (nebtypes.Node, error) { return nil, errors.New("not implemented") }
func (v fakeNebulaValue) AsEdge() (nebtypes.Edge, error) { return nil, errors.New("not implemented") }
func (v fakeNebulaValue) AsPath() (nebtypes.Path, error) { return nil, errors.New("not implemented") }
func (v fakeNebulaValue) AsDecimal() (nebtypes.Decimal, error) {
	return nil, errors.New("not implemented")
}
func (v fakeNebulaValue) AsGeography() (nebtypes.Geography, error) {
	return nil, errors.New("not implemented")
}
func (v fakeNebulaValue) AsEmbeddingVector() (nebtypes.EmbeddingVector, error) {
	return nil, errors.New("not implemented")
}

var _ nebtypes.Client = (*fakeNebulaClient)(nil)
var _ nebtypes.Result = (*fakeNebulaResult)(nil)
var _ nebtypes.Row = (fakeNebulaRow{})
var _ nebtypes.Value = (fakeNebulaValue{})
