# Runbook (Operations Guide)

## 1. Startup Procedure
Để khởi chạy ứng dụng (kèm database) ở local/production:
```bash
docker-compose up -d
```
Quá trình khởi chạy sẽ theo trình tự:
1. Postgres, Neo4j, Redis, NATS khởi động.
2. Zep Monolith khởi động. Supervisor bật lần lượt: User -> Thread -> Memory -> Graph -> Search -> Admin.
3. Cuối cùng, Gateway sẽ khởi động và bind vào `0.0.0.0:8080`.

## 2. Shutdown & Rollback
Ứng dụng hỗ trợ Graceful Shutdown.
```bash
docker-compose down
```
Khi nhận lệnh dừng (SIGTERM), Gateway sẽ ngừng nhận traffic mới. Các internal service xử lý nốt request đang dở dang và tắt theo thứ tự ngược lại (LIFO). Timeout mặc định là 10 giây.

## 3. Health Check
Kiểm tra sức khoẻ của gateway (public):
```bash
curl http://localhost:8080/healthz
```
*Expected: 200 OK*

## 4. Common Errors & Troubleshooting
| Lỗi (Log Message) | Nguyên nhân | Khắc phục |
|---|---|---|
| `connection refused` (Neo4j/Postgres) | DB chưa khởi động xong | Khởi động monolith sau khi infra đã `healthy` |
| `Supervisor: Error starting zep-user: port in use` | Port `9041` đang bị chiếm | Đổi `ZEP_USER_PORT` hoặc kill tiến trình đang chiếm port |
| `Timeout while stopping zep-search` | Tiến trình có job chạy quá 10s | Tăng supervisor shutdown timeout trong code |
