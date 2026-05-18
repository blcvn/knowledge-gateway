---
id: TASK-040
title: Tối ưu hoá Hiệu suất & Bundle Size (Enterprise Grade)
service: ui
type: task
status: done
source: enterprise-requirements
---

# TASK-040: Tối ưu hoá Hiệu suất & Bundle Size (Enterprise Grade)

## 1. Mục tiêu (Objective)
Đảm bảo hiệu năng tải trang và tương tác của Frontend đạt chuẩn Enterprise (điểm Lighthouse > 90), đáp ứng yêu cầu tải nhanh và mượt mà cho tập dữ liệu lớn.

## 2. Phạm vi công việc (Scope)
- **Code Splitting & Lazy Loading**: Cấu hình chia nhỏ bundle (Route-based splitting) sử dụng `React.lazy` và `Suspense`. 
- **Asset Optimization**: Tối ưu hóa ảnh (WebP, SVG), nén font chữ, và defer loading cho các tài nguyên không thiết yếu.
- **Render Optimization**: Tối ưu hóa render cycles sử dụng `useMemo`, `useCallback`, và `React.memo` cho các biểu đồ (Charts) và bảng dữ liệu (Data Tables) phức tạp.
- **Virtualization**: Áp dụng kỹ thuật Virtual List / Windowing (e.g., `@tanstack/react-virtual`) cho các danh sách dài hoặc bảng dữ liệu lớn.
- **Vite/Webpack Config**: Cấu hình build tool để gzip/brotli nén file, loại bỏ dead code (Tree shaking).

## 3. Tiêu chí nghiệm thu (Acceptance Criteria)
- [x] Kích thước Initial JavaScript Bundle (First Load) dưới 200KB (Gzipped).
- [x] Tốc độ First Contentful Paint (FCP) dưới 1.5s trên mạng 4G.
- [x] Time to Interactive (TTI) dưới 3s.
- [x] Table/List hiển thị hàng ngàn record không bị tụt FPS (dùng Virtualization).
- [x] Các module nặng (như Recharts, CodeMirror) chỉ được tải khi người dùng truy cập màn hình tương ứng.

## 4. Tài liệu tham khảo
- Web Vitals & Performance Standards.
