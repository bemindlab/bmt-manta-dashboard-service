package services

import (
	"context"
	"fmt"

	"github.com/bemindtech/bmt-manta-dashboard-service/internal/db"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CameraService ให้บริการเกี่ยวกับการจัดการกล้อง
type CameraService struct {
	DB *db.PostgresDB
}

// NewCameraService สร้าง CameraService ใหม่
func NewCameraService(postgres *db.PostgresDB) *CameraService {
	return &CameraService{
		DB: postgres,
	}
}

// CreateCamera สร้างกล้องใหม่
func (s *CameraService) CreateCamera(ctx context.Context, camera *models.Camera) error {
	// สร้าง ID ใหม่ถ้ายังไม่มี
	if camera.ID == "" {
		camera.ID = uuid.New().String()
	}

	// กำหนดสถานะเริ่มต้นเป็น active ถ้าไม่ระบุ
	if camera.Status == "" {
		camera.Status = "active"
	}

	// เพิ่มกล้องลงในฐานข้อมูลด้วย GORM
	// GORM จะจัดการกับ created_at และ updated_at โดยอัตโนมัติ
	if err := s.DB.DB.WithContext(ctx).Create(camera).Error; err != nil {
		return fmt.Errorf("ไม่สามารถสร้างกล้อง: %w", err)
	}

	return nil
}

// GetCamera ดึงข้อมูลกล้องตาม ID
func (s *CameraService) GetCamera(ctx context.Context, id, organizationID string) (*models.Camera, error) {
	var camera models.Camera
	
	result := s.DB.DB.WithContext(ctx).Where("id = ? AND organization_id = ?", id, organizationID).First(&camera)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ไม่พบกล้อง")
		}
		return nil, fmt.Errorf("ไม่สามารถดึงข้อมูลกล้อง: %w", result.Error)
	}

	return &camera, nil
}

// UpdateCamera อัปเดตข้อมูลกล้อง
func (s *CameraService) UpdateCamera(ctx context.Context, camera *models.Camera) error {
	// อัปเดตกล้องด้วย GORM
	// เลือกเฉพาะฟิลด์ที่ต้องการอัปเดต และอัปเดตเฉพาะกล้องที่เป็นขององค์กรนั้น
	result := s.DB.DB.WithContext(ctx).Model(&models.Camera{}).Where("id = ? AND organization_id = ?", camera.ID, camera.OrganizationID).Updates(map[string]interface{}{
		"name":     camera.Name,
		"location": camera.Location,
		"status":   camera.Status,
	})

	if result.Error != nil {
		return fmt.Errorf("ไม่สามารถอัปเดตกล้อง: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("ไม่พบกล้องที่ต้องการอัปเดต")
	}

	return nil
}

// DeleteCamera ลบกล้อง
func (s *CameraService) DeleteCamera(ctx context.Context, id, organizationID string) error {
	// เพิ่มการตรวจสอบว่ามีการใช้งานอยู่หรือไม่ เช่น ตรวจสอบจำนวนข้อมูล logs ก่อนลบ
	var logsCount int64
	if err := s.DB.DB.WithContext(ctx).Model(&models.PersonLog{}).Where("camera_id = ? AND organization_id = ?", id, organizationID).Count(&logsCount).Error; err != nil {
		return fmt.Errorf("ไม่สามารถตรวจสอบการใช้งานของกล้อง: %w", err)
	}

	if logsCount > 0 {
		return fmt.Errorf("ไม่สามารถลบกล้องที่มีข้อมูล logs อยู่ได้")
	}

	// ตรวจสอบว่ามีรูปภาพใบหน้าที่เกี่ยวข้องหรือไม่
	var imagesCount int64
	if err := s.DB.DB.WithContext(ctx).Model(&models.FaceImage{}).Where("camera_id = ? AND organization_id = ?", id, organizationID).Count(&imagesCount).Error; err != nil {
		return fmt.Errorf("ไม่สามารถตรวจสอบรูปภาพใบหน้าที่เกี่ยวข้องกับกล้อง: %w", err)
	}

	if imagesCount > 0 {
		return fmt.Errorf("ไม่สามารถลบกล้องที่มีรูปภาพใบหน้าอยู่ได้")
	}

	// ลบกล้องด้วย GORM (soft delete โดยใช้ DeletedAt field)
	result := s.DB.DB.WithContext(ctx).Where("id = ? AND organization_id = ?", id, organizationID).Delete(&models.Camera{})
	if result.Error != nil {
		return fmt.Errorf("ไม่สามารถลบกล้อง: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("ไม่พบกล้องที่ต้องการลบ")
	}

	return nil
}

// ListCameras ดึงรายการกล้องทั้งหมดขององค์กร
func (s *CameraService) ListCameras(ctx context.Context, organizationID string, page, pageSize int) ([]models.Camera, *models.Pagination, error) {
	// คำนวณ offset
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	// นับจำนวนกล้องทั้งหมดด้วย GORM
	var total int64
	if err := s.DB.DB.WithContext(ctx).Model(&models.Camera{}).Where("organization_id = ?", organizationID).Count(&total).Error; err != nil {
		return nil, nil, fmt.Errorf("ไม่สามารถนับจำนวนกล้อง: %w", err)
	}

	// ดึงข้อมูลกล้องด้วย GORM
	var cameras []models.Camera
	if err := s.DB.DB.WithContext(ctx).Where("organization_id = ?", organizationID).Order("name").Limit(pageSize).Offset(offset).Find(&cameras).Error; err != nil {
		return nil, nil, fmt.Errorf("ไม่สามารถดึงรายการกล้อง: %w", err)
	}

	// คำนวณจำนวนหน้าทั้งหมด
	totalPage := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPage++
	}

	// สร้างข้อมูล pagination
	pagination := &models.Pagination{
		Total:     int(total),
		Page:      page,
		PageSize:  pageSize,
		TotalPage: totalPage,
	}

	return cameras, pagination, nil
}