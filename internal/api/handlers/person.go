package handlers

import (
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/models"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/services"
	"github.com/gofiber/fiber/v2"
	"strconv"
)

// PersonHandler เป็นโครงสร้างสำหรับจัดการ API endpoints เกี่ยวกับข้อมูลบุคคล
type PersonHandler struct {
	PersonService *services.PersonService
}

// NewPersonHandler สร้าง PersonHandler ใหม่
func NewPersonHandler(personService *services.PersonService) *PersonHandler {
	return &PersonHandler{
		PersonService: personService,
	}
}

// GetPerson เป็น handler สำหรับดึงข้อมูลบุคคลตาม PersonHash
// @Summary Get person by person hash
// @Description Retrieve a person by their person_hash
// @Tags persons
// @Accept json
// @Produce json
// @Param person_hash path string true "Person Hash to retrieve"
// @Security ApiKeyAuth
// @Success 200 {object} models.Person
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 404 {object} ErrorResponse "Person not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/persons/{person_hash} [get]
func (h *PersonHandler) GetPerson(c *fiber.Ctx) error {
	// ดึง person_hash จาก path parameters
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

	// ดึงข้อมูลบุคคล
	person, err := h.PersonService.GetPerson(c.Context(), personHash, organizationID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(person)
}

// ListPersons เป็น handler สำหรับดึงรายการบุคคลทั้งหมด
// @Summary List all persons
// @Description Retrieve a list of all persons with pagination
// @Tags persons
// @Accept json
// @Produce json
// @Param page query int false "Page number to retrieve (starting from 1)" default(1)
// @Param page_size query int false "Number of items per page (max 100)" default(10)
// @Security ApiKeyAuth
// @Success 200 {object} PersonsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 500 {object} ErrorResponse
// @Router /api/persons [get]
func (h *PersonHandler) ListPersons(c *fiber.Ctx) error {
	// ดึง organization ID จาก context
	organizationID := c.Locals("organization_id").(string)
	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ไม่พบข้อมูลองค์กร",
		})
	}

	// แปลงค่า page ถ้ามี
	page := 1
	pageStr := c.Query("page", "1")
	if pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "รูปแบบของพารามิเตอร์ page ไม่ถูกต้อง",
			})
		}
		if page <= 0 {
			page = 1
		}
	}

	// แปลงค่า page_size ถ้ามี
	pageSize := 10
	pageSizeStr := c.Query("page_size", "10")
	if pageSizeStr != "" {
		var err error
		pageSize, err = strconv.Atoi(pageSizeStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "รูปแบบของพารามิเตอร์ page_size ไม่ถูกต้อง",
			})
		}
		if pageSize <= 0 {
			pageSize = 10
		}
		if pageSize > 100 {
			pageSize = 100
		}
	}

	// ดึงรายการบุคคล
	persons, pagination, err := h.PersonService.ListPersons(c.Context(), organizationID, page, pageSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// สร้าง response
	response := PersonsResponse{
		Data:       persons,
		Pagination: pagination,
	}

	return c.JSON(response)
}

// GetPersonStats เป็น handler สำหรับดึงข้อมูลสถิติของบุคคล
// @Summary Get person statistics
// @Description Retrieve statistics for a person
// @Tags persons
// @Accept json
// @Produce json
// @Param person_hash path string true "Person Hash to retrieve stats for"
// @Security ApiKeyAuth
// @Success 200 {object} models.PersonStats
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 404 {object} ErrorResponse "Person not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/persons/{person_hash}/stats [get]
func (h *PersonHandler) GetPersonStats(c *fiber.Ctx) error {
	// ดึง person_hash จาก path parameters
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

	// ดึงข้อมูลสถิติของบุคคล
	stats, err := h.PersonService.GetPersonStats(c.Context(), personHash, organizationID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(stats)
}

// DeletePerson เป็น handler สำหรับลบข้อมูลบุคคล
// @Summary Delete a person
// @Description Delete a person and all related data
// @Tags persons
// @Accept json
// @Produce json
// @Param person_hash path string true "Person Hash to delete"
// @Security ApiKeyAuth
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse "Unauthorized (invalid or missing API key)"
// @Failure 404 {object} ErrorResponse "Person not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/persons/{person_hash} [delete]
func (h *PersonHandler) DeletePerson(c *fiber.Ctx) error {
	// ดึง person_hash จาก path parameters
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

	// ลบข้อมูลบุคคล
	if err := h.PersonService.DeletePerson(c.Context(), personHash, organizationID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "ลบข้อมูลบุคคลสำเร็จ",
	})
}

// PersonsResponse เป็นโครงสร้างสำหรับส่งรายการบุคคลพร้อมกับข้อมูล pagination
type PersonsResponse struct {
	Data       []models.Person      `json:"data"`
	Pagination *models.Pagination `json:"pagination"`
}