Đây là một câu hỏi rất hay vì nhiều người dùng lẫn lộn Predictive Coding (PC) và Predictive Processing (PP).

Thực ra:

Predictive Processing (PP) là framework tổng quát (triết lý về cách não hoạt động).
Predictive Coding (PC) là cơ chế thần kinh mà não có thể dùng để hiện thực framework đó.

Có thể hình dung:

Predictive Processing
```text
│
├── Predictive Coding
├── Active Inference
├── Free Energy Principle
└── Bayesian Brain
```


Nếu PP là hệ điều hành, thì PC là một thuật toán chạy bên trong.

### 1. Não không xử lý từ dưới lên

Quan điểm cũ:

World

```text
↓
```


Eyes

```text
↓
```


Visual Cortex

```text
↓
```


Recognition

```text
↓
```


Action

Quan điểm mới:

```text
        Prediction

           ↓
```


World → Retina → Compare

```text
           ↑

    Prediction Error
```


Não luôn có một mô hình.

Nó luôn đoán.

### 2. Não có nhiều tầng

Ví dụ đơn giản:

Concept

```text
↓
```


Object

```text
↓
```


Shape

```text
↓
```


Edge

```text
↓
```


Pixel

Ví dụ khi nhìn con hổ.

Tiger

```text
↓
```


Head

```text
↓
```


Stripe

```text
↓
```


Edges

```text
↓
```


Pixels
```text
### 3. Điều gì xảy ra khi nhìn con hổ?
```

Bước 1

Ánh sáng đi vào mắt.

Retina gửi:

Pixels

lên V1.

Bước 2

Nhưng trước khi pixel tới...

Não đã dự đoán.

Tiger

```text
↓
```


Expected Shape

```text
↓
```


Expected Edge

```text
↓
```


Expected Pixels
Bước 3

Visual cortex so sánh

Prediction

vs

Reality

Ví dụ.

Prediction

Orange stripe

Reality

Orange stripe

```text
↓
```


Sai số nhỏ.

Nếu

Prediction

Tiger

Reality

Dog

```text
↓
```


Sai số lớn.

```text
↓
```


Model phải sửa.

### 4. Predictive Coding thực chất làm gì?

Thay vì gửi:

1 GB Pixel

Não chỉ gửi:

Prediction Error

```text
\=
```


Reality − Prediction

Ví dụ.

Prediction

██████

Reality

█████▌

Error

  ▌

Chỉ cần gửi:

▌

Tiết kiệm rất nhiều năng lượng.

### 5. Vì sao điều này tiết kiệm năng lượng?

Giả sử bạn đang ngồi văn phòng.

Mỗi giây.

Visual input gần như giống nhau.

Nếu phải xử lý:

Monitor

Desk

Chair

Wall

100 lần mỗi giây.

```text
↓
```


Rất tốn.

Predictive Coding.

```text
↓
```


Prediction đúng.

```text
↓
```


Không cần xử lý lại.

Chỉ khi:

Boss bước vào.

```text
↓
```


Error tăng.

```text
↓
```


Attention bật.

### 6. Prediction Error đi đâu?

Prediction Error luôn đi lên.

Prediction luôn đi xuống.

Concept

```text
↓
```


Prediction

```text
↓
```


Object

```text
↓
```


Prediction

```text
↓
```


Edge

```text
↓
```


Prediction

```text
↓
```


Pixel

↑

Error

↑

Error

↑

Error

Đây là đặc điểm rất quan trọng.

### 7. Ví dụ nhận diện khuôn mặt

Bạn bước vào công ty.

Prediction.

Đây là Minh.

Visual input.

```text
↓
```


Đúng.

```text
↓
```


Error nhỏ.

```text
↓
```


Không cần tính nhiều.

Nếu:

Là người lạ.

```text
↓
```


Error tăng.

```text
↓
```


Các tầng trên cập nhật.

### 8. Khi đọc sách

Bạn đọc:

"Con hổ đang..."

Ngay lập tức.

Não dự đoán:

chạy
săn
gầm

Nếu câu tiếp:

"...đánh răng."

```text
↓
```


Prediction Error cực lớn.

```text
↓
```


Bạn bật cười.

Đây là lý do punchline của truyện cười hoạt động.

### 9. Attention nằm ở đâu?

Đây là điểm thú vị.

Attention không phải:

"Tập trung."

Mà gần hơn với:

Độ tin cậy của prediction error.

Ví dụ.

Bạn ở quán café.

Có:

tiếng nhạc
tiếng xe
bạn đang nói chuyện.

Não đặt:

Friend Voice

Precision = 1.0

Xe ngoài đường

Precision = 0.05

```text
↓
```


Prediction error của xe gần như bị bỏ.

### 10. Learning diễn ra thế nào?

