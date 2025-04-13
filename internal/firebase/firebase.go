package firebase

import (
	"context"
	"fmt"
	"log"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"google.golang.org/api/option"

	"github.com/bemindtech/bmt-manta-dashboard-service/config"
)

// FirebaseClient เป็นโครงสร้างที่เก็บการเชื่อมต่อกับ Firebase
type FirebaseClient struct {
	App    *firebase.App
	DB     *db.Client
	Config *config.Config
}

// NewFirebaseClient สร้างและเชื่อมต่อกับ Firebase
func NewFirebaseClient(cfg *config.Config) (*FirebaseClient, error) {
	// ตรวจสอบว่ามีไฟล์ credentials
	if cfg.FirebaseCredentialsFile == "" {
		return nil, fmt.Errorf("ไม่พบไฟล์ Firebase credentials")
	}

	// สร้างตัวเลือกการตั้งค่า Firebase
	opt := option.WithCredentialsFile(cfg.FirebaseCredentialsFile)
	
	// กำหนดค่า Firebase
	config := &firebase.Config{
		ProjectID:   cfg.FirebaseProjectID,
		DatabaseURL: fmt.Sprintf("https://%s.firebaseio.com", cfg.FirebaseProjectID),
	}

	// สร้าง Firebase App
	app, err := firebase.NewApp(context.Background(), config, opt)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถสร้าง Firebase app: %w", err)
	}

	// สร้าง Realtime Database client
	dbClient, err := app.Database(context.Background())
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถสร้าง Firebase Realtime Database client: %w", err)
	}

	log.Println("เชื่อมต่อกับ Firebase Realtime Database สำเร็จ")

	return &FirebaseClient{
		App:    app,
		DB:     dbClient,
		Config: cfg,
	}, nil
}

// ListenForNewLogs เริ่มการรับฟังข้อมูลใหม่จาก Firebase และส่งไปยัง channel
func (fc *FirebaseClient) ListenForNewLogs(ctx context.Context, logsPath string) (<-chan map[string]interface{}, error) {
	logsChan := make(chan map[string]interface{})
	
	// เริ่ม goroutine สำหรับ polling
	go func() {
		defer close(logsChan)

		var lastTimestamp int64
		ticker := time.NewTicker(5 * time.Second) // polling ทุก 5 วินาที
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// ดึงข้อมูลใหม่
				ref := fc.DB.NewRef(logsPath).OrderByChild("timestamp").StartAt(lastTimestamp + 1)
				var data map[string]map[string]interface{}
				if err := ref.Get(ctx, &data); err != nil {
					log.Printf("ไม่สามารถดึงข้อมูลใหม่: %v", err)
					continue
				}

				// ส่งข้อมูลใหม่ไปยัง channel
				for _, value := range data {
					if ts, ok := value["timestamp"].(float64); ok {
						if int64(ts) > lastTimestamp {
							lastTimestamp = int64(ts)
						}
					}
					logsChan <- value
				}
			}
		}
	}()

	return logsChan, nil
}

// GetRecentLogs ดึงข้อมูล logs ล่าสุดจาก Firebase
func (fc *FirebaseClient) GetRecentLogs(ctx context.Context, logsPath string, limit int) ([]map[string]interface{}, error) {
	ref := fc.DB.NewRef(logsPath).OrderByChild("timestamp").LimitToLast(limit)
	
	var data map[string]map[string]interface{}
	if err := ref.Get(ctx, &data); err != nil {
		return nil, fmt.Errorf("ไม่สามารถดึงข้อมูล logs: %w", err)
	}

	// แปลงจาก map เป็น slice
	logs := make([]map[string]interface{}, 0, len(data))
	for key, value := range data {
		value["id"] = key
		logs = append(logs, value)
	}

	return logs, nil
} 