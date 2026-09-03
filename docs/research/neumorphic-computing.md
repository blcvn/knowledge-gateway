```text
Neuromorphic computing là một trong những hướng nghiên cứu mà mình nghĩ sẽ ảnh hưởng rất lớn đến AI trong 10–20 năm tới. Nếu GPU là "bộ não giả lập bằng số học", thì neuromorphic computing cố gắng xây dựng phần cứng hoạt động theo các nguyên lý vật lý tương tự não, thay vì mô phỏng não trên CPU/GPU.
```


### 1. Neuromorphic Computing là gì?

Neuromorphic Computing =

Thiết kế phần cứng mà việc tính toán diễn ra giống neuron thật.

Không phải:

Neuron
```text
↓
```


Tensor

```text
↓
```


GPU

```text
↓
```


Multiply

mà là

Neuron

```text
↓
```


Điện tích

```text
↓
```


Spike

```text
↓
```


Synapse

```text
↓
```


Spike

Tức là mỗi transistor không còn chỉ là logic gate, mà trở thành một phần của neuron hoặc synapse.

### 2. Tại sao cần Neuromorphic?

GPU hiện nay rất mạnh nhưng có nhược điểm.

Ví dụ:

GPT

Matrix Multiply

```text
↓
```


HBM Memory

```text
↓
```


Matrix Multiply

```text
↓
```


HBM

```text
↓
```


Matrix Multiply

GPU phải liên tục:

đọc memory
ghi memory
nhân ma trận

Đây gọi là von Neumann bottleneck.

Não thì khác.

Trong não:

Neuron

```text
↓
```


Synapse

```text
↓
```


Neuron

```text
↓
```


Synapse

Memory nằm ngay tại synapse.

Không có:

CPU

```text
↓
```


RAM

```text
↓
```


CPU

```text
↓
```


RAM

Điều này tiết kiệm rất nhiều năng lượng.

### 3. Nguyên lý hoạt động

Một neuron neuromorphic thường mô phỏng:

Input spike

```text
↓
```


Membrane potential

```text
↓
```


Threshold

```text
↓
```


Fire

```text
↓
```


Reset

Giống neuron sinh học.

Ví dụ:

0

```text
↓
```


0.2

```text
↓
```


0.4

```text
↓
```


0.7

```text
↓
```


1.0

```text
↓
```


Spike

```text
↓
```


0

Không cần phép nhân ma trận lớn.

### 4. Một neuron neuromorphic

Có thể mô hình như:

Input A

```text
↓
```


Input B

```text
↓
```


Input C

```text
↓
```


Capacitor

```text
↓
```


Threshold

```text
↓
```


Spike

Điện tích được tích trong capacitor.

Khi vượt ngưỡng:

Capacitor xả.

Điều này gần giống màng neuron thật.

### 5. Synapse trong Neuromorphic

Synapse thường lưu:

Weight

Nhưng thay vì:

float32

nó có thể là:

điện trở
memristor
điện tích
phase change material

Tức là:

memory chính là phần tử vật lý.

### 6. Event Driven

GPU

Clock

```text
↓
```


Tính toàn bộ mạng

```text
↓
```


Clock

```text
↓
```


Tính toàn bộ mạng

Neuromorphic

Không có spike

```text
↓
```


Không làm gì

```text
↓
```


Có spike

```text
↓
```


Chỉ neuron liên quan hoạt động

Giống não.

Điều này gọi là:

Event Driven Computing

### 7. Vì sao cực kỳ tiết kiệm điện?

Ví dụ.

GPU

1 tỷ neuron

```text
↓
```


Mỗi bước đều tính

Não

1 tỷ neuron

```text
↓
```


Chỉ vài %

```text
↓
```


Fire

Đa số neuron:

Sleeping

Không tiêu điện nhiều.

Đây gọi là

Sparse Computation.

### 8. Kiến trúc tổng thể

Một chip neuromorphic thường gồm:

Core

Neuron

Neuron

Neuron

```text
↓
```


Router

```text
↓
```


Core

Neuron

Neuron

```text
↓
```


Router

