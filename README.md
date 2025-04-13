# MANTA Dashboard Microservice – API Specification

**Project Name:** MANTA Dashboard API  
**Version:** 0.1
**Date:** 2025-04-11

---

## 1. Objective

สร้างระบบ Microservice API ด้วยภาษา Go โดยใช้ Fiber framework เพื่อใช้ในการ:

- ดึงข้อมูลจากระบบกล้องตรวจจับคน (ผ่าน Firebase หรือฐานข้อมูล)
- วิเคราะห์และจัดทำข้อมูลในรูปแบบ summary/heatmap เพื่อใช้ใน dashboard
- ให้บริการข้อมูลในรูปแบบ JSON สำหรับแอปบนเว็บและมือถือ

---

## 2. Technology Stack

| Component       | Technology                                       |
| --------------- | ------------------------------------------------ |
| Language        | Go                                               |
| Framework       | [Fiber](https://gofiber.io/)                     |
| Firebase Client | firebase-admin-go                                |
| Cache           | Redis (optional)                                 |
| Realtime DB     | Firebase Realtime DB หรือ Firestore              |
| SQL Database    | PostgreSQL                                       |
| Deployment      | Docker                                           |
| Auth            | API Key / JWT (optional)                         |
| Frontend        | Web (React, Vue), Mobile (Flutter, React Native) |

---

## 3. วิธีการใช้งาน

### การติดตั้งและการใช้งาน

#### สิ่งที่ต้องมีก่อนการติดตั้ง

1. Git
2. Docker และ Docker Compose
3. Go (เฉพาะกรณีที่ต้องการพัฒนาต่อ)
4. Firebase project และไฟล์ credentials

#### การติดตั้ง

1. โคลนโปรเจคนี้

```bash
git clone https://github.com/bemindtech/bmt-manta-dashboard-service.git
cd bmt-manta-dashboard-service
```

2. ตั้งค่าไฟล์ Firebase credentials

```bash
mkdir -p config
# วางไฟล์ firebase-credentials.json ใน ./config/
```

3. สร้างไฟล์ .env จาก .env.example

```bash
cp .env.example .env
# แก้ไขการตั้งค่าใน .env ตามที่ต้องการ
```

4. รันระบบด้วย Docker Compose

```bash
docker-compose up -d
```

5. ตรวจสอบว่าระบบทำงานได้ถูกต้อง

```bash
curl http://localhost:8080/api/health
```

### การพัฒนาต่อ

1. ติดตั้ง dependencies

```bash
go mod download
```

2. รันระบบในโหมดพัฒนา

```bash
go run cmd/api/main.go
```

3. รันการทดสอบ

```bash
go test ./internal/...
```

4. สร้าง build สำหรับนำไปใช้งาน

```bash
go build -o manta-dashboard-api ./cmd/api
```

---

## 4. API Documentation

The service provides a Swagger UI for interactive API documentation. When the service is running, you can access the Swagger documentation at:

```
http://localhost:8080/docs/index.html
```

### Available Endpoints

#### Basic Information
- **GET /api** - Get basic API information
- **GET /api/health** - Check API health status

#### Statistics and Logs
- **GET /api/logs** - Retrieve person detection logs with filtering options
- **GET /api/summary** - Get daily summary statistics
- **GET /api/heatmap** - Get heatmap data by time period
- **GET /api/person-stats** - Get new vs. returning person statistics

#### Organizations
- **GET /api/organizations** - List all organizations
- **POST /api/organizations** - Create a new organization
- **GET /api/organizations/:id** - Get organization details
- **PUT /api/organizations/:id** - Update organization details
- **DELETE /api/organizations/:id** - Delete an organization

#### Cameras
- **GET /api/cameras** - List all cameras
- **POST /api/cameras** - Create a new camera
- **GET /api/cameras/:id** - Get camera details
- **PUT /api/cameras/:id** - Update camera details
- **DELETE /api/cameras/:id** - Delete a camera

#### Face Images
- **POST /api/faces** - Upload a face image
- **GET /api/faces/:person_hash** - Get all face images for a person
- **DELETE /api/faces/image/:id** - Delete a face image

#### Persons
- **GET /api/persons** - List all persons
- **GET /api/persons/:person_hash** - Get person details
- **GET /api/persons/:person_hash/stats** - Get person statistics
- **DELETE /api/persons/:person_hash** - Delete a person

### Authentication

All endpoints except `/api` and `/api/health` require API key authentication. 
The API key must be provided in the `X-API-Key` header.

### Endpoints Details

#### `GET /api/summary`

- Summary รายวัน/สัปดาห์ของจำนวนคน
- Parameters:

  - `date`: วันที่ต้องการดูข้อมูล (รูปแบบ YYYY-MM-DD) ถ้าไม่ระบุจะใช้วันปัจจุบัน

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

#### `GET /api/heatmap`

- แสดงช่วงเวลาที่มีคนเยอะ/น้อย
- Parameters:

  - `date`: วันที่ต้องการดูข้อมูล (รูปแบบ YYYY-MM-DD) ถ้าไม่ระบุจะใช้วันปัจจุบัน

- Response:

```json
[
  {
    "hour": "08:00",
    "count": 12
  },
  {
    "hour": "09:00",
    "count": 24
  },
  {
    "hour": "10:00",
    "count": 46
  },
  {
    "hour": "11:00",
    "count": 30
  }
]
```

---

#### `GET /api/person-stats`

- เปรียบเทียบคนใหม่กับคนซ้ำ
- Parameters:

  - `date`: วันที่ต้องการดูข้อมูล (รูปแบบ YYYY-MM-DD) ถ้าไม่ระบุจะใช้วันปัจจุบัน

- Response:

```json
{
  "new": 71,
  "repeat": 29
}
```

---

#### `GET /api/logs`

- ดึง log ดิบจาก Firebase หรือ local
- Parameters:

  - `from`: เวลาเริ่มต้น (รูปแบบ YYYY-MM-DDTHH:MM:SS)
  - `to`: เวลาสิ้นสุด (รูปแบบ YYYY-MM-DDTHH:MM:SS)
  - `camera_id`: รหัสกล้อง (optional)
  - `person_id`: รหัสบุคคล (optional)
  - `page`: หน้าที่ต้องการ (เริ่มต้นที่ 1)
  - `page_size`: จำนวนรายการต่อหน้า (เริ่มต้นที่ 10, สูงสุด 100)

- Response:

```json
{
  "data": [
    {
      "id": "abcd1234",
      "timestamp": "2025-04-11T14:30:22Z",
      "person_hash": "7d82aef9",
      "camera_id": "cam_001",
      "is_new_person": false,
      "created_at": "2025-04-11T14:30:25Z"
    }
  ],
  "pagination": {
    "total": 138,
    "page": 1,
    "page_size": 10,
    "total_page": 14
  }
}
```

---

## 5. การจัดการความปลอดภัย

### Authentication

API นี้ใช้ API Key ในการยืนยันตัวตน ซึ่งสามารถส่งได้ 2 รูปแบบ:

1. HTTP Header: `X-API-Key: your-api-key`
2. Query parameter: `?api_key=your-api-key`

### Rate Limiting

เพื่อป้องกันการใช้งาน API มากเกินไป ระบบจะจำกัดจำนวนคำขอตามการตั้งค่าใน .env ซึ่งระบบจะส่ง header ต่อไปนี้กลับมาด้วย:

- `X-RateLimit-Limit`: จำนวนคำขอสูงสุดต่อช่วงเวลา
- `X-RateLimit-Remaining`: จำนวนคำขอที่เหลือในช่วงเวลานี้
- `X-RateLimit-Reset`: เวลาที่จะรีเซ็ตการนับ (Unix timestamp)

## 6. การติดตั้งบนระบบ Production

1. แก้ไขการตั้งค่าความปลอดภัยใน .env สำหรับระบบ Production

   - เปลี่ยน API_KEY เป็นค่าที่ซับซ้อน
   - เปลี่ยน JWT_SECRET เป็นค่าที่ซับซ้อน
   - ตั้งค่า PostgreSQL และ Redis ให้มีรหัสผ่านที่ปลอดภัย

2. รันระบบด้วย Docker Compose

```bash
docker-compose up -d
```

3. ตรวจสอบว่าระบบทำงานได้ถูกต้อง

```bash
curl http://your-server:8080/api/health
```

---

## 7. Testing

โปรเจคนี้มีการทดสอบครอบคลุมทั้งในส่วนของ services และ handlers โดยใช้ Go testing framework และ testify

### การรันทดสอบ

รันทดสอบทั้งหมด:

```bash
go test ./internal/...
```

รันทดสอบและแสดงรายละเอียด:

```bash
go test -v ./internal/...
```

รันทดสอบเฉพาะ service:

```bash
go test ./internal/services/...
```

รันทดสอบเฉพาะ handler:

```bash
go test ./internal/api/handlers/...
```

## 8. Future Enhancements

- เพิ่ม websocket สำหรับส่งข้อมูลแบบ real-time
- เพิ่ม endpoint แสดงความหนาแน่นตามตำแหน่ง (heatmap บนแผนที่)
- Export CSV/PDF จาก dashboard
- เพิ่มเติมการทดสอบที่ครอบคลุมมากขึ้น
- เพิ่มระบบ CI/CD และ automated testing
