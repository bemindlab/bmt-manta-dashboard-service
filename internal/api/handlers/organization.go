package handlers

import (
	"strconv"

	"github.com/bemindtech/bmt-manta-dashboard-service/internal/models"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/services"
	"github.com/gofiber/fiber/v2"
)

// OrganizationHandler เป็นโครงสร้างสำหรับจัดการ API endpoints เกี่ยวกับองค์กร
type OrganizationHandler struct {
	OrganizationService *services.OrganizationService
}

// NewOrganizationHandler สร้าง OrganizationHandler ใหม่
func NewOrganizationHandler(organizationService *services.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{
		OrganizationService: organizationService,
	}
}

// GetOrganizations ดึงรายการองค์กรทั้งหมด
// @Summary Get all organizations
// @Description Retrieve a list of all organizations with pagination
// @Tags organizations
// @Accept json
// @Produce json
// @Param page query int false "Page number (starting from 1)" default(1)
// @Param page_size query int false "Items per page (max 100)" default(10)
// @Security ApiKeyAuth
// @Success 200 {object} ListOrganizationsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 500 {object} ErrorResponse
// @Router /api/organizations [get]
func (h *OrganizationHandler) GetOrganizations(c *fiber.Ctx) error {
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

	// ดึงรายการองค์กร
	organizations, pagination, err := h.OrganizationService.ListOrganizations(c.Context(), page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// สร้าง response
	response := ListOrganizationsResponse{
		Data:       organizations,
		Pagination: pagination,
	}

	return c.JSON(response)
}

// GetOrganization ดึงข้อมูลองค์กรตาม ID
// @Summary Get organization by ID
// @Description Retrieve a specific organization by its ID
// @Tags organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Security ApiKeyAuth
// @Success 200 {object} models.Organization
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 404 {object} ErrorResponse "Organization not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/organizations/{id} [get]
func (h *OrganizationHandler) GetOrganization(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ต้องระบุรหัสองค์กร",
		})
	}

	// ดึงข้อมูลองค์กร
	organization, err := h.OrganizationService.GetOrganization(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(organization)
}

// CreateOrganization สร้างองค์กรใหม่
// @Summary Create a new organization
// @Description Create a new organization with the provided details
// @Tags organizations
// @Accept json
// @Produce json
// @Param organization body models.Organization true "Organization details"
// @Security ApiKeyAuth
// @Success 201 {object} models.Organization
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 500 {object} ErrorResponse
// @Router /api/organizations [post]
func (h *OrganizationHandler) CreateOrganization(c *fiber.Ctx) error {
	// แปลงข้อมูลจาก request
	var organization models.Organization
	if err := c.BodyParser(&organization); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "รูปแบบข้อมูลไม่ถูกต้อง",
		})
	}

	// ตรวจสอบข้อมูลจำเป็น
	if organization.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ต้องระบุชื่อองค์กร",
		})
	}

	// สร้างองค์กรใหม่
	if err := h.OrganizationService.CreateOrganization(c.Context(), &organization); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(organization)
}

// UpdateOrganization อัปเดตข้อมูลองค์กร
// @Summary Update an organization
// @Description Update an existing organization with the provided details
// @Tags organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Param organization body models.Organization true "Updated organization details"
// @Security ApiKeyAuth
// @Success 200 {object} models.Organization
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 404 {object} ErrorResponse "Organization not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/organizations/{id} [put]
func (h *OrganizationHandler) UpdateOrganization(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ต้องระบุรหัสองค์กร",
		})
	}

	// ดึงข้อมูลองค์กรปัจจุบัน
	organization, err := h.OrganizationService.GetOrganization(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// แปลงข้อมูลจาก request
	var updatedOrg models.Organization
	if err := c.BodyParser(&updatedOrg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "รูปแบบข้อมูลไม่ถูกต้อง",
		})
	}

	// ตรวจสอบข้อมูลจำเป็น
	if updatedOrg.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ต้องระบุชื่อองค์กร",
		})
	}

	// อัปเดตข้อมูลที่เปลี่ยนแปลง
	organization.Name = updatedOrg.Name
	organization.Description = updatedOrg.Description

	// บันทึกการเปลี่ยนแปลง
	if err := h.OrganizationService.UpdateOrganization(c.Context(), organization); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(organization)
}

// DeleteOrganization ลบองค์กร
// @Summary Delete an organization
// @Description Delete an organization by ID
// @Tags organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Security ApiKeyAuth
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 404 {object} ErrorResponse "Organization not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/organizations/{id} [delete]
func (h *OrganizationHandler) DeleteOrganization(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ต้องระบุรหัสองค์กร",
		})
	}

	// ลบองค์กร
	err := h.OrganizationService.DeleteOrganization(c.Context(), id)
	if err != nil {
		if err.Error() == "ไม่พบองค์กรที่ต้องการลบ" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if err.Error() == "ไม่สามารถลบองค์กรที่มีข้อมูลอยู่ได้" {
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
		"message": "ลบองค์กรสำเร็จ",
	})
}

// ListOrganizationsResponse โครงสร้างสำหรับส่งข้อมูลรายการองค์กรพร้อมข้อมูลการแบ่งหน้า
type ListOrganizationsResponse struct {
	Data       []models.Organization `json:"data"`
	Pagination *models.Pagination    `json:"pagination"`
}

// SuccessResponse โครงสร้างสำหรับส่งข้อมูลว่าทำงานสำเร็จ
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}