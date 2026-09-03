# kg-sample-import

Tool import KG sample data (CSV nodes/edges) vào Neo4j theo namespace `app_id` + `tenant_id`.

## 1) Chạy Neo4j local bằng Docker

Chạy nhanh một container Neo4j độc lập:

```bash
docker run -d \
  --name neo4j-local \
  -p 7474:7474 \
  -p 7687:7687 \
  -e NEO4J_AUTH=neo4j/neo4jpassword \
  neo4j:5.15-community
```

Thông tin kết nối:
- Browser: `http://localhost:7474`
- Bolt: `bolt://localhost:7687`
- Username: `neo4j`
- Password: `neo4jpassword`

Lệnh quản lý nhanh:

```bash
docker logs -f neo4j-local
docker stop neo4j-local
docker start neo4j-local
docker rm -f neo4j-local
```

## 2) Chạy importer

Từ thư mục gốc repo `ba-agent-system`:

```bash
cd services/ai-kg-service/kgs-platform
NEO4J_PASSWORD=neo4jpassword \
GOWORK=off GOCACHE=/tmp/go-build-cache \
go run ./cmd/kg-sample-import \
  --neo4j-uri bolt://localhost:7687 \
  --neo4j-user neo4j \
  --app-id ba-agent-system \
  --tenant-id default \
  --clear-scope \
  --ensure-schema
```

Mặc định tool sẽ đọc:
- `documents/ai-orchestrator/kg-sample-data/query_1-2026-04-07_62330-nodes.csv`
- `documents/ai-orchestrator/kg-sample-data/query_1-2026-04-07_62346-edges.csv`

## 3) Dùng custom CSV nodes/edges

Nếu bạn muốn import từ file CSV khác, chỉ định bằng `--nodes-csv` và `--edges-csv`:

```bash
go run ./cmd/kg-sample-import \
  --neo4j-uri bolt://localhost:7687 \
  --neo4j-user neo4j \
  --neo4j-pass neo4jpassword \
  --nodes-csv /abs/path/custom-nodes.csv \
  --edges-csv /abs/path/custom-edges.csv
```

Nếu muốn ép toàn bộ data về một `document_id` mới khi import:

```bash
go run ./cmd/kg-sample-import \
  --neo4j-uri bolt://localhost:7687 \
  --neo4j-user neo4j \
  --neo4j-pass neo4jpassword \
  --nodes-csv /abs/path/custom-nodes.csv \
  --edges-csv /abs/path/custom-edges.csv \
  --document-id doc-custom-001
```

Yêu cầu format CSV:
- `nodes.csv` bắt buộc có các cột: `id,document_id,reference_id,type,summary,description,source_id,metadata`
- `edges.csv` bắt buộc có các cột: `id,document_id,source_id,target_id,type,reason`
- `metadata` trong `nodes.csv` nên là JSON string hợp lệ (tool vẫn import nếu không hợp lệ, nhưng sẽ log `metadata_parse_errors`)

Lưu ý:
- Nên dùng đường dẫn tuyệt đối cho custom CSV để tránh nhầm `cwd`.
- Edge có `source_id`/`target_id` không tồn tại trong `nodes.csv` sẽ bị skip.

## 4) Các flag chính

- `--neo4j-uri`: URI Neo4j (default `bolt://localhost:7687`)
- `--neo4j-user`: user Neo4j (default `neo4j`)
- `--neo4j-pass`: password Neo4j (hoặc dùng env `NEO4J_PASSWORD`)
- `--neo4j-db`: database name (optional)
- `--nodes-csv`: path file nodes CSV
- `--edges-csv`: path file edges CSV
- `--app-id`: namespace app
- `--tenant-id`: namespace tenant
- `--document-id`: override `document_id` cho toàn bộ rows
- `--batch-size`: số bản ghi/batch (default `500`)
- `--clear-scope`: xóa data scope hiện tại trước khi import
- `--ensure-schema`: tạo constraint/index cần thiết trên `:Entity`

## 5) Kiểm tra sau import

Mở Neo4j Browser và chạy:

```cypher
MATCH (n:Entity {app_id:'ba-agent-system', tenant_id:'default'})
RETURN count(n) AS nodes;
```

```cypher
MATCH (:Entity {app_id:'ba-agent-system', tenant_id:'default'})-[r]->(:Entity {app_id:'ba-agent-system', tenant_id:'default'})
RETURN count(r) AS edges;
```

```cypher
MATCH p=(a:Entity {app_id:'ba-agent-system', tenant_id:'default'})-[r]->(b:Entity {app_id:'ba-agent-system', tenant_id:'default'})
RETURN p
LIMIT 300;
```
