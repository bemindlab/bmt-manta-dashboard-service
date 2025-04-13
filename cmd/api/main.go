package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bemindtech/bmt-manta-dashboard-service/config"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/api"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/db"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/firebase"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/swaggo/fiber-swagger"
	_ "github.com/bemindtech/bmt-manta-dashboard-service/docs" // ส่งออก docs
)

// @title BMT Manta Dashboard Service API
// @version 1.0
// @description API for BMT Manta Dashboard Service
// @host localhost:8080
// @BasePath /api
// @schemes http https
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
func main() {
	// โหลดการตั้งค่า
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("ไม่สามารถโหลดการตั้งค่า: %v", err)
	}

	// เชื่อมต่อกับ PostgreSQL
	postgres, err := db.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("ไม่สามารถเชื่อมต่อกับ PostgreSQL: %v", err)
	}
	defer postgres.Close()

	// สร้างตารางที่จำเป็น
	if err := postgres.InitTables(); err != nil {
		log.Fatalf("ไม่สามารถสร้างตาราง: %v", err)
	}

	// เชื่อมต่อกับ Redis (ถ้ามี)
	var redisClient *db.RedisClient
	if cfg.RedisHost != "" {
		redisClient, err = db.NewRedisClient(cfg)
		if err != nil {
			log.Printf("ไม่สามารถเชื่อมต่อกับ Redis: %v", err)
			log.Println("ระบบจะทำงานโดยไม่มี Redis cache")
		} else {
			defer redisClient.Close()
			log.Println("เชื่อมต่อกับ Redis สำเร็จ")
		}
	} else {
		log.Println("ไม่ได้กำหนดค่า Redis ระบบจะทำงานโดยไม่มี Redis cache")
	}

	// เชื่อมต่อกับ Firebase
	firebaseClient, err := firebase.NewFirebaseClient(cfg)
	if err != nil {
		log.Printf("ไม่สามารถเชื่อมต่อกับ Firebase: %v", err)
		log.Println("ระบบจะทำงานโดยไม่มีการซิงค์ข้อมูลจาก Firebase")
	} else {
		log.Println("เชื่อมต่อกับ Firebase สำเร็จ")
	}

	// สร้าง service
	statsService := services.NewStatsService(postgres, redisClient)

	// เริ่มต้นการซิงค์ข้อมูลจาก Firebase (ถ้ามี)
	if firebaseClient != nil {
		syncService := services.NewSyncService(postgres, firebaseClient)
		ctx := context.Background()

		// ซิงค์ข้อมูลเก่า
		if err := syncService.BackfillLogs(ctx, "logs", 1000); err != nil {
			log.Printf("ไม่สามารถซิงค์ข้อมูลเก่าจาก Firebase: %v", err)
		}

		// เริ่มการรับฟังข้อมูลใหม่
		if err := syncService.StartSync(ctx, "logs"); err != nil {
			log.Printf("ไม่สามารถเริ่มการซิงค์ข้อมูลจาก Firebase: %v", err)
		}
	}

	// สร้างแอปพลิเคชัน Fiber
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// ส่งข้อผิดพลาดในรูปแบบ JSON
			code := fiber.StatusInternalServerError

			// ตรวจสอบว่าเป็น fiber.Error หรือไม่
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// เพิ่ม Swagger route
	app.Get("/docs/*", fiberSwagger.WrapHandler)

	// ตั้งค่าเส้นทาง API
	api.SetupRoutes(app, cfg, postgres, statsService)

	// สร้าง channel สำหรับรับสัญญาณ interrupt
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	// เริ่มต้น server ในอีก goroutine
	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("เกิดข้อผิดพลาดในการเริ่มต้น server: %v", err)
		}
	}()

	log.Printf("เริ่มต้น server ที่พอร์ต %s\n", cfg.Port)

	// รอสัญญาณ shutdown
	<-shutdownChan
	log.Println("กำลังปิด server...")

	// ให้เวลาในการปิดการเชื่อมต่อ
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ปิด server
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatalf("เกิดข้อผิดพลาดในการปิด server: %v", err)
	}

	log.Println("ปิด server สำเร็จ")
} 