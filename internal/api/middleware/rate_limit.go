package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/bemindtech/bmt-manta-dashboard-service/config"
	"github.com/gofiber/fiber/v2"
)

// RateLimiter เป็นโครงสร้างสำหรับการจำกัดอัตราการเรียกใช้
type RateLimiter struct {
	max      int           // จำนวนคำขอสูงสุดต่อช่วงเวลา
	duration time.Duration // ช่วงเวลาจำกัด
	ips      map[string]*IPRateLimit
	mu       sync.RWMutex
}

// IPRateLimit เก็บข้อมูลการจำกัดอัตราสำหรับแต่ละ IP
type IPRateLimit struct {
	count    int       // จำนวนคำขอปัจจุบัน
	resetAt  time.Time // เวลาที่จะรีเซ็ตการนับ
}

// NewRateLimiter สร้าง RateLimiter ใหม่
func NewRateLimiter(max int, duration time.Duration) *RateLimiter {
	return &RateLimiter{
		max:      max,
		duration: duration,
		ips:      make(map[string]*IPRateLimit),
	}
}

// NewRateLimitMiddleware สร้าง middleware สำหรับการจำกัดอัตราการเรียกใช้
func NewRateLimitMiddleware(cfg *config.Config) fiber.Handler {
	limiter := NewRateLimiter(cfg.RateLimitMax, cfg.RateLimitDuration)

	// เริ่มการทำงานเพื่อล้างข้อมูลเก่า
	go limiter.cleanupTask()

	return func(c *fiber.Ctx) error {
		// ดึง IP address ของผู้ใช้
		ip := c.IP()
		if ip == "" {
			ip = "unknown"
		}

		// ตรวจสอบและบันทึกการเรียกใช้
		remaining, resetAt, allowed := limiter.allow(ip)

		// เพิ่ม header สำหรับข้อมูล rate limit
		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.RateLimitMax))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))

		// ถ้าเกินจำนวนที่กำหนด ให้ส่งข้อผิดพลาด
		if !allowed {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "เกินจำนวนคำขอที่กำหนด โปรดลองใหม่ภายหลัง",
			})
		}

		return c.Next()
	}
}

// allow ตรวจสอบและบันทึกการเรียกใช้จาก IP
func (rl *RateLimiter) allow(ip string) (int, time.Time, bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// ถ้าเป็น IP ใหม่หรือหมดเวลาแล้ว ให้สร้างใหม่
	limit, exists := rl.ips[ip]
	if !exists || now.After(limit.resetAt) {
		rl.ips[ip] = &IPRateLimit{
			count:    1,
			resetAt:  now.Add(rl.duration),
		}
		return rl.max - 1, now.Add(rl.duration), true
	}

	// ตรวจสอบว่าเกินจำนวนที่กำหนดหรือไม่
	if limit.count >= rl.max {
		return 0, limit.resetAt, false
	}

	// เพิ่มจำนวนคำขอ
	limit.count++
	return rl.max - limit.count, limit.resetAt, true
}

// cleanupTask ทำความสะอาดข้อมูลเก่าทุกๆ 5 นาที
func (rl *RateLimiter) cleanupTask() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup ลบข้อมูลที่หมดอายุแล้ว
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, limit := range rl.ips {
		if now.After(limit.resetAt) {
			delete(rl.ips, ip)
		}
	}
} 