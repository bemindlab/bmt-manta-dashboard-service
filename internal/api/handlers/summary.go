package handlers

import (
	"time"

	"github.com/bemindtech/bmt-manta-dashboard-service/internal/services"
	"github.com/gofiber/fiber/v2"
)

// SummaryHandler เป็นโครงสร้างสำหรับจัดการ API endpoints เกี่ยวกับการสรุปข้อมูล
type SummaryHandler struct {
	StatsService *services.StatsService
}

// NewSummaryHandler สร้าง SummaryHandler ใหม่
func NewSummaryHandler(statsService *services.StatsService) *SummaryHandler {
	return &SummaryHandler{
		StatsService: statsService,
	}
}

// GetDailySummary เป็น handler สำหรับดึงข้อมูลสรุปรายวัน
// @Summary Get daily summary statistics
// @Description Retrieve total, new, and returning people counts for the specified date
// @Tags summary
// @Accept json
// @Produce json
// @Param date query string false "Date to retrieve data for (format YYYY-MM-DD). If not specified, current date will be used."
// @Security ApiKeyAuth
// @Success 200 {object} models.DailySummary
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 500 {object} ErrorResponse
// @Router /api/summary [get]
func (h *SummaryHandler) GetDailySummary(c *fiber.Ctx) error {
	// ดึงพารามิเตอร์ date จาก query string
	date := c.Query("date")
	if date == "" {
		// ถ้าไม่ระบุวันที่ ใช้วันปัจจุบัน
		date = time.Now().Format("2006-01-02")
	}

	// ตรวจสอบรูปแบบวันที่
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "รูปแบบวันที่ไม่ถูกต้อง โปรดใช้รูปแบบ YYYY-MM-DD",
		})
	}

	// ดึง organization ID จาก context
	organizationID := c.Locals("organization_id").(string)
	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ไม่พบข้อมูลองค์กร",
		})
	}

	// ดึงข้อมูลสรุปรายวัน
	summary, err := h.StatsService.GetDailySummary(c.Context(), date, organizationID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(summary)
}

// GetHeatmap เป็น handler สำหรับดึงข้อมูลความหนาแน่นตามช่วงเวลา
// @Summary Get heatmap data by time period
// @Description Retrieve people count data by hour for the specified date
// @Tags summary
// @Accept json
// @Produce json
// @Param date query string false "Date to retrieve data for (format YYYY-MM-DD). If not specified, current date will be used."
// @Security ApiKeyAuth
// @Success 200 {array} models.HeatmapData
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 500 {object} ErrorResponse
// @Router /api/heatmap [get]
func (h *SummaryHandler) GetHeatmap(c *fiber.Ctx) error {
	// ดึงพารามิเตอร์ date จาก query string
	date := c.Query("date")
	if date == "" {
		// ถ้าไม่ระบุวันที่ ใช้วันปัจจุบัน
		date = time.Now().Format("2006-01-02")
	}

	// ตรวจสอบรูปแบบวันที่
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "รูปแบบวันที่ไม่ถูกต้อง โปรดใช้รูปแบบ YYYY-MM-DD",
		})
	}

	// ดึง organization ID จาก context
	organizationID := c.Locals("organization_id").(string)
	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ไม่พบข้อมูลองค์กร",
		})
	}

	// ดึงข้อมูลความหนาแน่น
	heatmap, err := h.StatsService.GetHeatmapData(c.Context(), date, organizationID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(heatmap)
}

// GetPersonStats เป็น handler สำหรับดึงข้อมูลสถิติคนใหม่และคนซ้ำ
// @Summary Get new vs. returning person statistics
// @Description Retrieve statistics about new vs. returning people for the specified date
// @Tags summary
// @Accept json
// @Produce json
// @Param date query string false "Date to retrieve data for (format YYYY-MM-DD). If not specified, current date will be used."
// @Security ApiKeyAuth
// @Success 200 {object} models.PersonStats
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 500 {object} ErrorResponse
// @Router /api/person-stats [get]
func (h *SummaryHandler) GetPersonStats(c *fiber.Ctx) error {
	// ดึงพารามิเตอร์ date จาก query string
	date := c.Query("date")
	if date == "" {
		// ถ้าไม่ระบุวันที่ ใช้วันปัจจุบัน
		date = time.Now().Format("2006-01-02")
	}

	// ตรวจสอบรูปแบบวันที่
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "รูปแบบวันที่ไม่ถูกต้อง โปรดใช้รูปแบบ YYYY-MM-DD",
		})
	}

	// ดึง organization ID จาก context
	organizationID := c.Locals("organization_id").(string)
	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ไม่พบข้อมูลองค์กร",
		})
	}

	// ดึงข้อมูลสถิติ
	stats, err := h.StatsService.GetPersonStats(c.Context(), date, organizationID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
} 