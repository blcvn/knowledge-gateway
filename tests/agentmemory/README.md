# agentmemory — Python Test Scripts

Bộ Python scripts để **sinh dữ liệu**, **push lên server**, và **thử nghiệm sử dụng memory** của hệ thống agentmemory.

## Cấu trúc

```
tests/agentmemory/
├── .env.example          ← Template cấu hình (copy → .env)
├── .env                  ← Cấu hình thực (gitignored)
├── config.py             ← Module đọc .env, dùng chung cho tất cả scripts
│
├── 01_generate_data.py   ← STEP 1: Sinh dữ liệu test
├── 02_push_data.py       ← STEP 2: Push dữ liệu lên server
├── 03_test_memory.py     ← STEP 3: Thử nghiệm sử dụng memory
├── run_all.py            ← Pipeline runner (chạy 3 steps liên tiếp)
│
├── data/                 ← Output (gitignored)
│   ├── sessions.json
│   ├── observations.jsonl
│   ├── memories.json
│   ├── search_queries.json
│   ├── manifest.json
│   ├── push_report.json
│   └── test_results.json
│
├── test-cases/           ← Test case documents (markdown)
├── designs/              ← Test design documents
└── requirements/         ← Test requirement documents
```

## Setup

### 1. Yêu cầu

```bash
# Python 3.8+
python3 --version

# Cài thư viện
pip install requests
```

### 2. Cấu hình

```bash
cd tests/agentmemory
cp .env.example .env
# Sửa .env với thông tin server thực
```

**Các biến quan trọng trong `.env`:**

| Biến | Mô tả | Default |
|------|-------|---------|
| `AGENTMEMORY_URL` | URL server agentmemory | `http://localhost:3111` |
| `AGENTMEMORY_SECRET` | Bearer token (để trống nếu local mode) | *(trống)* |
| `TEST_PROJECT` | Tên project test | `vnp-test-project` |
| `AGENT_ID` | Agent ID | `test-agent-python` |
| `GEN_SESSION_COUNT` | Số sessions sinh ra | `5` |
| `GEN_OBS_PER_SESSION` | Số observations/session | `20` |
| `GEN_MEMORY_COUNT` | Số memories sinh ra | `10` |

## Chạy

### Chạy toàn bộ pipeline

```bash
cd tests/agentmemory
python3 run_all.py
```

### Chạy từng bước

```bash
# Step 1: Sinh dữ liệu (không cần server)
python3 01_generate_data.py

# Step 2: Push lên server
python3 02_push_data.py
python3 02_push_data.py --dry-run          # Không gửi request thực
python3 02_push_data.py --sessions-only    # Chỉ push sessions
python3 02_push_data.py --obs-only         # Chỉ push observations
python3 02_push_data.py --memories-only    # Chỉ push memories
python3 02_push_data.py --batch-delay-ms 200  # Giảm tốc (ms)

# Step 3: Kiểm thử
python3 03_test_memory.py                  # Chạy tất cả suites
python3 03_test_memory.py --suite health   # Chỉ health check
python3 03_test_memory.py --suite search   # Chỉ search tests
python3 03_test_memory.py --suite remember # Chỉ remember/recall
python3 03_test_memory.py --suite lifecycle # Chỉ session lifecycle
python3 03_test_memory.py --verbose        # In chi tiết response
```

### Tùy chọn pipeline

```bash
python3 run_all.py --skip-generate    # Dùng data đã có
python3 run_all.py --skip-push        # Không push lại
python3 run_all.py --dry-push         # Push dry-run
python3 run_all.py --suite health     # Chỉ test health
python3 run_all.py --verbose          # Verbose
```

## Test Suites (`03_test_memory.py`)

| Suite | Test Cases | Mô tả |
|-------|-----------|-------|
| `health` | TC-019-001, TC-019-003 | Server health, no sensitive leak |
| `search` | TC-004, TC-005, TC-006 | BM25 search, limit, empty query |
| `remember` | TC-007-001, TC-007-002 | Lưu và recall memory |
| `lifecycle` | TC-001, TC-002 | Session start → observe → end |
| `replay` | TC-016-005 | Session list API |
| `validation` | TC-020, TC-015 | Auth, input validation |

## Output

Sau khi chạy, các file được tạo trong `data/`:

| File | Nội dung |
|------|----------|
| `manifest.json` | Metadata về data đã sinh |
| `sessions.json` | 5 session objects |
| `observations.jsonl` | 100 observations (1 dòng/obs) |
| `memories.json` | 10 memory objects |
| `search_queries.json` | 10 query mẫu |
| `push_report.json` | Kết quả push (ok/fail counts) |
| `test_results.json` | Kết quả test (passed/failed + details) |

## Ghi chú

- `01_generate_data.py` hoạt động **không cần server** — chỉ sinh file JSON/JSONL
- `02_push_data.py` cần server đang chạy; dùng `--dry-run` để test logic mà không cần server
- `03_test_memory.py` tự động tạo session tạm trong quá trình test; các session này không ảnh hưởng dữ liệu production
- Tất cả scripts đọc cấu hình từ `.env` qua `config.py`
