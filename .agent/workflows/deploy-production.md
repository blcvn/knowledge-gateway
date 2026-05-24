---
description: Deploy VNP Design Platform lên Production (b4.openledger.vn + b6.openledger.vn)
---

# Quy Trình Deploy Production — VNP Design Platform

**App Server:** 172.20.2.37 | **Gateway:** 172.20.2.16 (103.67.184.32) | **Domains:** b4.openledger.vn (OpenUI) + b6.openledger.vn (Preview)

---

## Kiến Trúc Triển Khai

```
[User] → b4.openledger.vn → [nginx :443 @ 172.20.2.16]
                                ├── / → openui-frontend :8089 @ 172.20.2.37
                                ├── /api/openui/* → openui-backend :18080 @ 172.20.2.37
                                └── /api/* → preview-all :18095 @ 172.20.2.37

[User] → b6.openledger.vn → [nginx :443 @ 172.20.2.16]
                                └── / → preview-frontend :8088 @ 172.20.2.37
                                        └── /api/* → preview-backend :18090 (internal)
```

---

## Điều Kiện Tiên Quyết

Trước khi bắt đầu, đảm bảo:
- Đã cài Docker + Docker Buildx trên MacOS local
- SSH access vào cả 2 server (`ssh ubuntu@172.20.2.37` và `ssh ubuntu@172.20.2.16`)
- DNS `b4.openledger.vn` và `b6.openledger.vn` đã trỏ về `103.67.184.32`

---

## A. DEPLOY LẦN ĐẦU (First-Time Setup)

### Bước 1: Build Go Binaries + Frontend

```bash
cd deploy/production
./deploy.sh build
```

### Bước 2: Cấu Hình `.env.production`

```bash
cp .env.production.example .env.production
# Điền: POSTGRES_PASSWORD, REDIS_PASSWORD, NEO4J_PASSWORD, JWT_SECRET, v.v.
nano .env.production
```

### Bước 3: Sync Artifacts + Start Containers trên App Server

```bash
./deploy.sh up
```

### Bước 4: Upload Nginx Config Lên Gateway (172.20.2.16)

```bash
./deploy.sh gateway-sync
```

> **Lưu ý:** Trước khi upload, cần đảm bảo thư mục `conf.d/` tồn tại trên Gateway:
> ```bash
> ssh ubuntu@172.20.2.16 "mkdir -p /home/ubuntu/vnp-qa-platform/proxy/conf.d"
> ```

### Bước 5: Lấy SSL Certificate (Certbot)

```bash
# Cần upload HTTP-only config trước nếu cert chưa có
./deploy.sh gateway-ssl        # Lấy cert cho cả b4 + b6
./deploy.sh gateway-ssl b4     # Chỉ b4.openledger.vn
./deploy.sh gateway-ssl b6     # Chỉ b6.openledger.vn
```

Hoặc thủ công trên Gateway:
```bash
ssh ubuntu@172.20.2.16
cd /home/ubuntu/vnp-qa-platform/proxy/
docker compose run --rm certbot certonly \
  --webroot --webroot-path /var/www/certbot \
  -d b4.openledger.vn \
  --email admin@openledger.vn \
  --agree-tos --no-eff-email

docker compose run --rm certbot certonly \
  --webroot --webroot-path /var/www/certbot \
  -d b6.openledger.vn \
  --email admin@openledger.vn \
  --agree-tos --no-eff-email

docker compose exec nginx nginx -s reload
```

### Bước 6: Kiểm Tra Tổng Thể

```bash
./deploy.sh health
```

---

## B. UPDATE PHIÊN BẢN MỚI (Routine Deployment)

### Option 1: Full Deploy (build + sync + restart)

```bash
./deploy.sh deploy
```

### Option 2: Chỉ Sync + Restart (không build lại)

```bash
./deploy.sh up
```

### Option 3: Build + Deploy Từng Phần

```bash
# Chỉ build backend
./deploy.sh build-backend                    # Tất cả Go binaries
./deploy.sh build-backend preview-backend    # Chỉ preview-backend
./deploy.sh build-backend preview-all        # Chỉ preview-all
./deploy.sh build-backend openui-backend     # Chỉ openui-backend

# Chỉ build frontend
./deploy.sh build-frontend                   # Tất cả (Preview + OpenUI)
./deploy.sh build-frontend preview           # Chỉ Preview frontend
./deploy.sh build-frontend openui            # Chỉ OpenUI frontend

# Sync + restart service cụ thể
./deploy.sh up preview-backend
./deploy.sh up openui-backend
./deploy.sh up preview-all
./deploy.sh up openui-frontend

# Skip options
SKIP_FRONTEND=1 ./deploy.sh deploy           # Bỏ qua frontend build
SKIP_BACKEND=1 ./deploy.sh deploy            # Bỏ qua backend build
SKIP_KGS=1 ./deploy.sh deploy               # Bỏ qua kgs-platform image
```

---

## C. QUẢN LÝ VẬN HÀNH (Operations)

### Xem Logs

```bash
# Tất cả services
./deploy.sh logs

# Service cụ thể
./deploy.sh logs preview-backend
./deploy.sh logs openui-backend
./deploy.sh logs preview-all
./deploy.sh logs openui-frontend
./deploy.sh logs preview-frontend
```

### Xem Trạng Thái

```bash
./deploy.sh status
```

### Restart / Reload

```bash
./deploy.sh restart                    # Restart tất cả
./deploy.sh restart openui-backend     # Restart service cụ thể
./deploy.sh reload openui-backend      # Force-recreate container
```

### Gateway Nginx

```bash
./deploy.sh gateway-sync               # Sync + reload nginx configs
./deploy.sh gateway-ssl                # Renew SSL cho tất cả domains
```

### SSH vào Server

```bash
./deploy.sh ssh
```

---

## D. XỬ LÝ SỰ CỐ NHANH

| Triệu chứng | Lệnh khắc phục |
|-------------|----------------|
| Container crash / không start | `./deploy.sh logs <service>` → đọc nguyên nhân |
| `exec format error` | `./deploy.sh build-backend` (cross-compile lại) |
| 502 Bad Gateway trên b4 | `./deploy.sh health` → kiểm tra openui-frontend/backend |
| 502 Bad Gateway trên b6 | `./deploy.sh health` → kiểm tra preview-frontend/backend |
| SSL expired | `./deploy.sh gateway-ssl` |
| Database không kết nối | SSH vào server, kiểm tra `.env.production` |
| Cần restart khẩn | `./deploy.sh restart` |
| Dừng toàn bộ | `./deploy.sh down` |

> Xem chi tiết tại: `deploy/production/specs/TROUBLESHOOTING.md`

---

## E. THAM KHẢO

| Tài liệu | Đường dẫn |
|----------|-----------|
| Hướng dẫn đầy đủ | `deploy/production/specs/END_TO_END_DEPLOYMENT_GUIDE.md` |
| Checklist Go-Live | `deploy/production/specs/DEPLOYMENT_CHECKLIST.md` |
| Troubleshooting | `deploy/production/specs/TROUBLESHOOTING.md` |
| Nginx: b4 (OpenUI) | `deploy/production/nginx-b4-openledger.conf` |
| Nginx: b6 (Preview) | `deploy/production/nginx-b6-openledger.conf` |
| Docker Compose | `deploy/production/docker-compose.prod.yaml` |
| Deploy Script | `deploy/production/deploy.sh` |
