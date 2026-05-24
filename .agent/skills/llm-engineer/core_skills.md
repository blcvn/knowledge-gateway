# Core Skills — LLM Engineering & Prompt Design

## Prompt Engineering

### Prompt Types & When to Use Each
| Type | Use Case | Example |
|---|---|---|
| **Zero-shot** | Well-defined task, LLM has strong priors | Simple classification of a text block |
| **Few-shot** | Domain-specific patterns LLM may not know | Extract business entities with custom schema |
| **Chain-of-Thought (CoT)** | Complex reasoning, multi-step logic | Infer relationships between extracted entities |
| **Structured Output (JSON mode)** | Any production pipeline output | All extraction tasks — always use JSON mode |
| **System + User split** | Persistent role/context vs per-request input | Agent persona in system; document in user turn |

### Prompt Design Principles
1. **Be Explicit, Not Implicit:** State the exact output format, every required field, and all edge case behaviors. Do not assume the model "knows" your intent.
2. **Constrain the Output Schema:** Always provide the exact JSON schema the model must follow. Include field names, types, and examples.
3. **Provide Negative Examples:** Tell the model what it should NOT do. This dramatically reduces hallucination of invented fields.
4. **One Task per Prompt:** A prompt that does two things does both poorly. Split complex tasks into sequential prompts.
5. **Seed with Examples:** Provide 2-5 high-quality examples of input → output pairs. Few-shot dramatically improves consistency.

### Production Prompt Template
```
[SYSTEM]
You are a {role}. Your task is to {precise task description}.

RULES:
- {Rule 1 — what to do}
- {Rule 2 — what NOT to do}
- If {edge case}, then {behavior}

OUTPUT FORMAT:
Return ONLY valid JSON matching this schema:
{
  "field_name": "type | description",
  ...
}

[USER]
{input document / text}
```

## Output Parsing & Validation
- **Always parse with schema validation:** Use Pydantic (Python) or `encoding/json` with strict structs (Go). Reject any output that does not match the schema.
- **Retry on parse failure:** If the model returns malformed JSON, retry once with an error correction prompt before failing.
- **Confidence Scoring:** For extractions where accuracy is critical, ask the model to include a `confidence: 0.0-1.0` score per entity. Reject low-confidence items for human review.
- **Hallucination Guard:** After extraction, cross-validate extracted entities against the source text. Any entity with zero text evidence is flagged as a potential hallucination.

## LLM Cost & Performance Optimization
- **Prompt Caching:** Cache prompt+response pairs for identical inputs (e.g., same document segment). Reduces cost and latency dramatically for repeated runs.
- **Token Counting:** Count tokens before sending. Chunk documents that exceed context window limits using sliding windows with overlap.
- **Model Tiering:** Use smaller, cheaper models (e.g., GPT-4o-mini) for preprocessing and classification tasks. Reserve large models (GPT-4o, Claude 3.5) for complex reasoning and extraction.
- **Batch API:** For offline pipeline stages (not user-facing), use the OpenAI/Anthropic Batch API for 50% cost reduction.

## Evaluation & Testing
- **Test Set:** Maintain a labeled ground-truth dataset of PRD segments with expected extraction outputs. Run every prompt change against this set.
- **Metrics:** Track Precision, Recall, and F1 per entity type (Actor, Entity, Action, Rule).
- **Regression Testing:** Any prompt change that decreases F1 by more than 2% on the test set must be reviewed and justified before deployment.
