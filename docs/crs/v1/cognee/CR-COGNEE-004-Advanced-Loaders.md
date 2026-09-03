# Change Request: CR-COGNEE-004 — Advanced Loaders & DLT Integration

**CR ID:** CR-COGNEE-004  
**Component:** `services/kg-service`  
**Priority:** Medium  
**Status:** Implemented  
**Reference:** Cognee PRD §4.1.1, SRS FR-ING-01, URD UR-ING-01  
**Spec:** `references/cognee/specs/services/02-cognee-ingestion.md`

---

## 1. Mô tả

Mở rộng **Extractor Registry** của `cognee-ingestion` với ba nhóm bộ xử lý mới:

1. **Layout-aware PDF** — phân tách bảng biểu, cột, header từ PDF phức tạp (thay cho plain text extraction).
2. **Web Scraping** — ingest trực tiếp từ URL với content cleaning.
3. **DLT (Data Load Tool) — Tabular FK edges** — khi ingest dữ liệu có foreign key, tự động tạo edges giữa rows mà không dùng LLM.

---

## 2. Vấn đề hiện tại

`Extractor Registry` hiện tại trong `services/cognee-ingestion/internal/adapter/extractor/registry.go` chỉ có:

| Extractor | File | Status |
|---|---|---|
| `TextExtractor` | `text.go` | ✅ Có |
| `PDFExtractor` | `pdf.go` (pdfcpu) | ✅ Có — nhưng flat text only |
| `HTMLExtractor` | `html.go` (colly/goquery) | ✅ Có |
| `DocxExtractor` | `docx.go` | ✅ Có |
| `CSVExtractor` | `csv.go` | ✅ Có — nhưng mỗi row thành một plain string |
| `WebExtractor` | `web.go` | ✅ Có — cơ bản |

**Thiếu:**
- PDF bảng biểu / đa cột (Docling-style layout parsing)
- Web readability nâng cao (loại bỏ nav, footer, ads)
- FK-edge extraction từ tabular data

---

## 3. Thay đổi đề xuất

### 3.1. [MODIFY] `internal/adapter/extractor/pdf.go` — Nâng cấp PDF Extractor

Thêm chế độ `layout` sử dụng thư viện Go xử lý PDF layout (ví dụ: `ledongthuc/pdfcpu` advanced mode hoặc gọi sidecar Python Docling qua HTTP):

```go
type PDFExtractor struct {
    Mode PDFExtractionMode  // PLAIN_TEXT | LAYOUT_AWARE
}

type PDFExtractionMode string
const (
    PDFPlainText    PDFExtractionMode = "PLAIN_TEXT"    // current behavior
    PDFLayoutAware  PDFExtractionMode = "LAYOUT_AWARE"  // new: tables, columns
)

// ExtractResult trả về structured blocks thay vì một string dài
type PDFBlock struct {
    Type    BlockType   // TEXT | TABLE | HEADING | LIST
    Content string
    PageNum int
    BBox    *BoundingBox
}
```

Khi `LAYOUT_AWARE`: Parse từng page, detect tables → convert thành Markdown table, detect headings → sử dụng làm chunk boundary.

### 3.2. [MODIFY] `internal/adapter/extractor/web.go` — Web Readability

```go
type WebExtractor struct {
    Readability bool   // true: dùng go-readability để clean content
    MaxDepth    int    // crawl depth (0 = single page only)
}
```

