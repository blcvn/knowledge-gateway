package admin

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/admin"
)

var ErrSnapshotDisabled = fmt.Errorf("snapshots are disabled")

type SnapshotUseCase struct {
	db      *pgxpool.Pool
	dataDir string
	enabled bool
}

func NewSnapshotUseCase(db *pgxpool.Pool, dataDir string, enabled bool) *SnapshotUseCase {
	return &SnapshotUseCase{db: db, dataDir: dataDir, enabled: enabled}
}

func (uc *SnapshotUseCase) Create(ctx context.Context) (*admin.SnapshotMeta, error) {
	if !uc.enabled {
		return nil, ErrSnapshotDisabled
	}

	stats, err := uc.collectStats(ctx)
	if err != nil {
		return nil, err
	}

	commitMsg := fmt.Sprintf("snapshot: %d sessions, %d memories (%s)",
		stats.Sessions, stats.Memories, time.Now().Format(time.RFC3339))

	cmds := [][]string{
		{"git", "-C", uc.dataDir, "add", "."},
		{"git", "-C", uc.dataDir, "commit", "-m", commitMsg, "--allow-empty"},
	}

	var commitHash string
	for _, cmd := range cmds {
		out, err := exec.CommandContext(ctx, cmd[0], cmd[1:]...).Output()
		if err != nil {
			// git commit may fail if nothing to commit; this is acceptable
			if cmd[2] == "commit" {
				commitHash = "no-changes"
			} else {
				return nil, err
			}
		}
		if len(out) > 0 && cmd[2] == "commit" {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 0 {
				commitHash = strings.TrimSpace(lines[0])
			}
		}
	}

	return &admin.SnapshotMeta{
		CommitHash: commitHash,
		Stats:      stats,
		CreatedAt:  time.Now(),
	}, nil
}

func (uc *SnapshotUseCase) collectStats(ctx context.Context) (admin.SnapshotStats, error) {
	var stats admin.SnapshotStats
	row := uc.db.QueryRow(ctx, `SELECT COUNT(*) FROM agent_sessions`)
	row.Scan(&stats.Sessions)
	row = uc.db.QueryRow(ctx, `SELECT COUNT(*) FROM raw_observations`)
	row.Scan(&stats.Observations)
	row = uc.db.QueryRow(ctx, `SELECT COUNT(*) FROM agent_memories WHERE is_latest = TRUE`)
	row.Scan(&stats.Memories)
	return stats, nil
}
