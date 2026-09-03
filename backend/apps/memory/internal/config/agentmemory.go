package config

import (
    "os"
    "strconv"
    "time"
)

type AgentMemoryConfig struct {
    // Observe
    MaxObsPerSession int
    DedupTTL         time.Duration
    InjectContext    bool
    TokenBudget      int

    // Memory lifecycle
    StrengthDefault    float64
    HalfLifeDays       int
    MaxMemoriesProject int
    AgentScope         string

    // Consolidation
    ConsolidationIntervalHours int
    MinProcedureFrequency      int
    LessonHalfLifeDays         int
    AutoCompress               bool
}

func LoadAgentMemoryConfig() AgentMemoryConfig {
    return AgentMemoryConfig{
        MaxObsPerSession:           getEnvInt("AGENTMEMORY_MAX_OBS_PER_SESSION", 500),
        DedupTTL:                   time.Duration(getEnvInt("AGENTMEMORY_DEDUP_TTL", 30)) * time.Second,
        InjectContext:               getEnvBool("AGENTMEMORY_INJECT_CONTEXT", false),
        TokenBudget:                 getEnvInt("AGENTMEMORY_TOKEN_BUDGET", 2000),
        StrengthDefault:             0.7,
        HalfLifeDays:                getEnvInt("AGENTMEMORY_HALF_LIFE_DAYS", 30),
        MaxMemoriesProject:          getEnvInt("AGENTMEMORY_MAX_MEMORIES_PROJECT", 1000),
        AgentScope:                  getEnvStr("AGENTMEMORY_AGENT_SCOPE", "shared"),
        ConsolidationIntervalHours:  getEnvInt("AGENTMEMORY_CONSOLIDATION_INTERVAL", 2),
        MinProcedureFrequency:       getEnvInt("AGENTMEMORY_MIN_PROCEDURE_FREQ", 3),
        LessonHalfLifeDays:          getEnvInt("AGENTMEMORY_LESSON_HALF_LIFE", 90),
        AutoCompress:                getEnvBool("AGENTMEMORY_AUTO_COMPRESS", false),
    }
}

func getEnvInt(key string, def int) int {
    if v := os.Getenv(key); v != "" {
        if n, err := strconv.Atoi(v); err == nil { return n }
    }
    return def
}

func getEnvBool(key string, def bool) bool {
    if v := os.Getenv(key); v != "" {
        return v == "true" || v == "1" || v == "yes"
    }
    return def
}

func getEnvStr(key, def string) string {
    if v := os.Getenv(key); v != "" { return v }
    return def
}
