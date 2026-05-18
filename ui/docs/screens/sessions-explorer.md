# Giao diện Sessions Explorer

Mô tả thiết kế giao diện chi tiết dự kiến cho màn hình quản lý phiên hội thoại (`sessions-explorer`).

## 1. Cấu trúc tổng quan
Màn hình áp dụng bố cục **Chia đôi (Split-pane Layout)**: Cột bên trái hiển thị danh sách các phiên (Sessions List) và cột bên phải chiếm phần lớn diện tích dùng để hiển thị chi tiết nội dung cuộc hội thoại (Session Replay Viewer).

### Khối 1: Left Panel - Danh sách Sessions
- **Header**: Thanh tìm kiếm (theo User ID, Agent ID) và bộ lọc theo ngày.
- **Danh sách Item**: 
  - Mỗi thẻ hiển thị: ID phiên (vd: `sess_abc123`), Avatar hoặc tên User/Agent.
  - Phụ đề hiển thị thời gian bắt đầu (Timestamp) và thời lượng (Duration: `12m 30s`).
  - Badge nhỏ hiển thị số lượng tin nhắn (`14 msgs`).
- Trạng thái Active: Thẻ đang được chọn có viền xanh dương và nền nổi bật.

### Khối 2: Right Panel - Session Replay Viewer
Khu vực này giống như một giao diện ứng dụng nhắn tin (Chat UI).
- **Session Metadata (Top Bar)**: Hiển thị thông tin tổng quan của phiên đang xem, nút "Export Log" và nút "Replay" (tái hiện lại tuần tự thời gian).
- **Chat Transcript**:
  - Giao diện bong bóng chat (Chat bubbles).
  - Phân biệt rõ màu sắc: 
    - Tin nhắn của **User**: Nằm bên phải, nền màu xanh lam (`bg-blue-600`).
    - Tin nhắn của **Agent**: Nằm bên trái, nền xám tối (`bg-[#2a2a35]`).
    - **System Log** (thông tin ẩn như "Agent retrieved 3 memories"): Căn giữa, text nhỏ màu vàng mờ.
  - Mỗi bong bóng chat có thời gian gửi (Timestamp) ở góc dưới.
