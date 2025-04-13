package services

import (
	"context"
	"fmt"

	"github.com/bemindtech/bmt-manta-dashboard-service/internal/db"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrganizationService ให้บริการเกี่ยวกับการจัดการองค์กร
type OrganizationService struct {
	DB *db.PostgresDB
}

// NewOrganizationService สร้าง OrganizationService ใหม่
func NewOrganizationService(postgres *db.PostgresDB) *OrganizationService {
	return &OrganizationService{
		DB: postgres,
	}
}

// CreateOrganization สร้างองค์กรใหม่
func (s *OrganizationService) CreateOrganization(ctx context.Context, org *models.Organization) error {
	// สร้าง ID ใหม่ถ้ายังไม่มี
	if org.ID == "" {
		org.ID = uuid.New().String()
	}

	// บันทึกลงฐานข้อมูลด้วย GORM
	if err := s.DB.DB.WithContext(ctx).Create(org).Error; err != nil {
		return fmt.Errorf("ไม่สามารถสร้างองค์กร: %w", err)
	}

	return nil
}

// GetOrganization ดึงข้อมูลองค์กรตาม ID
func (s *OrganizationService) GetOrganization(ctx context.Context, id string) (*models.Organization, error) {
	var org models.Organization

	// ดึงข้อมูลองค์กรด้วย GORM
	if err := s.DB.DB.WithContext(ctx).First(&org, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ไม่พบองค์กร")
		}
		return nil, fmt.Errorf("ไม่สามารถดึงข้อมูลองค์กร: %w", err)
	}

	return &org, nil
}

// UpdateOrganization อัปเดตข้อมูลองค์กร
func (s *OrganizationService) UpdateOrganization(ctx context.Context, org *models.Organization) error {
	// อัปเดตข้อมูลองค์กรด้วย GORM
	result := s.DB.DB.WithContext(ctx).Model(&models.Organization{}).
		Where("id = ?", org.ID).
		Updates(map[string]interface{}{
			"name":        org.Name,
			"description": org.Description,
		})

	if result.Error != nil {
		return fmt.Errorf("ไม่สามารถอัปเดตองค์กร: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("ไม่พบองค์กรที่ต้องการอัปเดต")
	}

	return nil
}

// DeleteOrganization ลบองค์กร
func (s *OrganizationService) DeleteOrganization(ctx context.Context, id string) error {
	// ตรวจสอบว่ามีข้อมูล logs ที่เกี่ยวข้องหรือไม่
	var logsCount int64
	if err := s.DB.DB.WithContext(ctx).Model(&models.PersonLog{}).
		Where("organization_id = ?", id).Count(&logsCount).Error; err != nil {
		return fmt.Errorf("ไม่สามารถตรวจสอบการใช้งานขององค์กร: %w", err)
	}

	if logsCount > 0 {
		return fmt.Errorf("ไม่สามารถลบองค์กรที่มีข้อมูลอยู่ได้")
	}

	// ตรวจสอบว่ามีรูปภาพใบหน้าที่เกี่ยวข้องหรือไม่
	var imagesCount int64
	if err := s.DB.DB.WithContext(ctx).Model(&models.FaceImage{}).
		Where("organization_id = ?", id).Count(&imagesCount).Error; err != nil {
		return fmt.Errorf("ไม่สามารถตรวจสอบรูปภาพใบหน้าที่เกี่ยวข้องกับองค์กร: %w", err)
	}

	if imagesCount > 0 {
		return fmt.Errorf("ไม่สามารถลบองค์กรที่มีรูปภาพใบหน้าอยู่ได้")
	}

	// ตรวจสอบว่ามีกล้องที่เกี่ยวข้องหรือไม่
	var camerasCount int64
	if err := s.DB.DB.WithContext(ctx).Model(&models.Camera{}).
		Where("organization_id = ?", id).Count(&camerasCount).Error; err != nil {
		return fmt.Errorf("ไม่สามารถตรวจสอบกล้องที่เกี่ยวข้องกับองค์กร: %w", err)
	}

	if camerasCount > 0 {
		return fmt.Errorf("ไม่สามารถลบองค์กรที่มีกล้องอยู่ได้")
	}

	// ลบ API keys ขององค์กร
	if err := s.DB.DB.WithContext(ctx).Where("organization_id = ?", id).Delete(&models.APIKey{}).Error; err != nil {
		return fmt.Errorf("ไม่สามารถลบ API keys ขององค์กร: %w", err)
	}

	// ลบผู้ใช้ขององค์กร
	if err := s.DB.DB.WithContext(ctx).Where("organization_id = ?", id).Delete(&models.User{}).Error; err != nil {
		return fmt.Errorf("ไม่สามารถลบผู้ใช้ขององค์กร: %w", err)
	}

	// ลบองค์กร
	result := s.DB.DB.WithContext(ctx).Delete(&models.Organization{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("ไม่สามารถลบองค์กร: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("ไม่พบองค์กรที่ต้องการลบ")
	}

	return nil
}

// ListOrganizations ดึงรายการองค์กรทั้งหมด
func (s *OrganizationService) ListOrganizations(ctx context.Context, page, pageSize int) ([]models.Organization, *models.Pagination, error) {
	// ตั้งค่า pagination
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	// นับจำนวนองค์กรทั้งหมด
	var total int64
	if err := s.DB.DB.WithContext(ctx).Model(&models.Organization{}).Count(&total).Error; err != nil {
		return nil, nil, fmt.Errorf("ไม่สามารถนับจำนวนองค์กร: %w", err)
	}

	// ดึงข้อมูลองค์กร
	var orgs []models.Organization
	if err := s.DB.DB.WithContext(ctx).
		Order("name").
		Limit(pageSize).
		Offset(offset).
		Find(&orgs).Error; err != nil {
		return nil, nil, fmt.Errorf("ไม่สามารถดึงรายการองค์กร: %w", err)
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

	return orgs, pagination, nil
}