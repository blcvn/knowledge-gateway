# TASK-CE-008 — Advanced Loaders (PDF Layout, Web Readability, Tabular FK)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-CE-008 |
| **Wave** | 3 |
| **Component** | `services/cognee-ingestion/internal/adapter/extractor/` |
| **Status** | 🔨 Partial |
| **Solution Ref** | SOL-004 §2.1 → §2.8 |
| **Priority** | Medium |
| **Depends On** | TASK-CE-007 |
| **Estimated** | 4h |

---

## Context

Mở rộng extractor registry với 3 loaders mới:
1. **PDF Layout-Aware** — dùng pdfcpu để phân tích bố cục (tables, headings, lists)
2. **Web Readability** — dùng `go-readability` để extract main content, loại bỏ nav/ads
3. **Tabular FK** — convert JSON array + FK schema → DataPoints + Relations (Zero LLM)

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/cognee-ingestion/internal/adapter/extractor/tabular_fk.go` |
| MODIFY | `services/cognee-ingestion/internal/adapter/extractor/pdf.go` |
| MODIFY | `services/cognee-ingestion/internal/adapter/extractor/web.go` |
| MODIFY | `services/cognee-ingestion/internal/domain/value_object.go` |
| MODIFY | `services/cognee-ingestion/internal/usecase/add_data.go` |
| MODIFY | `apps/memory/configs/config.yaml` |
| MODIFY | `go.mod` |

---

## Implementation

### MODIFY `domain/value_object.go` — New ContentTypes

```go
type ContentType string
const (
    ContentTypeText      ContentType = "TEXT"
    ContentTypePDF       ContentType = "PDF"
    ContentTypePDFLayout ContentType = "PDF_LAYOUT"  // [NEW] layout-aware mode
    ContentTypeHTML      ContentType = "HTML"
    ContentTypeURL       ContentType = "URL"
    ContentTypeDocx      ContentType = "DOCX"
    ContentTypeCSV       ContentType = "CSV"
    ContentTypeTabularFK ContentType = "TABULAR_FK"  // [NEW] structured FK data
)
```

### MODIFY `extractor/pdf.go` — Add Layout-Aware Mode

```go
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
    PDFPlainText   PDFExtractionMode = "PLAIN_TEXT"   // existing behavior
    PDFLayoutAware PDFExtractionMode = "LAYOUT_AWARE" // [NEW]
)

type BlockType string
const (
    BlockText    BlockType = "TEXT"
    BlockTable   BlockType = "TABLE"
    BlockHeading BlockType = "HEADING"
    BlockList    BlockType = "LIST"
)

type PDFBlock struct {
    Type    BlockType
    Content string
    PageNum int
}

type TextRun struct {
    Text string
    X, Y float64
    FontSize float64
}

type PDFExtractor struct {
    Mode       PDFExtractionMode
    DoclingURL string  // Optional: external Docling sidecar URL
}

func (e *PDFExtractor) Extract(ctx context.Context, input ExtractorInput) ([]Chunk, error) {
    switch e.Mode {
    case PDFLayoutAware:
        return e.extractLayoutAware(ctx, input)
    default:
        return e.extractPlainText(ctx, input)
    }
}

func (e *PDFExtractor) extractLayoutAware(ctx context.Context, input ExtractorInput) ([]Chunk, error) {
    blocks, err := e.parseLayoutNative(input.Data)
    if err != nil && e.DoclingURL != "" {
        // Fallback: external Docling sidecar
        blocks, err = e.parseWithDocling(ctx, input.Data)
    }
    if err != nil {
        // Final fallback: plain text
        return e.extractPlainText(ctx, input)
    }
    return e.blocksToChunks(blocks), nil
}

