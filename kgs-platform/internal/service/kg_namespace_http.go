package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	stdlog "log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/observability"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/server/middleware"
	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	kgEntitiesDefaultLimit = 100
	kgEntitiesMaxLimit     = 1000
	kgEdgesDefaultLimit    = 100
	kgEdgesMaxLimit        = 1000
	kgLookupDefaultLimit   = 10
	kgLookupMaxLimit       = 100
	kgNodesByLabelDefaultLimit = 100
	kgNodesByLabelMaxLimit     = 1000
)

type kgNamespaceRoute struct {
	Namespace string
	Suffix    string
}

type kgLookupRequest struct {
	EntityType string         `json:"entityType"`
	SourceFile string         `json:"sourceFile"`
	Properties map[string]any `json:"properties"`
	MatchMode  string         `json:"matchMode"`
	Limit      int            `json:"limit"`
}

func (s *GraphService) HandleKGNamespaceHTTP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	rec := newKGStatusWriter(w)
	operation := r.Method + " " + strings.TrimSpace(r.URL.Path)
	namespace := ""
	suffix := ""
	appCtxForLog := middleware.AppContext{}

	ctx, span := otel.Tracer("github.com/blcvn/knowledge-gateway/kgs-platform/service").Start(
		r.Context(),
		"kg.namespace.http",
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.target", r.URL.Path),
		),
	)
	defer span.End()
	r = r.WithContext(ctx)

	defer func() {
		status := rec.StatusCode()
		if namespace != "" {
			span.SetAttributes(attribute.String("kg.namespace", namespace))
		}
		if suffix != "" {
			span.SetAttributes(attribute.String("kg.route_suffix", suffix))
			operation = r.Method + " /kg/{ns}" + suffix
		}
		if appCtxForLog.AppID != "" {
			span.SetAttributes(
				attribute.String("kg.app_id", appCtxForLog.AppID),
				attribute.String("kg.tenant_id", appCtxForLog.TenantID),
				attribute.String("kg.org_id", appCtxForLog.OrgID),
			)
		}
		span.SetAttributes(attribute.Int("http.status_code", status))
		if status >= http.StatusBadRequest {
			span.SetStatus(codes.Error, http.StatusText(status))
		} else {
			span.SetStatus(codes.Ok, "ok")
		}

		observeErr := error(nil)
		if status >= http.StatusBadRequest {
			observeErr = kerrors.New(status, "ERR_HTTP", http.StatusText(status))
		}
		observability.ObserveRequest(operation, started, observeErr)

		traceID := kgTraceIDFromContext(ctx)
		if status >= http.StatusBadRequest {
			stdlog.Printf("[KGS][KGNamespaceHTTP] completed method=%s path=%s status=%d bytes=%d app_id=%s tenant_id=%s org_id=%s namespace=%s suffix=%s trace_id=%s duration=%s",
				r.Method, r.URL.Path, status, rec.BytesWritten(), appCtxForLog.AppID, appCtxForLog.TenantID, appCtxForLog.OrgID, namespace, suffix, traceID, time.Since(started))
			return
		}
		stdlog.Printf("[KGS][KGNamespaceHTTP] completed method=%s path=%s status=%d bytes=%d app_id=%s tenant_id=%s org_id=%s namespace=%s suffix=%s trace_id=%s duration=%s",
			r.Method, r.URL.Path, status, rec.BytesWritten(), appCtxForLog.AppID, appCtxForLog.TenantID, appCtxForLog.OrgID, namespace, suffix, traceID, time.Since(started))
	}()

	stdlog.Printf("[KGS][KGNamespaceHTTP] start method=%s path=%s remote_addr=%s trace_id=%s",
		r.Method, r.URL.Path, r.RemoteAddr, kgTraceIDFromContext(ctx))

	route, err := parseKGNamespaceRoute(r.URL.Path)
	if err != nil {
		writeKGHTTPError(rec, http.StatusNotFound, "ERR_NOT_FOUND", "endpoint not found")
		return
	}
	namespace = route.Namespace
	suffix = route.Suffix

	appCtx, status, err := resolveKGAppContext(r, route.Namespace)
	if err != nil {
		reason := "ERR_UNAUTHORIZED"
		if status == http.StatusForbidden {
			reason = "ERR_FORBIDDEN"
		}
		stdlog.Printf("[KGS][KGNamespaceHTTP] auth failed namespace=%s suffix=%s status=%d reason=%s trace_id=%s err=%v",
			namespace, suffix, status, reason, kgTraceIDFromContext(ctx), err)
		writeKGHTTPError(rec, status, reason, err.Error())
		return
	}
	appCtxForLog = appCtx

	r = r.WithContext(context.WithValue(r.Context(), middleware.AppContextKey, appCtx))

	switch route.Suffix {
	case "/graph/batch":
		s.BatchUpsertGraphHTTP(rec, r)
	case "/entities":
		if r.Method != http.MethodGet {
			writeKGHTTPError(rec, http.StatusMethodNotAllowed, "ERR_METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		s.listKGEntitiesHTTP(rec, r)
	case "/entities/query":
		if r.Method != http.MethodPost {
			writeKGHTTPError(rec, http.StatusMethodNotAllowed, "ERR_METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		s.queryNodesHTTP(rec, r)
	case "/entities/lookup":
		if r.Method != http.MethodPost {
			writeKGHTTPError(rec, http.StatusMethodNotAllowed, "ERR_METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		s.lookupKGEntitiesHTTP(rec, r)
	case "/edges":
		if r.Method != http.MethodGet {
			writeKGHTTPError(rec, http.StatusMethodNotAllowed, "ERR_METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		s.listKGEdgesHTTP(rec, r)
	case "/nodes-by-label":
		if r.Method != http.MethodGet {
			writeKGHTTPError(rec, http.StatusMethodNotAllowed, "ERR_METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		s.getNodesByLabelHTTP(rec, r)
	case "/stats":
		if r.Method != http.MethodGet {
			writeKGHTTPError(rec, http.StatusMethodNotAllowed, "ERR_METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		s.getNamespaceStatsHTTP(rec, r)
	default:
		writeKGHTTPError(rec, http.StatusNotFound, "ERR_NOT_FOUND", "endpoint not found")
	}
}

func (s *GraphService) listKGEntitiesHTTP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	if s.entityReader == nil {
		writeKGHTTPError(w, http.StatusInternalServerError, "ERR_NOT_CONFIGURED", "entity reader is not configured")
		return
	}

	appCtx, err := getAppContext(r.Context())
	if err != nil {
		writeKGHTTPError(w, http.StatusUnauthorized, "ERR_UNAUTHORIZED", err.Error())
		return
	}

	query := r.URL.Query()
	limit, err := parseLimit(query.Get("limit"), kgEntitiesDefaultLimit, kgEntitiesMaxLimit)
	if err != nil {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", err.Error())
		return
	}
	cursorID, err := decodeKGCursor(query.Get("cursor"), "entityId")
	if err != nil {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", err.Error())
		return
	}
	propertyKey, propertyValue, err := parsePropertyFilter(query.Get("propertyFilter"))
	if err != nil {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", err.Error())
		return
	}
	isDeleted, err := parseIsDeleted(query.Get("isDeleted"))
	if err != nil {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", err.Error())
		return
	}

	versionID := strings.TrimSpace(query.Get("versionId"))
	if err := validateOptionalUUID(versionID, "versionId"); err != nil {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", err.Error())
		return
	}
	stdlog.Printf("[KGS][KGNamespaceHTTP] ListEntities start app_id=%s tenant_id=%s entity_type=%s source_file=%s domain=%s provenance_type=%s version_id=%s property_key=%s limit=%d has_cursor=%t is_deleted=%t trace_id=%s",
		appCtx.AppID,
		appCtx.TenantID,
		strings.TrimSpace(query.Get("entityType")),
		strings.TrimSpace(query.Get("sourceFile")),
		strings.TrimSpace(query.Get("domain")),
		strings.TrimSpace(query.Get("provenanceType")),
		versionID,
		propertyKey,
		limit,
		cursorID != "",
		isDeleted,
		kgTraceIDFromContext(r.Context()),
	)

	entities, nextCursorID, hasMore, total, err := s.entityReader.ListEntities(
		r.Context(),
		appCtx.AppID,
		appCtx.TenantID,
		limit,
		cursorID,
		strings.TrimSpace(query.Get("entityType")),
		strings.TrimSpace(query.Get("sourceFile")),
		strings.TrimSpace(query.Get("domain")),
		strings.TrimSpace(query.Get("provenanceType")),
		versionID,
		propertyKey,
		propertyValue,
		isDeleted,
	)
	if err != nil {
		stdlog.Printf("[KGS][KGNamespaceHTTP] ListEntities failed app_id=%s tenant_id=%s limit=%d cursor=%t trace_id=%s err=%v",
			appCtx.AppID, appCtx.TenantID, limit, cursorID != "", kgTraceIDFromContext(r.Context()), err)
		writeKGHTTPError(w, http.StatusInternalServerError, "ERR_INTERNAL", err.Error())
		return
	}

	resp := map[string]any{
		"entities":   entities,
		"nextCursor": nil,
		"hasMore":    hasMore,
		"total":      total,
	}
	if hasMore && strings.TrimSpace(nextCursorID) != "" {
		resp["nextCursor"] = encodeKGCursor("entityId", nextCursorID)
	}
	stdlog.Printf("[KGS][KGNamespaceHTTP] ListEntities done app_id=%s tenant_id=%s returned=%d total=%d has_more=%t trace_id=%s duration=%s",
		appCtx.AppID, appCtx.TenantID, len(entities), total, hasMore, kgTraceIDFromContext(r.Context()), time.Since(started))
	writeKGHTTPSuccess(w, resp)
}

func (s *GraphService) lookupKGEntitiesHTTP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	if s.entityReader == nil {
		writeKGHTTPError(w, http.StatusInternalServerError, "ERR_NOT_CONFIGURED", "entity reader is not configured")
		return
	}

	appCtx, err := getAppContext(r.Context())
	if err != nil {
		writeKGHTTPError(w, http.StatusUnauthorized, "ERR_UNAUTHORIZED", err.Error())
		return
	}

	var req kgLookupRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_SCHEMA_INVALID", fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if len(req.Properties) == 0 {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_SCHEMA_INVALID", "properties is required")
		return
	}

	matchMode := strings.ToUpper(strings.TrimSpace(req.MatchMode))
	if matchMode == "" {
		matchMode = "ALL"
	}
	if matchMode != "ALL" && matchMode != "ANY" {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_SCHEMA_INVALID", "matchMode must be ALL or ANY")
		return
	}

	limit := req.Limit
	if limit == 0 {
		limit = kgLookupDefaultLimit
	}
	if limit < 0 || limit > kgLookupMaxLimit {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_SCHEMA_INVALID", fmt.Sprintf("limit must be between 1 and %d", kgLookupMaxLimit))
		return
	}

	properties := make(map[string]string, len(req.Properties))
	for key, val := range req.Properties {
		k := strings.TrimSpace(key)
		if k == "" {
			writeKGHTTPError(w, http.StatusBadRequest, "ERR_SCHEMA_INVALID", "properties contains empty key")
			return
		}
		properties[k] = fmt.Sprint(val)
	}
	stdlog.Printf("[KGS][KGNamespaceHTTP] LookupEntities start app_id=%s tenant_id=%s entity_type=%s source_file=%s match_mode=%s limit=%d properties=%d trace_id=%s",
		appCtx.AppID,
		appCtx.TenantID,
		strings.TrimSpace(req.EntityType),
		strings.TrimSpace(req.SourceFile),
		matchMode,
		limit,
		len(properties),
		kgTraceIDFromContext(r.Context()),
	)

	entities, total, err := s.entityReader.LookupEntities(
		r.Context(),
		appCtx.AppID,
		appCtx.TenantID,
		strings.TrimSpace(req.EntityType),
		strings.TrimSpace(req.SourceFile),
		matchMode,
		limit,
		properties,
	)
	if err != nil {
		stdlog.Printf("[KGS][KGNamespaceHTTP] LookupEntities failed app_id=%s tenant_id=%s match_mode=%s limit=%d trace_id=%s err=%v",
			appCtx.AppID, appCtx.TenantID, matchMode, limit, kgTraceIDFromContext(r.Context()), err)
		writeKGHTTPError(w, http.StatusInternalServerError, "ERR_INTERNAL", err.Error())
		return
	}
	stdlog.Printf("[KGS][KGNamespaceHTTP] LookupEntities done app_id=%s tenant_id=%s returned=%d total=%d trace_id=%s duration=%s",
		appCtx.AppID, appCtx.TenantID, len(entities), total, kgTraceIDFromContext(r.Context()), time.Since(started))

	writeKGHTTPSuccess(w, map[string]any{
		"entities": entities,
		"total":    total,
	})
}

func (s *GraphService) listKGEdgesHTTP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	if s.entityReader == nil {
		writeKGHTTPError(w, http.StatusInternalServerError, "ERR_NOT_CONFIGURED", "entity reader is not configured")
		return
	}

	appCtx, err := getAppContext(r.Context())
	if err != nil {
		writeKGHTTPError(w, http.StatusUnauthorized, "ERR_UNAUTHORIZED", err.Error())
		return
	}

	query := r.URL.Query()
	limit, err := parseLimit(query.Get("limit"), kgEdgesDefaultLimit, kgEdgesMaxLimit)
	if err != nil {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", err.Error())
		return
	}
	cursorID, err := decodeKGCursor(query.Get("cursor"), "edgeId")
	if err != nil {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", err.Error())
		return
	}
	isDeleted, err := parseIsDeleted(query.Get("isDeleted"))
	if err != nil {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", err.Error())
		return
	}

	fromEntityID := strings.TrimSpace(query.Get("fromEntityId"))
	toEntityID := strings.TrimSpace(query.Get("toEntityId"))
	versionID := strings.TrimSpace(query.Get("versionId"))
	if err := validateOptionalUUID(versionID, "versionId"); err != nil {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", err.Error())
		return
	}
	stdlog.Printf("[KGS][KGNamespaceHTTP] ListEdges start app_id=%s tenant_id=%s relation_type=%s source_file=%s from=%s to=%s version_id=%s limit=%d has_cursor=%t is_deleted=%t trace_id=%s",
		appCtx.AppID,
		appCtx.TenantID,
		strings.TrimSpace(query.Get("relationType")),
		strings.TrimSpace(query.Get("sourceFile")),
		fromEntityID,
		toEntityID,
		versionID,
		limit,
		cursorID != "",
		isDeleted,
		kgTraceIDFromContext(r.Context()),
	)

	edges, nextCursorID, hasMore, total, err := s.entityReader.ListEdges(
		r.Context(),
		appCtx.AppID,
		appCtx.TenantID,
		limit,
		cursorID,
		strings.TrimSpace(query.Get("relationType")),
		strings.TrimSpace(query.Get("sourceFile")),
		fromEntityID,
		toEntityID,
		versionID,
		isDeleted,
	)
	if err != nil {
		stdlog.Printf("[KGS][KGNamespaceHTTP] ListEdges failed app_id=%s tenant_id=%s limit=%d cursor=%t trace_id=%s err=%v",
			appCtx.AppID, appCtx.TenantID, limit, cursorID != "", kgTraceIDFromContext(r.Context()), err)
		writeKGHTTPError(w, http.StatusInternalServerError, "ERR_INTERNAL", err.Error())
		return
	}

	resp := map[string]any{
		"edges":      edges,
		"nextCursor": nil,
		"hasMore":    hasMore,
		"total":      total,
	}
	if hasMore && strings.TrimSpace(nextCursorID) != "" {
		resp["nextCursor"] = encodeKGCursor("edgeId", nextCursorID)
	}
	stdlog.Printf("[KGS][KGNamespaceHTTP] ListEdges done app_id=%s tenant_id=%s returned=%d total=%d has_more=%t trace_id=%s duration=%s",
		appCtx.AppID, appCtx.TenantID, len(edges), total, hasMore, kgTraceIDFromContext(r.Context()), time.Since(started))
	writeKGHTTPSuccess(w, resp)
}

func parseKGNamespaceRoute(path string) (kgNamespaceRoute, error) {
	cleaned := strings.Trim(strings.TrimSpace(path), "/")
	if !strings.HasPrefix(cleaned, "kg/") {
		return kgNamespaceRoute{}, fmt.Errorf("invalid kg path")
	}
	rest := strings.TrimPrefix(cleaned, "kg/")
	patterns := []string{"/entities/lookup", "/entities/query", "/entities", "/edges", "/nodes-by-label", "/stats", "/graph/batch"}
	for _, suffix := range patterns {
		if !strings.HasSuffix(rest, suffix) {
			continue
		}
		namespacePart := strings.Trim(strings.TrimSuffix(rest, suffix), "/")
		if namespacePart == "" {
			return kgNamespaceRoute{}, fmt.Errorf("missing namespace")
		}
		if decoded, err := url.PathUnescape(namespacePart); err == nil {
			namespacePart = decoded
		}
		return kgNamespaceRoute{Namespace: namespacePart, Suffix: suffix}, nil
	}
	return kgNamespaceRoute{}, fmt.Errorf("unsupported kg endpoint")
}

func resolveKGAppContext(r *http.Request, namespace string) (middleware.AppContext, int, error) {
	if r == nil {
		return middleware.AppContext{}, http.StatusUnauthorized, fmt.Errorf("missing app context")
	}
	pathCtx, err := appContextFromNamespace(namespace)
	if err != nil {
		return middleware.AppContext{}, http.StatusUnauthorized, fmt.Errorf("missing app context")
	}

	if headerNS := strings.TrimSpace(r.Header.Get("X-KG-Namespace")); headerNS != "" && headerNS != namespace {
		return middleware.AppContext{}, http.StatusForbidden, fmt.Errorf("namespace does not match application context")
	}

	if appCtx, err := getAppContext(r.Context()); err == nil {
		expectedNS := biz.ComputeNamespace(appCtx.AppID, appCtx.TenantID, appCtx.OrgID)
		if expectedNS != namespace {
			return middleware.AppContext{}, http.StatusForbidden, fmt.Errorf("namespace does not match application context")
		}
		return appCtx, 0, nil
	}

	if !hasAnyAPIKeyHeader(r) {
		return middleware.AppContext{}, http.StatusUnauthorized, fmt.Errorf("missing API key")
	}
	return pathCtx, 0, nil
}

func hasAnyAPIKeyHeader(r *http.Request) bool {
	if r == nil {
		return false
	}
	if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
		return true
	}
	if strings.TrimSpace(r.Header.Get("X-API-Key")) != "" {
		return true
	}
	return false
}

func parseLimit(raw string, defaultValue, maxValue int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("limit must be an integer")
	}
	if value <= 0 || value > maxValue {
		return 0, fmt.Errorf("limit must be between 1 and %d", maxValue)
	}
	return value, nil
}

func parsePropertyFilter(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", nil
	}
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("propertyFilter must have format key:value")
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" || value == "" {
		return "", "", fmt.Errorf("propertyFilter must have non-empty key and value")
	}
	return key, value, nil
}

func parseIsDeleted(raw string) (bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("isDeleted must be a boolean")
	}
	return value, nil
}

func validateOptionalUUID(value string, field string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if _, err := uuid.Parse(value); err != nil {
		return fmt.Errorf("%s must be a valid UUID", field)
	}
	return nil
}

func decodeKGCursor(raw, field string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return "", fmt.Errorf("invalid cursor")
		}
	}
	payload := map[string]string{}
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return "", fmt.Errorf("invalid cursor")
	}
	id := strings.TrimSpace(payload[field])
	if id == "" {
		return "", fmt.Errorf("invalid cursor")
	}
	return id, nil
}

