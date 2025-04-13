package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bemindtech/bmt-manta-dashboard-service/internal/db"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/firebase"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SyncService เป็นโครงสร้างสำหรับการซิงค์ข้อมูลจาก Firebase ไปยัง PostgreSQL
type SyncService struct {
	DB           *db.PostgresDB
	Firebase     *firebase.FirebaseClient
	PersonService *PersonService
}

// NewSyncService สร้าง SyncService ใหม่
func NewSyncService(postgres *db.PostgresDB, firebaseClient *firebase.FirebaseClient) *SyncService {
	return &SyncService{
		DB:           postgres,
		Firebase:     firebaseClient,
		PersonService: NewPersonService(postgres),
	}
}

// StartSync เริ่มการซิงค์ข้อมูลจาก Firebase ไปยัง PostgreSQL
func (s *SyncService) StartSync(ctx context.Context, logsPath string) error {
	// เริ่มการรับฟังข้อมูลใหม่จาก Firebase
	logsChan, err := s.Firebase.ListenForNewLogs(ctx, logsPath)
	if err != nil {
		return fmt.Errorf("ไม่สามารถเริ่มการรับฟังข้อมูลจาก Firebase: %w", err)
	}

	// เริ่ม goroutine สำหรับการรับข้อมูลและซิงค์
	go func() {
		for {
			select {
			case data, ok := <-logsChan:
				if !ok {
					// channel ถูกปิด
					log.Println("การรับฟังข้อมูลจาก Firebase ถูกปิด")
					return
				}
				// ซิงค์ข้อมูลไปยัง PostgreSQL
				if err := s.syncLogToPostgres(ctx, data); err != nil {
					log.Printf("ไม่สามารถซิงค์ข้อมูลไปยัง PostgreSQL: %v", err)
				}
			case <-ctx.Done():
				// context ถูกยกเลิก
				log.Println("การซิงค์ข้อมูลถูกยกเลิก")
				return
			}
		}
	}()

	return nil
}

// BackfillLogs ดึงข้อมูลเก่าจาก Firebase และซิงค์ไปยัง PostgreSQL
func (s *SyncService) BackfillLogs(ctx context.Context, logsPath string, limit int) error {
	// ดึงข้อมูล logs ล่าสุดจาก Firebase
	logs, err := s.Firebase.GetRecentLogs(ctx, logsPath, limit)
	if err != nil {
		return fmt.Errorf("ไม่สามารถดึงข้อมูลจาก Firebase: %w", err)
	}

	// ซิงค์ข้อมูลไปยัง PostgreSQL
	for _, log := range logs {
		if err := s.syncLogToPostgres(ctx, log); err != nil {
			return fmt.Errorf("ไม่สามารถซิงค์ข้อมูลไปยัง PostgreSQL: %w", err)
		}
	}

	return nil
}

