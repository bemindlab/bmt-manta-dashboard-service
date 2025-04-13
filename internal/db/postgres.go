package db

import (
	"fmt"
	"log"

	"github.com/bemindtech/bmt-manta-dashboard-service/config"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/models"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgresDB is a struct that holds the GORM database connection
type PostgresDB struct {
	DB *gorm.DB
}

// NewPostgresDB creates and connects to a PostgreSQL database using GORM
func NewPostgresDB(cfg *config.Config) (*PostgresDB, error) {
	// Create DSN (Data Source Name)
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.PostgresHost,
		cfg.PostgresPort,
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresDB,
		cfg.PostgresSSLMode,
	)

	// Configure GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	// Connect to the database
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถเชื่อมต่อกับฐานข้อมูล PostgreSQL: %w", err)
	}

	// Get the underlying SQL DB to verify connectivity
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถเข้าถึง SQL DB: %w", err)
	}

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ไม่สามารถ ping ฐานข้อมูล PostgreSQL: %w", err)
	}

	log.Println("เชื่อมต่อกับฐานข้อมูล PostgreSQL สำเร็จ")

	return &PostgresDB{DB: db}, nil
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	sqlDB, err := p.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// InitTables creates all required tables using GORM auto migration
func (p *PostgresDB) InitTables() error {
	// Auto migrate all models - GORM will create tables, indexes, etc.
	err := p.DB.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.APIKey{},
		&models.Camera{},
		&models.PersonLog{},
		&models.FaceImage{},
		&models.Person{},
	)
	if err != nil {
		return fmt.Errorf("ไม่สามารถ migrate ฐานข้อมูล: %w", err)
	}

	log.Println("สร้างตารางทั้งหมดสำเร็จ (ถ้ายังไม่มี)")

	// Check if we need to create a default organization and API key
	var count int64
	p.DB.Model(&models.Organization{}).Count(&count)

	if count == 0 {
		// Create a default organization
		defaultOrg := models.Organization{
			Base: models.Base{
				ID: uuid.New().String(),
			},
			Name:        "Default Organization",
			Description: "Default organization created during initialization",
		}

		if err := p.DB.Create(&defaultOrg).Error; err != nil {
			return fmt.Errorf("ไม่สามารถสร้างองค์กรเริ่มต้น: %w", err)
		}

		// Create a default API key for the organization
		defaultAPIKey := models.APIKey{
			Base: models.Base{
				ID: uuid.New().String(),
			},
			KeyValue:      uuid.New().String(),
			Description:   "Default API key for default organization",
			OrganizationID: defaultOrg.ID,
		}

		if err := p.DB.Create(&defaultAPIKey).Error; err != nil {
			return fmt.Errorf("ไม่สามารถสร้าง API key เริ่มต้น: %w", err)
		}

		log.Printf("สร้างองค์กรเริ่มต้นและ API key สำเร็จ (API Key: %s)", defaultAPIKey.KeyValue)
	}

	return nil
}