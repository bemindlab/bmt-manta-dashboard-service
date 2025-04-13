package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/bemindtech/bmt-manta-dashboard-service/config"
	"github.com/go-redis/redis/v8"
)

// RedisClient เป็นโครงสร้างที่เก็บการเชื่อมต่อกับ Redis
type RedisClient struct {
	Client *redis.Client
	Config *config.Config
}

// NewRedisClient สร้างและเชื่อมต่อกับ Redis
func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// ทดสอบการเชื่อมต่อ
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถเชื่อมต่อกับ Redis: %w", err)
	}

	log.Println("เชื่อมต่อกับ Redis สำเร็จ")

	return &RedisClient{
		Client: client,
		Config: cfg,
	}, nil
}

// Close ปิดการเชื่อมต่อกับ Redis
func (r *RedisClient) Close() error {
	return r.Client.Close()
}

// Set เก็บข้อมูลใน Redis โดยมีเวลาหมดอายุ
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// แปลงข้อมูลเป็น JSON
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("ไม่สามารถแปลงข้อมูลเป็น JSON: %w", err)
	}

	// เก็บข้อมูลใน Redis
	return r.Client.Set(ctx, key, jsonData, expiration).Err()
}

// Get ดึงข้อมูลจาก Redis
func (r *RedisClient) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	// ดึงข้อมูลจาก Redis
	data, err := r.Client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// ไม่พบข้อมูล
			return false, nil
		}
		return false, fmt.Errorf("ไม่สามารถดึงข้อมูลจาก Redis: %w", err)
	}

	// แปลงข้อมูล JSON กลับเป็นโครงสร้าง
	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return false, fmt.Errorf("ไม่สามารถแปลงข้อมูล JSON กลับเป็นโครงสร้าง: %w", err)
	}

	return true, nil
}

// Delete ลบข้อมูลจาก Redis
func (r *RedisClient) Delete(ctx context.Context, key string) error {
	return r.Client.Del(ctx, key).Err()
}

// FlushAll ลบข้อมูลทั้งหมดใน Redis
func (r *RedisClient) FlushAll(ctx context.Context) error {
	return r.Client.FlushAll(ctx).Err()
} 