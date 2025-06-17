package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"log"

	"github.com/LikhithMar14/BidZy/pkg/types"
	"github.com/redis/go-redis/v9"
)

// TokenBucket represents an in-memory token bucket
type TokenBucket struct {
	tokens       float64
	lastRefresh  time.Time
	rate         float64 // tokens per second
	capacity     int     // maximum tokens
	mu           sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(rate float64, capacity int) *TokenBucket {
	return &TokenBucket{
		tokens:      float64(capacity),
		lastRefresh: time.Now(),
		rate:        rate,
		capacity:    capacity,
	}
}

// TryConsume attempts to consume tokens from the bucket
func (tb *TokenBucket) TryConsume(tokens int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefresh).Seconds()
	
	// Add tokens based on elapsed time
	tb.tokens = min(float64(tb.capacity), tb.tokens+elapsed*tb.rate)
	tb.lastRefresh = now

	// Check if we have enough tokens
	if tb.tokens >= float64(tokens) {
		tb.tokens -= float64(tokens)
		return true
	}
	
	return false
}

// GetTokens returns current token count (for debugging)
func (tb *TokenBucket) GetTokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(tb.lastRefresh).Seconds()
	tb.tokens = min(float64(tb.capacity), tb.tokens+elapsed*tb.rate)
	tb.lastRefresh = now
	
	return tb.tokens
}

// HybridRateLimiter combines in-memory buckets with Redis for distributed limiting
type HybridRateLimiter struct {
	// In-memory buckets for high-frequency requests
	buckets    sync.Map
	rate       float64
	capacity   int
	
	// Redis for distributed coordination (optional)
	rdb        *redis.Client
	useRedis   bool
	
	// Configuration
	cleanupInterval time.Duration
	bucketTTL       time.Duration
	debug          bool
	
	// Cleanup goroutine control
	stopCleanup chan struct{}
	cleanupWG   sync.WaitGroup
}

// NewHybridRateLimiter creates a new hybrid rate limiter
func NewHybridRateLimiter(rate float64, capacity int, rdb *redis.Client) *HybridRateLimiter {
	rl := &HybridRateLimiter{
		rate:            rate,
		capacity:        capacity,
		rdb:             rdb,
		useRedis:        rdb != nil,
		cleanupInterval: 1 * time.Minute,
		bucketTTL:       5 * time.Minute,
		debug:           false,
		stopCleanup:     make(chan struct{}),
	}
	
	// Start cleanup goroutine
	rl.startCleanup()
	
	return rl
}

// SetDebug enables/disables debug logging
func (rl *HybridRateLimiter) SetDebug(debug bool) {
	rl.debug = debug
}

// startCleanup starts the background cleanup goroutine
func (rl *HybridRateLimiter) startCleanup() {
	rl.cleanupWG.Add(1)
	go func() {
		defer rl.cleanupWG.Done()
		ticker := time.NewTicker(rl.cleanupInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				rl.cleanupOldBuckets()
			case <-rl.stopCleanup:
				return
			}
		}
	}()
}

// cleanupOldBuckets removes old unused buckets
func (rl *HybridRateLimiter) cleanupOldBuckets() {
	cutoff := time.Now().Add(-rl.bucketTTL)
	
	rl.buckets.Range(func(key, value interface{}) bool {
		if bucket, ok := value.(*TokenBucket); ok {
			bucket.mu.Lock()
			if bucket.lastRefresh.Before(cutoff) {
				rl.buckets.Delete(key)
				if rl.debug {
					log.Printf("RateLimit: Cleaned up old bucket for key: %s", key)
				}
			}
			bucket.mu.Unlock()
		}
		return true
	})
}

// Stop stops the cleanup goroutine
func (rl *HybridRateLimiter) Stop() {
	close(rl.stopCleanup)
	rl.cleanupWG.Wait()
}

// getBucket gets or creates a token bucket for the given key
func (rl *HybridRateLimiter) getBucket(key string) *TokenBucket {
	if bucket, ok := rl.buckets.Load(key); ok {
		return bucket.(*TokenBucket)
	}
	
	newBucket := NewTokenBucket(rl.rate, rl.capacity)
	actual, _ := rl.buckets.LoadOrStore(key, newBucket)
	return actual.(*TokenBucket)
}