Tích hợp [`go-shiori/go-readability`](https://github.com/go-shiori/go-readability) để extract nội dung chính, loại bỏ nav/footer/ads trước khi return text.

### 3.3. [NEW] `internal/adapter/extractor/tabular_fk.go` — Tabular FK Edge Extractor

```go
// TabularFKExtractor nhận JSON array (tabular data) và sinh ra
// DataPoint list + Relation list dựa trên schema.fk_relations config
type TabularFKExtractor struct{}

type TabularDataInput struct {
    Rows      []map[string]any   `json:"rows"`
    Schema    TabularSchema      `json:"schema"`
}

type TabularSchema struct {
    IDField   string             `json:"id_field"`
    FKRelations []FKRelation     `json:"fk_relations"`
}

type FKRelation struct {
    FromField  string  `json:"from_field"`   // field name in this table
    ToDataset  string  `json:"to_dataset"`   // target dataset name
    EdgeLabel  string  `json:"edge_label"`   // e.g. "authored_by"
}

// Output: []DataPoint (1 per row) với Relations populated từ FK fields
// → Gọi sang AddDataPoints usecase (CR-003)
```

### 3.4. [MODIFY] `internal/usecase/add_data.go`

Bổ sung content type detection logic:
```go
// Nếu source là URL: chạy WebExtractor(Readability=true)
// Nếu source là PDF với config layout=true: chạy PDFExtractor(LAYOUT_AWARE)
// Nếu source là JSON array + schema.fk_relations: route sang TabularFKExtractor
```

### 3.5. [MODIFY] `internal/domain/value_object.go`

```go
type ContentType string
const (
    // existing ...
    ContentTypePDFLayout  ContentType = "PDF_LAYOUT"    // [NEW]
    ContentTypeTabularFK  ContentType = "TABULAR_FK"    // [NEW]
)
```

### 3.6. API Changes

**[MODIFY]** `POST /api/v1/cognee/add` — thêm optional config:
```json
{
  "dataset_name": "annual_reports",
  "items": [{"url": "https://example.com/report.pdf", "config": {"pdf_mode": "LAYOUT_AWARE"}}]
}
```

```json
{
  "dataset_name": "employees",
  "items": [
    {
      "content_type": "TABULAR_FK",
      "data": {
        "rows": [{"id": "e1", "dept_id": "d1", "name": "Alice"}],
        "schema": {
          "id_field": "id",
          "fk_relations": [{"from_field": "dept_id", "to_dataset": "departments", "edge_label": "works_in"}]
        }
      }
    }
  ]
}
```

---

## 4. Traceability

| Item | Ref |
|---|---|
| Modified files | `extractor/pdf.go`, `extractor/web.go` |
| New file | `extractor/tabular_fk.go` |
| gRPC port | `cognee-ingestion:9011` — `AddData` method |
| External dep | `go-shiori/go-readability` |
| External dep | (optional) Docling Python sidecar HTTP |
| LLM usage | None (TabularFK path zero-LLM) |
| Depends on | CR-COGNEE-003 (DataPoint + AddDataPoints for tabular path) |

---

## 5. Acceptance Criteria

- [x] Upload PDF chứa bảng biểu với mode `LAYOUT_AWARE`, kết quả extract có blocks type `TABLE` (không phải flat text).
- [x] Ingest URL web page: nội dung chính được extract chính xác, không có nav/footer noise.
- [x] Ingest JSON array với FK schema: mỗi row thành 1 DataPoint, relations giữa các rows được tạo theo FK config.
- [x] TabularFK ingest không gọi LLM (verify qua Bifrost metrics).
- [x] Backward compatible: Các ContentType cũ (PDF plain, HTML, TEXT, CSV, DOCX) không bị ảnh hưởng.

---

## 6. Implementation Notes

**Implemented in:** `services/kg-service` (MERGE-P2-T2)
Các thay đổi đã được thực hiện ở lớp Domain Entity (Gateway Proxy) cho phép đẩy config trích xuất cấu trúc phức tạp xuống Cognee Python service.

| File | Change |
|------|--------|
| `services/kg-service/internal/domain/cognee/entity.go` | `[NEW]` `ContentTypePDFLayout`, `ContentTypeTabularFK` |
| `services/kg-service/internal/domain/cognee/entity.go` | `[MODIFY]` `DataItem` hỗ trợ `Config map[string]any` |
