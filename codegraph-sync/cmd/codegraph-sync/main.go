package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kg-service/codegraph-sync/internal/bridge"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: codegraph-sync <build|sync|sync:dry|mcp>")
		os.Exit(2)
	}

	cfg, err := bridge.LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "build":
		if err := bridge.EnsureBuildLayout(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "build error:", err)
			os.Exit(1)
		}
	case "sync":
		runSync(cfg, false, hasArg("--full"))
	case "sync:dry":
		runSync(cfg, true, false)
	case "mcp":
		if err := bridge.RunMCP(context.Background(), cfg); err != nil {
			fmt.Fprintln(os.Stderr, "mcp error:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: codegraph-sync <build|sync|sync:dry|mcp>")
		os.Exit(2)
	}
}

func runSync(cfg bridge.Config, dryRun bool, fullReindex bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	report, err := bridge.SyncProject(ctx, cfg, dryRun, fullReindex)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sync error:", err)
		os.Exit(1)
	}

	mode := "sync"
	if dryRun {
		mode = "sync:dry"
	}
	fmt.Fprintf(os.Stdout, "[%s] project=%s nodes=%d relationships=%d created=%d updated=%d skipped_relationships=%d duration=%s\n",
		mode,
		report.ProjectID,
		report.NodeCount,
		report.RelationshipCount,
		report.CreatedNodes,
		report.UpdatedNodes,
		report.SkippedRelationships,
		report.Duration.Truncate(time.Millisecond),
	)
	if report.OutputDir != "" {
		fmt.Fprintf(os.Stdout, "[%s] state=%s\n", mode, filepath.Clean(report.OutputDir))
	}
}

func hasArg(flag string) bool {
	for _, arg := range os.Args[2:] {
		if strings.TrimSpace(arg) == flag {
			return true
		}
	}
	return false
}
