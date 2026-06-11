package middleware

import (
	"context"
	"strconv"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/redis/go-redis/v9"
)

// RateLimiter uses Redis sliding window to rate limit requests per AppID
func RateLimiter(quotaProvider QuotaProvider, store RateLimitStore) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			appCtx, ok := AppContextFromContext(ctx)
			if !ok || appCtx.AppID == "" || store == nil {
				return handler(ctx, req)
			}

			limit, err := quotaProvider.GetQuotaLimit(ctx, appCtx.AppID, 1000)
			if err != nil {
				limit = 1000
			}

			minute := time.Now().UTC().Format("200601021504")
			key := "kgs:ratelimit:" + appCtx.AppID + ":" + minute
			count, err := store.Increment(ctx, key, time.Minute+5*time.Second)
			if err != nil {
				return nil, kerrors.InternalServer("ERR_RATE_LIMIT", "failed to evaluate rate limit")
			}
			if count > limit {
				if tr, ok := transport.FromServerContext(ctx); ok {
					tr.ReplyHeader().Set("Retry-After", "60")
				}
				return nil, kerrors.New(429, "ERR_RATE_LIMIT", "request rate limit exceeded")
			}
			return handler(ctx, req)
		}
	}
}

type QuotaProvider interface {
	GetQuotaLimit(ctx context.Context, appID string, fallback int64) (int64, error)
}

type RateLimitStore interface {
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

type redisRateLimitStore struct {
	redisCli *redis.Client
}

func NewRedisRateLimitStore(redisCli *redis.Client) RateLimitStore {
	if redisCli == nil {
		return nil
	}
	return &redisRateLimitStore{redisCli: redisCli}
}

func (s *redisRateLimitStore) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	res, err := rateLimitLua.Run(ctx, s.redisCli, []string{key}, int(ttl.Seconds())).Result()
	if err != nil {
		return 0, err
	}
	switch v := res.(type) {
	case int64:
		return v, nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, kerrors.InternalServer("ERR_RATE_LIMIT", "invalid rate limit counter response")
	}
}

var rateLimitLua = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return current
`)
