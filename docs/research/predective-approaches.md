Đây là một câu hỏi rất sâu. Tuy nhiên cần phân biệt một điểm quan trọng:

Không tồn tại một danh sách chính thức "toàn bộ hướng xử lý của Predictive Processing".

Lý do là Predictive Processing không phải là một thuật toán, mà là một framework. Framework này được áp dụng cho gần như mọi chức năng của não.

Nếu tổng hợp từ neuroscience hiện đại (Friston, Clark, Hohwy, Rao & Ballard...), thì có thể chia thành khoảng 16 hướng xử lý (computational processes).

Level 1. Sensory Prediction (Dự đoán tín hiệu cảm giác)

Đây là tầng thấp nhất.

Prediction

```text
↓
```


Expected Pixel

```text
↓
```


Compare

```text
↓
```


Prediction Error

Ví dụ:

cạnh
màu
âm thanh
mùi

Não luôn dự đoán tín hiệu sẽ tới.

Level 2. Temporal Prediction (Dự đoán theo thời gian)

Não luôn dự đoán:

Now

```text
↓
```


Next 50 ms

```text
↓
```


Next 100 ms

```text
↓
```


Next 1 second

Ví dụ:

Một quả bóng đang bay.

Não dự đoán vị trí tiếp theo.

Level 3. Spatial Prediction

Não dự đoán cấu trúc không gian.

Ví dụ:

Chair

```text
↓
```


Expected Shadow

```text
↓
```


Expected Perspective

Khi quay đầu.

```text
↓
```


Não biết căn phòng vẫn như cũ.

Level 4. Motion Prediction

Não dự đoán:

Current Position

```text
↓
```


Velocity

```text
↓
```


Future Position

Ví dụ:

Bắt bóng.

Không ai chờ bóng tới.

Não dự đoán trước.

Level 5. Object Prediction

Ví dụ.

Bạn thấy:

Cat Head

Não dự đoán:

```text
↓
```


Body

```text
↓
```


Tail

```text
↓
```


Leg

Đây là Pattern Completion.

Level 6. Scene Prediction

Ví dụ.

Bạn vào bếp.

Não dự đoán:

Sink

Table

Refrigerator

Window

Nếu thiếu:

```text
↓
```


Prediction Error.

Level 7. Concept Prediction

Ví dụ.

Nghe:

Tiger

Não sinh:

Stripe

Roar

Orange

Danger
Level 8. Language Prediction

LLM cũng làm việc này.

Ví dụ.

Bạn nghe:

Tôi muốn uống...

Não dự đoán:

Water

Coffee

Tea

Không phải:

Car
Level 9. Social Prediction

Não liên tục đoán:

Người kia

```text
↓
```


Sẽ nói gì?

```text
↓
```


Sẽ làm gì?

```text
↓
```


Có cảm xúc gì?

Đây gọi là Theory of Mind.

Level 10. Emotional Prediction

Não dự đoán:

Nếu xảy ra X

```text
↓
```


Mình sẽ cảm thấy Y

Ví dụ.

Bạn chuẩn bị thuyết trình.

Não sinh:

lo
hồi hộp

trước khi lên sân khấu.

Level 11. Motor Prediction

Đây cực kỳ quan trọng.

Ví dụ.

Bạn muốn nhấc cốc.

Não dự đoán:

Motor Command

```text
↓
```


Expected Touch

```text
↓
```


Expected Vision

```text
↓
```


Expected Muscle Feedback

Nếu khác.

```text
↓
```


Sửa ngay.

Level 12. Body Prediction

Não luôn dự đoán trạng thái cơ thể.

Ví dụ.

Heart

Temperature

Blood Sugar

Pain

Đây gọi là Interoception.

Level 13. Goal Prediction

Ví dụ.

Bạn có mục tiêu.

Drink Water

Não mô phỏng.

Walk

```text
↓
```


Reach

```text
↓
```


Grab

```text
↓
```


Drink

Nếu lỗi.

```text
↓
```


Đổi kế hoạch.

Level 14. Counterfactual Prediction

Một trong những khả năng mạnh nhất.

