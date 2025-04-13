package services

import (
	"context"
	"fmt"
	"time"

	"github.com/bemindtech/bmt-manta-dashboard-service/internal/db"
	"github.com/bemindtech/bmt-manta-dashboard-service/internal/models"
)

// StatsService เป็นโครงสร้างสำหรับการวิเคราะห์ข้อมูล
type StatsService struct {
	DB    *db.PostgresDB
	Redis *db.RedisClient
}

// NewStatsService สร้าง StatsService ใหม่
func NewStatsService(postgres *db.PostgresDB, redis *db.RedisClient) *StatsService {
	return &StatsService{
		DB:    postgres,
		Redis: redis,
	}
}

// GetDailySummary ดึงข้อมูลสรุปรายวัน
func (s *StatsService) GetDailySummary(ctx context.Context, date string, organizationID string) (*models.DailySummary, error) {
	// ตรวจสอบใน Redis cache ก่อน
	cacheKey := fmt.Sprintf("daily_summary:%s:%s", organizationID, date)
	var summary models.DailySummary
	
	// ดึงข้อมูลจาก cache
	found, err := s.Redis.Get(ctx, cacheKey, &summary)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถดึงข้อมูลจาก Redis: %w", err)
	}

	// ถ้าพบใน cache ให้ใช้เลย
	if found {
		return &summary, nil
	}

	// ถ้าไม่พบใน cache ต้องดึงจากฐานข้อมูล
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("รูปแบบวันที่ไม่ถูกต้อง: %w", err)
	}

	// สร้างช่วงเวลาสำหรับวันนั้น
	startOfDay := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, parsedDate.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// ใช้ GORM ในการดึงข้อมูลและประมวลผล
	var total int64
	var newCount int64
	var repeatCount int64

	// ดึงจำนวนรายการทั้งหมด
	if err := s.DB.DB.WithContext(ctx).Model(&models.PersonLog{}).
		Where("timestamp >= ? AND timestamp < ? AND organization_id = ?", startOfDay, endOfDay, organizationID).
		Count(&total).Error; err != nil {
		return nil, fmt.Errorf("ไม่สามารถดึงข้อมูลสรุปรายวัน (total): %w", err)
	}

	// ดึงจำนวนคนใหม่
	if err := s.DB.DB.WithContext(ctx).Model(&models.PersonLog{}).
		Where("timestamp >= ? AND timestamp < ? AND organization_id = ? AND is_new_person = ?", startOfDay, endOfDay, organizationID, true).
		Count(&newCount).Error; err != nil {
		return nil, fmt.Errorf("ไม่สามารถดึงข้อมูลสรุปรายวัน (new count): %w", err)
	}

	// ดึงจำนวนคนซ้ำ
	if err := s.DB.DB.WithContext(ctx).Model(&models.PersonLog{}).
		Where("timestamp >= ? AND timestamp < ? AND organization_id = ? AND is_new_person = ?", startOfDay, endOfDay, organizationID, false).
		Count(&repeatCount).Error; err != nil {
		return nil, fmt.Errorf("ไม่สามารถดึงข้อมูลสรุปรายวัน (repeat count): %w", err)
	}

	// สร้างข้อมูลสรุป
	summary = models.DailySummary{
		Date:   date,
		Total:  int(total),
		New:    int(newCount),
		Repeat: int(repeatCount),
	}

	// บันทึกใน Redis
	if err := s.Redis.Set(ctx, cacheKey, summary, 1*time.Hour); err != nil {
		return nil, fmt.Errorf("ไม่สามารถบันทึกข้อมูลใน Redis: %w", err)
	}

	return &summary, nil
}

// GetHeatmapData ดึงข้อมูลความหนาแน่นตามช่วงเวลา
func (s *StatsService) GetHeatmapData(ctx context.Context, date string, organizationID string) ([]models.HeatmapData, error) {
	// ตรวจสอบใน Redis cache ก่อน
	cacheKey := fmt.Sprintf("heatmap:%s:%s", organizationID, date)
	var heatmap []models.HeatmapData
	
	// ดึงข้อมูลจาก cache
	found, err := s.Redis.Get(ctx, cacheKey, &heatmap)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถดึงข้อมูลจาก Redis: %w", err)
	}

	// ถ้าพบใน cache ให้ใช้เลย
	if found {
		return heatmap, nil
	}

	// ถ้าไม่พบใน cache ต้องดึงจากฐานข้อมูล
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("รูปแบบวันที่ไม่ถูกต้อง: %w", err)
	}

	// สร้างช่วงเวลาสำหรับวันนั้น
	startOfDay := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, parsedDate.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// ยังคงต้องใช้ Raw SQL เนื่องจาก GORM ไม่สนับสนุนฟังก์ชัน TO_CHAR โดยตรง
	// แต่เราจะใช้ GORM Raw method แทนการใช้ SQL driver โดยตรง
	var result []struct {
		Hour  string
		Count int
	}

	// ดึงข้อมูลโดยใช้ GORM Raw
	if err := s.DB.DB.WithContext(ctx).Raw(`
		SELECT 
			TO_CHAR(timestamp, 'HH24:00') as hour,
			COUNT(*) as count
		FROM person_logs
		WHERE timestamp >= ? AND timestamp < ? AND organization_id = ?
		GROUP BY hour
		ORDER BY hour
	`, startOfDay, endOfDay, organizationID).Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("ไม่สามารถดึงข้อมูล heatmap: %w", err)
	}

	// แปลงข้อมูลเป็น HeatmapData
	heatmap = make([]models.HeatmapData, len(result))
	for i, item := range result {
		heatmap[i] = models.HeatmapData{
			Hour:  item.Hour,
			Count: item.Count,
		}
	}

	// บันทึกใน Redis
	if err := s.Redis.Set(ctx, cacheKey, heatmap, 1*time.Hour); err != nil {
		return nil, fmt.Errorf("ไม่สามารถบันทึกข้อมูลใน Redis: %w", err)
	}

	return heatmap, nil
}