Giả sử trẻ chưa biết hổ.

Prediction.

Big Cat

Reality.

Orange

Stripe

```text
↓
```


Error.

```text
↓
```


Não cập nhật.

```text
↓
```


Khái niệm mới.

Tiger

Đây chính là học.

Không phải lưu ảnh.

Mà sửa mô hình.

### 11. Memory trong Predictive Processing

Điểm này rất khác cách nghĩ truyền thống.

Memory không phải:

Read File

Mà là:

Run Prediction

Ví dụ.

Bạn nhớ nhà.

Não không mở ảnh.

Nó sinh:

sofa
cửa
cầu thang
mùi

Tất cả được generate.

### 12. Hallucination

Hallucination có thể xảy ra khi:

Prediction

Sensory Input.

Ví dụ.

Người rất lo lắng.

Prediction.

Có người gọi mình.

```text
↓
```


Một tiếng động nhỏ.

```text
↓
```


Prediction lấn át.

```text
↓
```


Não tạo:

"Có người gọi."

### 13. Dream

Khi ngủ REM.

Sensory input gần như tắt.

```text
↓
```


Prediction vẫn chạy.

```text
↓
```


Không có Error để sửa.

```text
↓
```


World Model tự sinh thế giới.

```text
↓
```


Dream.

### 14. AI hiện nay khác ở đâu?

Transformer.

Input

```text
↓
```


Output

Não.

Predict

```text
↓
```


Compare

```text
↓
```


Error

```text
↓
```


Update

```text
↓
```


Predict Again

Não chạy vòng lặp này khoảng hàng chục đến hàng trăm lần mỗi giây, trên nhiều cấp độ khác nhau (thị giác, thính giác, vận động, ngôn ngữ...).

### 15. Một ví dụ hoàn chỉnh: Bạn xem một bộ phim

Đây là ví dụ tổng hợp hầu hết những gì chúng ta đã trao đổi.

```text
Giả sử phim có 24 khung hình/giây.
```


Mỗi frame thực chất là ảnh tĩnh.

Frame 1

Người đứng

```text
↓
```


Não dự đoán

Người sẽ bước tiếp

Frame 2

Người tiến lên một chút

```text
↓
```


Prediction đúng.

```text
↓
```


Error rất nhỏ.

```text
↓
```


Não kết luận:

Có một người đang đi.

Điều thú vị là não không hề "thấy" chuyển động trực tiếp. Chuyển động là một giả thuyết (hypothesis) mà world model tạo ra để giải thích chuỗi ảnh rời rạc.

### 16. Góc nhìn sâu hơn: Prediction không chỉ về cảm giác mà còn về hành động

Đây là bước tiếp theo của Predictive Processing, gọi là Active Inference.

Não không chỉ dự đoán:

"Mình sẽ thấy gì?"

Mà còn dự đoán:

"Nếu mình làm X thì chuyện gì sẽ xảy ra?"

Ví dụ khi bạn muốn cầm một cốc nước:

Ý định uống nước
```text
│
▼
```

Não dự đoán:
"Tay sẽ nâng lên"
"Tay sẽ chạm cốc"
"Cảm giác ở ngón tay sẽ thay đổi"
```text
│
▼
```

Motor cortex phát lệnh
```text
│
▼
```

Cơ bắp di chuyển
```text
│
▼
```

Tín hiệu cảm giác quay trở lại
```text
│
▼
```

So sánh với dự đoán
```text
│
├── Khớp → tiếp tục
└── Không khớp → điều chỉnh lực và quỹ đạo
```


Điều này giải thích vì sao con người có thể cầm một chiếc cốc rất chính xác dù không cần "tính toán" từng góc khớp. Não liên tục dự đoán – so sánh – hiệu chỉnh trong một vòng lặp kín.

Kết luận

Nếu phải tóm tắt Predictive Coding bằng một câu thì sẽ là:

Mỗi tầng của não cố gắng giải thích hoạt động của tầng thấp hơn bằng một mô hình dự đoán; chỉ phần không giải thích được (prediction error) mới được truyền lên trên.

Và nếu phải tóm tắt Predictive Processing bằng một câu thì sẽ là:

Toàn bộ não là một cỗ máy xây dựng và liên tục cập nhật "mô hình sinh" (generative world model) của thế giới. Nhận thức, trí nhớ, tưởng tượng, chú ý, học tập và hành động đều là các quá trình khác nhau sử dụng cùng một mô hình này.

Đó là lý do ngày càng nhiều nhà khoa học cho rằng bản chất của trí thông minh không phải là lưu trữ thông tin, mà là xây dựng một mô hình đủ tốt để có thể dự đoán, mô phỏng và thích nghi với thế giới. Đây cũng là một trong những hướng nghiên cứu AGI được quan tâm nhất hiện nay.
