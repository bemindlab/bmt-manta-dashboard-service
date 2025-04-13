# ขั้นตอนการ build
FROM golang:1.21-alpine AS builder

# ติดตั้ง dependencies
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

# ตั้งค่า working directory
WORKDIR /app

# คัดลอกไฟล์ go.mod และ go.sum ก่อน
COPY go.mod go.sum ./

# ดาวน์โหลด dependencies
RUN go mod download

# คัดลอกโค้ดทั้งหมด
COPY . .

# build แอปพลิเคชัน
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o manta-dashboard-api ./cmd/api

# ขั้นตอนการสร้าง runtime
FROM alpine:3.16

# ติดตั้ง dependencies ที่จำเป็น
RUN apk --no-cache add ca-certificates tzdata

# ตั้งค่า timezone
ENV TZ=Asia/Bangkok

# สร้าง non-root user
RUN adduser -D -g '' appuser

# สร้างโฟลเดอร์ที่จำเป็น
RUN mkdir -p /app/config
WORKDIR /app

# คัดลอกไบนารีจากขั้นตอนก่อนหน้า
COPY --from=builder /app/manta-dashboard-api .

# ตั้งค่าให้ไฟล์ทำงานได้
RUN chmod +x ./manta-dashboard-api

# ตั้งค่าไฟล์เป็นเจ้าของโดย appuser
RUN chown -R appuser:appuser /app

# สลับไปใช้ non-root user
USER appuser

# เปิดพอร์ตที่จำเป็น
EXPOSE 8080

# คำสั่งที่ใช้เริ่มต้นแอปพลิเคชัน
CMD ["./manta-dashboard-api"] 