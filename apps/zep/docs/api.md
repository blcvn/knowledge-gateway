# Zep Monolith API Reference

Do Zep Monolith là một bản nhúng (embedded version) của Gateway và các Services, toàn bộ cấu trúc API REST/gRPC tuân thủ theo đúng đặc tả API của Gateway.

## Endpoints Công Khai (Qua Gateway)

* **Base URL:** `http://localhost:8080` (hoặc domain của bạn)

Tất cả các tài liệu API chi tiết như:
- `/api/v1/users`
- `/api/v1/threads`
- `/api/v1/memories`
- `/api/v1/graph`
- `/api/v1/search`

...đều kế thừa trực tiếp từ Swagger/OpenAPI spec của module `gateway`.

## Internal Endpoints (Chỉ dành cho localhost)

Các cổng gRPC nội bộ KHÔNG ĐƯỢC expose ra ngoài internet:
- `:9041` - User gRPC
- `:9042` - Thread gRPC
- `:9043` - Memory gRPC
- `:9044` - Graph gRPC
- `:9045` - Search gRPC
- `:9046` - Admin gRPC

Mọi external client phải kết nối qua cổng `8080` (REST) hoặc `8081` (gRPC public) của Gateway.
