# Giao diện Memory Explorer

Mô tả chi tiết kiến trúc giao diện hiện tại của màn hình Memory Explorer (dựa trên source code `MemoryExplorer.tsx`).

## 1. Cấu trúc tổng quan
Màn hình chia làm 3 phần chính theo chiều dọc, được khóa thanh cuộn ở root (`overflow-hidden flex flex-col`) và chỉ cho phép cuộn ở phần danh sách kết quả.

### Khối 1: Header
- Tiêu đề: **Memory Explorer**.
- Phụ đề: "Unified search across all memory types".
- Phân cách bằng viền mờ bên dưới (`border-b border-white/10`).

### Khối 2: Search & Filters Toolbar
Khối tương tác chính của người dùng, bao gồm:
1. **Thanh tìm kiếm (Search Bar)**:
   - Icon Search kính lúp ở bên trái.
   - Placeholder: "Semantic search, hybrid search, or graph query...".
   - Hiệu ứng Focus viền xanh (`focus:ring-blue-500/50`).
2. **Nút Bộ lọc (Filter Button)**:
   - Đặt cạnh thanh tìm kiếm, có viền bo góc, biểu tượng Filter.
3. **Tab Chuyển đổi Loại Trí nhớ (Memory Type Tabs)**:
   - Dãy nút bấm ngang: **All**, **Episodic**, **Semantic**, **Conversational**, **Procedural**.
   - Trạng thái Active: Có viền và chữ màu xanh lam nổi bật (`bg-blue-500/20 text-blue-400`).
   - Mỗi tab đều có Badge hiển thị số lượng bộ nhớ tương ứng (vd: `742`, `145`).

### Khối 3: Danh sách Kết quả (Results List)
Khu vực hiển thị kết quả tìm kiếm, có thể cuộn dọc (`overflow-y-auto p-6`).

**Cấu trúc một Thẻ Kết Quả (MemoryResultCard):**
1. **Header Thẻ**:
   - Icon phân loại: Một ô vuông bo góc chứa icon màu sắc chuyên biệt (Tím cho Episodic, Xanh lam cho Semantic, Xanh lục cho Conversational, Cam cho Procedural).
   - Title: Tiêu đề in đậm của bộ nhớ.
   - Timestamp: Thời gian lưu.
   - Confidence: Điểm tự tin (vd: 94% confidence) hiển thị màu xanh lá ở góc phải.
2. **Nội dung (Summary)**: 
   - Đoạn văn bản ngắn mô tả/tóm tắt lại nội dung bộ nhớ.
3. **Thực thể (Entities)**: 
   - Danh sách các thực thể liên quan được hiển thị dưới dạng Pill (bo góc, viền mờ `bg-white/5 border-white/10`).
4. **Footer Thẻ**:
   - Meta data: Tên Source (Engine nguồn) và ID Session.
   - Nút Hành động: "View Details →" màu xanh lam ở góc dưới bên phải để mở chi tiết (có thể trigger Side Inspector sau này).
