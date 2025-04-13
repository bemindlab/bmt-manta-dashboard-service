package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bemindtech/bmt-manta-dashboard-service/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockPersonService เป็น mock สำหรับ PersonService
type MockPersonService struct {
	mock.Mock
}

// GetPerson implements PersonService.GetPerson
func (m *MockPersonService) GetPerson(ctx context.Context, personHash, organizationID string) (*models.Person, error) {
	args := m.Called(ctx, personHash, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Person), args.Error(1)
}

// ListPersons implements PersonService.ListPersons
func (m *MockPersonService) ListPersons(ctx context.Context, organizationID string, page, pageSize int) ([]models.Person, *models.Pagination, error) {
	args := m.Called(ctx, organizationID, page, pageSize)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]models.Person), args.Get(1).(*models.Pagination), args.Error(2)
}

// CreateOrUpdatePerson implements PersonService.CreateOrUpdatePerson
func (m *MockPersonService) CreateOrUpdatePerson(ctx context.Context, personLog *models.PersonLog) (*models.Person, error) {
	args := m.Called(ctx, personLog)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Person), args.Error(1)
}

// GetPersonStats implements PersonService.GetPersonStats
func (m *MockPersonService) GetPersonStats(ctx context.Context, personHash, organizationID string) (*models.PersonStats, error) {
	args := m.Called(ctx, personHash, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PersonStats), args.Error(1)
}

// DeletePerson implements PersonService.DeletePerson
func (m *MockPersonService) DeletePerson(ctx context.Context, personHash, organizationID string) error {
	args := m.Called(ctx, personHash, organizationID)
	return args.Error(0)
}

// setupTest สร้าง Fiber app และ MockPersonService สำหรับการทดสอบ
func setupHandlerTest() (*fiber.App, *MockPersonService) {
	mockService := new(MockPersonService)
	handler := NewPersonHandler(mockService)

	app := fiber.New()
	
	// สร้าง middleware จำลองเพื่อตั้งค่า context ที่จำเป็น
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("organization_id", "test-org-123")
		return c.Next()
	})

	// ตั้งค่าเส้นทาง
	app.Get("/persons/:person_hash", handler.GetPerson)
	app.Get("/persons", handler.ListPersons)
	app.Get("/persons/:person_hash/stats", handler.GetPersonStats)
	app.Delete("/persons/:person_hash", handler.DeletePerson)

	return app, mockService
}

