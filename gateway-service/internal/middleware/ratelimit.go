package middleware

import (
	"gateway-service/internal/config"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	config *config.RateLimitConfig

	// Global rate limiter
	globalLimiter *rate.Limiter

	// Service-specific rate limiters
	serviceLimiters sync.Map
}

func NewRateLimiter(cfg *config.RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config: cfg,
		globalLimiter: rate.NewLimiter(
			rate.Every(time.Minute/time.Duration(cfg.Global.RequestsPerMinute)),
			cfg.Global.Burst,
		),
	}
	return rl
}

func (rl *RateLimiter) getOrCreateServiceLimiter(serviceName string) *rate.Limiter {
	serviceLimit, exists := rl.config.ServiceLimits[serviceName]
	if !exists {
		return nil
	}

	limiter, _ := rl.serviceLimiters.LoadOrStore(serviceName, rate.NewLimiter(
		rate.Every(time.Minute/time.Duration(serviceLimit.RequestsPerMinute)),
		serviceLimit.Burst,
	))
	return limiter.(*rate.Limiter)
}

func (rl *RateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		serviceName, _ := c.Locals("service_name").(string)

		if !rl.globalLimiter.Allow() {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Global rate limit exceeded",
			})
		}

		if serviceLimiter := rl.getOrCreateServiceLimiter(serviceName); serviceLimiter != nil {
			if !serviceLimiter.Allow() {
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
					"error": "Service rate limit exceeded",
				})
			}
		}

		return c.Next()
	}
}
func ServiceName(name string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("service_name", name)
		return c.Next()
	}
}
