# P7 — AI Power User

> **Vai trò:** Người dùng cuối tương tác với AI assistant hàng ngày cho công việc, học tập, sáng tạo.
> **Tần suất:** Hàng ngày, nhiều giờ/ngày.

---

## Pain Points

### PP-P7-01 — AI quên mọi thứ sau mỗi session — không cá nhân hóa

**Mô tả:**
User đã dùng AI assistant 6 tháng, nhưng AI vẫn "không biết gì" về họ. Phải nhắc đi nhắc lại: "Tôi là developer, tôi dùng macOS, tôi prefer TypeScript hơn JavaScript, tôi đang làm project X..."

**Hậu quả thực tế:**
- Cảm giác frustrating: AI "ngốc" dù đã dùng lâu
- Mất thời gian "briefing" AI mỗi session
- AI đưa ra suggestions không phù hợp với context của user

**Features giải quyết:**
- [F05] Profile Memory (Memobase YOLO Engine):
  - Tự động extract từ conversations: preference/fact/goal/habit
  - "Tôi prefer TypeScript" → lưu vào profile category=preference
  - Mỗi lần chat mới → context assembly inject profile relevant
- [F07] Adaptive Memory: facts về user tự động update khi có thông tin mới

---

### PP-P7-02 — Không thể "dạy" AI nhớ điều gì đó vĩnh viễn

**Mô tả:**
User muốn nói "Hãy nhớ: tôi vegetarian, không suggest món thịt bao giờ." Nhưng sau 1 tuần AI lại suggest thịt. Không có mechanism để user biết AI đang nhớ gì và enforce rules.

**Features giải quyết:**
- [F07] Supermemory: `POST /v1/sm/memories` với `forgetAfter=never` — permanent memory
- [F05] Memobase: profile category=fact — high-persistence facts
- [F18] User Profiles Console: user tự xem/edit profile của mình

---

### PP-P7-03 — Không thể "quên" thông tin đã lưu

**Mô tả:**
User thay đổi ý kiến, hoặc muốn AI quên thông tin nhạy cảm đã chia sẻ. Nhưng không có cách nào để xóa specific memory — chỉ có thể "bắt đầu chat mới" (xóa hết).

**Features giải quyết:**
- [F01] `POST /v1/memory/forget` — xóa specific memory
- [F18] User Profiles Console: xem và delete specific profile entries
- [F22] GDPR self-service: user request xóa toàn bộ data của mình

---

### PP-P7-04 — Không minh bạch về AI đang biết gì về mình

**Mô tả:**
"AI đang nhớ gì về tôi?" — không có câu trả lời. Users không biết AI đang có data gì, từ nguồn nào. Gây lo ngại về privacy.

**Features giải quyết:**
- [F18] User Profiles Console:
  - Xem structured profile: `{key, value, category, score}`
  - Xem event timeline: khi nào AI học được gì
  - `GET /v1/console/profiles/{user_id}/events`
- [F16] Memory Explorer: xem tất cả memories liên quan đến user

---

## Summary

| Pain | Giải pháp |
|---|---|
| AI không nhớ / không cá nhân hóa | Memobase profile (YOLO engine) |
| Không thể dạy AI nhớ vĩnh viễn | Supermemory permanent memory |
| Không thể xóa specific memory | /v1/memory/forget + profile editor |
| Không biết AI biết gì về mình | User Profiles Console + Memory Explorer |
