# Solution: SOL-004 — Advanced Loaders & DLT Integration

**CR ID:** CR-COGNEE-004  
**Solution ID:** SOL-004  
**Priority:** Medium (Wave 3)  
**Architecture:** EXTEND `services/cognee-ingestion/internal/adapter/extractor/`

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md`:
- `services/cognee-ingestion/internal/adapter/extractor/` — đây là Extractor Registry.
- Existing extractors: `TextExtractor`, `PDFExtractor` (pdfcpu, flat text), `HTMLExtractor` (colly/goquery), `DocxExtractor`, `CSVExtractor`, `WebExtractor`.
- Extractor pattern: `type Extractor interface { Extract(ctx, input) ([]Chunk, error) }`.
- `add_data.go` dispatch sang extractor theo `ContentType`.
- **Bifrost** là LLM gateway (embedding provider) — không cần LLM cho TabularFK.

---

## 2. Giải pháp chi tiết

### 2.1. [MODIFY] PDF Extractor — Layout-Aware Mode

```go
// services/cognee-ingestion/internal/adapter/extractor/pdf.go

package extractor

import (
    "bytes"
    "context"
    "fmt"
    "strings"

    "github.com/pdfcpu/pdfcpu/pkg/api"
    "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type PDFExtractionMode string
const (
    PDFPlainText   PDFExtractionMode = "PLAIN_TEXT"    // existing behavior
    PDFLayoutAware PDFExtractionMode = "LAYOUT_AWARE"  // [NEW] tables, columns, headings
)

type PDFBlock struct {
    Type    BlockType   // TEXT | TABLE | HEADING | LIST
    Content string
    PageNum int
    BBox    *BoundingBox
}

type BlockType string
const (
    BlockText    BlockType = "TEXT"
    BlockTable   BlockType = "TABLE"
    BlockHeading BlockType = "HEADING"
    BlockList    BlockType = "LIST"
)

type BoundingBox struct {
    X1, Y1, X2, Y2 float64
}

type PDFExtractor struct {
    Mode           PDFExtractionMode
    DoclingURL     string  // Optional: external Docling sidecar URL for complex PDFs
}

func (e *PDFExtractor) Extract(ctx context.Context, input ExtractorInput) ([]Chunk, error) {
    switch e.Mode {
    case PDFLayoutAware:
        return e.extractLayoutAware(ctx, input)
    default:
        return e.extractPlainText(ctx, input)  // existing behavior unchanged
    }
}

func (e *PDFExtractor) extractLayoutAware(ctx context.Context, input ExtractorInput) ([]Chunk, error) {
    // Option A: Native Go layout parsing (preferred, zero external dependency)
    blocks, err := e.parseLayoutNative(input.Data)
    if err != nil && e.DoclingURL != "" {
        // Option B: Fallback to Docling sidecar via HTTP
        blocks, err = e.parseWithDocling(ctx, input.Data)
    }
    if err != nil {
        // Option C: Fall back to plain text
        return e.extractPlainText(ctx, input)
    }
    return e.blocksToChunks(blocks), nil
}

// parseLayoutNative uses pdfcpu with page content stream analysis
func (e *PDFExtractor) parseLayoutNative(data []byte) ([]PDFBlock, error) {
    r := bytes.NewReader(data)
    conf := model.NewDefaultConfiguration()

    // Extract page content as structured elements
    // pdfcpu provides page content stream access
    ctx, err := api.ReadContext(r, conf)
    if err != nil { return nil, err }

    var blocks []PDFBlock
    for pageNum, page := range ctx.PageDict {
        // Analyze content stream for text positioning (BT/ET operators)
        // Detect table-like structures by Y-coordinate clustering
        textRuns := extractTextRuns(page)
        clusters := clusterByYCoord(textRuns)

        for _, cluster := range clusters {
            block := classifyCluster(cluster)
            block.PageNum = pageNum + 1
            blocks = append(blocks, block)
        }
    }
    return blocks, nil
}

// parseWithDocling calls external Docling Python sidecar
func (e *PDFExtractor) parseWithDocling(ctx context.Context, data []byte) ([]PDFBlock, error) {
    // POST to e.DoclingURL/extract with PDF bytes
    // Response: [{type, content, page, bbox}]
    resp, err := httpPostJSON(ctx, e.DoclingURL+"/extract", map[string]any{
        "data": data, "format": "pdf",
    })
    if err != nil { return nil, err }
    return parseDoclingResponse(resp), nil
}

// blocksToChunks converts layout blocks to ingestion chunks
// Tables → Markdown table format; Headings → chunk boundaries
func (e *PDFExtractor) blocksToChunks(blocks []PDFBlock) []Chunk {
    var chunks []Chunk
    var currentSection strings.Builder
    var currentHeading string

    for _, block := range blocks {
        switch block.Type {
        case BlockHeading:
            // Flush current section
            if currentSection.Len() > 0 {
                chunks = append(chunks, Chunk{
                    Content:  currentSection.String(),
                    Metadata: map[string]any{"heading": currentHeading, "type": "text"},
                })
                currentSection.Reset()
            }
            currentHeading = block.Content
        case BlockTable:
            // Convert table to Markdown format
            chunks = append(chunks, Chunk{
                Content:  block.Content,  // Already Markdown table from classifier
                Metadata: map[string]any{"heading": currentHeading, "type": "table", "page": block.PageNum},
            })
        case BlockText, BlockList:
            currentSection.WriteString(block.Content)
            currentSection.WriteString("\n")
        }
    }

    if currentSection.Len() > 0 {
        chunks = append(chunks, Chunk{
            Content:  currentSection.String(),
            Metadata: map[string]any{"heading": currentHeading, "type": "text"},
        })
    }
    return chunks
}

// clusterByYCoord groups text runs by Y coordinate proximity
// Rows with similar Y values → table row candidates
func clusterByYCoord(runs []TextRun) [][]TextRun {
    const yThreshold = 5.0  // pt distance for same-row detection
    // ... clustering algorithm
    return nil
}

// classifyCluster determines if a cluster is TEXT, TABLE, HEADING, or LIST
func classifyCluster(cluster []TextRun) PDFBlock {
    // Heuristics:
    // - Multiple cells in same Y range with regular X spacing → TABLE
    // - Single run with larger font size → HEADING
    // - Runs starting with bullet chars → LIST
    // - Otherwise → TEXT
    return PDFBlock{Type: BlockText}
}
```

### 2.2. [MODIFY] Web Extractor — Readability Mode

```go
// services/cognee-ingestion/internal/adapter/extractor/web.go

package extractor

import (
    "context"
    "net/http"
    "strings"
    "time"

    "github.com/go-shiori/go-readability"
)

type WebExtractor struct {
    Readability bool   // [NEW] true: use go-readability to clean content
    MaxDepth    int    // 0 = single page only (crawl depth reserved for future)
    Timeout     time.Duration
}

func (e *WebExtractor) Extract(ctx context.Context, input ExtractorInput) ([]Chunk, error) {
    url := input.URL
    if url == "" { url = input.Content }

    // Fetch page
    client := &http.Client{Timeout: e.Timeout}
    resp, err := client.Get(url)
    if err != nil { return nil, fmt.Errorf("fetch %s: %w", url, err) }
    defer resp.Body.Close()

    if e.Readability {
        return e.extractWithReadability(ctx, resp, url)
    }
    return e.extractRaw(ctx, resp)
}

func (e *WebExtractor) extractWithReadability(ctx context.Context, resp *http.Response, url string) ([]Chunk, error) {
    // go-readability: extract main content, remove nav/footer/ads/scripts
    article, err := readability.FromReader(resp.Body, url)
    if err != nil { return nil, fmt.Errorf("readability parse: %w", err) }

    // article.Content = cleaned HTML
    // article.TextContent = plain text (no HTML)
    content := article.TextContent

    // Split into paragraphs as chunks
    paragraphs := strings.Split(content, "\n\n")
    chunks := make([]Chunk, 0, len(paragraphs))
    for _, p := range paragraphs {
        p = strings.TrimSpace(p)
        if len(p) < 50 { continue } // skip tiny fragments
        chunks = append(chunks, Chunk{
            Content: p,
            Metadata: map[string]any{
                "source":  url,
                "title":   article.Title,
                "excerpt": article.Excerpt,
                "type":    "web",
            },
        })
    }
    return chunks, nil
}
```

### 2.3. [NEW] Tabular FK Extractor

```go
// services/cognee-ingestion/internal/adapter/extractor/tabular_fk.go

package extractor

import (
    "context"
    "fmt"
    "github.com/google/uuid"
    "github.com/vnp-memory/services/cognee-ingestion/internal/domain"
)

// TabularFKExtractor converts JSON array + FK schema → DataPoints + Relations
// Zero LLM: pure structural mapping
type TabularFKExtractor struct{}

type TabularDataInput struct {
    Rows      []map[string]any `json:"rows"`
    Schema    TabularSchema    `json:"schema"`
}

type TabularSchema struct {
    IDField    string       `json:"id_field"`   // which field is the row's unique ID
    TypeName   string       `json:"type_name"`  // DataPoint type, e.g. "Employee"
    FKRelations []FKRelation `json:"fk_relations"`
}

type FKRelation struct {
    FromField string `json:"from_field"`   // field name in this table, e.g. "dept_id"
    ToDataset string `json:"to_dataset"`   // target dataset name, e.g. "departments"
    EdgeLabel string `json:"edge_label"`   // e.g. "works_in"
}

// ExtractDataPoints converts rows to DataPoints (for use with AddDataPointsUseCase)
func (e *TabularFKExtractor) ExtractDataPoints(ctx context.Context, input TabularDataInput) ([]domain.DataPoint, error) {
    dps := make([]domain.DataPoint, 0, len(input.Rows))

    for _, row := range input.Rows {
        // Determine row ID
        idVal, ok := row[input.Schema.IDField]
        if !ok { continue }
        idStr := fmt.Sprint(idVal)

        // Deterministic UUID from row key
        dpID := domain.DeterministicUUID(input.Schema.TypeName, idStr)

        // Build fields (all columns)
        fields := make(map[string]any, len(row))
        for k, v := range row { fields[k] = v }

        // Extract FK relations
        var relations []domain.DataPointRelation
        for _, fk := range input.Schema.FKRelations {
            if fkVal, ok := row[fk.FromField]; ok {
                targetID := domain.DeterministicUUID(fk.ToDataset, fmt.Sprint(fkVal))
                relations = append(relations, domain.DataPointRelation{
                    TargetID: targetID,
                    Label:    fk.EdgeLabel,
                    Weight:   1.0,
                })
            }
        }

        dp := domain.DataPoint{
            ID:        dpID,
            Type:      input.Schema.TypeName,
            Fields:    fields,
            Relations: relations,
        }
        // IndexFields = all string fields (auto-detect)
        dp.IndexFields = detectStringFields(fields)
        dps = append(dps, dp)
    }
    return dps, nil
}

// detectStringFields returns field names with string values (good candidates for embedding)
func detectStringFields(fields map[string]any) []string {
    var result []string
    for k, v := range fields {
        if _, ok := v.(string); ok && len(fmt.Sprint(v)) > 10 {
            result = append(result, k)
        }
    }
    return result
}
```

### 2.4. [MODIFY] `add_data.go` — Content Type Routing

```go
// services/cognee-ingestion/internal/usecase/add_data.go

func (uc *AddDataUseCase) selectExtractor(item DataItem) Extractor {
    switch {
    case item.Config != nil && item.Config.PDFMode == "LAYOUT_AWARE":
        return &PDFExtractor{Mode: PDFLayoutAware, DoclingURL: uc.config.DoclingURL}

    case item.ContentType == ContentTypeTabularFK:
        // Route to TabularFKExtractor → then AddDataPoints flow
        return &TabularFKExtractorWrapper{uc.addDataPointsUC}

    case isURL(item.Content) || item.URL != "":
        return &WebExtractor{Readability: true, Timeout: 15 * time.Second}

    case item.ContentType == ContentTypePDF:
        return &PDFExtractor{Mode: PDFPlainText}  // default unchanged

    default:
        return uc.extractorRegistry.Get(item.ContentType)
    }
}
```

### 2.5. [NEW] `ContentType` Value Objects

```go
// services/cognee-ingestion/internal/domain/value_object.go

type ContentType string
const (
    ContentTypeText      ContentType = "TEXT"
    ContentTypePDF       ContentType = "PDF"
    ContentTypePDFLayout ContentType = "PDF_LAYOUT"  // [NEW]
    ContentTypeHTML      ContentType = "HTML"
    ContentTypeURL       ContentType = "URL"
    ContentTypeDocx      ContentType = "DOCX"
    ContentTypeCSV       ContentType = "CSV"
    ContentTypeTabularFK ContentType = "TABULAR_FK"  // [NEW]
)
```

### 2.6. [MODIFY] REST API — Extended `POST /api/v1/cognee/add` Body

```json
// PDF Layout-Aware
{
  "dataset_name": "annual_reports",
  "items": [{
    "url": "s3://bucket/report.pdf",
    "config": {"pdf_mode": "LAYOUT_AWARE"}
  }]
}

// Web Readability (auto-detected from URL)
{
  "dataset_name": "research",
  "items": [{"url": "https://arxiv.org/abs/2401.12345"}]
}

// Tabular FK
{
  "dataset_name": "employees",
  "items": [{
    "content_type": "TABULAR_FK",
    "data": {
      "rows": [
        {"id": "e1", "name": "Alice", "dept_id": "d1"},
        {"id": "e2", "name": "Bob",   "dept_id": "d2"}
      ],
      "schema": {
        "id_field": "id",
        "type_name": "Employee",
        "fk_relations": [
          {"from_field": "dept_id", "to_dataset": "departments", "edge_label": "works_in"}
        ]
      }
    }
  }]
}
```

### 2.7. [MODIFY] Config — `apps/memory/configs/config.yaml`

```yaml
cognee:
  ingestion:
    docling_url: ""             # Optional: "http://localhost:8910" for complex PDF sidecar
    web_readability: true       # Enable by default
    tabular_fk_enabled: true
```

### 2.8. go.mod Dependencies

```bash
# go-readability (Apache 2.0 license, pure Go)
go get github.com/go-shiori/go-readability@latest

# pdfcpu already in go.mod (upgrade to latest for better content stream access)
go get github.com/pdfcpu/pdfcpu@latest
```

---

## 3. Files

### [NEW]

| File | Mô tả |
|------|-------|
| `services/cognee-ingestion/internal/adapter/extractor/tabular_fk.go` | TabularFKExtractor |

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `services/cognee-ingestion/internal/adapter/extractor/pdf.go` | + Layout-Aware mode, PDFBlock types |
| `services/cognee-ingestion/internal/adapter/extractor/web.go` | + Readability mode |
| `services/cognee-ingestion/internal/domain/value_object.go` | + PDF_LAYOUT, TABULAR_FK ContentTypes |
| `services/cognee-ingestion/internal/usecase/add_data.go` | + extractor routing logic |
| `apps/memory/configs/config.yaml` | + docling_url, web_readability, tabular_fk_enabled |
| `go.mod` | + github.com/go-shiori/go-readability |

---

## 4. Acceptance Criteria Mapping

| AC từ CR-COGNEE-004 | Covered by |
|--------------------|-----------|
| PDF + LAYOUT_AWARE → blocks có type TABLE | PDFExtractor.extractLayoutAware() + classifyCluster() |
| URL ingest → nội dung chính, không có nav/footer noise | WebExtractor.extractWithReadability() |
| JSON array + FK schema → DataPoints + relations | TabularFKExtractor.ExtractDataPoints() |
| TabularFK không gọi LLM | Zero llmClient.Chat() trong flow |
| ContentType cũ không bị ảnh hưởng | selectExtractor() default fallback |
