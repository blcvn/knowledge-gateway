package bridge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const defaultDomainID = "code-graph"

type Config struct {
	ProjectPath           string
	ProjectID             string
	CodeGraphDBPath       string
	KGServiceURL          string
	KGAPIKey              string
	KGDomainID            string
	KGVisibility          string
	StateDir              string
	TemplateDomainID      string
	DefaultTopK           int
	TemplateTimeoutSec    int
	NodeBatchSize         int
	RelationshipBatchSize int
}

func LoadConfig() (Config, error) {
	projectPath := envOr("PROJECT_PATH", ".")
	projectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return Config{}, err
	}

	projectID := envOr("PROJECT_ID", filepath.Base(projectPath))
	kgURL := strings.TrimRight(envOr("KG_SERVICE_URL", ""), "/")
	if kgURL == "" {
		return Config{}, errors.New("KG_SERVICE_URL is required")
	}
	apiKey := strings.TrimSpace(os.Getenv("KG_API_KEY"))
	if apiKey == "" {
		return Config{}, errors.New("KG_API_KEY is required")
	}

	codegraphDBPath := strings.TrimSpace(os.Getenv("CODEGRAPH_DB_PATH"))
	if codegraphDBPath == "" {
		codegraphDBPath = filepath.Join(projectPath, ".codegraph", "codegraph.db")
	} else if !filepath.IsAbs(codegraphDBPath) {
		codegraphDBPath = filepath.Join(projectPath, codegraphDBPath)
	}

	stateDir := envOr("KG_STATE_DIR", filepath.Join("codegraph-sync", ".state"))
	if !filepath.IsAbs(stateDir) {
		stateDir = filepath.Join(projectPath, stateDir)
	}

	defaultTopK := 10
	if raw := strings.TrimSpace(os.Getenv("KG_DEFAULT_TOP_K")); raw != "" {
		if value, convErr := strconv.Atoi(raw); convErr == nil && value > 0 {
			defaultTopK = value
		}
	}

	timeoutSec := 30
	if raw := strings.TrimSpace(os.Getenv("KG_TEMPLATE_TIMEOUT_SEC")); raw != "" {
		if value, convErr := strconv.Atoi(raw); convErr == nil && value > 0 {
			timeoutSec = value
		}
	}

	nodeBatchSize := 200
	if raw := strings.TrimSpace(os.Getenv("KG_SYNC_NODE_BATCH_SIZE")); raw != "" {
		if value, convErr := strconv.Atoi(raw); convErr == nil && value > 0 {
			nodeBatchSize = value
		}
	}

	relationshipBatchSize := 200
	if raw := strings.TrimSpace(os.Getenv("KG_SYNC_RELATIONSHIP_BATCH_SIZE")); raw != "" {
		if value, convErr := strconv.Atoi(raw); convErr == nil && value > 0 {
			relationshipBatchSize = value
		}
	}

	return Config{
		ProjectPath:           projectPath,
		ProjectID:             projectID,
		CodeGraphDBPath:       codegraphDBPath,
		KGServiceURL:          kgURL,
		KGAPIKey:              apiKey,
		KGDomainID:            envOr("KG_DOMAIN_ID", defaultDomainID),
		KGVisibility:          envOr("KG_VISIBILITY", "private"),
		StateDir:              stateDir,
		TemplateDomainID:      envOr("KG_TEMPLATE_DOMAIN_ID", defaultDomainID),
		DefaultTopK:           defaultTopK,
		TemplateTimeoutSec:    timeoutSec,
		NodeBatchSize:         nodeBatchSize,
		RelationshipBatchSize: relationshipBatchSize,
	}, nil
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func (c Config) StatePath() string {
	return filepath.Join(c.StateDir, sanitizePathComponent(c.ProjectID)+".json")
}

func sanitizePathComponent(value string) string {
	value = strings.ReplaceAll(value, string(filepath.Separator), "_")
	value = strings.ReplaceAll(value, ":", "_")
	value = strings.TrimSpace(value)
	if value == "" {
		return "project"
	}
	return value
}

func (c Config) String() string {
	return fmt.Sprintf("project=%s projectPath=%s domain=%s service=%s", c.ProjectID, c.ProjectPath, c.KGDomainID, c.KGServiceURL)
}
