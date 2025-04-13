package services

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/db"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupTest สร้าง mock database และ PersonService สำหรับการทดสอบ
func setupTest(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *PersonService, func()) {
	// สร้าง sqlmock
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	// สร้าง gorm.DB ด้วย mock database
	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 mockDB,
		PreferSimpleProtocol: true,
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	// สร้าง PostgresDB wrapper
	postgresDB := &db.PostgresDB{
		DB: gormDB,
	}

	// สร้าง PersonService
	service := NewPersonService(postgresDB)

	// สร้าง function สำหรับการ cleanup
	cleanup := func() {
		mockDB.Close()
	}

	return mockDB, mock, service, cleanup
}

// TestGetPerson ทดสอบเมธอด GetPerson
func TestGetPerson(t *testing.T) {
	_, mock, service, cleanup := setupTest(t)
	defer cleanup()

	ctx := context.Background()
	personHash := "abc123"
	organizationID := "org123"
	now := time.Now()

	// Mock ข้อมูลที่จะ return จาก database
	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "person_hash", "first_seen", "last_seen", "visit_count", "organization_id"}).
		AddRow("person-uuid", now, now, nil, personHash, now, now, 5, organizationID)

	// Mock GORM จะใช้ query ประมาณนี้สำหรับ First
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "persons" WHERE person_hash = $1 AND organization_id = $2 AND "persons"."deleted_at" IS NULL ORDER BY "persons"."id" LIMIT 1`)).
		WithArgs(personHash, organizationID).
		WillReturnRows(rows)

	// Mock rows สำหรับการ preload FaceImages
	faceImageRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "person_hash", "image_url", "organization_id"}).
		AddRow("face-uuid-1", now, now, nil, personHash, "http://example.com/face1.jpg", organizationID).
		AddRow("face-uuid-2", now, now, nil, personHash, "http://example.com/face2.jpg", organizationID)

	// Mock GORM query สำหรับ Preload FaceImages
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "face_images" WHERE "face_images"."person_hash" = $1 AND "face_images"."deleted_at" IS NULL`)).
		WithArgs(personHash).
		WillReturnRows(faceImageRows)

	// เรียกใช้ฟังก์ชันที่ต้องการทดสอบ
	person, err := service.GetPerson(ctx, personHash, organizationID)

	// ตรวจสอบว่าไม่มี error และได้ข้อมูลที่ถูกต้อง
	assert.NoError(t, err)
	assert.NotNil(t, person)
	assert.Equal(t, personHash, person.PersonHash)
	assert.Equal(t, organizationID, person.OrganizationID)
	assert.Equal(t, 5, person.VisitCount)
	assert.Len(t, person.FaceImages, 2)

	// ตรวจสอบว่าไม่มี expectations ใด ๆ ที่ไม่ได้ถูกเรียกใช้
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestListPersons ทดสอบเมธอด ListPersons
func TestListPersons(t *testing.T) {
	_, mock, service, cleanup := setupTest(t)
	defer cleanup()

	ctx := context.Background()
	organizationID := "org123"
	page := 1
	pageSize := 10
	now := time.Now()

	// Mock การนับจำนวนรายการทั้งหมดสำหรับ pagination
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(15)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "persons" WHERE organization_id = $1 AND "persons"."deleted_at" IS NULL`)).
		WithArgs(organizationID).
		WillReturnRows(countRows)

	// Mock ข้อมูลบุคคลที่จะ return
	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "person_hash", "first_seen", "last_seen", "visit_count", "organization_id"}).
		AddRow("person-uuid-1", now, now, nil, "person1", now, now, 5, organizationID).
		AddRow("person-uuid-2", now, now, nil, "person2", now, now, 3, organizationID)

	// Mock query สำหรับการดึงข้อมูลบุคคล
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "persons" WHERE organization_id = $1 AND "persons"."deleted_at" IS NULL ORDER BY last_seen DESC LIMIT 10 OFFSET 0`)).
		WithArgs(organizationID).
		WillReturnRows(rows)

	// Mock ข้อมูล FaceImages สำหรับ person ที่ 1
	faceImagesRows1 := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "person_hash", "image_url", "organization_id"}).
		AddRow("face-uuid-1", now, now, nil, "person1", "http://example.com/face1.jpg", organizationID)
	
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "face_images" WHERE "face_images"."person_hash" = $1 AND "face_images"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT 1`)).
		WithArgs("person1").
		WillReturnRows(faceImagesRows1)

	// Mock ข้อมูล FaceImages สำหรับ person ที่ 2
	faceImagesRows2 := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "person_hash", "image_url", "organization_id"}).
		AddRow("face-uuid-2", now, now, nil, "person2", "http://example.com/face2.jpg", organizationID)
	
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "face_images" WHERE "face_images"."person_hash" = $1 AND "face_images"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT 1`)).
		WithArgs("person2").
		WillReturnRows(faceImagesRows2)

	// เรียกใช้ฟังก์ชันที่ต้องการทดสอบ
	persons, pagination, err := service.ListPersons(ctx, organizationID, page, pageSize)

	// ตรวจสอบว่าไม่มี error
	assert.NoError(t, err)
	assert.NotNil(t, persons)
	assert.NotNil(t, pagination)
	
	// ตรวจสอบข้อมูล pagination
	assert.Equal(t, 15, pagination.Total)
	assert.Equal(t, 1, pagination.Page)
	assert.Equal(t, 10, pagination.PageSize)
	assert.Equal(t, 2, pagination.TotalPage)
	
	// ตรวจสอบข้อมูล persons
	assert.Len(t, persons, 2)
	assert.Equal(t, "person1", persons[0].PersonHash)
	assert.Equal(t, 5, persons[0].VisitCount)
	assert.Equal(t, "person2", persons[1].PersonHash)
	assert.Equal(t, 3, persons[1].VisitCount)
	
	// ตรวจสอบว่าแต่ละ person มี FaceImage
	assert.Len(t, persons[0].FaceImages, 1)
	assert.Equal(t, "http://example.com/face1.jpg", persons[0].FaceImages[0].ImageURL)
	assert.Len(t, persons[1].FaceImages, 1)
	assert.Equal(t, "http://example.com/face2.jpg", persons[1].FaceImages[0].ImageURL)

	// ตรวจสอบว่าไม่มี expectations ใด ๆ ที่ไม่ได้ถูกเรียกใช้
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateOrUpdatePerson_Create ทดสอบเมธอด CreateOrUpdatePerson ในกรณีสร้างบุคคลใหม่
func TestCreateOrUpdatePerson_Create(t *testing.T) {
	_, mock, service, cleanup := setupTest(t)
	defer cleanup()

	ctx := context.Background()
	personHash := "new-person-123"
	organizationID := "org123"
	now := time.Now()

	// สร้าง personLog สำหรับทดสอบ
	personLog := &models.PersonLog{
		Base: models.Base{
			ID: uuid.New().String(),
		},
		PersonHash:     personHash,
		CameraID:       "camera123",
		OrganizationID: organizationID,
		Timestamp:      now,
	}

	// Mock กรณีที่ค้นหาบุคคลไม่พบ (สร้างใหม่)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "persons" WHERE person_hash = $1 AND organization_id = $2 AND "persons"."deleted_at" IS NULL ORDER BY "persons"."id" LIMIT 1`)).
		WithArgs(personHash, organizationID).
		WillReturnError(gorm.ErrRecordNotFound)

	// Mock การ insert ข้อมูลบุคคลใหม่
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "persons" ("id","created_at","updated_at","deleted_at","person_hash","first_seen","last_seen","visit_count","organization_id") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING "id"`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), nil, personHash, now, now, 1, organizationID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New().String()))
	mock.ExpectCommit()

	// เรียกใช้ฟังก์ชันที่ต้องการทดสอบ
	person, err := service.CreateOrUpdatePerson(ctx, personLog)

	// ตรวจสอบว่าไม่มี error
	assert.NoError(t, err)
	assert.NotNil(t, person)
	assert.Equal(t, personHash, person.PersonHash)
	assert.Equal(t, organizationID, person.OrganizationID)
	assert.Equal(t, 1, person.VisitCount)
	assert.Equal(t, now, person.FirstSeen)
	assert.Equal(t, now, person.LastSeen)
	
	// ตรวจสอบว่า IsNewPerson ถูกตั้งค่าเป็น true เนื่องจากเป็นบุคคลใหม่
	assert.True(t, personLog.IsNewPerson)

	// ตรวจสอบว่าไม่มี expectations ใด ๆ ที่ไม่ได้ถูกเรียกใช้
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateOrUpdatePerson_Update ทดสอบเมธอด CreateOrUpdatePerson ในกรณีอัปเดตบุคคลที่มีอยู่แล้ว
func TestCreateOrUpdatePerson_Update(t *testing.T) {
	_, mock, service, cleanup := setupTest(t)
	defer cleanup()

	ctx := context.Background()
	personHash := "existing-person-123"
	organizationID := "org123"
	now := time.Now()
	oldTime := now.Add(-24 * time.Hour) // 1 วันก่อน

	// สร้าง personLog สำหรับทดสอบ
	personLog := &models.PersonLog{
		Base: models.Base{
			ID: uuid.New().String(),
		},
		PersonHash:     personHash,
		CameraID:       "camera123",
		OrganizationID: organizationID,
		Timestamp:      now,
	}

	// Mock ข้อมูลบุคคลที่มีอยู่แล้ว
	personRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "person_hash", "first_seen", "last_seen", "visit_count", "organization_id"}).
		AddRow("person-uuid", oldTime, oldTime, nil, personHash, oldTime, oldTime, 5, organizationID)

	// Mock การค้นหาบุคคลที่มีอยู่แล้ว
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "persons" WHERE person_hash = $1 AND organization_id = $2 AND "persons"."deleted_at" IS NULL ORDER BY "persons"."id" LIMIT 1`)).
		WithArgs(personHash, organizationID).
		WillReturnRows(personRows)

	// Mock การอัปเดตข้อมูลบุคคล
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "persons" SET "last_seen"=$1,"visit_count"=visit_count + 1,"updated_at"=$2 WHERE person_hash = $3 AND organization_id = $4 AND "persons"."deleted_at" IS NULL`)).
		WithArgs(now, sqlmock.AnyArg(), personHash, organizationID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// เรียกใช้ฟังก์ชันที่ต้องการทดสอบ
	person, err := service.CreateOrUpdatePerson(ctx, personLog)

	// ตรวจสอบว่าไม่มี error
	assert.NoError(t, err)
	assert.NotNil(t, person)
	assert.Equal(t, personHash, person.PersonHash)
	assert.Equal(t, organizationID, person.OrganizationID)
	
	// ตรวจสอบว่า IsNewPerson ถูกตั้งค่าเป็น false เนื่องจากเป็นบุคคลที่มีอยู่แล้ว
	assert.False(t, personLog.IsNewPerson)

	// ตรวจสอบว่าไม่มี expectations ใด ๆ ที่ไม่ได้ถูกเรียกใช้
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetPersonStats ทดสอบเมธอด GetPersonStats
func TestGetPersonStats(t *testing.T) {
	_, mock, service, cleanup := setupTest(t)
	defer cleanup()

	ctx := context.Background()
	personHash := "person123"
	organizationID := "org123"
	now := time.Now()

	// Mock ข้อมูลบุคคล
	personRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "person_hash", "first_seen", "last_seen", "visit_count", "organization_id"}).
		AddRow("person-uuid", now, now, nil, personHash, now, now, 5, organizationID)

	// Mock การค้นหาบุคคล
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "persons" WHERE person_hash = $1 AND organization_id = $2 AND "persons"."deleted_at" IS NULL ORDER BY "persons"."id" LIMIT 1`)).
		WithArgs(personHash, organizationID).
		WillReturnRows(personRows)

	// Mock การนับจำนวน PersonLog
	logCountRows := sqlmock.NewRows([]string{"count"}).AddRow(10)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "person_logs" WHERE person_hash = $1 AND organization_id = $2 AND "person_logs"."deleted_at" IS NULL`)).
		WithArgs(personHash, organizationID).
		WillReturnRows(logCountRows)

	// Mock การนับจำนวน FaceImage
	faceCountRows := sqlmock.NewRows([]string{"count"}).AddRow(3)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "face_images" WHERE person_hash = $1 AND organization_id = $2 AND "face_images"."deleted_at" IS NULL`)).
		WithArgs(personHash, organizationID).
		WillReturnRows(faceCountRows)

	// เรียกใช้ฟังก์ชันที่ต้องการทดสอบ
	stats, err := service.GetPersonStats(ctx, personHash, organizationID)

	// ตรวจสอบว่าไม่มี error
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	
	// ตรวจสอบค่าสถิติ
	assert.Equal(t, 1, stats.New) // จำนวนคนใหม่ ถูกตั้งค่าเป็น 1 เสมอ
	assert.Equal(t, 9, stats.Repeat) // จำนวนการพบซ้ำ (นับจำนวน log ทั้งหมดลบด้วย 1)
	assert.Equal(t, organizationID, stats.OrganizationID)

	// ตรวจสอบว่าไม่มี expectations ใด ๆ ที่ไม่ได้ถูกเรียกใช้
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeletePerson ทดสอบเมธอด DeletePerson
func TestDeletePerson(t *testing.T) {
	_, mock, service, cleanup := setupTest(t)
	defer cleanup()

	ctx := context.Background()
	personHash := "person123"
	organizationID := "org123"

	// Mock transaction
	mock.ExpectBegin()
	
	// Mock การลบรูปภาพใบหน้าที่เกี่ยวข้อง
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "face_images" SET "deleted_at"=$1 WHERE person_hash = $2 AND organization_id = $3 AND "face_images"."deleted_at" IS NULL`)).
		WithArgs(sqlmock.AnyArg(), personHash, organizationID).
		WillReturnResult(sqlmock.NewResult(0, 3)) // สมมติว่าลบ 3 รูปภาพ
	
	// Mock การลบบันทึก PersonLog ที่เกี่ยวข้อง
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "person_logs" SET "deleted_at"=$1 WHERE person_hash = $2 AND organization_id = $3 AND "person_logs"."deleted_at" IS NULL`)).
		WithArgs(sqlmock.AnyArg(), personHash, organizationID).
		WillReturnResult(sqlmock.NewResult(0, 10)) // สมมติว่าลบ 10 บันทึก
	
	// Mock การลบข้อมูลบุคคล
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "persons" SET "deleted_at"=$1 WHERE person_hash = $2 AND organization_id = $3 AND "persons"."deleted_at" IS NULL`)).
		WithArgs(sqlmock.AnyArg(), personHash, organizationID).
		WillReturnResult(sqlmock.NewResult(0, 1)) // สมมติว่าลบ 1 บุคคล
	
	// Mock commit transaction
	mock.ExpectCommit()

	// เรียกใช้ฟังก์ชันที่ต้องการทดสอบ
	err := service.DeletePerson(ctx, personHash, organizationID)

	// ตรวจสอบว่าไม่มี error
	assert.NoError(t, err)

	// ตรวจสอบว่าไม่มี expectations ใด ๆ ที่ไม่ได้ถูกเรียกใช้
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeletePerson_NotFound ทดสอบเมธอด DeletePerson กรณีไม่พบบุคคลที่ต้องการลบ
func TestDeletePerson_NotFound(t *testing.T) {
	_, mock, service, cleanup := setupTest(t)
	defer cleanup()

	ctx := context.Background()
	personHash := "nonexistent-person"
	organizationID := "org123"

	// Mock transaction
	mock.ExpectBegin()
	
	// Mock การลบรูปภาพใบหน้าที่เกี่ยวข้อง (ไม่พบ)
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "face_images" SET "deleted_at"=$1 WHERE person_hash = $2 AND organization_id = $3 AND "face_images"."deleted_at" IS NULL`)).
		WithArgs(sqlmock.AnyArg(), personHash, organizationID).
		WillReturnResult(sqlmock.NewResult(0, 0)) // ไม่พบรูปภาพ
	
	// Mock การลบบันทึก PersonLog ที่เกี่ยวข้อง (ไม่พบ)
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "person_logs" SET "deleted_at"=$1 WHERE person_hash = $2 AND organization_id = $3 AND "person_logs"."deleted_at" IS NULL`)).
		WithArgs(sqlmock.AnyArg(), personHash, organizationID).
		WillReturnResult(sqlmock.NewResult(0, 0)) // ไม่พบบันทึก
	
	// Mock การลบข้อมูลบุคคล (ไม่พบ)
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "persons" SET "deleted_at"=$1 WHERE person_hash = $2 AND organization_id = $3 AND "persons"."deleted_at" IS NULL`)).
		WithArgs(sqlmock.AnyArg(), personHash, organizationID).
		WillReturnResult(sqlmock.NewResult(0, 0)) // ไม่พบบุคคล
	
	// Mock rollback transaction เนื่องจากไม่พบบุคคลที่ต้องการลบ
	mock.ExpectRollback()

	// เรียกใช้ฟังก์ชันที่ต้องการทดสอบ
	err := service.DeletePerson(ctx, personHash, organizationID)

	// ตรวจสอบว่ามี error "ไม่พบข้อมูลบุคคลที่ต้องการลบ"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ไม่พบข้อมูลบุคคลที่ต้องการลบ")

	// ตรวจสอบว่าไม่มี expectations ใด ๆ ที่ไม่ได้ถูกเรียกใช้
	assert.NoError(t, mock.ExpectationsWereMet())
}