Các core gửi spike cho nhau.

Không gửi tensor.

### 9. Giao tiếp

Thông điệp chỉ là

Neuron ID

Fire Time

Ví dụ

Neuron 145

Fire

12.4 ms

Không cần truyền cả vector lớn.

### 10. Learning

Có hai hướng.

Offline Learning

GPU train

```text
↓
```


Weight

```text
↓
```


Download vào chip

Chip chỉ inference.

Online Learning

Chip tự học.

Ví dụ

Spike

```text
↓
```


STDP

```text
↓
```


Weight tăng

Giống não.

### 11. STDP

Một cơ chế nổi tiếng là Spike-Timing-Dependent Plasticity (STDP).

Nếu

Neuron A

```text
↓
```


Neuron B

A luôn bắn trước B

```text
↓
```


Weight tăng.

Nếu

B

```text
↓
```


A

Weight giảm.

Không cần Backpropagation.

### 12. Memristor

Một công nghệ rất được kỳ vọng.

Memristor:

Voltage

```text
↓
```


Resistance thay đổi

```text
↓
```


Tự nhớ trạng thái

Nó giống synapse thật.

Nếu dòng điện chạy nhiều

```text
↓
```


Resistance đổi.

Chính là learning.

### 13. Các chip nổi tiếng
```text
    Intel Loihi
```


Đặc điểm

hỗ trợ SNN
học online
event-driven
cực ít điện
IBM TrueNorth
1 triệu neuron

256 triệu synapse

Công suất chỉ khoảng 70 mW.

BrainChip Akida

Hướng tới

Edge AI
IoT
Camera
```text
### 14. So sánh với GPU
```

GPU Neuromorphic
Đồng bộ theo clock Event-driven
Tensor Spike
Backprop STDP hoặc cơ chế cục bộ
Memory tách compute Memory gần hoặc nằm trong synapse
Tiêu thụ điện cao Rất tiết kiệm
Rất mạnh cho LLM hiện nay Rất phù hợp cho cảm biến và xử lý thời gian thực
```text
### 15. Hạn chế hiện nay
```


Neuromorphic vẫn chưa thay thế GPU vì:

Thiếu thuật toán huấn luyện hiệu quả tương đương backpropagation cho nhiều bài toán lớn.
Spiking Neural Networks (SNNs) còn kém các Transformer trong nhiều tác vụ ngôn ngữ và thị giác.
Hệ sinh thái phần mềm chưa trưởng thành bằng NVIDIA CUDA và các framework AI phổ biến.
Phần cứng giữa các hãng chưa có chuẩn chung.
```text
### 16. Tương lai: từ LLM đến "AI Brain"
```


Nếu nhìn từ góc độ kiến trúc, có thể thấy một xu hướng thú vị:

GPU
```text
│
├── Rất giỏi tính toán ma trận lớn
├── Phù hợp Transformer, LLM
└── Tốn điện
```


Trong khi đó:

Neuromorphic
```text
│
├── Event-driven
├── Có khả năng học liên tục (continual learning)
├── Xử lý dữ liệu cảm biến theo thời gian thực
├── Tiết kiệm năng lượng
└── Phù hợp với robot và Edge AI
```


Nhiều nhà nghiên cứu dự đoán tương lai sẽ không phải GPU hoặc neuromorphic, mà là hệ thống lai (hybrid):

```text
GPU/TPU đảm nhiệm huấn luyện các mô hình nền tảng lớn như LLM.
```

Neuromorphic chips xử lý tín hiệu cảm biến, trí nhớ ngắn hạn, học trực tuyến, phản xạ nhanh và điều khiển robot.
Hai loại phần cứng giao tiếp với nhau giống như vỏ não (neocortex) và các hệ thống thần kinh khác phối hợp trong não người.

Đây cũng là lý do neuromorphic computing được xem là một trong những hướng hứa hẹn nhất để xây dựng các AI agent tự chủ, có khả năng học liên tục từ môi trường với mức tiêu thụ năng lượng rất thấp—điều mà các hệ thống AI hiện nay vẫn còn gặp nhiều thách thức.
