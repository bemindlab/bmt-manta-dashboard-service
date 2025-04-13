package handlers

import (
	"strconv"

	"github.com/bemindtech/bmt-manta-dashboard-service/internal/models"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/services"
	"github.com/gofiber/fiber/v2"
)

// CameraHandler เป็นโครงสร้างสำหรับจัดการ API endpoints เกี่ยวกับกล้อง
type CameraHandler struct {
	CameraService *services.CameraService
}

// NewCameraHandler สร้าง CameraHandler ใหม่
func NewCameraHandler(cameraService *services.CameraService) *CameraHandler {
	return &CameraHandler{
		CameraService: cameraService,
	}
}

// GetCameras ดึงรายการกล้องทั้งหมดขององค์กร
// @Summary Get all cameras
// @Description Retrieve a list of all cameras in the organization with pagination
// @Tags cameras
// @Accept json
// @Produce json
// @Param page query int false "Page number (starting from 1)" default(1)
// @Param page_size query int false "Items per page (max 100)" default(10)
// @Security ApiKeyAuth
// @Success 200 {object} ListCamerasResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 500 {object} ErrorResponse
// @Router /api/cameras [get]
func (h *CameraHandler) GetCameras(c *fiber.Ctx) error {
	// ดึง organization ID จาก context
	organizationID := c.Locals("organization_id").(string)
	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ไม่พบข้อมูลองค์กร",
		})
	}

	// ดึงค่า pagination จาก query parameters
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(c.Query("page_size", "10"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// ดึงรายการกล้อง
	cameras, pagination, err := h.CameraService.ListCameras(c.Context(), organizationID, page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// สร้าง response
	response := ListCamerasResponse{
		Data:       cameras,
		Pagination: pagination,
	}

	return c.JSON(response)
}

// GetCamera ดึงข้อมูลกล้องตาม ID
// @Summary Get camera by ID
// @Description Retrieve a specific camera by its ID
// @Tags cameras
// @Accept json
// @Produce json
// @Param id path string true "Camera ID"
// @Security ApiKeyAuth
// @Success 200 {object} models.Camera
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 404 {object} ErrorResponse "Camera not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/cameras/{id} [get]
func (h *CameraHandler) GetCamera(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ต้องระบุรหัสกล้อง",
		})
	}

	// ดึง organization ID จาก context
	organizationID := c.Locals("organization_id").(string)
	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ไม่พบข้อมูลองค์กร",
		})
	}

	// ดึงข้อมูลกล้อง
	camera, err := h.CameraService.GetCamera(c.Context(), id, organizationID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(camera)
}

// CreateCamera สร้างกล้องใหม่
// @Summary Create a new camera
// @Description Create a new camera in the organization
// @Tags cameras
// @Accept json
// @Produce json
// @Param camera body models.Camera true "Camera details"
// @Security ApiKeyAuth
// @Success 201 {object} models.Camera
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 500 {object} ErrorResponse
// @Router /api/cameras [post]
func (h *CameraHandler) CreateCamera(c *fiber.Ctx) error {
	// แปลงข้อมูลจาก request
	var camera models.Camera
	if err := c.BodyParser(&camera); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "รูปแบบข้อมูลไม่ถูกต้อง",
		})
	}

	// ตรวจสอบข้อมูลจำเป็น
	if camera.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ต้องระบุชื่อกล้อง",
		})
	}

	// ดึง organization ID จาก context
	organizationID := c.Locals("organization_id").(string)
	if organizationID == "" {
		if camera.OrganizationID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "ต้องระบุรหัสองค์กร",
			})
		}
	} else {
		// ถ้ามี organization ID ใน context ให้ใช้ค่านั้น
		camera.OrganizationID = organizationID
	}

	// สร้างกล้องใหม่
	if err := h.CameraService.CreateCamera(c.Context(), &camera); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(camera)
}

// UpdateCamera อัปเดตข้อมูลกล้อง
// @Summary Update a camera
// @Description Update an existing camera's details
// @Tags cameras
// @Accept json
// @Produce json
// @Param id path string true "Camera ID"
// @Param camera body models.Camera true "Updated camera details"
// @Security ApiKeyAuth
// @Success 200 {object} models.Camera
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 404 {object} ErrorResponse "Camera not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/cameras/{id} [put]
func (h *CameraHandler) UpdateCamera(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ต้องระบุรหัสกล้อง",
		})
	}

	// ดึง organization ID จาก context
	organizationID := c.Locals("organization_id").(string)
	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ไม่พบข้อมูลองค์กร",
		})
	}

	// ดึงข้อมูลกล้องปัจจุบัน
	camera, err := h.CameraService.GetCamera(c.Context(), id, organizationID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// แปลงข้อมูลจาก request
	var updatedCamera models.Camera
	if err := c.BodyParser(&updatedCamera); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "รูปแบบข้อมูลไม่ถูกต้อง",
		})
	}

	// ตรวจสอบข้อมูลจำเป็น
	if updatedCamera.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ต้องระบุชื่อกล้อง",
		})
	}

	// อัปเดตข้อมูลที่เปลี่ยนแปลง
	camera.Name = updatedCamera.Name
	camera.Location = updatedCamera.Location
	if updatedCamera.Status != "" {
		camera.Status = updatedCamera.Status
	}

	// บันทึกการเปลี่ยนแปลง
	if err := h.CameraService.UpdateCamera(c.Context(), camera); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(camera)
}

// DeleteCamera ลบกล้อง
// @Summary Delete a camera
// @Description Delete a camera by ID
// @Tags cameras
// @Accept json
// @Produce json
// @Param id path string true "Camera ID"
// @Security ApiKeyAuth
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 404 {object} ErrorResponse "Camera not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/cameras/{id} [delete]
func (h *CameraHandler) DeleteCamera(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ต้องระบุรหัสกล้อง",
		})
	}

	// ดึง organization ID จาก context
	organizationID := c.Locals("organization_id").(string)
	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ไม่พบข้อมูลองค์กร",
		})
	}

	// ลบกล้อง
	err := h.CameraService.DeleteCamera(c.Context(), id, organizationID)
	if err != nil {
		if err.Error() == "ไม่พบกล้องที่ต้องการลบ" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if err.Error() == "ไม่สามารถลบกล้องที่มีข้อมูล logs อยู่ได้" || 
		   err.Error() == "ไม่สามารถลบกล้องที่มีรูปภาพใบหน้าอยู่ได้" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "ลบกล้องสำเร็จ",
	})
}

// ListCamerasResponse โครงสร้างสำหรับส่งข้อมูลรายการกล้องพร้อมข้อมูลการแบ่งหน้า
type ListCamerasResponse struct {
	Data       []models.Camera     `json:"data"`
	Pagination *models.Pagination `json:"pagination"`
}