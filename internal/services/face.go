package services

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/bemindtech/bmt-manta-dashboard-service/internal/db"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/models"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/storage"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FaceService ให้บริการเกี่ยวกับการจัดการใบหน้า
type FaceService struct {
	DB      *db.PostgresDB
	Storage storage.StorageService
}

// NewFaceService สร้าง FaceService ใหม่
func NewFaceService(postgres *db.PostgresDB, storage storage.StorageService) *FaceService {
	return &FaceService{
		DB:      postgres,
		Storage: storage,
	}
}

// UploadFaceImage อัปโหลดรูปภาพใบหน้า
func (s *FaceService) UploadFaceImage(ctx context.Context, file *multipart.FileHeader, personHash, cameraID, organizationID string) (*models.FaceImage, error) {
	// ตรวจสอบว่ามีรหัสกล้องหรือไม่
	if cameraID == "" {
		return nil, fmt.Errorf("ต้องระบุรหัสกล้อง")
	}

	// ตรวจสอบว่ามีรหัสบุคคลหรือไม่
	if personHash == "" {
		return nil, fmt.Errorf("ต้องระบุรหัสบุคคล")
	}

	// ตรวจสอบว่ามีองค์กรหรือไม่
	if organizationID == "" {
		return nil, fmt.Errorf("ต้องระบุองค์กร")
	}

	// ตรวจสอบไฟล์
	if file == nil {
		return nil, fmt.Errorf("ไม่มีไฟล์ที่อัปโหลด")
	}

	// อัปโหลดไฟล์ไปยังระบบจัดเก็บ
	imageURL, err := s.Storage.UploadFaceImage(ctx, file, personHash, organizationID)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถอัปโหลดรูปภาพ: %w", err)
	}

	// สร้างข้อมูลในฐานข้อมูล
	faceImage := &models.FaceImage{
		Base: models.Base{
			ID: uuid.New().String(),
		},
		PersonHash:     personHash,
		ImageURL:       imageURL,
		OrganizationID: organizationID,
		CameraID:       cameraID,
	}

	// บันทึกลงฐานข้อมูลด้วย GORM
	if err := s.DB.DB.WithContext(ctx).Create(faceImage).Error; err != nil {
		// ถ้าบันทึกไม่สำเร็จ ให้ลบไฟล์ที่อัปโหลดไปแล้ว
		_ = s.Storage.DeleteFaceImage(ctx, imageURL)
		return nil, fmt.Errorf("ไม่สามารถบันทึกข้อมูลรูปภาพ: %w", err)
	}

	return faceImage, nil
}

// GetFaceImages ดึงรูปภาพใบหน้าตามรหัสบุคคล
func (s *FaceService) GetFaceImages(ctx context.Context, personHash, organizationID string, page, pageSize int) ([]models.FaceImage, *models.Pagination, error) {
	// คำนวณ offset
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	// สร้าง query ด้วย GORM
	query := s.DB.DB.WithContext(ctx).Model(&models.FaceImage{}).Where("person_hash = ? AND organization_id = ?", personHash, organizationID)

	// นับจำนวนรายการทั้งหมด
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, nil, fmt.Errorf("ไม่สามารถนับจำนวนรูปภาพ: %w", err)
	}

	// ดึงข้อมูลรูปภาพด้วย GORM
	var images []models.FaceImage
	if err := query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&images).Error; err != nil {
		return nil, nil, fmt.Errorf("ไม่สามารถดึงข้อมูลรูปภาพ: %w", err)
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

	return images, pagination, nil
}

// DeleteFaceImage ลบรูปภาพใบหน้า
func (s *FaceService) DeleteFaceImage(ctx context.Context, id, organizationID string) error {
	// ดึงข้อมูลรูปภาพก่อนลบ เพื่อใช้ลบไฟล์
	var faceImage models.FaceImage
	result := s.DB.DB.WithContext(ctx).Where("id = ? AND organization_id = ?", id, organizationID).First(&faceImage)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return fmt.Errorf("ไม่พบรูปภาพที่ต้องการลบ")
		}
		return fmt.Errorf("ไม่สามารถดึงข้อมูลรูปภาพ: %w", result.Error)
	}

	// เก็บ URL ของรูปภาพไว้เพื่อใช้ในการลบไฟล์
	imageURL := faceImage.ImageURL

	// ลบข้อมูลในฐานข้อมูลด้วย GORM
	result = s.DB.DB.WithContext(ctx).Where("id = ? AND organization_id = ?", id, organizationID).Delete(&models.FaceImage{})
	if result.Error != nil {
		return fmt.Errorf("ไม่สามารถลบข้อมูลรูปภาพในฐานข้อมูล: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("ไม่พบรูปภาพที่ต้องการลบ")
	}

	// ลบไฟล์
	if err := s.Storage.DeleteFaceImage(ctx, imageURL); err != nil {
		// บันทึก log แต่ไม่ return error เพราะข้อมูลในฐานข้อมูลถูกลบไปแล้ว
		fmt.Printf("ไม่สามารถลบไฟล์รูปภาพ %s: %v\n", imageURL, err)
	}

	return nil
}