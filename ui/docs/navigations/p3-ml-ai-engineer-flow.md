# User Navigation Flow: P3 - ML/AI Engineer

## Vai trò & Mục tiêu
ML/AI Engineer (P3) quan tâm đến chất lượng (Quality) của Graph, mức độ chính xác của Semantic Search, và việc thiết kế Ontology phù hợp với nghiệp vụ.

## Flow 1: Khám phá và Đánh giá Knowledge Graph
**Ngữ cảnh**: Cần phân tích xem Graph có bị phân mảnh (nhiều Node rời rạc) hoặc quá dày đặc (dense) không.
1. **Truy cập `Graph Studio`**:
   - Bật chế độ lọc (Filter) chỉ hiển thị một loại Ontology Class cụ thể (vd: `Company` hoặc `Product`).
   - Sử dụng **Timeline Slider** ở đáy màn hình, kéo lùi về 1 tháng trước rồi ấn nút Play để quan sát cách các cụm (Clusters) thực thể phát triển và kết nối với nhau theo thời gian.
   - Đánh giá xem thuật toán Entity Resolution có gom cụm đúng các node đồng nghĩa hay không.

## Flow 2: Tối ưu hóa Vector Pipeline
**Ngữ cảnh**: Cần biết quá trình Chunking và Embedding có hoạt động hiệu quả và đủ nhanh không.
1. **Truy cập `Pipelines Monitor`**:
   - Xem sơ đồ React Flow của Pipeline xử lý.
   - Chú ý vào bước `Embedding Generation` và `Chunking`. Xem throughput (số chunk/sec) hiện tại.
2. **Truy cập `Memory Explorer`**:
   - Tìm kiếm một chuỗi văn bản mẫu.
   - Quan sát xem Memory card hiện ra có Confidence Score bao nhiêu. Nếu quá thấp, có thể cần đổi model Embedding.

## Flow 3: Chạy Evaluation Pipeline & So sánh
**Ngữ cảnh**: Cần đo lường xem phương pháp Graph RAG có tốt hơn RAG truyền thống cho domain hiện tại hay không.
1. **Truy cập `Context Debugger`**:
   - Chạy 2 truy vấn với cùng một Prompt nhưng chọn Engine khác nhau.
   - Quan sát tab **Evaluation** (nếu có) để xem chỉ số Completeness và Accuracy.
   - Đưa ra quyết định tinh chỉnh hệ số Retrieval.