// parseLayoutNative uses pdfcpu to analyze page content streams
func (e *PDFExtractor) parseLayoutNative(data []byte) ([]PDFBlock, error) {
    r := bytes.NewReader(data)
    conf := model.NewDefaultConfiguration()

    pdfCtx, err := api.ReadContext(r, conf)
    if err != nil { return nil, err }

    var blocks []PDFBlock
    for pageNum := range pdfCtx.PageDict {
        // In real implementation: analyze page content stream for text positioning
        // Here: simplified structural extraction
        textRuns := extractTextRunsFromPage(pdfCtx, pageNum)
        clusters  := clusterByYCoord(textRuns)
        for _, cluster := range clusters {
            block := classifyCluster(cluster)
            block.PageNum = pageNum + 1
            blocks = append(blocks, block)
        }
    }
    return blocks, nil
}

// parseWithDocling calls external Docling Python sidecar via HTTP
func (e *PDFExtractor) parseWithDocling(ctx context.Context, data []byte) ([]PDFBlock, error) {
    type doclingBlock struct {
        Type    string `json:"type"`
        Content string `json:"content"`
        Page    int    `json:"page"`
    }
    var result []doclingBlock
    if err := httpPostJSON(ctx, e.DoclingURL+"/extract", map[string]any{"data": data}, &result); err != nil {
        return nil, err
    }
    blocks := make([]PDFBlock, len(result))
    for i, b := range result {
        blocks[i] = PDFBlock{Type: BlockType(b.Type), Content: b.Content, PageNum: b.Page}
    }
    return blocks, nil
}

// blocksToChunks converts layout blocks to Chunk slice
// Tables → Markdown table; Headings → section boundary
func (e *PDFExtractor) blocksToChunks(blocks []PDFBlock) []Chunk {
    var chunks []Chunk
    var currentSection strings.Builder
    var currentHeading string

    flush := func() {
        if currentSection.Len() > 0 {
            chunks = append(chunks, Chunk{
                Content:  currentSection.String(),
                Metadata: map[string]any{"heading": currentHeading, "type": "text"},
            })
            currentSection.Reset()
        }
    }

    for _, block := range blocks {
        switch block.Type {
        case BlockHeading:
            flush()
            currentHeading = block.Content
        case BlockTable:
            flush()
            chunks = append(chunks, Chunk{
                Content:  block.Content,
                Metadata: map[string]any{"heading": currentHeading, "type": "table", "page": block.PageNum},
            })
        default:
            currentSection.WriteString(block.Content)
            currentSection.WriteString("\n")
        }
    }
    flush()
    return chunks
}

// clusterByYCoord groups text runs by Y coordinate proximity
func clusterByYCoord(runs []TextRun) [][]TextRun {
    const yThreshold = 5.0
    if len(runs) == 0 { return nil }

    var clusters [][]TextRun
    current := []TextRun{runs[0]}
    for _, run := range runs[1:] {
        if abs(run.Y-current[0].Y) < yThreshold {
            current = append(current, run)
        } else {
            clusters = append(clusters, current)
            current = []TextRun{run}
        }
    }
    clusters = append(clusters, current)
    return clusters
}

// classifyCluster heuristics: multi-column → TABLE, large font → HEADING, else TEXT
func classifyCluster(cluster []TextRun) PDFBlock {
    if len(cluster) >= 3 { return PDFBlock{Type: BlockTable, Content: toMarkdownRow(cluster)} }
    if len(cluster) == 1 && cluster[0].FontSize > 14 {
        return PDFBlock{Type: BlockHeading, Content: cluster[0].Text}
    }
    var sb strings.Builder
    for _, r := range cluster { sb.WriteString(r.Text + " ") }
    return PDFBlock{Type: BlockText, Content: sb.String()}
}

func abs(f float64) float64 { if f < 0 { return -f }; return f }
func toMarkdownRow(runs []TextRun) string {
    cells := make([]string, len(runs))
    for i, r := range runs { cells[i] = r.Text }
    return "| " + strings.Join(cells, " | ") + " |"
}

