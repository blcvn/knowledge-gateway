Đây là một câu hỏi rất hay, vì nó chạm đến cách neuron hoạt động ở mức "hệ thống". Câu trả lời ngắn gọn là:

Thông thường, một điện thế hoạt động (action potential) được khởi phát tại axon hillock sẽ lan truyền dọc toàn bộ axon và đến tất cả các axon terminals của neuron đó. Nó không chỉ đi đến một phần của axon.

Cơ chế thông thường

Giả sử một neuron có 1 axon nhưng cuối axon phân thành nhiều nhánh:

```text
           Soma
            │
     Axon Hillock
            │

================│================
│
┌─────┼─────┐
│ │ │
```

Terminal A
Terminal B
Terminal C

Khi axon hillock quyết định phát xung:

Action Potential

```text
↓

=========================>

↓
```


A ✓

B ✓

C ✓

Thông thường, cả A, B và C đều nhận được cùng một điện thế hoạt động.

Vậy neuron có gửi cùng một "thông điệp" đến mọi nơi không?

Về tín hiệu điện, gần như là có. Một action potential là tín hiệu "tất cả hoặc không có gì" (all-or-none). Nó không yếu dần khi truyền dọc axon bình thường.

Tuy nhiên, hiệu quả ở từng đầu tận cùng (terminal) có thể khác nhau.

Ví dụ:

Terminal A → giải phóng nhiều chất dẫn truyền

Terminal B → giải phóng ít hơn

Terminal C → tạm thời không giải phóng

Sự khác biệt này có thể do:

số lượng túi chứa chất dẫn truyền thần kinh,
xác suất giải phóng (release probability),
điều hòa bởi các thụ thể trước synapse (presynaptic receptors),
lịch sử hoạt động gần đây của terminal.

Vì vậy, cùng một xung điện nhưng tác động lên từng neuron đích có thể không giống nhau.

Có khi nào chỉ một số nhánh được kích hoạt không?

Có, nhưng đây là các trường hợp đặc biệt và là chủ đề nghiên cứu hiện đại.

Ví dụ:

### 1. Branch-specific modulation

Một số nhánh axon có thể bị điều hòa cục bộ bởi:

các neuron khác,
tế bào thần kinh đệm (glia),
chất dẫn truyền thần kinh tại chỗ.

Điều này làm một nhánh truyền tín hiệu hiệu quả hơn nhánh khác.

### 2. Axonal branch failure

Ở một số neuron rất dài hoặc hoạt động ở tần số cao, điện thế hoạt động đôi khi không truyền thành công vào tất cả các nhánh nhỏ.

Điều này không phổ biến nhưng đã được quan sát trong một số loại neuron.

### 3. Presynaptic inhibition

Một đầu tận cùng có thể bị ức chế ngay trước khi giải phóng chất dẫn truyền.

Khi đó:

Action potential

```text
↓
```


Terminal A → truyền

Terminal B → bị ức chế

Terminal C → truyền

Như vậy, xung điện vẫn đến cả ba nhánh, nhưng chỉ một số nhánh thực sự truyền tín hiệu hóa học sang neuron tiếp theo.

Liên hệ với AI

Đây là một điểm khác biệt lớn giữa neuron sinh học và neuron trong AI.

Trong mạng neuron nhân tạo:

Neuron

```text
↓
```


Output

```text
↓
```


Mọi kết nối đều nhận cùng một giá trị

Trong não:

Neuron

```text
↓
```


Action Potential

```text
↓
```


Terminal A → truyền mạnh

Terminal B → truyền yếu

Terminal C → có thể bị điều hòa

Nghĩa là mỗi đầu ra của cùng một neuron có thể được điều chỉnh độc lập ở mức synapse, dù chúng đều nhận cùng một xung điện từ axon.

Kết luận

Có thể hình dung neuron như một công ty:

Soma là ban điều hành.
Axon hillock là nơi ra quyết định: "Có gửi thông điệp hay không?"
Axon là tuyến vận chuyển thông điệp.
Axon terminals là các chi nhánh giao hàng.

Khi ban điều hành quyết định gửi một thông điệp, toàn bộ tuyến vận chuyển đều mang cùng một thông điệp đến tất cả các chi nhánh. Tuy nhiên, mỗi chi nhánh có thể quyết định giao nhiều, giao ít hoặc tạm thời không giao, tùy theo trạng thái cục bộ của chính đầu tận cùng đó và các tín hiệu điều hòa mà nó nhận được. Đây là một trong những cơ chế giúp não vừa có tín hiệu thống nhất, vừa có khả năng điều khiển rất tinh vi ở từng kết nối riêng lẻ.
