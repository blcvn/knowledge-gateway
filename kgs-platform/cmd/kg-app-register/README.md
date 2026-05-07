# kg-app-register

CLI tool để register app mới vào `ai-kg-service` (`kgs-platform`) và trả về `app_id` để cấu hình cho service client (ví dụ `ai-agent-executor` với biến `KG_APP_ID`).

## Inputs bắt buộc

- `-app-name`: tên app cần register
- `-owner`: owner của app (ví dụ `system@internal`)

## Inputs tùy chọn

- `-description`: mô tả app
- `-base-url`: địa chỉ HTTP của kgs-platform (default: `http://localhost:8000`, hoặc env `KGS_BASE_URL`)
- `-timeout`: timeout mỗi HTTP call (default: `20s`)
- `-reuse-existing`: mặc định `true`; nếu app cùng `app_name + owner` đã tồn tại thì trả `app_id` cũ thay vì tạo app mới
- `-format`: `app_id | env | json` (default: `app_id`)

## Quick Start

Từ thư mục `services/ai-kg-service/kgs-platform`:

```bash
GOWORK=off go run ./cmd/kg-app-register \
  -base-url http://localhost:18000 \
  -app-name ai-agent-executor \
  -owner system@internal
```

Output mặc định chỉ là `app_id` (UUID), ví dụ:

```text
7599854e-994c-408b-b126-e381ad4cb07a
```

In theo format env để copy thẳng vào config:

```bash
GOWORK=off go run ./cmd/kg-app-register \
  -base-url http://localhost:18000 \
  -app-name ai-agent-executor \
  -owner system@internal \
  -format env
```

Ví dụ output:

```text
KG_APP_ID=7599854e-994c-408b-b126-e381ad4cb07a
```

## Build Binary

```bash
cd services/ai-kg-service/kgs-platform
GOWORK=off GOCACHE=/tmp/gocache go build -o ./bin/kg-app-register ./cmd/kg-app-register
./bin/kg-app-register -base-url http://localhost:18000 -app-name ai-agent-executor -owner system@internal
```
