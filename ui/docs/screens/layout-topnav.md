# Giao diện Layout Top Navigation (TopNav)

Mô tả chi tiết kiến trúc giao diện hiện tại của thanh điều hướng phía trên (dựa trên source code `TopNav.tsx`).

## 1. Cấu trúc tổng quan
TopNav là thanh ngang nằm sát trên cùng của khu vực làm việc (nằm bên phải Sidebar), có chiều cao cố định (`h-16`) và sử dụng cùng màu nền tối (`bg-[#1a1a1f]`).

Giao diện được chia thành 3 phần rõ rệt (Trái, Giữa, Phải).

### Khối 1: Left - Tenant & Environment Selector
Chứa các thành phần chuyển đổi bối cảnh môi trường:
1. **Tenant Selector**: 
   - Một nút bấm (button) hiển thị tên Tenant hiện tại (VD: `Acme Corporation`).
   - Có chấm tròn màu xanh lá (chỉ báo Active) ở đầu và mũi tên chỉ xuống (`ChevronDown`) ở cuối.
2. **Environment Switcher**: 
   - Hiển thị Môi trường hiện tại (VD: `Production`).
   - Có chấm báo trạng thái thay đổi màu linh hoạt: Đỏ (Production), Vàng (Staging), Xanh (các môi trường khác).

### Khối 2: Center - Global Search Bar
Thanh tìm kiếm toàn cục, được đặt ở chính giữa màn hình với kích thước rộng (`max-w-2xl`):
- Bao gồm icon Search (kính lúp) ở bên trái.
- Placeholder text: "Search memories, sessions, entities...".
- Cạnh phải của thanh tìm kiếm có một phím tắt gợi ý được bo viền (kbd): `⌘K` chỉ dẫn cách gọi nhanh search.
- Khi focus vào input, viền chuyển thành dạng phát sáng màu xanh (`focus:ring-blue-500/50`).

### Khối 3: Right - Actions & Profile
Góc trên bên phải chứa các thao tác nhanh và thông tin cá nhân người dùng:
1. **Notification Button (Chuông)**: Icon Bell kèm chấm đỏ (chỉ báo có thông báo mới) nổi ở góc trên.
2. **AI Assistant Button (Bot)**: Icon Bot để gọi nhanh hỗ trợ AI.
3. **Profile Dropdown**:
   - Avatar hiển thị hai chữ cái đầu của tên người dùng (VD: "AD") trên nền gradient từ Xanh sang Tím (`from-blue-500 to-purple-500`).
   - Icon `ChevronDown` báo hiệu có menu sổ xuống.