func encodeKGCursor(field, value string) string {
	payload := map[string]string{field: strings.TrimSpace(value)}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func writeKGHTTPSuccess(w http.ResponseWriter, payload map[string]any) {
	if payload == nil {
		payload = map[string]any{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeKGHTTPError(w http.ResponseWriter, status int, reason, message string) {
	if status <= 0 {
		status = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":     status,
		"reason":   reason,
		"message":  message,
		"metadata": map[string]any{},
	})
}

type kgStatusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func newKGStatusWriter(w http.ResponseWriter) *kgStatusWriter {
	return &kgStatusWriter{ResponseWriter: w, status: http.StatusOK}
}

func (w *kgStatusWriter) WriteHeader(code int) {
	if code > 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *kgStatusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

func (w *kgStatusWriter) StatusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *kgStatusWriter) BytesWritten() int {
	return w.bytes
}

func kgTraceIDFromContext(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return ""
	}
	return spanCtx.TraceID().String()
}

func (s *GraphService) getNodesByLabelHTTP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	if s.entityReader == nil {
		writeKGHTTPError(w, http.StatusInternalServerError, "ERR_NOT_CONFIGURED", "entity reader is not configured")
		return
	}

	appCtx, err := getAppContext(r.Context())
	if err != nil {
		writeKGHTTPError(w, http.StatusUnauthorized, "ERR_UNAUTHORIZED", err.Error())
		return
	}

	query := r.URL.Query()
	label := strings.TrimSpace(query.Get("label"))
	if label == "" {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", "label query parameter is required")
		return
	}

	limit, err := parseLimit(query.Get("limit"), kgNodesByLabelDefaultLimit, kgNodesByLabelMaxLimit)
	if err != nil {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", err.Error())
		return
	}
	cursorID, err := decodeKGCursor(query.Get("cursor"), "entityId")
	if err != nil {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_BAD_REQUEST", err.Error())
		return
	}

	stdlog.Printf("[KGS][KGNamespaceHTTP] GetNodesByLabel start app_id=%s tenant_id=%s label=%s limit=%d has_cursor=%t trace_id=%s",
		appCtx.AppID, appCtx.TenantID, label, limit, cursorID != "", kgTraceIDFromContext(r.Context()))

	entities, nextCursorID, hasMore, total, err := s.entityReader.GetNodesByLabel(
		r.Context(),
		appCtx.AppID,
		appCtx.TenantID,
		label,
		limit,
		cursorID,
	)
	if err != nil {
		stdlog.Printf("[KGS][KGNamespaceHTTP] GetNodesByLabel failed app_id=%s tenant_id=%s label=%s limit=%d trace_id=%s err=%v",
			appCtx.AppID, appCtx.TenantID, label, limit, kgTraceIDFromContext(r.Context()), err)
		writeKGHTTPError(w, http.StatusInternalServerError, "ERR_INTERNAL", err.Error())
		return
	}

	// Convert entities to GraphNode format for compatibility with proto types
	nodes := make([]map[string]any, 0, len(entities))
	for _, entity := range entities {
		node := map[string]any{
			"id":    entity["id"],
			"label": entity["entity_type"],
		}
		if propsJSON, err := json.Marshal(entity); err == nil {
			node["properties_json"] = string(propsJSON)
		}
		node["properties"] = entity
		nodes = append(nodes, node)
	}

	resp := map[string]any{
		"nodes":      nodes,
		"total":      total,
		"nextCursor": nil,
		"hasMore":    hasMore,
	}
	if hasMore && strings.TrimSpace(nextCursorID) != "" {
		resp["nextCursor"] = encodeKGCursor("entityId", nextCursorID)
	}
	stdlog.Printf("[KGS][KGNamespaceHTTP] GetNodesByLabel done app_id=%s tenant_id=%s label=%s returned=%d total=%d has_more=%t trace_id=%s duration=%s",
		appCtx.AppID, appCtx.TenantID, label, len(nodes), total, hasMore, kgTraceIDFromContext(r.Context()), time.Since(started))
	writeKGHTTPSuccess(w, resp)
}

func (s *GraphService) getNamespaceStatsHTTP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	if s.entityReader == nil {
		writeKGHTTPError(w, http.StatusInternalServerError, "ERR_NOT_CONFIGURED", "entity reader is not configured")
		return
	}

	appCtx, err := getAppContext(r.Context())
	if err != nil {
		writeKGHTTPError(w, http.StatusUnauthorized, "ERR_UNAUTHORIZED", err.Error())
		return
	}

	query := r.URL.Query()
	labelsRaw := strings.TrimSpace(query.Get("labels"))
	var labels []string
	if labelsRaw != "" {
		for _, l := range strings.Split(labelsRaw, ",") {
			if trimmed := strings.TrimSpace(l); trimmed != "" {
				labels = append(labels, trimmed)
			}
		}
	}

	stdlog.Printf("[KGS][KGNamespaceHTTP] GetNamespaceStats start app_id=%s tenant_id=%s labels=%v trace_id=%s",
		appCtx.AppID, appCtx.TenantID, labels, kgTraceIDFromContext(r.Context()))

	totalNodes, totalEdges, byLabel, err := s.entityReader.GetNamespaceStats(r.Context(), appCtx.AppID, appCtx.TenantID, labels)
	if err != nil {
		stdlog.Printf("[KGS][KGNamespaceHTTP] GetNamespaceStats failed app_id=%s tenant_id=%s trace_id=%s err=%v",
			appCtx.AppID, appCtx.TenantID, kgTraceIDFromContext(r.Context()), err)
		writeKGHTTPError(w, http.StatusInternalServerError, "ERR_INTERNAL", err.Error())
		return
	}

	stdlog.Printf("[KGS][KGNamespaceHTTP] GetNamespaceStats done app_id=%s tenant_id=%s total_nodes=%d total_edges=%d label_count=%d trace_id=%s duration=%s",
		appCtx.AppID, appCtx.TenantID, totalNodes, totalEdges, len(byLabel), kgTraceIDFromContext(r.Context()), time.Since(started))

	// Convert map to sorted list for JSON output
	type labelEntry struct {
		Label string `json:"label"`
		Count int64  `json:"count"`
	}
	labelList := make([]labelEntry, 0, len(byLabel))
	for label, count := range byLabel {
		labelList = append(labelList, labelEntry{Label: label, Count: count})
	}
	sort.Slice(labelList, func(i, j int) bool { return labelList[i].Label < labelList[j].Label })

	writeKGHTTPSuccess(w, map[string]any{
		"total_nodes": totalNodes,
		"total_edges": totalEdges,
		"by_label":    labelList,
	})
}

