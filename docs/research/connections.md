Đúng. Nếu xét ở mức vật lý, thì nhận định của bạn gần như chính xác và cũng là quan điểm chủ đạo của khoa học thần kinh hiện nay:

Thông tin đầu vào không tạo thêm nhiều neuron (node) mới trong neocortex, mà chủ yếu làm thay đổi kiến trúc kết nối (connectivity) và trạng thái của các kết nối giữa các neuron.

Tuy nhiên, có một điểm cần bổ sung: không chỉ connections thay đổi, mà bản thân từng connection cũng thay đổi về tính chất.

Có thể mô hình hóa như một đồ thị (graph)

Nếu coi não là một graph:

Neuron (Node)
```text
│
```

Synapse (Edge)

Ban đầu:

```text
A ---- B

C ---- D
```


Sau khi học:

```text
A ===== B
\\ │
\\ │
\\ │
C=====D
```


Điều thay đổi gồm:

xuất hiện kết nối mới
một số kết nối biến mất
kết nối mạnh lên hoặc yếu đi
nhiều đường truyền mới được hình thành

Các node (neuron) phần lớn vẫn là các neuron cũ.

Về mặt vật lý, connection thay đổi như thế nào?

Một synapse không phải là một "dây điện" cố định. Nó có thể thay đổi ở nhiều cấp độ:

Độ mạnh của synapse (synaptic weight)
Đây là thay đổi phổ biến nhất. Cùng một tín hiệu nhưng synapse có thể truyền mạnh hơn hoặc yếu hơn.
Số lượng thụ thể (receptors)
Ví dụ, số lượng thụ thể glutamate trên neuron sau synapse có thể tăng hoặc giảm, làm thay đổi hiệu quả truyền tín hiệu.
Hình dạng của dendritic spine
Các gai dendrite có thể:
mọc thêm
lớn lên
co lại
biến mất
Hình thành synapse mới
Hai neuron trước đây chưa kết nối có thể tạo synapse mới.
Loại bỏ synapse không dùng
Các kết nối ít được sử dụng có thể bị cắt tỉa (synaptic pruning).
Node có hoàn toàn không đổi không?

Không hoàn toàn.

Ở người trưởng thành:

Khoảng 86 tỷ neuron hầu hết được tạo trước khi sinh.
Trong neocortex, số lượng neuron mới được sinh ra sau khi trưởng thành là rất ít.

Tuy nhiên, neuron vẫn thay đổi:

biểu hiện gen
loại và số lượng protein
số lượng nhánh dendrite
số lượng synapse
tính hưng phấn (excitability)

Do đó, một neuron sau khi học không còn hoàn toàn giống chính nó trước khi học.

Tại sao thay đổi connection lại đủ để lưu thông tin?

Hãy tưởng tượng mạng gồm 1 triệu neuron.

Nếu mỗi neuron kết nối với khoảng 10.000 neuron khác thì sẽ có khoảng:

1,000,000 × 10,000

≈ 10^10 connections

Thông tin chủ yếu nằm ở:

neuron nào kết nối với neuron nào
mức độ mạnh của từng kết nối
thời điểm và kiểu kích hoạt của chúng

Điều này tương tự như mạng Internet:

Router không cần thay đổi.
Điều thay đổi là bảng định tuyến và lưu lượng giữa các router.
Đây cũng là ý tưởng của AI hiện đại

Trong mạng nơ-ron nhân tạo:

Neuron
```text
│
```

Weight
```text
│
```

Neuron

Khi huấn luyện:

số lượng node gần như giữ nguyên
kiến trúc thường giữ nguyên
weight thay đổi

Não cũng có điểm tương đồng:

AI Não
Weight Độ mạnh synapse
```text
Gradient update Plasticity (LTP/LTD và các cơ chế khác)
```

Training Học tập và trải nghiệm
Epoch Lặp lại, replay khi ngủ, ôn tập

Khác biệt là não sinh học linh hoạt hơn nhiều. Não không chỉ điều chỉnh "trọng số" mà còn có thể:

tạo hoặc loại bỏ synapse,
thay đổi cấu trúc dendrite,
thay đổi mức biểu hiện gen,
điều chỉnh các chất dẫn truyền thần kinh,
và ở một số vùng như hippocampus, còn có thể sinh thêm neuron mới trong suốt cuộc đời.
Có thể xem trí nhớ là "thông tin được mã hóa trong cấu trúc"

Một cách khái quát, có thể coi:

Thông tin từ môi trường
```text
↓
```

Hoạt động của mạng neuron
```text
↓
```

Thay đổi vật lý của synapse và neuron
```text
↓
```

Mạng kết nối mới
```text
↓
```

Khả năng tái tạo lại thông tin trong tương lai

Theo quan điểm này, trí nhớ không phải là một "dữ liệu" được cất ở một vị trí cố định, mà là một mẫu (pattern) của các thay đổi vật lý trong mạng neuron. Khi một phần của mẫu được kích hoạt, hoạt động lan truyền qua mạng và tái tạo lại ký ức hoặc kiến thức liên quan. Đây là một trong những nguyên lý nền tảng của khoa học thần kinh hiện đại.