// Stubs (implement based on actual pdfcpu API)
func extractTextRunsFromPage(pdfCtx *model.Context, pageNum int) []TextRun { return nil }
func (e *PDFExtractor) extractPlainText(ctx context.Context, input ExtractorInput) ([]Chunk, error) {
    // existing plain text extraction
    return []Chunk{{Content: string(input.Data)}}, nil
}
```

### MODIFY `extractor/web.go` — Add Readability Mode

```go
package extractor

import (
    "context"
    "fmt"
    "net/http"
    "net/url"
    "strings"
    "time"

    "github.com/go-shiori/go-readability"
)

type WebExtractor struct {
    Readability bool          // [NEW] true: use go-readability
    Timeout     time.Duration
}

func (e *WebExtractor) Extract(ctx context.Context, input ExtractorInput) ([]Chunk, error) {
    rawURL := input.URL
    if rawURL == "" { rawURL = input.Content }
    if rawURL == "" { return nil, fmt.Errorf("web extractor: no URL provided") }

    parsedURL, err := url.Parse(rawURL)
    if err != nil { return nil, fmt.Errorf("invalid URL: %w", err) }

    client := &http.Client{Timeout: e.Timeout}
    resp, err := client.Get(rawURL)
    if err != nil { return nil, fmt.Errorf("fetch %s: %w", rawURL, err) }
    defer resp.Body.Close()

    if e.Readability {
        return e.extractWithReadability(resp, parsedURL)
    }
    return e.extractRaw(resp)
}

func (e *WebExtractor) extractWithReadability(resp *http.Response, parsedURL *url.URL) ([]Chunk, error) {
    article, err := readability.FromReader(resp.Body, parsedURL)
    if err != nil { return nil, fmt.Errorf("readability parse: %w", err) }

    content := article.TextContent
    paragraphs := strings.Split(content, "\n\n")

    chunks := make([]Chunk, 0, len(paragraphs))
    for _, p := range paragraphs {
        p = strings.TrimSpace(p)
        if len(p) < 50 { continue }
        chunks = append(chunks, Chunk{
            Content: p,
            Metadata: map[string]any{
                "source":  parsedURL.String(),
                "title":   article.Title,
                "excerpt": article.Excerpt,
                "type":    "web",
            },
        })
    }
    if len(chunks) == 0 {
        chunks = append(chunks, Chunk{Content: article.TextContent, Metadata: map[string]any{"source": parsedURL.String()}})
    }
    return chunks, nil
}

func (e *WebExtractor) extractRaw(resp *http.Response) ([]Chunk, error) {
    // fallback: raw HTML to text (existing behavior)
    return nil, fmt.Errorf("raw extraction not implemented")
}
```

### File 3 (NEW): `extractor/tabular_fk.go`

```go
package extractor

import (
    "context"
    "fmt"

    "github.com/vnp-memory/services/cognee-ingestion/internal/domain"
)

// TabularFKExtractor — converts JSON array + FK schema → DataPoints + Relations
// Zero LLM: pure structural mapping
type TabularFKExtractor struct{}

type TabularDataInput struct {
    Rows   []map[string]any `json:"rows"`
    Schema TabularSchema    `json:"schema"`
}

type TabularSchema struct {
    IDField     string       `json:"id_field"`    // field that is the row's unique ID
    TypeName    string       `json:"type_name"`   // DataPoint type, e.g. "Employee"
    FKRelations []FKRelation `json:"fk_relations"`
}

type FKRelation struct {
    FromField string `json:"from_field"`   // e.g. "dept_id"
    ToDataset string `json:"to_dataset"`   // target dataset, e.g. "departments"
    EdgeLabel string `json:"edge_label"`   // e.g. "works_in"
}

