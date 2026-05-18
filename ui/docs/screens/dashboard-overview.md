# Giao diện Dashboard Overview

Mô tả chi tiết kiến trúc giao diện hiện tại của màn hình Dashboard (dựa trên source code `Dashboard.tsx`).

## 1. Cấu trúc tổng quan
Màn hình được chia làm 4 khối theo chiều dọc, có thể cuộn (scrollable) với class `flex-1 overflow-y-auto p-6 space-y-6`. Nền tối đồng nhất chuẩn Dark Mode.

### Khối 1: Page Header
- Tiêu đề: **Platform Overview**.
- Phụ đề: "Enterprise Cognitive Infrastructure Control Plane" (màu text mờ `text-white/50`).

### Khối 2: KPI Cards (4 Cột)
Hiển thị 4 chỉ số cốt lõi dạng thẻ (Cards) với nền gradient nổi bật ở icon:
- **Active Agents**: Icon Users, dải màu Xanh lam nhạt (Blue/Cyan).
- **Recall Latency**: Icon Activity, dải màu Tím/Hồng (Purple/Pink).
- **Context Savings**: Icon Zap, dải màu Xanh lục (Green/Emerald).
- **Graph Growth**: Icon Database, dải màu Cam/Đỏ (Orange/Red).
- *Chi tiết hiển thị*: Mỗi thẻ có giá trị chính, phần trăm thay đổi (`change`) màu xanh lá và tên chỉ số.

### Khối 3: Memory Flow Visualization (24h)
- Một biểu đồ dạng miền (`AreaChart` từ thư viện Recharts) hiển thị dữ liệu hoạt động trong 24h.
- Có 3 lớp dữ liệu xếp chồng với gradient mờ:
  - **Ingest**: Màu xanh lam (`#3b82f6`).
  - **Recall**: Màu xanh lục (`#10b981`).
  - **Embeddings**: Màu tím (`#8b5cf6`).
- Có chú thích (Legend) nằm ở dưới đáy biểu đồ dạng chấm tròn kèm text.

### Khối 4: 2 Cột (Engine Health & Memory Distribution)
Khối này sử dụng CSS Grid 2 cột (`grid-cols-2 gap-6`).
1. **Engine Health**: 
   - Danh sách các Engine lõi (Graphiti, Cognee, Zep, OpenViking, KGS).
   - Mỗi Engine hiển thị dấu chấm báo trạng thái (Xanh lá = Healthy, Vàng = Warning, Đỏ = Critical).
   - Hiển thị thông số Latency và thông số Hàng đợi (Queue: Q) trong khung nổi (`bg-white/10`).
2. **Memory Type Distribution**: 
   - Biểu đồ cột (`BarChart`) phân bố số lượng cho Episodic (Tím), Semantic (Xanh lam), Conversational (Xanh lục), Procedural (Cam).

### Khối 5: Recent Activity
- Danh sách sự kiện gần nhất dạng List.
- Mỗi sự kiện có một dấu chấm tròn chỉ báo loại sự kiện (Success = Xanh, Warning = Vàng, Error = Đỏ, Info = Xanh lam).
- Thông tin gồm: Hành động (Action), Tên Tenant, và Thời gian (vd: "2 min ago"). Có nút "View All" ở góc trên bên phải.
