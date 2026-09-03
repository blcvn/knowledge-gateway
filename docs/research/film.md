Đây là một câu hỏi rất sâu. Thực ra nó liên quan trực tiếp đến Predictive Processing, temporal integration (tích hợp theo thời gian) và Bayesian inference trong não.

Điểm quan trọng nhất là:

Não không cố gắng tái tạo chính xác thế giới. Não cố gắng xây dựng mô hình có xác suất cao nhất về thế giới từ dữ liệu luôn thiếu và nhiễu.

Nói cách khác:

Não không nhìn thấy thực tại; não suy luận về thực tại.

Đó là lý do phim ảnh, ảo thuật và cả nhiều ảo giác thị giác đều hoạt động.

### 1. Não không nhận toàn bộ thế giới

Giả sử bạn nhìn căn phòng.

Bạn nghĩ mình thấy:

Căn phòng đầy đủ.

Thực tế võng mạc chỉ có vùng trung tâm (fovea) có độ phân giải rất cao; vùng ngoại vi mờ hơn nhiều. Mắt còn liên tục thực hiện các chuyển động rất nhỏ (saccades), nên dữ liệu đến não bị chia thành nhiều "mẫu" rời rạc theo thời gian.

Não nhận được một chuỗi dữ liệu không hoàn chỉnh:

Frame 1

```text
↓
```


Frame 2

```text
↓
```


Frame 3

Sau đó tự ghép thành một cảnh liên tục.

### 2. Bộ phim là ví dụ hoàn hảo

Một bộ phim 24 fps thực chất là:

Image

```text
↓
```


Image

```text
↓
```


Image

```text
↓
```


Image

Không có chuyển động.

Chỉ có các ảnh tĩnh.

Nhưng não tạo ra:

Walking

Tại sao?

Vì não luôn giả định:

Thế giới thường thay đổi liên tục chứ không nhảy loạn.

Đây là một prior (tiên nghiệm) rất mạnh.

### 3. Predictive Processing giải thích phim

Giả sử bạn xem:

Frame A

```text
↓
```


Frame B

Frame A:

Người đứng đây.

Frame B:

Người đứng lệch 5 cm.

Não không nghĩ:

"Có hai người khác nhau."

Não dự đoán:

Người vừa di chuyển.

Prediction error rất nhỏ.

```text
↓
```


Não kết luận:

Có chuyển động.

### 4. Não luôn chọn lời giải đơn giản nhất

Giả sử có hai khả năng.

Khả năng 1

Một người

```text
↓
```


đi từng bước

Khả năng 2

100 người khác nhau

```text
↓
```


liên tục xuất hiện

Khả năng nào xác suất cao hơn?

Rõ ràng là (1).

Não luôn chọn mô hình có xác suất cao nhất.

Đây gần với nguyên lý Bayesian inference.

### 5. Tính liên tục theo thời gian (Temporal continuity)

Não có một giả định rất mạnh:

Hiện tại

≈

1 giây trước

Tức là:

Thế giới thường thay đổi từ từ.

Nếu không có bằng chứng rõ ràng, não sẽ giả định sự liên tục.

Ví dụ.

Bạn thấy:

Ô tô

```text
↓
```


Ô tô

```text
↓
```


Ô tô

Dù giữa các khung hình có khoảng trống, bạn vẫn thấy đó là cùng một chiếc xe.

### 6. Object permanence

Một ví dụ khác.

Con mèo đi sau cái ghế.

Bạn vẫn tin:

Cat

```text
↓
```


Behind Chair

Bạn không nghĩ:

Cat

```text
↓
```


Biến mất

```text
↓
```


Con khác xuất hiện

Não giữ mô hình của đối tượng ngay cả khi không còn nhìn thấy.

### 7. Đây chính là "world model"

Trong não.

Bạn không lưu:

Pixels

Bạn lưu:

World

Ví dụ.

Room

```text
├── Table

├── Chair

├── Computer
```


Khi quay đầu.

Bạn không nhìn thấy bàn.

Nhưng mô hình vẫn có:

Table

Khi quay lại.

```text
↓
```


Prediction đúng.

```text
↓
```


Error nhỏ.

### 8. Tại sao cắt cảnh trong phim vẫn hiểu?

Ví dụ.

Cảnh 1

Người mở cửa.

Cắt.

