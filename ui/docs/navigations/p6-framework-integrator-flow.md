# User Navigation Flow: P6 - AI Framework Integrator (Secondary)

## Vai trò & Mục tiêu
Các kỹ sư muốn tích hợp VNP Memory vào trong các Framework phổ biến (LangChain, LlamaIndex, CrewAI) bằng cách viết các Wrapper/Connector.

## Flow 1: Cấu hình Webhook & Lấy thông tin SDK
**Ngữ cảnh**: Integrator cần cấu hình để VNP Memory bắn sự kiện (Webhook) trả về hệ thống CrewAI của họ khi một Graph Node mới được tạo ra.
1. **Truy cập `API & SDK Manager`**:
   - Chuyển sang Tab **Webhooks**.
   - Thêm Endpoint URL mới (vd: `https://crewai-agent.local/vnp-webhook`).
   - Chọn các Event cần lắng nghe: `memory.created`, `graph.node_added`.
2. **Khám phá SDK Specs**:
   - (Nếu Console có tích hợp) Mở tab **SDK Documentation**.
   - Copy đoạn code mẫu (Boilerplate) bằng Python/TypeScript để khởi tạo `VNPMemoryClient`.
   - Xem cấu trúc Payload của REST API để parse JSON trong Framework của mình.