// TestGetPerson ทดสอบฟังก์ชัน GetPerson
func TestGetPersonHandler(t *testing.T) {
	app, mockService := setupHandlerTest()
	
	// สร้างข้อมูลจำลอง
	now := time.Now()
	personHash := "test-person-123"
	organizationID := "test-org-123"
	
	person := &models.Person{
		Base: models.Base{
			ID:        uuid.New().String(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		PersonHash:     personHash,
		FirstSeen:      now,
		LastSeen:       now,
		VisitCount:     5,
		OrganizationID: organizationID,
		FaceImages: []models.FaceImage{
			{
				Base: models.Base{
					ID:        uuid.New().String(),
					CreatedAt: now,
					UpdatedAt: now,
				},
				PersonHash:     personHash,
				ImageURL:       "https://example.com/face1.jpg",
				OrganizationID: organizationID,
			},
		},
	}

	// กำหนดพฤติกรรมของ mock service
	mockService.On("GetPerson", mock.Anything, personHash, organizationID).Return(person, nil)

	// สร้าง HTTP request
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/persons/%s", personHash), nil)
	req.Header.Set("Content-Type", "application/json")

	// ทดสอบ
	resp, err := app.Test(req)
	require.NoError(t, err)
	
	// ตรวจสอบ status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// อ่าน response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	
	// แปลง JSON response เป็น Person
	var responsePerson models.Person
	err = json.Unmarshal(body, &responsePerson)
	require.NoError(t, err)
	
	// ตรวจสอบข้อมูล
	assert.Equal(t, person.PersonHash, responsePerson.PersonHash)
	assert.Equal(t, person.OrganizationID, responsePerson.OrganizationID)
	assert.Equal(t, person.VisitCount, responsePerson.VisitCount)
	assert.Len(t, responsePerson.FaceImages, 1)
	
	// ตรวจสอบว่า mock ถูกเรียกตามที่คาดหวัง
	mockService.AssertExpectations(t)
}

// TestGetPerson_Error ทดสอบฟังก์ชัน GetPerson กรณีเกิดข้อผิดพลาด
func TestGetPersonHandler_Error(t *testing.T) {
	app, mockService := setupHandlerTest()
	
	personHash := "test-person-123"
	organizationID := "test-org-123"
	
	// กำหนดให้ service คืนค่า error
	mockService.On("GetPerson", mock.Anything, personHash, organizationID).
		Return(nil, errors.New("ไม่พบข้อมูลบุคคล"))

	// สร้าง HTTP request
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/persons/%s", personHash), nil)
	req.Header.Set("Content-Type", "application/json")

	// ทดสอบ
	resp, err := app.Test(req)
	require.NoError(t, err)
	
	// ตรวจสอบ status code
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	
	// อ่าน response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	
	// แปลง JSON response
	var errorResponse map[string]string
	err = json.Unmarshal(body, &errorResponse)
	require.NoError(t, err)
	
	// ตรวจสอบข้อความ error
	assert.Equal(t, "ไม่พบข้อมูลบุคคล", errorResponse["error"])
	
	// ตรวจสอบว่า mock ถูกเรียกตามที่คาดหวัง
	mockService.AssertExpectations(t)
}

// TestListPersons ทดสอบฟังก์ชัน ListPersons
func TestListPersonsHandler(t *testing.T) {
	app, mockService := setupHandlerTest()
	
	// สร้างข้อมูลจำลอง
	now := time.Now()
	organizationID := "test-org-123"
	
	persons := []models.Person{
		{
			Base: models.Base{
				ID:        uuid.New().String(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			PersonHash:     "person-1",
			FirstSeen:      now,
			LastSeen:       now,
			VisitCount:     5,
			OrganizationID: organizationID,
			FaceImages: []models.FaceImage{
				{
					Base: models.Base{
						ID:        uuid.New().String(),
						CreatedAt: now,
						UpdatedAt: now,
					},
					PersonHash:     "person-1",
					ImageURL:       "https://example.com/face1.jpg",
					OrganizationID: organizationID,
				},
			},
		},
		{
			Base: models.Base{
				ID:        uuid.New().String(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			PersonHash:     "person-2",
			FirstSeen:      now,
			LastSeen:       now,
			VisitCount:     3,
			OrganizationID: organizationID,
			FaceImages: []models.FaceImage{
				{
					Base: models.Base{
						ID:        uuid.New().String(),
						CreatedAt: now,
						UpdatedAt: now,
					},
					PersonHash:     "person-2",
					ImageURL:       "https://example.com/face2.jpg",
					OrganizationID: organizationID,
				},
			},
		},
	}
	
	pagination := &models.Pagination{
		Total:     2,
		Page:      1,
		PageSize:  10,
		TotalPage: 1,
	}

	// กำหนดพฤติกรรมของ mock service
	mockService.On("ListPersons", mock.Anything, organizationID, 1, 10).Return(persons, pagination, nil)

	// สร้าง HTTP request
	req := httptest.NewRequest(http.MethodGet, "/persons?page=1&page_size=10", nil)
	req.Header.Set("Content-Type", "application/json")

	// ทดสอบ
	resp, err := app.Test(req)
	require.NoError(t, err)
	
	// ตรวจสอบ status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// อ่าน response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	
	// แปลง JSON response
	var response PersonsResponse
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)
	
	// ตรวจสอบข้อมูล
	assert.Len(t, response.Data, 2)
	assert.Equal(t, "person-1", response.Data[0].PersonHash)
	assert.Equal(t, "person-2", response.Data[1].PersonHash)
	assert.Equal(t, 1, response.Pagination.Page)
	assert.Equal(t, 10, response.Pagination.PageSize)
	
	// ตรวจสอบว่า mock ถูกเรียกตามที่คาดหวัง
	mockService.AssertExpectations(t)
}

// TestGetPersonStats ทดสอบฟังก์ชัน GetPersonStats
func TestGetPersonStatsHandler(t *testing.T) {
	app, mockService := setupHandlerTest()
	
	// สร้างข้อมูลจำลอง
	personHash := "test-person-123"
	organizationID := "test-org-123"
	
	stats := &models.PersonStats{
		New:            1,
		Repeat:         5,
		OrganizationID: organizationID,
	}

	// กำหนดพฤติกรรมของ mock service
	mockService.On("GetPersonStats", mock.Anything, personHash, organizationID).Return(stats, nil)

	// สร้าง HTTP request
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/persons/%s/stats", personHash), nil)
	req.Header.Set("Content-Type", "application/json")

	// ทดสอบ
	resp, err := app.Test(req)
	require.NoError(t, err)
	
	// ตรวจสอบ status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// อ่าน response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	
	// แปลง JSON response
	var responseStats models.PersonStats
	err = json.Unmarshal(body, &responseStats)
	require.NoError(t, err)
	
	// ตรวจสอบข้อมูล
	assert.Equal(t, 1, responseStats.New)
	assert.Equal(t, 5, responseStats.Repeat)
	assert.Equal(t, organizationID, responseStats.OrganizationID)
	
	// ตรวจสอบว่า mock ถูกเรียกตามที่คาดหวัง
	mockService.AssertExpectations(t)
}

// TestDeletePerson ทดสอบฟังก์ชัน DeletePerson
func TestDeletePersonHandler(t *testing.T) {
	app, mockService := setupHandlerTest()
	
	// สร้างข้อมูลจำลอง
	personHash := "test-person-123"
	organizationID := "test-org-123"

	// กำหนดพฤติกรรมของ mock service
	mockService.On("DeletePerson", mock.Anything, personHash, organizationID).Return(nil)

	// สร้าง HTTP request
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/persons/%s", personHash), nil)
	req.Header.Set("Content-Type", "application/json")

	// ทดสอบ
	resp, err := app.Test(req)
	require.NoError(t, err)
	
	// ตรวจสอบ status code
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// อ่าน response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	
	// แปลง JSON response
	var response map[string]string
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)
	
	// ตรวจสอบข้อความตอบกลับ
	assert.Equal(t, "ลบข้อมูลบุคคลสำเร็จ", response["message"])
	
	// ตรวจสอบว่า mock ถูกเรียกตามที่คาดหวัง
	mockService.AssertExpectations(t)
}

// TestDeletePerson_Error ทดสอบฟังก์ชัน DeletePerson กรณีเกิดข้อผิดพลาด
func TestDeletePersonHandler_Error(t *testing.T) {
	app, mockService := setupHandlerTest()
	
	// สร้างข้อมูลจำลอง
	personHash := "test-person-123"
	organizationID := "test-org-123"

	// กำหนดให้ service คืนค่า error
	mockService.On("DeletePerson", mock.Anything, personHash, organizationID).
		Return(errors.New("ไม่พบข้อมูลบุคคลที่ต้องการลบ"))

	// สร้าง HTTP request
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/persons/%s", personHash), nil)
	req.Header.Set("Content-Type", "application/json")

	// ทดสอบ
	resp, err := app.Test(req)
	require.NoError(t, err)
	
	// ตรวจสอบ status code
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	
	// อ่าน response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	
	// แปลง JSON response
	var errorResponse map[string]string
	err = json.Unmarshal(body, &errorResponse)
	require.NoError(t, err)
	
	// ตรวจสอบข้อความ error
	assert.Equal(t, "ไม่พบข้อมูลบุคคลที่ต้องการลบ", errorResponse["error"])
	
	// ตรวจสอบว่า mock ถูกเรียกตามที่คาดหวัง
	mockService.AssertExpectations(t)
}