package orchestration

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "github.com/google/uuid"
    "vnp-memory/services/orchestration-service/internal/domain"
    "vnp-memory/services/orchestration-service/internal/usecase/port"
)

type SketchService struct {
    repo        port.ISketchRepo
    actionRepo  port.IActionRepo
    crystalRepo port.ICrystalRepo
    llm         port.ILLMProvider
}

func (s *SketchService) Promote(ctx context.Context, sketchID string) (*domain.Crystal, error) {
    sketch, err := s.repo.Get(ctx, sketchID)
    if err != nil { return nil, domain.ErrSketchNotFound }

    // Collect action results
    var summaries []string
    var actionIDs []string
    for _, aid := range sketch.ActionIDs {
        action, _ := s.actionRepo.Get(ctx, aid)
        if action != nil && action.Result != "" {
            summaries = append(summaries, action.Title+": "+action.Result)
            actionIDs = append(actionIDs, aid)
        }
    }

    crystal := domain.Crystal{
        ID:              uuid.New().String(),
        TenantID:        sketch.TenantID,
        SourceActionIDs: actionIDs,
        CreatedAt:       time.Now(),
    }

    // Try LLM crystal generation
    if s.llm != nil && len(summaries) > 0 {
        prompt := buildCrystalPrompt(summaries)
        resp, err := s.llm.Chat(ctx, "Generate a crystal memory from these action results.", prompt)
        if err == nil {
            var result struct {
                Narrative   string   `json:"narrative"`
                KeyOutcomes []string `json:"key_outcomes"`
                Lessons     []string `json:"lessons"`
            }
            if json.Unmarshal([]byte(resp), &result) == nil {
                crystal.Narrative   = result.Narrative
                crystal.KeyOutcomes = result.KeyOutcomes
                crystal.Lessons     = result.Lessons
            }
        }
    }

    // Graceful degrade: synthetic crystal
    if crystal.Narrative == "" {
        crystal.Narrative   = strings.Join(summaries, "; ")
        crystal.KeyOutcomes = summaries
    }

    if err := s.crystalRepo.Save(ctx, crystal); err != nil { return nil, err }
    s.repo.SetStatus(ctx, sketchID, "promoted")
    return &crystal, nil
}

func buildCrystalPrompt(summaries []string) string {
    return fmt.Sprintf(`Synthesize these action results into a crystal memory.
Return JSON: {"narrative": "...", "key_outcomes": [...], "lessons": [...]}

Actions:
%s`, strings.Join(summaries, "\n"))
}
