package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	visitors = make(map[string]*visitor)
	mu       sync.Mutex
	once     sync.Once
)

func startCleanup() {
	once.Do(func() {
		go func() {
			for {
				time.Sleep(5 * time.Minute)
				mu.Lock()
				for key, v := range visitors {
					if time.Since(v.lastSeen) > 10*time.Minute {
						delete(visitors, key)
					}
				}
				mu.Unlock()
			}
		}()
	})
}

func getVisitor(key string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[key]
	if !exists {
		// 10 requests per minute, burst of 5
		limiter := rate.NewLimiter(rate.Limit(10.0/60.0), 5)
		visitors[key] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func RateLimit() gin.HandlerFunc {
	startCleanup()

	return func(c *gin.Context) {
		key := c.GetHeader("X-Device-Hash")
		if key == "" {
			key = c.ClientIP()
		}

		limiter := getVisitor(key)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		c.Next()
	}
}