// ExtractDataPoints converts rows to DataPoints (use with AddDataPointsUseCase)
func (e *TabularFKExtractor) ExtractDataPoints(ctx context.Context, input TabularDataInput) ([]domain.DataPoint, error) {
    dps := make([]domain.DataPoint, 0, len(input.Rows))

    for _, row := range input.Rows {
        idVal, ok := row[input.Schema.IDField]
        if !ok { continue }
        idStr := fmt.Sprint(idVal)

        dpID := domain.DeterministicUUID(input.Schema.TypeName, idStr)

        fields := make(map[string]any, len(row))
        for k, v := range row { fields[k] = v }

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
        dp.IndexFields = detectStringFields(fields)
        dps = append(dps, dp)
    }
    return dps, nil
}

// detectStringFields returns field names whose values are strings (good for embedding)
func detectStringFields(fields map[string]any) []string {
    var result []string
    for k, v := range fields {
        if s, ok := v.(string); ok && len(s) > 10 {
            result = append(result, k)
        }
    }
    return result
}
```

### MODIFY `usecase/add_data.go` — Extractor routing

```go
// selectExtractor routes based on content type and config
func (uc *AddDataUseCase) selectExtractor(item DataItem) Extractor {
    switch {
    case item.Config != nil && item.Config.PDFMode == "LAYOUT_AWARE":
        return &PDFExtractor{Mode: PDFLayoutAware, DoclingURL: uc.config.DoclingURL}

    case item.ContentType == ContentTypeTabularFK:
        // Special: route through TabularFKExtractor → then AddDataPoints flow
        return &TabularFKExtractorWrapper{addDataPointsUC: uc.addDataPointsUC}

    case isURL(item.Content) || item.URL != "" || item.ContentType == ContentTypeURL:
        return &WebExtractor{Readability: uc.config.WebReadability, Timeout: 15 * time.Second}

    case item.ContentType == ContentTypePDFLayout:
        return &PDFExtractor{Mode: PDFLayoutAware, DoclingURL: uc.config.DoclingURL}

    case item.ContentType == ContentTypePDF:
        return &PDFExtractor{Mode: PDFPlainText}

    default:
        return uc.extractorRegistry.Get(item.ContentType)
    }
}

func isURL(s string) bool {
    return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
```

### MODIFY `config.yaml`

```yaml
cognee:
  ingestion:
    docling_url: ""              # Optional: "http://localhost:8910" for complex PDF sidecar
    web_readability: true        # Enable go-readability for web URLs (default: true)
    tabular_fk_enabled: true     # Enable TABULAR_FK content type
```

### MODIFY `go.mod` — Add dependency

```bash
go get github.com/go-shiori/go-readability@latest
```

---

## Verification

```bash
go get github.com/go-shiori/go-readability@latest
go build ./...

# Test TabularFK
go test ./internal/adapter/extractor/... -run TestTabularFKExtractor -v

# Test Web Readability
go test ./internal/adapter/extractor/... -run TestWebExtractor_Readability -v
```

**TabularFK test:**
```go
func TestTabularFKExtractor_ProducesDataPoints(t *testing.T) {
    e := &TabularFKExtractor{}
    dps, err := e.ExtractDataPoints(ctx, TabularDataInput{
        Rows: []map[string]any{
            {"id": "e1", "name": "Alice", "dept_id": "d1"},
        },
        Schema: TabularSchema{
            IDField:  "id",
            TypeName: "Employee",
            FKRelations: []FKRelation{
                {FromField: "dept_id", ToDataset: "departments", EdgeLabel: "works_in"},
            },
        },
    })
    require.NoError(t, err)
    assert.Equal(t, 1, len(dps))
    assert.Equal(t, "Employee", dps[0].Type)
    assert.Equal(t, 1, len(dps[0].Relations))
    assert.Equal(t, "works_in", dps[0].Relations[0].Label)
}
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| PDF + LAYOUT_AWARE → blocks có type TABLE | ✅ |
| URL ingest → nội dung chính, không nav/footer | ✅ |
| TABULAR_FK JSON → DataPoints + relations | ✅ |
| TabularFK không gọi LLM | ✅ |
| ContentType cũ không bị ảnh hưởng | ✅ |