// Middleware returns the rate limiting middleware
func (rl *HybridRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip WebSocket upgrade requests
		if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
			next.ServeHTTP(w, r)
			return
		}

		// Get client identifier
		key := rl.getClientKey(r)
		
		// Check rate limit using in-memory bucket
		allowed := rl.checkRateLimit(key, 1)
		
		if !allowed {
			if rl.debug {
				log.Printf("RateLimit: BLOCKED request for key=%s, path=%s", key, r.URL.Path)
			}
			
			bucket := rl.getBucket(key)
			remaining := int(bucket.GetTokens())
			
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.capacity))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		if rl.debug {
			bucket := rl.getBucket(key)
			log.Printf("RateLimit: ALLOWED request for key=%s, tokens=%.2f", key, bucket.GetTokens())
		}

		next.ServeHTTP(w, r)
	})
}

// checkRateLimit checks if request should be allowed
func (rl *HybridRateLimiter) checkRateLimit(key string, tokens int) bool {
	bucket := rl.getBucket(key)
	return bucket.TryConsume(tokens)
}

// getClientKey generates a unique key for rate limiting
func (rl *HybridRateLimiter) getClientKey(r *http.Request) string {
	// If user is authenticated, use user ID as the key
	if ctxUser := r.Context().Value("user"); ctxUser != nil {
		if claims, ok := ctxUser.(*types.UserClaims); ok {
			return "user:" + claims.ID
		}
	}

	// Otherwise use IP address
	ip := extractIP(r)
	return "ip:" + ip
}

// extractIP extracts the real IP address from the request
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header (common with proxies/load balancers)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if ip != "" {
			return ip
		}
	}

	// Check X-Real-IP header (common with nginx)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	// Fallback to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Reset resets the rate limit for a specific key
func (rl *HybridRateLimiter) Reset(key string) {
	rl.buckets.Delete(key)
	if rl.debug {
		log.Printf("RateLimit: Reset bucket for key: %s", key)
	}
}

// GetStats returns current statistics for a key
func (rl *HybridRateLimiter) GetStats(key string) map[string]interface{} {
	bucket := rl.getBucket(key)
	tokens := bucket.GetTokens()
	
	return map[string]interface{}{
		"key":       key,
		"tokens":    tokens,
		"rate":      rl.rate,
		"capacity":  rl.capacity,
		"remaining": int(tokens),
	}
}

// DebugHandler returns a handler for debugging rate limiter state
func (rl *HybridRateLimiter) DebugHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := rl.getClientKey(r)
		stats := rl.GetStats(key)
		
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
	"key": "%s",
	"tokens": %.2f,
	"rate": %.2f,
	"capacity": %d,
	"remaining": %d
}`, 
			stats["key"], 
			stats["tokens"], 
			stats["rate"], 
			stats["capacity"], 
			stats["remaining"])
	}
}

// StatsHandler returns overall statistics
func (rl *HybridRateLimiter) StatsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var bucketCount int
		rl.buckets.Range(func(key, value interface{}) bool {
			bucketCount++
			return true
		})
		
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
	"active_buckets": %d,
	"rate_per_second": %.2f,
	"capacity": %d,
	"cleanup_interval": "%s",
	"bucket_ttl": "%s"
}`, bucketCount, rl.rate, rl.capacity, rl.cleanupInterval, rl.bucketTTL)
	}
}

// Helper function for min
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Optional: Distributed rate limiter using Redis (for multi-instance deployments)
// This can be used alongside the in-memory limiter for global limits

const redisRateLimitScript = `
local key = KEYS[1]
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local current_time = tonumber(ARGV[3])

local current = redis.call('GET', key)
if current == false then
    redis.call('SET', key, 1)
    redis.call('EXPIRE', key, window)
    return {1, limit - 1}
else
    local count = tonumber(current)
    if count < limit then
        local new_count = redis.call('INCR', key)
        local ttl = redis.call('TTL', key)
        return {new_count, limit - new_count}
    else
        local ttl = redis.call('TTL', key)
        return {count, 0, ttl}
    end
end
`

// CheckGlobalLimit checks against a global Redis-based limit (optional)
func (rl *HybridRateLimiter) CheckGlobalLimit(key string, window int, limit int) (bool, error) {
	if !rl.useRedis {
		return true, nil
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	result, err := rl.rdb.Eval(ctx, redisRateLimitScript, 
		[]string{"global:" + key}, window, limit, time.Now().Unix()).Result()
	
	if err != nil {
		// Fail open on Redis errors
		return true, err
	}
	
	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) < 2 {
		return true, fmt.Errorf("unexpected result format")
	}
	
	remaining, ok := resultSlice[1].(int64)
	if !ok {
		return true, fmt.Errorf("invalid remaining value")
	}
	
	return remaining > 0, nil
}