package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/LikhithMar14/BidZy/pkg/types"
	"github.com/redis/go-redis/v9"
)

const luaTokenBucket = `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local last_tokens = tonumber(redis.call("HGET", key, "tokens") or capacity)
local last_refreshed = tonumber(redis.call("HGET", key, "timestamp") or 0)

local delta = math.max(0, now - last_refreshed)
local new_tokens = math.min(capacity, last_tokens + delta * rate)
local allowed = new_tokens >= requested
local new_tokens_after = new_tokens

if allowed then
	new_tokens_after = new_tokens - requested
end

redis.call("HSET", key, "tokens", new_tokens_after)
redis.call("HSET", key, "timestamp", now)
redis.call("EXPIRE", key, 10)

return allowed
`

type RedisRateLimiter struct {
	RDB       *redis.Client
	Rate      float64 // tokens/sec
	Capacity  int     // burst size
}

func NewRedisRateLimiter(rdb *redis.Client, rate float64, capacity int) *RedisRateLimiter {
	return &RedisRateLimiter{
		RDB:      rdb,
		Rate:     rate,
		Capacity: capacity,
	}
}
func (rl *RedisRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip WebSocket upgrade requests
		if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
			next.ServeHTTP(w, r)
			return
		}

		// Identify user by IP or Authenticated ID
		ip := strings.Split(r.RemoteAddr, ":")[0]
		key := ip
		if ctxUser := r.Context().Value("user"); ctxUser != nil {
			if claims, ok := ctxUser.(*types.UserClaims); ok {
				key = claims.ID
			}
		}
		now := time.Now().Unix()
		allowed, err := rl.RDB.Eval(context.Background(), luaTokenBucket, []string{"rate_limit:" + key},
			rl.Rate, rl.Capacity, now, 1,
		).Bool()

		if err != nil {
			http.Error(w, "Internal RateLimiter error", http.StatusInternalServerError)
			return
		}

		if !allowed {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
