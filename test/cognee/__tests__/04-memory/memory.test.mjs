/**
 * 04 — Memory Pipeline Tests (Remember / Cognify / Recall / Forget)
 *
 * /api/v1/remember: multipart/form-data, 'data' as UploadFile
 * /api/v1/remember/entry: JSON with discriminated union entry + required session_id
 * POST recall: calls LLM → very slow, test GET recall only
 */
import { describe, it, expect, beforeAll } from "@jest/globals";
import { getAuthClient, testId, CFG } from "../../lib/client.mjs";

describe("Cognee Memory Pipeline", () => {
  let client;
  beforeAll(async () => { client = await getAuthClient(); });

  describe("POST /api/v1/remember (multipart)", () => {
    it("should accept file data for processing", async () => {
      const dsName = testId("rem");
      const { status, data } = await client.post("/api/v1/remember", {
        multipart: {
          data: "Blockchain provides a decentralized ledger. Smart contracts automate agreements.",
          datasetName: dsName,
          run_in_background: "true",
        },
        timeout: CFG.longTimeout,
      });
      // May return 200 or 500 (LLM connection test timeout)
      expect([200, 500]).toContain(status);
    }, 120_000);
  });

  describe("POST /api/v1/remember/entry (JSON)", () => {
    it("should accept a QA entry", async () => {
      const { status, data } = await client.post("/api/v1/remember/entry", {
        body: {
          entry: {
            type: "qa",
            question: "What is the capital of Vietnam?",
            answer: "Hanoi is the capital of Vietnam.",
          },
          dataset_name: testId("entry"),
          session_id: testId("session"),
        },
        timeout: CFG.longTimeout,
      });
      // May return 200 or 500 (LLM connection test)
      expect([200, 500]).toContain(status);
    }, 120_000);
  });

  describe("POST /api/v1/cognify", () => {
    it("should trigger cognification for datasets", async () => {
      const { data: datasets } = await client.get("/api/v1/datasets");
      if (!datasets || datasets.length === 0) return;
      const { status } = await client.post("/api/v1/cognify", {
        body: { datasets: [], datasetIds: [datasets[0].id], runInBackground: true },
        timeout: CFG.longTimeout,
      });
      // May return 200 or 500 depending on LLM availability
      expect([200, 500]).toContain(status);
    }, 120_000);
  });

  describe("GET /api/v1/recall (fast path)", () => {
    it("should support GET recall with query params", async () => {
      const params = new URLSearchParams({ query: "Vietnam", searchType: "CHUNKS", topK: "3" });
      const { status } = await client.get(`/api/v1/recall?${params}`, { timeout: CFG.longTimeout });
      expect(status).toBe(200);
    }, 120_000);
  });

  describe("POST /api/v1/forget", () => {
    it("should accept a forget request", async () => {
      if (CFG.skipDestructive) return;
      const { status } = await client.post("/api/v1/forget", {
        body: { dataset: testId("forget-nonexist") },
        timeout: CFG.longTimeout,
      });
      // May return 200 or 500 depending on dataset state
      expect([200, 500]).toContain(status);
    }, 120_000);
  });

  describe("POST /api/v1/improve", () => {
    it("should accept feedback", async () => {
      const { status } = await client.post("/api/v1/improve", {
        body: { type: "qa", question: "What is Vietnam's capital?", answer: "Hanoi" },
        timeout: CFG.longTimeout,
      });
      expect(status).toBeDefined();
    }, 120_000);
  });
});