Cảnh 2

Người đã vào nhà.

Thực tế:

Không hề có cảnh bước qua cửa.

Nhưng não tự thêm:

Open

```text
↓
```


Walk

```text
↓
```


Enter

Điều này gọi là event completion.

Não tự điền khoảng trống.

### 9. Kuleshov Effect

Đây là một thí nghiệm kinh điển.

Cùng một khuôn mặt vô cảm.

Ghép với:

bát súp,
quan tài,
em bé.

Người xem nói:

Ông ấy đói.

Ông ấy buồn.

Ông ấy vui.

Trong khi khuôn mặt không hề thay đổi.

Não dùng ngữ cảnh để suy luận cảm xúc.

### 10. Change blindness

Có một hiện tượng nổi tiếng.

Hai ảnh:

Ảnh A

```text
↓
```


Ảnh B

Khác nhau rất lớn.

Nhưng nếu chen một khung trắng ở giữa.

```text
↓
```


Rất nhiều người không nhận ra.

Vì:

Não không lưu toàn bộ pixel.

Não lưu:

Concept
```text
### 11. Ảo thuật
```


Ảo thuật gia không "đánh lừa mắt".

Họ đánh lừa:

attention,
prediction,
memory.

Ví dụ.

Bạn dự đoán:

Đồng xu

```text
↓
```


tay phải

Trong khi thực tế:

tay trái

Prediction quá mạnh.

```text
↓
```


Não không kiểm tra lại.

### 12. Tại sao phim hoạt hình vẫn sống động?

Một nhân vật hoạt hình chỉ gồm:

vài nét vẽ,
màu sắc đơn giản.

Nhưng não bổ sung:

trọng lượng,
cảm xúc,
ý định,
tính cách.

Bạn đang xem mô hình mà não tự hoàn thiện, không chỉ những gì có trên màn hình.

### 13. Vì sao điều này lại có lợi cho tiến hóa?

Nếu não phải đợi đủ mọi dữ liệu mới đưa ra kết luận, phản ứng sẽ quá chậm.

Hãy tưởng tượng trong rừng:

Bạn thấy một vật thể màu vàng với vài vệt đen chuyển động trong bụi cây.

Có hai chiến lược:

Chờ nhìn thật rõ mới quyết định.
Dự đoán: "Có thể là hổ" và lùi lại.

Chiến lược (2) có thể tạo ra nhiều "báo động giả", nhưng chi phí của một lần nhầm là rất nhỏ, trong khi chi phí của việc bỏ sót một con hổ thật có thể là tử vong.

Vì vậy, tiến hóa ưu tiên một bộ não:

suy luận nhanh,
chấp nhận gần đúng,
liên tục cập nhật nếu có thêm bằng chứng.
```text
### 14. Liên hệ với AI
```


Đây là điểm rất khác giữa nhiều hệ AI hiện nay và não người.

LLM thường hoạt động như:

Input

```text
↓
```


Tính toán

```text
↓
```


Output

Trong khi não hoạt động gần hơn với:

World Model
```text
│
▼
```

Prediction
```text
│
▼
```

Sensory Input
```text
│
▼
```

Prediction Error
```text
│
▼
```

Update World Model

Điều quan trọng là world model tồn tại liên tục, ngay cả khi không có đầu vào. Vì vậy não có thể:

nối các cảnh phim thành một câu chuyện,
hiểu một đoạn hội thoại bị ngắt quãng,
đoán phần còn thiếu của một bức tranh,
và thậm chí tạo ra những ảo giác khi mô hình bên trong mạnh hơn dữ liệu cảm giác.

Đó là lý do một bộ phim chỉ là 24 ảnh tĩnh mỗi giây, nhưng với não, nó trở thành một thế giới liên tục có nhân vật, mục tiêu, cảm xúc và quan hệ nhân quả. Bộ não không bị "lừa" theo nghĩa đơn thuần; nó đang làm đúng nhiệm vụ mà tiến hóa đã tối ưu: xây dựng lời giải thích hợp lý nhất về thế giới từ thông tin luôn không đầy đủ. Đây cũng là lý do con người có thể hiểu truyện, phim, kịch, sơ đồ hay ký hiệu chỉ với một lượng dữ liệu rất nhỏ—vì phần lớn "thế giới" được tạo ra từ chính mô hình bên trong não.
