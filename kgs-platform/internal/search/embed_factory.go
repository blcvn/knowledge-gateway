package search

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	embeddingProviderDeterministic = "deterministic"
	embeddingProviderOpenAI        = "openai"
	embeddingProviderAIProxy       = "ai-proxy"
	embeddingProviderVNP           = "vnp"
	defaultOpenAIBaseURL           = "https://api.openai.com/v1"
	defaultAIProxyBaseURL          = "http://ai-proxy:8080"
	defaultAIProxyEmbedPath        = "/ai/embeddings"
	defaultOpenAIModel             = "text-embedding-3-small"
	defaultEmbeddingVectorSize     = 1536
	defaultEmbeddingTimeout        = 15 * time.Second
)

func NewEmbeddingClient(cfg *conf.Data, logger log.Logger) (EmbeddingClient, error) {
	provider := embeddingProviderDeterministic
	vectorSize := defaultVectorSize(cfg)
	if cfg != nil {
		if emb := cfg.GetEmbedding(); emb != nil {
			if raw := strings.TrimSpace(emb.GetProvider()); raw != "" {
				provider = strings.ToLower(raw)
			}
			if emb.GetVectorSize() > 0 {
				vectorSize = int(emb.GetVectorSize())
			}
		}
	}

	switch provider {
	case embeddingProviderDeterministic:
		return NewDeterministicEmbeddingClient(vectorSize), nil
	case embeddingProviderOpenAI:
		return newOpenAIEmbeddingClient(cfg, logger, vectorSize)
	case embeddingProviderAIProxy:
		return newAIProxyEmbeddingClient(cfg, logger, vectorSize)
	case embeddingProviderVNP:
		return newVNPEmbeddingClientFromConfig(cfg, logger, vectorSize)
	default:
		return nil, fmt.Errorf("unsupported embedding provider %q", provider)
	}
}

func newOpenAIEmbeddingClient(cfg *conf.Data, logger log.Logger, vectorSize int) (EmbeddingClient, error) {
	var emb *conf.Data_Embedding
	if cfg != nil {
		emb = cfg.GetEmbedding()
	}
	apiKey := ""
	model := defaultOpenAIModel
	baseURL := defaultOpenAIBaseURL
	timeout := defaultEmbeddingTimeout

	if emb != nil {
		apiKey = strings.TrimSpace(emb.GetApiKey())
		if raw := strings.TrimSpace(emb.GetModel()); raw != "" {
			model = raw
		}
		if raw := strings.TrimSpace(emb.GetBaseUrl()); raw != "" {
			baseURL = raw
		}
		if d := emb.GetTimeout(); d != nil && d.AsDuration() > 0 {
			timeout = d.AsDuration()
		}
	}

	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	}
	if apiKey == "" {
		return nil, fmt.Errorf("embedding provider=openai but api key is empty (embedding.api_key/OPENAI_API_KEY)")
	}

	if helper := log.NewHelper(logger); helper != nil {
		helper.Infof("embedding provider configured: %s model=%s", embeddingProviderOpenAI, model)
	}
	return NewOpenAIEmbeddingClient(baseURL, apiKey, model, vectorSize, timeout), nil
}

func newAIProxyEmbeddingClient(cfg *conf.Data, logger log.Logger, vectorSize int) (EmbeddingClient, error) {
	var emb *conf.Data_Embedding
	if cfg != nil {
		emb = cfg.GetEmbedding()
	}
	apiKey := ""
	model := defaultOpenAIModel
	baseURL := defaultAIProxyBaseURL
	path := defaultAIProxyEmbedPath
	timeout := defaultEmbeddingTimeout

	if emb != nil {
		apiKey = strings.TrimSpace(emb.GetApiKey())
		if raw := strings.TrimSpace(emb.GetModel()); raw != "" {
			model = raw
		}
		if raw := strings.TrimSpace(emb.GetBaseUrl()); raw != "" {
			baseURL = raw
		}
		if raw := strings.TrimSpace(emb.GetPath()); raw != "" {
			path = raw
		}
		if d := emb.GetTimeout(); d != nil && d.AsDuration() > 0 {
			timeout = d.AsDuration()
		}
	}

	if helper := log.NewHelper(logger); helper != nil {
		helper.Infof("embedding provider configured: %s model=%s endpoint=%s%s", embeddingProviderAIProxy, model, strings.TrimRight(baseURL, "/"), path)
	}
	return NewAIProxyEmbeddingClient(baseURL, path, apiKey, model, vectorSize, timeout), nil
}

func newVNPEmbeddingClientFromConfig(cfg *conf.Data, logger log.Logger, _ int) (EmbeddingClient, error) {
	var emb *conf.Data_Embedding
	if cfg != nil {
		emb = cfg.GetEmbedding()
	}
	apiKey := ""
	baseURL := defaultVNPEmbedURL
	timeout := defaultEmbeddingTimeout

	if emb != nil {
		apiKey = strings.TrimSpace(emb.GetApiKey())
		if raw := strings.TrimSpace(emb.GetBaseUrl()); raw != "" {
			baseURL = raw
		}
		if d := emb.GetTimeout(); d != nil && d.AsDuration() > 0 {
			timeout = d.AsDuration()
		}
	}

	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("VNP_EMBED_API_KEY"))
	}
	if apiKey == "" {
		return nil, fmt.Errorf("embedding provider=vnp but api key is empty (embedding.api_key/VNP_EMBED_API_KEY)")
	}

	if helper := log.NewHelper(logger); helper != nil {
		helper.Infof("embedding provider configured: %s endpoint=%s vectorSize=%d (fixed)", embeddingProviderVNP, baseURL, defaultVNPEmbedVectorSize)
	}
	return NewVNPEmbeddingClient(baseURL, apiKey, timeout), nil
}

func defaultVectorSize(cfg *conf.Data) int {
	if cfg != nil && cfg.GetQdrant() != nil && cfg.GetQdrant().GetVectorSize() > 0 {
		return int(cfg.GetQdrant().GetVectorSize())
	}
	return defaultEmbeddingVectorSize
}
