package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bemindtech/bmt-manta-dashboard-service/internal/models"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/services"
	"github.com/gofiber/fiber/v2"
)

// ErrorResponse เป็นโครงสร้างสำหรับส่งข้อผิดพลาด
type ErrorResponse struct {
	Error string `json:"error"`
}

// LogsHandler เป็นโครงสร้างสำหรับจัดการ API endpoints เกี่ยวกับข้อมูล logs
type LogsHandler struct {
	StatsService *services.StatsService
}

// NewLogsHandler สร้าง LogsHandler ใหม่
func NewLogsHandler(statsService *services.StatsService) *LogsHandler {
	return &LogsHandler{
		StatsService: statsService,
	}
}

// GetLogs เป็น handler สำหรับดึงข้อมูล logs
// @Summary Retrieve logs data with filtering
// @Description Retrieve person detection logs based on specified filters
// @Tags logs
// @Accept json
// @Produce json
// @Param from query string false "Start time for log retrieval (format YYYY-MM-DDTHH:MM:SS)"
// @Param to query string false "End time for log retrieval (format YYYY-MM-DDTHH:MM:SS)"
// @Param camera_id query string false "Camera ID to filter logs by"
// @Param person_id query string false "Person ID to filter logs by"
// @Param page query int false "Page number to retrieve (starting from 1)" default(1)
// @Param page_size query int false "Number of items per page (max 100)" default(10)
// @Security ApiKeyAuth
// @Success 200 {object} LogsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 500 {object} ErrorResponse
// @Router /api/logs [get]
func (h *LogsHandler) GetLogs(c *fiber.Ctx) error {
	// สร้าง filter จาก query parameters
	filter, err := parseLogsFilter(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// ดึง organization ID จาก context
	organizationID := c.Locals("organization_id").(string)
	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ไม่พบข้อมูลองค์กร",
		})
	}

	// กำหนดองค์กรให้กับ filter
	filter.OrganizationID = organizationID

	// ดึงข้อมูล logs
	logs, pagination, err := h.StatsService.GetLogs(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// สร้าง response
	response := LogsResponse{
		Data:       logs,
		Pagination: pagination,
	}

	return c.JSON(response)
}

// LogsResponse เป็นโครงสร้างสำหรับส่งข้อมูล logs พร้อมกับข้อมูล pagination
type LogsResponse struct {
	Data       []models.PersonLog   `json:"data"`
	Pagination *models.Pagination `json:"pagination"`
}

// parseLogsFilter แปลง query parameters เป็น LogFilter
func parseLogsFilter(c *fiber.Ctx) (models.LogFilter, error) {
	filter := models.LogFilter{}

	// แปลงค่า from ถ้ามี
	fromStr := c.Query("from")
	if fromStr != "" {
		from, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return filter, fmt.Errorf("รูปแบบของพารามิเตอร์ from ไม่ถูกต้อง โปรดใช้รูปแบบ YYYY-MM-DDTHH:MM:SS")
		}
		filter.From = from
	}

	// แปลงค่า to ถ้ามี
	toStr := c.Query("to")
	if toStr != "" {
		to, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return filter, fmt.Errorf("รูปแบบของพารามิเตอร์ to ไม่ถูกต้อง โปรดใช้รูปแบบ YYYY-MM-DDTHH:MM:SS")
		}
		filter.To = to
	}

	// ดึงค่า camera_id
	filter.CameraID = c.Query("camera_id")

	// ดึงค่า person_id
	filter.PersonID = c.Query("person_id")

	// แปลงค่า page ถ้ามี
	pageStr := c.Query("page", "1")
	if pageStr != "" {
		var err error
		filter.Page, err = strconv.Atoi(pageStr)
		if err != nil {
			return filter, fmt.Errorf("รูปแบบของพารามิเตอร์ page ไม่ถูกต้อง")
		}
		if filter.Page <= 0 {
			filter.Page = 1
		}
	}

	// แปลงค่า page_size ถ้ามี
	pageSizeStr := c.Query("page_size", "10")
	if pageSizeStr != "" {
		var err error
		filter.PageSize, err = strconv.Atoi(pageSizeStr)
		if err != nil {
			return filter, fmt.Errorf("รูปแบบของพารามิเตอร์ page_size ไม่ถูกต้อง")
		}
		if filter.PageSize <= 0 {
			filter.PageSize = 10
		}
		if filter.PageSize > 100 {
			filter.PageSize = 100
		}
	}

	return filter, nil
} 