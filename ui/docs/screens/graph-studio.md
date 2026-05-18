# Giao diện Graph Studio

Mô tả chi tiết kiến trúc giao diện hiện tại của màn hình Graph Studio (dựa trên source code `GraphStudio.tsx`).

## 1. Cấu trúc tổng quan
Màn hình được chia làm 2 phần chính: Header mỏng ở trên và Vùng Canvas tương tác chiếm toàn bộ không gian còn lại (`flex-1 relative`).

### Khối 1: Header
- Nằm trên cùng với tiêu đề **Graph Studio** và phụ đề "Visual knowledge graph exploration".
- Cạnh phải Header là cụm 3 nút thao tác nhanh: **Zoom In**, **Zoom Out**, và **Maximize** (Phóng to toàn màn hình).

### Khối 2: Graph Canvas (Vùng Tương Tác Cốt Lõi)
Sử dụng công nghệ SVG để vẽ lưới và các node đồ thị. Gồm các thành phần xếp chồng lên nhau (Overlay):

#### 2.1 Lớp Nền & SVG Đồ thị
- **Background**: Màu tối sâu (`#0a0a0f`) kèm Pattern đường lưới (Grid line mờ).
- **Edges (Các cạnh)**: Các đường line kết nối giữa các Node với màu sắc khác nhau, độ mờ (opacity) 0.6.
- **Nodes (Thực thể)**: 
  - Các hình tròn đại diện cho thực thể (User, Project, Task, Team, Document).
  - Có viền màu (Stroke) đại diện cho loại node (Xanh, Tím, Cam, Hồng).
  - Label text nằm ngay phía dưới Node.

#### 2.2 Stats Overlay (Góc trên bên trái)
Ba thẻ thống kê nổi nhỏ gọn (`backdrop-blur-sm` hiệu ứng kính kính mờ):
- **Nodes**: Số lượng đỉnh (vd: 1,247).
- **Edges**: Số lượng cạnh (vd: 3,821).
- **Clusters**: Số lượng cụm (vd: 24).

#### 2.3 Entity Inspector Sidebar (Góc trên bên phải)
Một bảng thông tin động, nổi lên phía trên canvas, hiển thị chi tiết của Node hiện tại đang được chọn:
- Tiêu đề: **Entity Inspector**.
- Thông tin chính: Type, Ontology Class (Loại Schema).
- **Confidence**: Thanh tiến trình (Progress bar) trực quan màu xanh lam (ví dụ: 92%).
- Số Node liên kết.
- **Related Facts**: Danh sách các khối nhỏ (`bg-white/5`) hiển thị thông tin thực tế dạng text (VD: "Works on Project Alpha").

#### 2.4 Timeline Controls (Góc dưới ở giữa)
Thanh công cụ trượt thời gian nổi ở mép dưới màn hình (Floating Bottom Bar):
- Nút Play/Pause để giả lập mô phỏng chạy theo thời gian.
- Text hiển thị mốc bắt đầu (2026-01-01) và kết thúc (2026-05-13).
- Một thanh trượt (`input type="range"`) cho phép kéo tay để "duyệt" lịch sử tri thức (Temporal playback).
