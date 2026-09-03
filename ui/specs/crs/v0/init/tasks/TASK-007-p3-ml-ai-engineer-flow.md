---
id: TASK-007
title: Implement User Navigation Flow: P3 - ML/AI Engineer
service: ui
type: task
status: done
source: docs/navigations/p3-ml-ai-engineer-flow.md
---

# TASK-007: Triển khai User Navigation Flow: P3 - ML/AI Engineer

## 1. Mục tiêu (Objective)
- Triển khai các yêu cầu và đặc tả kỹ thuật từ tài liệu: `ui/docs/navigations/p3-ml-ai-engineer-flow.md`
- Đảm bảo 100% tính năng và giao diện được mô tả trong tài liệu được áp dụng vào source code.

## 2. Phạm vi công việc (Scope)
Dưới đây là chi tiết công việc được trích xuất từ tài liệu gốc:

**## Vai trò & Mục tiêu**
ML/AI Engineer (P3) quan tâm đến chất lượng (Quality) của Graph, mức độ chính xác của Semantic Search, và việc thiết kế Ontology phù hợp với nghiệp vụ....

**## Flow 1: Khám phá và Đánh giá Knowledge Graph**
**Ngữ cảnh**: Cần phân tích xem Graph có bị phân mảnh (nhiều Node rời rạc) hoặc quá dày đặc (dense) không.
1. **Truy cập `Graph Studio`**:
   - Bật chế độ lọc (Filter) chỉ hiển thị một loại Ontology Class cụ thể (vd: `Company` hoặc `Product`).
   - Sử dụng **Timeline Slider** ở đáy màn hình, kéo lùi về 1 tháng trước rồi ấn nút Play để quan sát cách các cụm (Clusters) thực thể phát triển và kết nối với nhau theo thời gian.
   - Đánh giá xem thuật toán Entity Resolution có gom cụm đúng các node đồ...

**## Flow 2: Tối ưu hóa Vector Pipeline**
**Ngữ cảnh**: Cần biết quá trình Chunking và Embedding có hoạt động hiệu quả và đủ nhanh không.
1. **Truy cập `Pipelines Monitor`**:
   - Xem sơ đồ React Flow của Pipeline xử lý.
   - Chú ý vào bước `Embedding Generation` và `Chunking`. Xem throughput (số chunk/sec) hiện tại.
2. **Truy cập `Memory Explorer`**:
   - Tìm kiếm một chuỗi văn bản mẫu.
   - Quan sát xem Memory card hiện ra có Confidence Score bao nhiêu. Nếu quá thấp, có thể cần đổi model Embedding....

**## Flow 3: Chạy Evaluation Pipeline & So sánh**
**Ngữ cảnh**: Cần đo lường xem phương pháp Graph RAG có tốt hơn RAG truyền thống cho domain hiện tại hay không.
1. **Truy cập `Context Debugger`**:
   - Chạy 2 truy vấn với cùng một Prompt nhưng chọn Engine khác nhau.
   - Quan sát tab **Evaluation** (nếu có) để xem chỉ số Completeness và Accuracy.
   - Đưa ra quyết định tinh chỉnh hệ số Retrieval....

## 3. Tiêu chí nghiệm thu (Acceptance Criteria)
- [x] Code tuân thủ theo đúng chuẩn của dự án.
- [x] Giao diện (nếu có) hiển thị đúng theo mô tả trong document.
- [x] Mọi chức năng/luồng tương tác trong tài liệu đều hoạt động chính xác.
- [x] Build thành công và không phá vỡ các luồng (flows) hiện tại.


### 💎 Enterprise & Product-Grade UI/UX Standards
- [x] **Premium Aesthetics**: Giao diện mang cảm giác cao cấp (premium). Tránh dùng màu sắc cơ bản. Ưu tiên dùng hệ màu HSL mượt mà, dark mode sâu sắc (deep dark), hiệu ứng gradient tinh tế và glassmorphism.
- [x] **Typography**: Sử dụng modern typography (Inter, Roboto, Outfit). Layout tuân thủ chặt chẽ spacing grid system, UI không bị chật chội hoặc lỏng lẻo.
- [x] **Dynamic & Responsive**: Tích hợp các micro-animations, hiệu ứng hover, focus states, và transition mượt mà giúp giao diện "sống động" và phản hồi cao.
- [x] **Enterprise Completeness**: Xử lý triệt để loading states, empty states, error boundaries, và accessible (a11y) đầy đủ.

## 4. Tài liệu tham khảo
- [Source Document](../../../docs/navigations/p3-ml-ai-engineer-flow.md)
