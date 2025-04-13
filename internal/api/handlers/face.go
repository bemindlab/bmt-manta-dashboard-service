package handlers

import (
	"strconv"

	"github.com/bemindtech/bmt-manta-dashboard-service/internal/models"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/services"
	"github.com/gofiber/fiber/v2"
)

// FaceHandler เป็นโครงสร้างสำหรับจัดการ API endpoints เกี่ยวกับรูปภาพใบหน้า
type FaceHandler struct {
	FaceService *services.FaceService
}

// NewFaceHandler สร้าง FaceHandler ใหม่
func NewFaceHandler(faceService *services.FaceService) *FaceHandler {
	return &FaceHandler{
		FaceService: faceService,
	}
}

// UploadFaceImage เป็น handler สำหรับอัปโหลดรูปภาพใบหน้า
// @Summary Upload a face image
// @Description Upload an image of a person's face for training AI models
// @Tags faces
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Face image file"
// @Param person_hash formData string true "Person's unique hash"
// @Param camera_id formData string true "ID of the camera that captured the image"
// @Param organization_id formData string true "Organization ID"
// @Security ApiKeyAuth
// @Success 201 {object} models.FaceImage
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 413 {object} ErrorResponse "File too large"
// @Failure 500 {object} ErrorResponse
// @Router /api/faces [post]
func (h *FaceHandler) UploadFaceImage(c *fiber.Ctx) error {
	// ตรวจสอบขนาดไฟล์และประเภทไฟล์
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ไม่พบไฟล์ในการอัปโหลด",
		})
	}

	// ขนาดไฟล์ไม่เกิน 5MB
	if file.Size > 5*1024*1024 {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"error": "ขนาดไฟล์ต้องไม่เกิน 5MB",
		})
	}

	// ดึงข้อมูลจาก form
	personHash := c.FormValue("person_hash")
	if personHash == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ต้องระบุรหัสบุคคล (person_hash)",
		})
	}

	cameraID := c.FormValue("camera_id")
	if cameraID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ต้องระบุรหัสกล้อง (camera_id)",
		})
	}

	// ดึง organization ID จาก context หรือ form
	organizationID := c.Locals("organization_id").(string)
	if organizationID == "" {
		organizationID = c.FormValue("organization_id")
		if organizationID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "ต้องระบุรหัสองค์กร (organization_id)",
			})
		}
	}

	// อัปโหลดรูปภาพ
	faceImage, err := h.FaceService.UploadFaceImage(c.Context(), file, personHash, cameraID, organizationID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(faceImage)
}

// GetFaceImages เป็น handler สำหรับดึงรูปภาพใบหน้าตามรหัสบุคคล
// @Summary Get face images by person hash
// @Description Retrieve face images for a specific person
// @Tags faces
// @Accept json
// @Produce json
// @Param person_hash path string true "Person's unique hash"
// @Param page query int false "Page number (starting from 1)" default(1)
// @Param page_size query int false "Items per page (max 100)" default(10)
// @Security ApiKeyAuth
// @Success 200 {object} FaceImagesResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 500 {object} ErrorResponse
// @Router /api/faces/{person_hash} [get]
func (h *FaceHandler) GetFaceImages(c *fiber.Ctx) error {
	personHash := c.Params("person_hash")
	if personHash == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ต้องระบุรหัสบุคคล",
		})
	}

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

	// ดึงรูปภาพ
	images, pagination, err := h.FaceService.GetFaceImages(c.Context(), personHash, organizationID, page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// สร้าง response
	response := FaceImagesResponse{
		Data:       images,
		Pagination: pagination,
	}

	return c.JSON(response)
}

// DeleteFaceImage เป็น handler สำหรับลบรูปภาพใบหน้า
// @Summary Delete a face image
// @Description Delete a specific face image by ID
// @Tags faces
// @Accept json
// @Produce json
// @Param id path string true "Face image ID"
// @Security ApiKeyAuth
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 404 {object} ErrorResponse "Face image not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/faces/image/{id} [delete]
func (h *FaceHandler) DeleteFaceImage(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ต้องระบุรหัสรูปภาพ",
		})
	}

	// ดึง organization ID จาก context
	organizationID := c.Locals("organization_id").(string)
	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ไม่พบข้อมูลองค์กร",
		})
	}

	// ลบรูปภาพ
	err := h.FaceService.DeleteFaceImage(c.Context(), id, organizationID)
	if err != nil {
		if err.Error() == "ไม่พบรูปภาพที่ต้องการลบ" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "ลบรูปภาพสำเร็จ",
	})
}

// FaceImagesResponse โครงสร้างสำหรับส่งข้อมูลรายการรูปภาพใบหน้าพร้อมข้อมูลการแบ่งหน้า
type FaceImagesResponse struct {
	Data       []models.FaceImage  `json:"data"`
	Pagination *models.Pagination `json:"pagination"`
}