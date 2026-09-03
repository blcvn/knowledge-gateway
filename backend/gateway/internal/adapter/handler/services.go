package handler

import (
	"log/slog"
	"net/http"

	"github.com/vnp-community/vnp-memory/gateway/internal/usecase/port"
)

// CogneeHandler handles /v1/cognee/* routes.
type CogneeHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewCogneeHandler(registry port.ServiceRegistry, logger *slog.Logger) *CogneeHandler {
	return &CogneeHandler{registry: registry, logger: logger}
}

func (h *CogneeHandler) CreateDataset(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "cognee-ingestion", h.logger)(w, r)
}

func (h *CogneeHandler) UploadData(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "cognee-ingestion", h.logger)(w, r)
}

func (h *CogneeHandler) Cognify(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "cognee-cognify", h.logger)(w, r)
}

func (h *CogneeHandler) Search(w http.ResponseWriter, r *http.Request) {
	// Body is forwarded verbatim to cognee-search.
	// Supported fields: query, strategies, dataset_name, node_sets (CR-002),
	// top_k, save_interaction (CR-005), feedback_for, feedback_score, feedback_text.
	ForwardToService(h.registry, "cognee-search", h.logger)(w, r)
}

// GraphitiHandler handles /v1/graphiti/* routes.
type GraphitiHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewGraphitiHandler(registry port.ServiceRegistry, logger *slog.Logger) *GraphitiHandler {
	return &GraphitiHandler{registry: registry, logger: logger}
}

func (h *GraphitiHandler) IngestEpisode(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "graphiti-ingestion", h.logger)(w, r)
}

func (h *GraphitiHandler) Search(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "graphiti-search", h.logger)(w, r)
}

func (h *GraphitiHandler) GetNode(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "graphiti-store", h.logger)(w, r)
}

func (h *GraphitiHandler) GetEdge(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "graphiti-store", h.logger)(w, r)
}

// MemobaseHandler handles /v1/memobase/* routes.
type MemobaseHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewMemobaseHandler(registry port.ServiceRegistry, logger *slog.Logger) *MemobaseHandler {
	return &MemobaseHandler{registry: registry, logger: logger}
}

func (h *MemobaseHandler) InsertBlob(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memobase-ingestion", h.logger)(w, r)
}

func (h *MemobaseHandler) Flush(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memobase-ingestion", h.logger)(w, r)
}

func (h *MemobaseHandler) GetContext(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memobase-context", h.logger)(w, r)
}

func (h *MemobaseHandler) GetProfiles(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memobase-context", h.logger)(w, r)
}

func (h *MemobaseHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "vnp-event", h.logger)(w, r)
}

// OpenVikingHandler handles /v1/ov/* routes.
type OpenVikingHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewOpenVikingHandler(registry port.ServiceRegistry, logger *slog.Logger) *OpenVikingHandler {
	return &OpenVikingHandler{registry: registry, logger: logger}
}

func (h *OpenVikingHandler) ReadFile(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "ov-fs", h.logger)(w, r)
}

func (h *OpenVikingHandler) WriteFile(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "ov-fs", h.logger)(w, r)
}

func (h *OpenVikingHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "ov-fs", h.logger)(w, r)
}

func (h *OpenVikingHandler) Tree(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "ov-fs", h.logger)(w, r)
}

func (h *OpenVikingHandler) Grep(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "ov-fs", h.logger)(w, r)
}

func (h *OpenVikingHandler) Search(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "ov-search", h.logger)(w, r)
}

func (h *OpenVikingHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "ov-session", h.logger)(w, r)
}

func (h *OpenVikingHandler) AddMessage(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "ov-session", h.logger)(w, r)
}

func (h *OpenVikingHandler) CommitSession(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "ov-session", h.logger)(w, r)
}

func (h *OpenVikingHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "ov-resource", h.logger)(w, r)
}

// ZepHandler handles /v1/zep/* routes.
type ZepHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewZepHandler(registry port.ServiceRegistry, logger *slog.Logger) *ZepHandler {
	return &ZepHandler{registry: registry, logger: logger}
}

func (h *ZepHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "zep-user", h.logger)(w, r)
}

func (h *ZepHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "zep-user", h.logger)(w, r)
}

func (h *ZepHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "zep-user", h.logger)(w, r)
}

func (h *ZepHandler) PutMemory(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "zep-memory", h.logger)(w, r)
}

func (h *ZepHandler) GetMemory(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "zep-memory", h.logger)(w, r)
}

func (h *ZepHandler) GraphSearch(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "zep-search", h.logger)(w, r)
}

func (h *ZepHandler) SessionSearch(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "zep-search", h.logger)(w, r)
}

func (h *ZepHandler) AddFact(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "zep-graph", h.logger)(w, r)
}

func (h *ZepHandler) SetOntology(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "zep-graph", h.logger)(w, r)
}

// SMHandler handles /v1/sm/* (Supermemory) routes.
type SMHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewSMHandler(registry port.ServiceRegistry, logger *slog.Logger) *SMHandler {
	return &SMHandler{registry: registry, logger: logger}
}

func (h *SMHandler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "sm-document", h.logger)(w, r)
}

func (h *SMHandler) GetDocument(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "sm-document", h.logger)(w, r)
}

func (h *SMHandler) CreateMemory(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "sm-memory", h.logger)(w, r)
}

func (h *SMHandler) Search(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "sm-search", h.logger)(w, r)
}

func (h *SMHandler) RAG(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "sm-search", h.logger)(w, r)
}

func (h *SMHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "sm-profile", h.logger)(w, r)
}

func (h *SMHandler) CreateConnection(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "sm-connector", h.logger)(w, r)
}

func (h *SMHandler) SyncConnection(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "sm-connector", h.logger)(w, r)
}

func (h *SMHandler) CreateSpace(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "sm-project", h.logger)(w, r)
}

// AdminHandler handles /v1/admin/* routes.
type AdminHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewAdminHandler(registry port.ServiceRegistry, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{registry: registry, logger: logger}
}

func (h *AdminHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

func (h *AdminHandler) IssueAPIKey(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

func (h *AdminHandler) Health(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

func (h *AdminHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}
