package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/bemindtech/bmt-manta-dashboard-service/config"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/db"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// JWTClaims เป็นโครงสร้างสำหรับข้อมูลใน JWT token
type JWTClaims struct {
	UserID         string `json:"user_id"`
	Role           string `json:"role"`
	OrganizationID string `json:"organization_id"`
	jwt.RegisteredClaims
}

// NewAuthMiddleware สร้าง middleware สำหรับการตรวจสอบ JWT token
func NewAuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// ดึง Authorization header
		authHeader := c.Get("Authorization")
		
		// ตรวจสอบรูปแบบ header
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "ไม่มี Authorization header",
			})
		}

		// ตรวจสอบว่าเป็น Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "รูปแบบ Authorization header ไม่ถูกต้อง",
			})
		}

		// ดึง token
		tokenString := parts[1]

		// ตรวจสอบ token
		token, err := jwt.ParseWithClaims(
			tokenString,
			&JWTClaims{},
			func(token *jwt.Token) (interface{}, error) {
				// ตรวจสอบว่าใช้ algorithm HS256
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("signing method ไม่ถูกต้อง: %v", token.Header["alg"])
				}
				return []byte(cfg.JWTSecret), nil
			},
		)

		// ตรวจสอบว่ามีข้อผิดพลาดในการตรวจสอบ token
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fmt.Sprintf("token ไม่ถูกต้อง: %v", err),
			})
		}

		// ตรวจสอบว่า token ยังไม่หมดอายุ
		claims, ok := token.Claims.(*JWTClaims)
		if !ok || !token.Valid || claims.ExpiresAt.Before(time.Now()) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "token ไม่ถูกต้องหรือหมดอายุ",
			})
		}

		// เก็บข้อมูลผู้ใช้ไว้ใน context
		c.Locals("user_id", claims.UserID)
		c.Locals("role", claims.Role)
		c.Locals("organization_id", claims.OrganizationID)

		return c.Next()
	}
}

// NewOrganizationAuthMiddleware สร้าง middleware สำหรับตรวจสอบการเข้าถึงข้อมูลขององค์กร
func NewOrganizationAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// ดึง organization ID จาก request
		requestOrgID := c.Params("organization_id")
		if requestOrgID == "" {
			requestOrgID = c.Query("organization_id")
		}
		
		// ดึงข้อมูลผู้ใช้จาก context
		userOrgID, ok := c.Locals("organization_id").(string)
		if !ok || userOrgID == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "ไม่มีข้อมูลองค์กรของผู้ใช้",
			})
		}
		
		// ดึง role ของผู้ใช้
		userRole, _ := c.Locals("role").(string)
		
		// ถ้าเป็น admin ให้เข้าถึงข้อมูลองค์กรไหนก็ได้
		if userRole == "admin" {
			return c.Next()
		}
		
		// ตรวจสอบว่าผู้ใช้เป็นสมาชิกองค์กรที่ต้องการเข้าถึงข้อมูล
		if requestOrgID != userOrgID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "ไม่มีสิทธิ์เข้าถึงข้อมูลขององค์กรนี้",
			})
		}
		
		return c.Next()
	}
}

// NewAPIKeyMiddleware สร้าง middleware สำหรับการตรวจสอบ API key
func NewAPIKeyMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// ดึง API key จาก header
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			// ถ้าไม่มีใน header ให้ดูใน query parameter
			apiKey = c.Query("api_key")
		}

		// ตรวจสอบว่ามี API key
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "ไม่มี API key",
			})
		}

		// ตรวจสอบว่า API key ถูกต้อง
		if apiKey != cfg.APIKey {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "API key ไม่ถูกต้อง",
			})
		}

		return c.Next()
	}
}

// NewAPIKeyWithOrgMiddleware สร้าง middleware สำหรับการตรวจสอบ API key และองค์กร
func NewAPIKeyWithOrgMiddleware(postgres *db.PostgresDB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// ดึง API key จาก header
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			// ถ้าไม่มีใน header ให้ดูใน query parameter
			apiKey = c.Query("api_key")
		}

		// ตรวจสอบว่ามี API key
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "ไม่มี API key",
			})
		}

		// ตรวจสอบ API key และดึงข้อมูลองค์กร
		var apiKeyRecord models.APIKey
		result := postgres.DB.Where("key_value = ?", apiKey).
			Where("expires_at IS NULL OR expires_at > NOW()").
			First(&apiKeyRecord)
		
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "API key ไม่ถูกต้องหรือหมดอายุ",
				})
			}
			// เกิดข้อผิดพลาดในการค้นหา API key
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "เกิดข้อผิดพลาดในการตรวจสอบ API key",
			})
		}

		// เก็บข้อมูลองค์กรไว้ใน context
		c.Locals("organization_id", apiKeyRecord.OrganizationID)

		return c.Next()
	}
}