// syncLogToPostgres ซิงค์ข้อมูล log จาก Firebase ไปยัง PostgreSQL
func (s *SyncService) syncLogToPostgres(ctx context.Context, data map[string]interface{}) error {
	// แปลงข้อมูลจาก Firebase
	id, _ := data["id"].(string)
	if id == "" {
		id = uuid.New().String()
	}

	// ตรวจสอบว่ามีข้อมูลที่จำเป็นครบถ้วน
	timestamp, ok := data["timestamp"].(float64)
	if !ok {
		return fmt.Errorf("ไม่พบหรือรูปแบบของ timestamp ไม่ถูกต้อง")
	}

	personHash, ok := data["person_hash"].(string)
	if !ok || personHash == "" {
		return fmt.Errorf("ไม่พบหรือรูปแบบของ person_hash ไม่ถูกต้อง")
	}

	cameraID, ok := data["camera_id"].(string)
	if !ok || cameraID == "" {
		return fmt.Errorf("ไม่พบหรือรูปแบบของ camera_id ไม่ถูกต้อง")
	}

	// แปลง timestamp เป็น time.Time
	timestampTime := time.Unix(int64(timestamp), 0)
	
	// ดึงข้อมูลกล้องเพื่อหา organization_id
	var camera models.Camera
	result := s.DB.DB.WithContext(ctx).First(&camera, "id = ?", cameraID)
	if result.Error != nil {
		// ถ้าไม่พบกล้อง ใช้ organization ID เริ่มต้น
		// ซึ่งในกรณีนี้ควรจะกำหนด organization_id เริ่มต้นไว้ (อาจจะเป็นองค์กรเริ่มต้น)
		log.Printf("ไม่พบข้อมูลกล้อง %s, จะใช้ organization_id เริ่มต้น", cameraID)
		// องค์กรเริ่มต้นควรมีแล้วจากการ seed ข้อมูล
		var defaultOrg models.Organization
		if err := s.DB.DB.WithContext(ctx).First(&defaultOrg).Error; err != nil {
			return fmt.Errorf("ไม่พบข้อมูลองค์กรเริ่มต้น: %w", err)
		}
		camera.OrganizationID = defaultOrg.ID
	}

	// ตรวจสอบว่ามี log นี้อยู่ในฐานข้อมูลแล้วหรือไม่
	var existingLog models.PersonLog
	result = s.DB.DB.WithContext(ctx).Where(
		"person_hash = ? AND camera_id = ? AND timestamp = ?",
		personHash, cameraID, timestampTime,
	).First(&existingLog)

	// ถ้ามีข้อมูลอยู่แล้ว ไม่ต้องเพิ่มใหม่
	if result.Error == nil {
		return nil
	}

	// ถ้าไม่พบข้อมูล (ErrRecordNotFound) ให้เพิ่มใหม่
	if result.Error != gorm.ErrRecordNotFound {
		return fmt.Errorf("ไม่สามารถตรวจสอบข้อมูลใน PostgreSQL: %w", result.Error)
	}

	// ตรวจสอบว่าเป็นคนใหม่หรือคนซ้ำ
	var count int64
	if err := s.DB.DB.WithContext(ctx).Model(&models.PersonLog{}).Where(
		"person_hash = ? AND timestamp < ?",
		personHash, timestampTime,
	).Count(&count).Error; err != nil {
		return fmt.Errorf("ไม่สามารถตรวจสอบประวัติบุคคล: %w", err)
	}

	// กำหนดว่าเป็นคนใหม่หรือไม่
	isNewPerson := count == 0

	// สร้าง log ใหม่
	newLog := models.PersonLog{
		Base: models.Base{
			ID:        id,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Timestamp:     timestampTime,
		PersonHash:    personHash,
		CameraID:      cameraID,
		IsNewPerson:   isNewPerson,
		OrganizationID: camera.OrganizationID,
	}

	// เพิ่มข้อมูลใน PostgreSQL
	if err := s.DB.DB.WithContext(ctx).Create(&newLog).Error; err != nil {
		return fmt.Errorf("ไม่สามารถเพิ่มข้อมูลใน PostgreSQL: %w", err)
	}

	// อัปเดตหรือสร้างข้อมูลบุคคล
	if _, personErr := s.PersonService.CreateOrUpdatePerson(ctx, &newLog); personErr != nil {
		log.Printf("ไม่สามารถอัปเดตข้อมูลบุคคล: %v", personErr)
		// ดำเนินการต่อแม้จะมีข้อผิดพลาดในการอัปเดตข้อมูลบุคคล
	}

	log.Printf("ซิงค์ข้อมูล log %s สำเร็จ (คนใหม่: %v)", id, isNewPerson)
	return nil
}

// SyncPersonLogToFirebase ซิงค์ข้อมูล log จาก PostgreSQL ไปยัง Firebase (ถ้าจำเป็น)
func (s *SyncService) SyncPersonLogToFirebase(ctx context.Context, personLog models.PersonLog, logsPath string) error {
	// สร้างข้อมูลสำหรับบันทึกใน Firebase
	data := map[string]interface{}{
		"timestamp":   personLog.Timestamp.Unix(),
		"person_hash": personLog.PersonHash,
		"camera_id":   personLog.CameraID,
	}

	// บันทึกข้อมูลใน Firebase
	ref := s.Firebase.DB.NewRef(fmt.Sprintf("%s/%s", logsPath, personLog.ID))
	if err := ref.Set(ctx, data); err != nil {
		return fmt.Errorf("ไม่สามารถบันทึกข้อมูลใน Firebase: %w", err)
	}

	log.Printf("ซิงค์ข้อมูล log %s ไปยัง Firebase สำเร็จ", personLog.ID)
	return nil
} 