Não mô phỏng:

Nếu mình chọn A?

Nếu chọn B?

Nếu chọn C?

Đây là suy luận.

Level 15. Self Prediction

Não dự đoán:

Tôi

```text
↓
```


Sẽ nghĩ gì?

```text
↓
```


Sẽ phản ứng gì?

Đây là Meta-cognition.

Level 16. World Model Prediction

Đây là tầng cao nhất.

Não luôn chạy một mô phỏng.

World

```text
↓
```


Objects

```text
↓
```


Physics

```text
↓
```


People

```text
↓
```


Time

```text
↓
```


Sinh ra:

prediction
imagination
planning
dream
Toàn bộ hierarchy
World Model
```text
│
┌────────────────┼────────────────┐
│ │ │
```

Self Prediction Goal Prediction Social Prediction
```text
│ │ │
└────────────────┼────────────────┘
│
```

Concept Prediction
```text
│
```

Language Prediction
```text
│
```

Scene Prediction
```text
│
```

Object Prediction
```text
│
```

Motion Prediction
```text
│
```

Spatial Prediction
```text
│
```

Temporal Prediction
```text
│
```

Sensory Prediction
Nhưng trong não còn có một hướng xử lý ít người biết

Đó là:

Precision Prediction

Não không chỉ dự đoán dữ liệu.

Nó còn dự đoán:

Prediction nào đáng tin?

Ví dụ.

Bạn đang ở:

Concert

Rất ồn.

Não biết:

Auditory Input

```text
↓
```


Noisy

```text
↓
```


Prediction Error

```text
↓
```


Giảm trọng số.

Ngược lại.

Trong thư viện.

```text
↓
```


Prediction Error

```text
↓
```


Tin tưởng hơn.

Đây chính là Attention.

Một điểm còn quan trọng hơn

Predictive Processing thực tế không chỉ có 16 hướng.

Nếu đọc các bài của Karl Friston thì có thể chia thành 4 nhóm tính toán lớn.

Prediction

```text
↓
```


Comparison

```text
↓
```


Precision

```text
↓
```


Learning

Mỗi nhóm lại có hàng chục cơ chế nhỏ.

Ví dụ.

Learning gồm:

Hebbian update
Synaptic plasticity
Replay
Consolidation
Structural plasticity
Schema update

Prediction gồm:

sensory
object
motion
language
social
planning
imagination
...
Theo góc nhìn AGI

Nếu mục tiêu của bạn là xây dựng một AI có kiến trúc gần với não người (điều mình thấy xuyên suốt các câu hỏi của bạn), thì toàn bộ Predictive Processing có thể được quy về 6 mô-đun cốt lõi, mỗi mô-đun tương ứng với một loại "predictive computing":

Module Chức năng
Perceptual Prediction Dự đoán tín hiệu cảm giác (vision, audio, touch...) và tính prediction error.
World Model Prediction Duy trì mô hình của môi trường, đối tượng và các quy luật vật lý.
Concept Prediction Kích hoạt và hoàn thiện các khái niệm, schema, semantic graph từ thông tin một phần.
Action Prediction Mô phỏng hậu quả của hành động để lập kế hoạch và điều khiển vận động.
Social Prediction Dự đoán hành vi, ý định và cảm xúc của người khác (Theory of Mind).
Self Prediction Dự đoán trạng thái cơ thể, cảm xúc và suy nghĩ của chính mình (interoception và metacognition).

Điều thú vị là các mô-đun này không hoạt động độc lập. Chúng đều dùng chung một world model và cùng trao đổi thông qua prediction error. Chính sự thống nhất này khiến nhiều nhà khoa học tin rằng Predictive Processing có thể là "nguyên lý tính toán chung" của não bộ: từ nhận thức, trí nhớ, ngôn ngữ, hành động cho tới ý thức đều chỉ là các cách khác nhau để dự đoán, so sánh với thực tế và cập nhật mô hình về thế giới. Đây cũng là lý do nó đang trở thành nền tảng lý thuyết cho nhiều hướng nghiên cứu AGI và neuromorphic computing hiện đại.