type queryNodesRequest struct {
	Labels         []string          `json:"labels"`
	PropertyEq     map[string]string `json:"property_eq"`
	PropertyExists []string          `json:"property_exists"`
	OrderBy        string            `json:"order_by"`
	Limit          int               `json:"limit"`
	Offset         int               `json:"offset"`
}

func (s *GraphService) queryNodesHTTP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	if s.entityReader == nil {
		writeKGHTTPError(w, http.StatusInternalServerError, "ERR_NOT_CONFIGURED", "entity reader is not configured")
		return
	}

	appCtx, err := getAppContext(r.Context())
	if err != nil {
		writeKGHTTPError(w, http.StatusUnauthorized, "ERR_UNAUTHORIZED", err.Error())
		return
	}

	var req queryNodesRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeKGHTTPError(w, http.StatusBadRequest, "ERR_SCHEMA_INVALID", fmt.Sprintf("invalid request body: %v", err))
		return
	}

	stdlog.Printf("[KGS][KGNamespaceHTTP] QueryNodes start app_id=%s tenant_id=%s labels=%v property_eq_keys=%d property_exists=%v limit=%d offset=%d trace_id=%s",
		appCtx.AppID, appCtx.TenantID, req.Labels, len(req.PropertyEq), req.PropertyExists, req.Limit, req.Offset, kgTraceIDFromContext(r.Context()))

	filter := data.QueryNodesFilter{
		Labels:         req.Labels,
		PropertyEq:     req.PropertyEq,
		PropertyExists: req.PropertyExists,
		OrderBy:        req.OrderBy,
		Limit:          req.Limit,
		Offset:         req.Offset,
	}
	entities, total, err := s.entityReader.QueryNodes(r.Context(), appCtx.AppID, appCtx.TenantID, filter)
	if err != nil {
		stdlog.Printf("[KGS][KGNamespaceHTTP] QueryNodes failed app_id=%s tenant_id=%s trace_id=%s err=%v",
			appCtx.AppID, appCtx.TenantID, kgTraceIDFromContext(r.Context()), err)
		writeKGHTTPError(w, http.StatusInternalServerError, "ERR_INTERNAL", err.Error())
		return
	}

	stdlog.Printf("[KGS][KGNamespaceHTTP] QueryNodes done app_id=%s tenant_id=%s returned=%d total=%d trace_id=%s duration=%s",
		appCtx.AppID, appCtx.TenantID, len(entities), total, kgTraceIDFromContext(r.Context()), time.Since(started))
	writeKGHTTPSuccess(w, map[string]any{
		"entities": entities,
		"total":    total,
	})
}
