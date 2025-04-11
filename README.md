# MANTA Dashboard Microservice – API Specification  
**Project Name:** MANTA Dashboard API  
**Version:** 1.0  
**Date:** 2025-04-11

---

## 1. Objective
สร้างระบบ Microservice API ด้วยภาษา Go โดยใช้ Fiber framework เพื่อใช้ในการ:
- ดึงข้อมูลจากระบบกล้องตรวจจับคน (ผ่าน Firebase หรือฐานข้อมูล)
- วิเคราะห์และจัดทำข้อมูลในรูปแบบ summary/heatmap เพื่อใช้ใน dashboard
- ให้บริการข้อมูลในรูปแบบ JSON สำหรับแอปบนเว็บและมือถือ

---

## 2. Technology Stack

| Component | Technology |
|----------|------------|
| Language | Go |
| Framework | [Fiber](https://gofiber.io/) |
| Firebase Client | firebase-admin-go |
| Cache | Redis (optional) |
| Database | Firebase Realtime DB หรือ Firestore |
| Deployment | Docker |
| Auth | API Key / JWT (optional) |
| Frontend | Web (React, Vue), Mobile (Flutter, React Native) |

---

## 3. API Endpoints

### `GET /api/summary`
- Summary รายวัน/สัปดาห์ของจำนวนคน
- Response:
```json
{
  "date": "2025-04-10",
  "total": 138,
  "new": 94,
  "repeat": 44
}
```

---

### `GET /api/heatmap`
- แสดงช่วงเวลาที่มีคนเยอะ/น้อย
- Response:
```json
{
  "08:00": 12,
  "09:00": 24,
  "10:00": 46,
  "11:00": 30
}
```

---

### `GET /api/person-stats`
- เปรียบเทียบคนใหม่กับคนซ้ำ
- Response:
```json
{
  "new": 71,
  "repeat": 29
}
```

---

### `GET /api/logs?from=...&to=...`
- ดึง log ดิบจาก Firebase หรือ local
- Response:
```json
[
  {
    "timestamp": "2025-04-11T14:30:22",
    "person_hash": "7d82aef9",
    "camera_id": "cam_001"
  }
]
```

---

## 4. Project Structure

```
manta-dashboard-api/
├── main.go
├── routes/
│   ├── summary.go
│   ├── heatmap.go
│   └── logs.go
├── services/
│   ├── firebase.go
│   ├── stats.go
│   └── cache.go
├── models/
│   └── log.go
├── utils/
│   └── time.go
├── config/
│   └── config.go
├── go.mod
└── Dockerfile
```

---

## 5. Sample Code

### `main.go`
```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "manta-dashboard-api/routes"
)

func main() {
    app := fiber.New()

    app.Get("/api/summary", routes.GetSummary)
    app.Get("/api/heatmap", routes.GetHeatmap)
    app.Get("/api/logs", routes.GetLogs)

    app.Listen(":8080")
}
```

### `routes/summary.go`
```go
package routes

import (
    "github.com/gofiber/fiber/v2"
    "manta-dashboard-api/services"
)

func GetSummary(c *fiber.Ctx) error {
    stats := services.CalculateDailySummary()
    return c.JSON(stats)
}
```

---

## 6. Integration with Frontend

### Web App
- React, Next.js, Vue
- ใช้ Chart.js หรือ ECharts เพื่อวาดกราฟ

### Mobile App
- Flutter, React Native
- เรียก API จาก microservice เพื่อนำไปแสดงผล

---

## 7. Deployment

- ใช้ Docker build สำหรับ production:
```Dockerfile
FROM golang:1.20-alpine
WORKDIR /app
COPY . .
RUN go build -o main .
CMD ["./main"]
```

- Docker Compose (optional):
```yaml
version: '3.8'
services:
  manta-api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - FIREBASE_CREDENTIALS=/app/credentials.json
    volumes:
      - ./credentials.json:/app/credentials.json
```

---

## 8. Optional Features

- ระบบ Auth ด้วย JWT หรือ API Key
- Rate limiting สำหรับความปลอดภัย
- ใช้ Redis caching สำหรับข้อมูล summary ซ้ำ

---

## 9. Future Enhancements

- เพิ่ม websocket สำหรับส่งข้อมูลแบบ real-time
- เพิ่ม endpoint แสดงความหนาแน่นตามตำแหน่ง (heatmap บนแผนที่)
- Export CSV/PDF จาก dashboard
