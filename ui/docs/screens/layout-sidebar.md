# Giao diện Layout Sidebar

Mô tả chi tiết kiến trúc giao diện hiện tại của thanh điều hướng Sidebar (dựa trên source code `Sidebar.tsx`).

## 1. Cấu trúc tổng quan
Sidebar là một cột dọc cố định nằm ở bên trái toàn bộ ứng dụng, có chiều rộng cố định (`w-64`) và chiếm toàn bộ chiều cao (`h-full`). 
Sử dụng nền màu xám đen (`bg-[#1a1a1f]`) phân tách với màn hình chính bằng một đường viền dọc mờ (`border-r border-white/10`).

### Khối 1: Logo & Branding (Header)
- Khu vực trên cùng của Sidebar.
- Hiển thị Text Logo: **VNP Memory** (font-semibold, màu trắng).
- Phụ đề nhỏ ngay bên dưới: "Control Plane" (màu trắng trong suốt 50%).
- Có một đường viền ngang chia cắt mỏng ở dưới (`border-b`).

### Khối 2: Menu Navigation (Main Content)
Vùng hiển thị danh sách các mục điều hướng, có khả năng cuộn dọc độc lập nếu danh sách quá dài (`overflow-y-auto`). Bao gồm 10 mục:
1. **Overview** (Icon: LayoutDashboard)
2. **Memory Explorer** (Icon: Database)
3. **Graph Studio** (Icon: Network)
4. **Sessions** (Icon: MessageSquare)
5. **Governance** (Icon: Shield)
6. **Pipelines** (Icon: GitBranch)
7. **Infrastructure** (Icon: Server)
8. **Observability** (Icon: Activity)
9. **API & SDK** (Icon: Key)
10. **Settings** (Icon: Settings)

**Hiệu ứng tương tác (Active/Hover States):**
- Trạng thái bình thường: Text và Icon màu xám nhạt (`text-white/70`). Khi Hover (rê chuột), nền đổi sang mờ nhẹ (`hover:bg-white/5`) và chữ sáng lên.
- Trạng thái được chọn (Active): Nút bấm đổi sang nền màu xanh dương nhạt (`bg-blue-500/20`), text màu xanh (`text-blue-400`), và có đường viền mỏng bao quanh (`border-blue-500/30`).

### Khối 3: Footer
- Nằm cố định ở đáy Sidebar, cách biệt bằng đường viền ngang (`border-t border-white/10`).
- Hiển thị thông tin phiên bản và phân hệ: "v1.0.0 • Enterprise Edition" với font chữ siêu nhỏ (`text-xs`).
