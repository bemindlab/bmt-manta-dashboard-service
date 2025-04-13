package api

import (
	"github.com/bemindtech/bmt-manta-dashboard-service/config"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/api/handlers"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/api/middleware"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/db"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/services"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// SetupRoutes ตั้งค่าเส้นทาง API ทั้งหมด
func SetupRoutes(app *fiber.App, cfg *config.Config, postgres *db.PostgresDB, statsService *services.StatsService) {
	// ใช้ middleware พื้นฐาน
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000,http://localhost:8080", // อนุญาตเฉพาะ origins ที่กำหนด
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-API-Key",
		ExposeHeaders:    "Content-Length",
		AllowCredentials: false, // ปิดการใช้งาน credentials เพื่อความปลอดภัย
		MaxAge:           3600,  // cache preflight requests for 1 hour
	}))

	// สร้าง services
	organizationService := services.NewOrganizationService(postgres)
	cameraService := services.NewCameraService(postgres)
	storageService, err := storage.NewStorageService(cfg)
	if err != nil {
		app.Use(func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "ไม่สามารถเริ่มต้นบริการจัดเก็บข้อมูล: " + err.Error(),
			})
		})
		return
	}
	faceService := services.NewFaceService(postgres, storageService)
	personService := services.NewPersonService(postgres)

	// สร้าง handlers
	summaryHandler := handlers.NewSummaryHandler(statsService)
	logsHandler := handlers.NewLogsHandler(statsService)
	organizationHandler := handlers.NewOrganizationHandler(organizationService)
	cameraHandler := handlers.NewCameraHandler(cameraService)
	faceHandler := handlers.NewFaceHandler(faceService)
	personHandler := handlers.NewPersonHandler(personService)

	// กำหนดเส้นทาง API
	api := app.Group("/api")

	// สร้าง middleware สำหรับ API key ที่มีการจัดเก็บในฐานข้อมูล
	apiKeyWithOrgMiddleware := middleware.NewAPIKeyWithOrgMiddleware(postgres)

	// สร้าง middleware สำหรับตรวจสอบการเข้าถึงข้อมูลขององค์กร
	orgAuthMiddleware := middleware.NewOrganizationAuthMiddleware()

	// เส้นทางที่ต้องการ API Key
	apiKeyProtected := api.Group("/", apiKeyWithOrgMiddleware)

	// ตั้งค่าเส้นทาง API สำหรับข้อมูลสรุป
	apiKeyProtected.Get("/summary", summaryHandler.GetDailySummary)
	apiKeyProtected.Get("/heatmap", summaryHandler.GetHeatmap)
	apiKeyProtected.Get("/person-stats", summaryHandler.GetPersonStats)
	apiKeyProtected.Get("/logs", logsHandler.GetLogs)

	// ตั้งค่าเส้นทาง API สำหรับจัดการองค์กร
	organizations := apiKeyProtected.Group("/organizations")
	organizations.Get("/", organizationHandler.GetOrganizations)
	organizations.Post("/", organizationHandler.CreateOrganization)
	organizations.Get("/:id", orgAuthMiddleware, organizationHandler.GetOrganization)
	organizations.Put("/:id", orgAuthMiddleware, organizationHandler.UpdateOrganization)
	organizations.Delete("/:id", orgAuthMiddleware, organizationHandler.DeleteOrganization)

	// ตั้งค่าเส้นทาง API สำหรับจัดการกล้อง
	cameras := apiKeyProtected.Group("/cameras")
	cameras.Get("/", cameraHandler.GetCameras)
	cameras.Post("/", cameraHandler.CreateCamera)
	cameras.Get("/:id", cameraHandler.GetCamera)
	cameras.Put("/:id", cameraHandler.UpdateCamera)
	cameras.Delete("/:id", cameraHandler.DeleteCamera)

	// ตั้งค่าเส้นทาง API สำหรับจัดการรูปภาพใบหน้า
	faces := apiKeyProtected.Group("/faces")
	faces.Post("/", faceHandler.UploadFaceImage)
	faces.Get("/:person_hash", faceHandler.GetFaceImages)
	faces.Delete("/image/:id", faceHandler.DeleteFaceImage)
	
	// ตั้งค่าเส้นทาง API สำหรับจัดการข้อมูลบุคคล
	persons := apiKeyProtected.Group("/persons")
	persons.Get("/", personHandler.ListPersons)
	persons.Get("/:person_hash", personHandler.GetPerson)
	persons.Get("/:person_hash/stats", personHandler.GetPersonStats)
	persons.Delete("/:person_hash", personHandler.DeletePerson)

	// เส้นทางสำหรับตรวจสอบสถานะ API
	// @Summary Check API health
	// @Description Returns the health status of the API
	// @Tags info
	// @Produce json
	// @Success 200 {object} map[string]string
	// @Router /api/health [get]
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
		})
	})

	// เส้นทางสำหรับข้อมูล API
	// @Summary Get API information
	// @Description Returns the name and version of the API
	// @Tags info
	// @Produce json
	// @Success 200 {object} map[string]string
	// @Router /api [get]
	api.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"name":    "MANTA Dashboard API",
			"version": "0.1",
		})
	})
}