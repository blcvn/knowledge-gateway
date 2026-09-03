package admin

import "time"

type HealthSnapshot struct {
    Status        string         `json:"status"`   // "healthy" | "degraded" | "critical"
    Timestamp     time.Time      `json:"timestamp"`
    UptimeSeconds float64        `json:"uptime_seconds"`
    Workers       []WorkerStatus `json:"workers"`
    Memory        MemoryUsage    `json:"memory"`
    CPU           CPUUsage       `json:"cpu"`
    Indexes       IndexHealth    `json:"indexes"`
    Connections   ConnHealth     `json:"connections"`
    Alerts        []string       `json:"alerts,omitempty"`
}

type WorkerStatus struct {
    Name    string    `json:"name"`
    Status  string    `json:"status"`  // "running" | "stopped" | "error"
    LastRun time.Time `json:"last_run,omitempty"`
    Error   string    `json:"error,omitempty"`
}

type MemoryUsage struct {
    HeapMB    float64 `json:"heap_mb"`
    AllocMB   float64 `json:"alloc_mb"`
    GoroutineCount int `json:"goroutine_count"`
}

type CPUUsage struct {
    NumCPU    int     `json:"num_cpu"`
    GoMaxProcs int    `json:"go_max_procs"`
}

type IndexHealth struct {
    BM25Documents   int       `json:"bm25_documents"`
    VectorDocuments int       `json:"vector_documents"`
    LastPersisted   time.Time `json:"last_persisted,omitempty"`
    Status          string    `json:"status"`
}

type ConnHealth struct {
    PostgreSQL    string `json:"postgresql"`
    NATS         string `json:"nats"`
    Bifrost      string `json:"bifrost,omitempty"`
    ObserveSearch string `json:"observe_search"`
}

type DiagnosticCheck struct {
    Check      string `json:"check"`
    Status     string `json:"status"`     // "ok" | "warning" | "error"
    Message    string `json:"message"`
    Suggestion string `json:"suggestion,omitempty"`
}

type SnapshotMeta struct {
    CommitHash string        `json:"commit_hash"`
    Stats      SnapshotStats `json:"stats"`
    CreatedAt  time.Time     `json:"created_at"`
}

type SnapshotStats struct {
    Sessions     int `json:"sessions"`
    Observations int `json:"observations"`
    Memories     int `json:"memories"`
}
