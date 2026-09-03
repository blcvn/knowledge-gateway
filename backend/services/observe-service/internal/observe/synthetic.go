package observe

import (
    "fmt"
    "strings"

    "github.com/vnp-memory/services/observe-service/internal/domain"
)

// syntheticCompress converts RawObservation → CompressedObservation without LLM
func syntheticCompress(raw domain.RawObservation) domain.CompressedObservation {
    obs := domain.CompressedObservation{
        SessionID: raw.SessionID,
        TenantID:  raw.TenantID,
        AgentID:   raw.AgentID,
        Timestamp: raw.Timestamp,
        Confidence: 0.8,
    }
    
    switch domain.HookType(raw.HookType) {
    case domain.HookPostToolUse:
        obs.ObsType   = string(deriveObsType(raw.ToolName))
        obs.Title     = fmt.Sprintf("%s: %s", raw.ToolName, extractFirstLine(raw.ToolOutput))
        obs.Files     = extractFilePaths(raw.ToolInput, raw.ToolOutput)
        obs.Facts     = extractFacts(raw.ToolOutput)
        obs.Importance = 0.5

    case domain.HookPostToolFailure:
        obs.ObsType    = string(domain.ObsError)
        obs.Title      = fmt.Sprintf("ERROR in %s: %s", raw.ToolName, extractErrorMsg(raw.ToolOutput))
        obs.Facts      = []string{extractErrorMsg(raw.ToolOutput)}
        obs.Importance = 0.8

    case domain.HookPromptSubmit:
        obs.ObsType    = string(domain.ObsConversation)
        obs.Title      = truncate(raw.UserPrompt, 80)
        obs.Importance = 0.3

    case domain.HookTaskCompleted:
        obs.ObsType    = string(domain.ObsDecision)
        obs.Title      = "Task completed"
        obs.Importance = 0.7

    default:
        obs.ObsType    = string(domain.ObsToolCall)
        obs.Title      = fmt.Sprintf("[%s] %s", raw.HookType, raw.ToolName)
        obs.Importance = 0.4
    }
    
    obs.Concepts = extractConcepts(obs.Title, obs.Facts)
    return obs
}

func deriveObsType(toolName string) domain.ObsType {
    switch {
    case strings.Contains(toolName, "write") || strings.Contains(toolName, "create"):
        return domain.ObsFileWrite
    case strings.Contains(toolName, "read") || strings.Contains(toolName, "view"):
        return domain.ObsFileRead
    case strings.Contains(toolName, "search") || strings.Contains(toolName, "grep"):
        return domain.ObsSearch
    case strings.Contains(toolName, "run") || strings.Contains(toolName, "exec"):
        return domain.ObsExec
    case strings.Contains(toolName, "git"):
        return domain.ObsCommit
    default:
        return domain.ObsToolCall
    }
}

func extractFirstLine(data []byte) string {
    if len(data) == 0 { return "" }
    s := strings.Split(string(data), "\n")[0]
    return truncate(s, 60)
}

func extractErrorMsg(data []byte) string {
    s := string(data)
    if idx := strings.Index(s, "error"); idx >= 0 {
        return truncate(s[idx:], 80)
    }
    return truncate(s, 80)
}

func truncate(s string, n int) string {
    if len(s) <= n { return s }
    return s[:n] + "..."
}

func extractFilePaths(input, output []byte) []string {
    var paths []string
    for _, b := range [][]byte{input, output} {
        s := string(b)
        for _, word := range strings.Fields(s) {
            if strings.HasPrefix(word, "/") && strings.Contains(word, ".") {
                paths = append(paths, word)
            }
        }
    }
    return paths
}

func extractFacts(output []byte) []string {
    if len(output) == 0 { return nil }
    lines := strings.Split(string(output), "\n")
    var facts []string
    for _, l := range lines {
        l = strings.TrimSpace(l)
        if len(l) > 10 && len(l) < 200 { facts = append(facts, l) }
        if len(facts) >= 5 { break }
    }
    return facts
}

func extractConcepts(title string, facts []string) []string {
    words := strings.Fields(title)
    for _, f := range facts { words = append(words, strings.Fields(f)...) }
    seen := map[string]bool{}
    var concepts []string
    for _, w := range words {
        w = strings.ToLower(strings.Trim(w, ".,;:()[]\"'"))
        if len(w) >= 4 && !seen[w] && !isStopword(w) {
            seen[w] = true
            concepts = append(concepts, w)
            if len(concepts) >= 8 { break }
        }
    }
    return concepts
}

var stopwords = map[string]bool{"this": true, "that": true, "with": true, "from": true, "have": true, "will": true, "been": true}
func isStopword(w string) bool { return stopwords[w] }
func detectModality(rawJSON []byte) string {
    if strings.Contains(string(rawJSON), "base64") { return "image" }
    return "text"
}
