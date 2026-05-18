# User Navigation Flow: P8 - Product Manager (Secondary)

## Vai trò & Mục tiêu
Quản lý Sản phẩm (PM) sử dụng VNP Memory để phân tích Insights (hiểu biết) từ tệp người dùng, xem xét mức độ tăng trưởng và tối ưu chi phí (Tokens).

## Flow 1: Theo dõi Tăng trưởng và Chi phí
**Ngữ cảnh**: PM muốn biết lượng người dùng và độ lớn của Database bộ nhớ tăng bao nhiêu trong tháng qua để lên kế hoạch ngân sách.
1. **Truy cập `Dashboard Overview`**:
   - Tập trung vào chỉ số **Graph Growth** (Số lượng Node/Edge) để đánh giá độ dày của tri thức.
   - Theo dõi chỉ số **Context Savings** (%) để tính toán lượng Token LLM (chi phí OpenAI/Anthropic) đã tiết kiệm được nhờ có bộ nhớ dài hạn thay vì gửi toàn bộ lịch sử chat.
   - Nhìn biểu đồ 24h để biết khung giờ cao điểm (Peak hours) mà User hoạt động.

## Flow 2: Phân tích Hành vi Người dùng (Insights)
**Ngữ cảnh**: Đội ngũ Product cần biết người dùng đang dùng Agent để làm những tác vụ gì nhiều nhất.
1. **Truy cập `Memory Explorer`**:
   - Sử dụng Filter lọc các Memory theo loại `Procedural` (Cách người dùng yêu cầu Agent làm việc) hoặc `Semantic` (Sự kiện người dùng đề cập).
   - Đọc tóm tắt (Summary) để thu thập Insight.
2. **Truy cập `Sessions Explorer`**:
   - Đọc ngẫu nhiên (Sample) một vài đoạn chat Replay thực tế giữa User và Agent để hiểu rõ Pain points và cách Agent xử lý.

## Flow 3: Xem Structured User Profile
**Ngữ cảnh**: PM muốn rút xuất chân dung khách hàng (User Profile) tổng hợp từ hàng trăm đoạn chat rời rạc.
1. **Truy cập `Memory Explorer` / `Graph Studio`**:
   - Lọc theo một nhóm User cụ thể.
   - Xem các Entity thuộc Class `Profile` hoặc `Preference` đã được LLM cấu trúc hóa.
   - Tổng hợp các Topic/Sub_topic mà người dùng hay quan tâm nhất để định hướng tính năng tiếp theo cho sản phẩm.
