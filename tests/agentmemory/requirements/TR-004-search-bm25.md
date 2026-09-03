# TR-004: BM25 Search Engine Test Requirements

**Module:** SearchIndex (BM25)  
**Nguồn:** SRS §3.5 (FR-SEARCH-001..005), Architecture §5.1, TDD §3.1  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## Mô tả

Test requirements cho BM25 inverted index — full-text keyword search engine với CJK support, stemming và synonym expansion.

**File:** `src/state/search-index.ts`

---

## TR-004-BM25-001 — Basic exact-term search
🔴 P0 | `[UNIT]` | **FR-SEARCH-001**

**Given:** SearchIndex với 1 document:
```
id: "obs1"
title: "Fixed N+1 query in UserRepository"
facts: ["Changed findAll() to use eager loading"]
concepts: ["database", "ORM", "query", "N+1"]
```
**When:** `idx.search("query", 10)` được gọi  
**Then:**
- `results[0].obsId = "obs1"`
- `results[0].score > 0`

**Traceability:** TDD §3.1, SRS §3.5 FR-SEARCH-001

---

## TR-004-BM25-002 — Synonym expansion: "auth" matches "authentication"
🔴 P0 | `[UNIT]` | **FR-SEARCH-001**

**Given:** Document có concept "authentication"  
**When:** `idx.search("auth")` được gọi  
**Then:**
- Document được tìm thấy (synonym expansion)
- Score = BM25_score × 0.7 (30% discount cho synonym)

**Traceability:** TDD §3.1, Architecture §5.1

---

## TR-004-BM25-003 — Synonym expansion: "database" matches "db"
🟠 P1 | `[UNIT]`

**Given:** Document có term "database"  
**When:** `idx.search("db")`  
**Then:** Document được tìm thấy qua synonym expansion

**Traceability:** TDD §3.1, Architecture §5.1

---

## TR-004-BM25-004 — Prefix matching
🟠 P1 | `[UNIT]`

**Given:** Document có term "authentication"  
**When:** `idx.search("authen")` (prefix)  
**Then:**
- Document được tìm thấy
- Score = BM25_score × 0.5 (50% discount cho prefix)

**Traceability:** TDD §3.1, Architecture §5.1

---

## TR-004-BM25-005 — Porter stemmer: "running" matches "run"
🟠 P1 | `[UNIT]`

**Given:** Document có term "running tests"  
**When:** `idx.search("run tests")`  
**Then:** Document được tìm thấy (Porter stemmer normalize cả 2)

**Traceability:** TDD §3.1, Architecture §5.1

---

## TR-004-BM25-006 — CJK text: Japanese segmentation
🟠 P1 | `[UNIT]` | **FR-SEARCH-001**

**Given:** Document với Japanese text  
**When:** `idx.search("日本語テスト")`  
**Then:** CJK bigram segmentation được áp dụng, document được tìm thấy

**Traceability:** SRS §3.5 FR-SEARCH-001, TDD §3.1

---

## TR-004-BM25-007 — Multi-term query ranking
🔴 P0 | `[UNIT]`

**Given:** 3 documents:
- doc1: có cả "database" và "performance" (2 terms)
- doc2: chỉ có "database" (1 term)
- doc3: không có term nào

**When:** `idx.search("database performance")`  
**Then:** `results` xếp hạng: doc1 > doc2 > doc3

**Traceability:** TDD §3.1 BM25 scoring

---

## TR-004-BM25-008 — IDF: rare terms có weight cao hơn common terms
🟠 P1 | `[UNIT]`

**Given:** 100 documents, 99 có từ "the", 1 có từ "xenova"  
**When:** `idx.search("xenova")`  
**Then:**
- Document với "xenova" được rank cao nhất
- IDF("xenova") >> IDF("the")

**Traceability:** TDD §3.1 BM25 formula, Architecture §5.1

---

## TR-004-BM25-009 — Document length normalization
🟠 P1 | `[UNIT]`

**Given:** 2 documents với cùng term frequency cho "auth":
- doc1: 500 terms total (long document)
- doc2: 10 terms total (short document)

**When:** `idx.search("auth")`  
**Then:** doc2 được rank cao hơn doc1 (length normalization penalizes long docs)

**Traceability:** TDD §3.1 BM25 formula (b=0.75)

---

## TR-004-BM25-010 — Limit parameter
🟡 P2 | `[UNIT]`

**Given:** 20 matching documents  
**When:** `idx.search("query", 5)`  
**Then:** Chỉ trả về 5 results (top-5 by score)

**Traceability:** TDD §3.1

---

## TR-004-BM25-011 — Empty query
🟡 P2 | `[UNIT]`

**Given:** SearchIndex với data  
**When:** `idx.search("", 10)`  
**Then:** Trả về empty array `[]`, không throw error

**Traceability:** TDD §3.1

---

## TR-004-BM25-012 — add() và remove()
🔴 P0 | `[UNIT]`

**Given:** Document được add vào index  
**When:** `idx.remove(obsId)` được gọi  
**Then:**
- Document không còn xuất hiện trong search results
- Inverted index được cập nhật
- `totalDocLength` giảm đúng

**Traceability:** TDD §3.1

---

## TR-004-BM25-013 — Serialization v2 format
🔴 P0 | `[UNIT]`

**Given:** SearchIndex với 10 documents  
**When:** `idx.serialize()` rồi `SearchIndex.deserialize(json)` trên index mới  
**Then:**
- Tất cả documents xuất hiện trong search results
- Scores giống như trước serialization
- Format JSON có key `"v": 2`

**Traceability:** TDD §3.1, Architecture §5.1

---

## TR-004-BM25-014 — Serialization: ~50KB per 1000 docs
🟡 P2 | `[PERF]`

**Given:** 1000 documents được add  
**When:** `idx.serialize()`  
**Then:** JSON string size ≤ 55KB

**Traceability:** Architecture §5.1

---

## TR-004-BM25-015 — Concurrent add thread safety
🔴 P0 | `[UNIT]`

**Given:** SearchIndex instance  
**When:** 50 documents được add đồng thời  
**Then:**
- `entries.size = 50`
- Không có lost updates
- `search()` trả về kết quả đúng sau khi tất cả add hoàn thành

**Traceability:** TDD §3.1, Architecture §3.4

---

## TR-004-BM25-016 — Rebuild index từ KV
🟠 P1 | `[INT]`

**Given:** KV có 100 CompressedObservations, SearchIndex empty  
**When:** `rebuildIndex(kv)` chạy  
**Then:**
- 100 documents được add vào index
- Search hoạt động đúng cho tất cả

**Traceability:** SRS §4.1, Architecture §9.3

---

## TR-004-BM25-017 — Index rebuild performance
🟡 P2 | `[PERF]` | **FR-SEARCH-005**

**Given:** 1000 observations trong KV  
**When:** Full index rebuild  
**Then:** Hoàn thành trong < 30 giây

**Traceability:** SRS §4.1

---

## TR-004-BM25-018 — Sorting: sortedTerms invalidation
🟡 P2 | `[UNIT]`

**Given:** `sortedTerms` đã được cache  
**When:** Một document mới được add  
**Then:** `sortedTerms = null` (invalidated), được rebuild lazily khi prefix search cần

**Traceability:** TDD §3.1

---

## TR-004-BM25-019 — Search: semantic relevance (functional test)
🔴 P0 | `[UNIT]` | **FR-SEARCH-001**

**Given:** Document với title="Fixed N+1 database query in UserRepository"  
**When:** `idx.search("database performance optimization")`  
**Then:** Document xuất hiện trong top-5 results

**Traceability:** SRS §3.5, TDD §14.2
