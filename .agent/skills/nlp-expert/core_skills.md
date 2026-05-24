# Core Skills — Natural Language Processing (NLP)

## Text Preprocessing Pipeline

### Stage 1: Document Normalization
```
Raw PRD/URD Text
    → Language detection (langdetect / LLM)
    → Unicode normalization (NFC)
    → Whitespace normalization (collapse multiple spaces, normalize newlines)
    → Section segmentation (regex + heading detection)
    → Output: [{section_type: "Overview|Actors|Functional|API", content: "..."}]
```

### Stage 2: Sentence & Block Segmentation
- **Rule-Based Segmentation:** Split on section headers (`##`, `###`, numbered lists) using markdown structure.
- **Semantic Segmentation:** Use LLM to split dense paragraphs into atomic logical statements (one concept per chunk).
- **Block Classification:** Classify each block as `Actor`, `Flow`, `Rule`, `Constraint`, or `General` using few-shot classification.

### Stage 3: Named Entity Recognition (NER) for Business Domains
Custom NER labels for PRD documents:
| Label | Example | Detection Strategy |
|---|---|---|
| `ACTOR` | "Admin", "Customer", "System" | Capitalized nouns that perform actions |
| `ENTITY` | "Order", "Invoice", "Product" | Domain objects that have fields/state |
| `ACTION` | "approve", "submit", "generate" | Active verbs in functional requirement sentences |
| `STATE` | "Pending", "Approved", "Rejected" | Adjectives/past-participles describing entity status |
| `FIELD` | "email", "amount", "created_at" | Property-like nouns associated with entities |
| `CONSTRAINT` | "must", "shall", "required" | Modal verbs indicating business rules |

## Entity Deduplication & Normalization

### The Deduplication Problem
PRDs often refer to the same concept multiple ways:
- "Đơn hàng", "Order", "đơn", "purchase" → all mean the same Entity
- "Người dùng", "User", "khách hàng", "Customer" → may or may not be the same Actor

### Deduplication Strategy
1. **Translate to canonical language** (English) using LLM for normalization.
2. **Generate embeddings** for each entity mention using `text-embedding-3-small`.
3. **Compute cosine similarity** — pairs with similarity > 0.92 are candidate duplicates.
4. **LLM confirmation:** For each candidate pair, ask LLM: "Are these the same concept in this domain context? Yes/No."
5. **Merge:** Keep the most descriptive name as canonical; alias all others to it.

## Relationship Inference

### Dependency Parsing for Relationship Extraction
Use dependency parsing to extract subject-verb-object triples:
```
"Admin approves the Order" → (Actor: Admin) -[PERFORMS: approves]-> (Entity: Order)
"Order transitions to Approved when payment is confirmed" 
  → (State: Pending) -[TRANSITIONS_TO: {condition: "payment confirmed"}]-> (State: Approved)
```

### Business Rule Extraction (IF/THEN patterns)
Detect conditional patterns in text and extract structured rules:
- Pattern: `"IF <condition> THEN <action>"` → `{condition: "...", action: "..."}`
- Pattern: `"<entity> must <constraint>"` → `{entity: "...", constraint: "...", type: "MUST"}`
- Pattern: `"only <actor> can <action>"` → `{actor: "...", action: "...", permission: "EXCLUSIVE"}`
