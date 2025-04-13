package services

import (
	"context"
	"fmt"
	"time"

	"github.com/bemindtech/bmt-manta-dashboard-service/internal/db"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PersonService ให้บริการเกี่ยวกับการจัดการข้อมูลบุคคล
type PersonService struct {
	DB *db.PostgresDB
}

// NewPersonService สร้าง PersonService ใหม่
func NewPersonService(postgres *db.PostgresDB) *PersonService {
	return &PersonService{
		DB: postgres,
	}
}

// GetPerson ดึงข้อมูลบุคคลตาม PersonHash
func (s *PersonService) GetPerson(ctx context.Context, personHash, organizationID string) (*models.Person, error) {
	var person models.Person

	// ดึงข้อมูลบุคคลด้วย GORM รวมถึงรูปภาพใบหน้า
	result := s.DB.DB.WithContext(ctx).
		Preload("FaceImages").
		Where("person_hash = ? AND organization_id = ?", personHash, organizationID).
		First(&person)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ไม่พบข้อมูลบุคคล")
		}
		return nil, fmt.Errorf("ไม่สามารถดึงข้อมูลบุคคล: %w", result.Error)
	}

	return &person, nil
}

// ListPersons ดึงรายการบุคคลทั้งหมดขององค์กร
func (s *PersonService) ListPersons(ctx context.Context, organizationID string, page, pageSize int) ([]models.Person, *models.Pagination, error) {
	// จัดการ pagination
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	// นับจำนวนบุคคลทั้งหมด
	var total int64
	if err := s.DB.DB.WithContext(ctx).Model(&models.Person{}).
		Where("organization_id = ?", organizationID).
		Count(&total).Error; err != nil {
		return nil, nil, fmt.Errorf("ไม่สามารถนับจำนวนบุคคล: %w", err)
	}

	// ดึงข้อมูลบุคคล
	var persons []models.Person
	if err := s.DB.DB.WithContext(ctx).
		Preload("FaceImages", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Limit(1)
		}).
		Where("organization_id = ?", organizationID).
		Order("last_seen DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&persons).Error; err != nil {
		return nil, nil, fmt.Errorf("ไม่สามารถดึงรายการบุคคล: %w", err)
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

	return persons, pagination, nil
}

// CreateOrUpdatePerson สร้างหรืออัปเดตข้อมูลบุคคล
func (s *PersonService) CreateOrUpdatePerson(ctx context.Context, personLog *models.PersonLog) (*models.Person, error) {
	// ตรวจสอบว่ามีบุคคลนี้ในฐานข้อมูลแล้วหรือไม่
	var person models.Person
	result := s.DB.DB.WithContext(ctx).
		Where("person_hash = ? AND organization_id = ?", personLog.PersonHash, personLog.OrganizationID).
		First(&person)

	isNewPerson := result.Error == gorm.ErrRecordNotFound
	now := time.Now()

	if isNewPerson {
		// สร้างบุคคลใหม่
		person = models.Person{
			Base: models.Base{
				ID: uuid.New().String(),
			},
			PersonHash:     personLog.PersonHash,
			FirstSeen:      personLog.Timestamp,
			LastSeen:       personLog.Timestamp,
			VisitCount:     1,
			OrganizationID: personLog.OrganizationID,
		}

		if err := s.DB.DB.WithContext(ctx).Create(&person).Error; err != nil {
			return nil, fmt.Errorf("ไม่สามารถสร้างข้อมูลบุคคลใหม่: %w", err)
		}
	} else if result.Error != nil {
		// เกิดข้อผิดพลาดอื่นๆ
		return nil, fmt.Errorf("ไม่สามารถตรวจสอบข้อมูลบุคคล: %w", result.Error)
	} else {
		// อัปเดตข้อมูลบุคคลที่มีอยู่แล้ว
		updates := map[string]interface{}{
			"last_seen":   personLog.Timestamp,
			"visit_count": gorm.Expr("visit_count + 1"),
			"updated_at":  now,
		}

		if err := s.DB.DB.WithContext(ctx).Model(&person).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("ไม่สามารถอัปเดตข้อมูลบุคคล: %w", err)
		}
	}

	// อัปเดตสถานะ IsNewPerson ใน PersonLog ตามผลการตรวจสอบ
	personLog.IsNewPerson = isNewPerson

	return &person, nil
}

// GetPersonStats ดึงข้อมูลสถิติของบุคคล
func (s *PersonService) GetPersonStats(ctx context.Context, personHash, organizationID string) (*models.PersonStats, error) {
	// ดึงข้อมูลบุคคล
	var person models.Person
	if err := s.DB.DB.WithContext(ctx).
		Where("person_hash = ? AND organization_id = ?", personHash, organizationID).
		First(&person).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ไม่พบข้อมูลบุคคล")
		}
		return nil, fmt.Errorf("ไม่สามารถดึงข้อมูลบุคคล: %w", err)
	}

	// นับจำนวนการพบบุคคลนี้
	var visitCount int64
	if err := s.DB.DB.WithContext(ctx).Model(&models.PersonLog{}).
		Where("person_hash = ? AND organization_id = ?", personHash, organizationID).
		Count(&visitCount).Error; err != nil {
		return nil, fmt.Errorf("ไม่สามารถนับจำนวนการพบบุคคล: %w", err)
	}

	// นับจำนวนรูปภาพใบหน้าของบุคคลนี้
	var faceCount int64
	if err := s.DB.DB.WithContext(ctx).Model(&models.FaceImage{}).
		Where("person_hash = ? AND organization_id = ?", personHash, organizationID).
		Count(&faceCount).Error; err != nil {
		return nil, fmt.Errorf("ไม่สามารถนับจำนวนรูปภาพใบหน้า: %w", err)
	}

	// สร้างข้อมูลสถิติ
	stats := &models.PersonStats{
		New:           1, // จำนวนคนใหม่ (ในกรณีนี้คือ 1 คน)
		Repeat:        int(visitCount) - 1, // จำนวนการพบซ้ำ (ลบการพบครั้งแรกออก)
		OrganizationID: organizationID,
	}

	return stats, nil
}

// DeletePerson ลบข้อมูลบุคคลและรูปภาพที่เกี่ยวข้อง
func (s *PersonService) DeletePerson(ctx context.Context, personHash, organizationID string) error {
	// ใช้ transaction เพื่อให้แน่ใจว่าการลบทั้งหมดสำเร็จหรือล้มเหลวพร้อมกัน
	return s.DB.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// ลบรูปภาพใบหน้าที่เกี่ยวข้อง
		if err := tx.Where("person_hash = ? AND organization_id = ?", personHash, organizationID).
			Delete(&models.FaceImage{}).Error; err != nil {
			return fmt.Errorf("ไม่สามารถลบรูปภาพใบหน้า: %w", err)
		}

		// ลบข้อมูล person logs ที่เกี่ยวข้อง
		if err := tx.Where("person_hash = ? AND organization_id = ?", personHash, organizationID).
			Delete(&models.PersonLog{}).Error; err != nil {
			return fmt.Errorf("ไม่สามารถลบข้อมูล logs: %w", err)
		}

		// ลบข้อมูลบุคคล
		result := tx.Where("person_hash = ? AND organization_id = ?", personHash, organizationID).
			Delete(&models.Person{})
		
		if result.Error != nil {
			return fmt.Errorf("ไม่สามารถลบข้อมูลบุคคล: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("ไม่พบข้อมูลบุคคลที่ต้องการลบ")
		}

		return nil
	})
}