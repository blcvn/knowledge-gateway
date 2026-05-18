# Configuration Reference

Zep Monolith sử dụng Unified Configuration. Bạn có thể cấu hình ứng dụng thông qua các biến môi trường (Environment Variables) hoặc file `.env`.

## Cấu Hình Port (Internal & Gateway)

| Biến | Ý nghĩa | Default | Required |
|---|---|---|---|
| `ZEP_GATEWAY_PORT` | Cổng cho public REST API (Gateway) | `8080` | No |
| `ZEP_USER_PORT` | Cổng localhost cho zep-user | `9041` | No |
| `ZEP_THREAD_PORT` | Cổng localhost cho zep-thread | `9042` | No |
| `ZEP_MEMORY_PORT` | Cổng localhost cho zep-memory | `9043` | No |
| `ZEP_GRAPH_PORT` | Cổng localhost cho zep-graph | `9044` | No |
| `ZEP_SEARCH_PORT` | Cổng localhost cho zep-search | `9045` | No |
| `ZEP_ADMIN_PORT` | Cổng localhost cho zep-admin | `9046` | No |

## Cấu Hình Database & Infra

Các cấu hình này sẽ được truyền xuyên suốt cho tất cả các embedded services.

| Biến | Ý nghĩa | Ví dụ | Required |
|---|---|---|---|
| `DB_HOST` | Địa chỉ PostgreSQL (pgvector) | `postgres` | Yes |
| `REDIS_HOST` | Địa chỉ Redis server | `redis` | Yes |
| `NATS_URL` | NATS JetStream URL | `nats://nats:4222` | Yes |
| `NEO4J_URI` | Địa chỉ Neo4j Graph DB | `neo4j://neo4j:7687` | Yes |
| `NEO4J_AUTH` | Tài khoản xác thực Neo4j | `neo4j/password` | Yes |

*Lưu ý:* Khi chạy qua `docker-compose up`, các biến hạ tầng sẽ tự động được gán thành tên container nội bộ (e.g. `postgres`, `nats`).