// GetPersonStats ดึงข้อมูลสถิติคนใหม่และคนซ้ำ
func (s *StatsService) GetPersonStats(ctx context.Context, date string, organizationID string) (*models.PersonStats, error) {
	// ตรวจสอบใน Redis cache ก่อน
	cacheKey := fmt.Sprintf("person_stats:%s:%s", organizationID, date)
	var stats models.PersonStats
	
	// ดึงข้อมูลจาก cache
	found, err := s.Redis.Get(ctx, cacheKey, &stats)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถดึงข้อมูลจาก Redis: %w", err)
	}

	// ถ้าพบใน cache ให้ใช้เลย
	if found {
		return &stats, nil
	}

	// ถ้าไม่พบใน cache ต้องดึงจากฐานข้อมูล
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("รูปแบบวันที่ไม่ถูกต้อง: %w", err)
	}

	// สร้างช่วงเวลาสำหรับวันนั้น
	startOfDay := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, parsedDate.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// ใช้ GORM สำหรับการดึงข้อมูล
	var newCount int64
	var repeatCount int64

	// นับจำนวนคนใหม่
	if err := s.DB.DB.WithContext(ctx).Model(&models.PersonLog{}).
		Where("timestamp >= ? AND timestamp < ? AND organization_id = ? AND is_new_person = ?", 
			startOfDay, endOfDay, organizationID, true).
		Count(&newCount).Error; err != nil {
		return nil, fmt.Errorf("ไม่สามารถดึงข้อมูลสถิติคนใหม่: %w", err)
	}

	// นับจำนวนคนซ้ำ
	if err := s.DB.DB.WithContext(ctx).Model(&models.PersonLog{}).
		Where("timestamp >= ? AND timestamp < ? AND organization_id = ? AND is_new_person = ?", 
			startOfDay, endOfDay, organizationID, false).
		Count(&repeatCount).Error; err != nil {
		return nil, fmt.Errorf("ไม่สามารถดึงข้อมูลสถิติคนซ้ำ: %w", err)
	}

	// สร้างข้อมูลสถิติ
	stats = models.PersonStats{
		New:    int(newCount),
		Repeat: int(repeatCount),
	}

	// บันทึกใน Redis
	if err := s.Redis.Set(ctx, cacheKey, stats, 1*time.Hour); err != nil {
		return nil, fmt.Errorf("ไม่สามารถบันทึกข้อมูลใน Redis: %w", err)
	}

	return &stats, nil
}

// GetLogs ดึงข้อมูล logs ตามเงื่อนไข
func (s *StatsService) GetLogs(ctx context.Context, filter models.LogFilter) ([]models.PersonLog, *models.Pagination, error) {
	// จัดการ pagination
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 10
	}

	// คำนวณ offset
	offset := (filter.Page - 1) * filter.PageSize

	// สร้าง query ด้วย GORM
	query := s.DB.DB.WithContext(ctx).Model(&models.PersonLog{})

	// เพิ่มเงื่อนไขการค้นหา
	if !filter.From.IsZero() {
		query = query.Where("timestamp >= ?", filter.From)
	}
	if !filter.To.IsZero() {
		query = query.Where("timestamp < ?", filter.To)
	}
	if filter.CameraID != "" {
		query = query.Where("camera_id = ?", filter.CameraID)
	}
	if filter.PersonID != "" {
		query = query.Where("person_hash = ?", filter.PersonID)
	}
	if filter.OrganizationID != "" {
		query = query.Where("organization_id = ?", filter.OrganizationID)
	}

	// นับจำนวนรายการทั้งหมด
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, nil, fmt.Errorf("ไม่สามารถนับจำนวนรายการทั้งหมด: %w", err)
	}

	// ดึงข้อมูลพร้อม pagination
	var logs []models.PersonLog
	if err := query.Order("timestamp DESC").
		Limit(filter.PageSize).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, nil, fmt.Errorf("ไม่สามารถดึงข้อมูล logs: %w", err)
	}

	// คำนวณจำนวนหน้าทั้งหมด
	totalPage := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPage++
	}

	// สร้างข้อมูล pagination
	pagination := &models.Pagination{
		Total:     int(total),
		Page:      filter.Page,
		PageSize:  filter.PageSize,
		TotalPage: totalPage,
	}

	return logs, pagination, nil
} 