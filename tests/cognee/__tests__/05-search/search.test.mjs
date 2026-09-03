/**
 * 05 — Search & Retrieval Tests
 *
 * POST /search and POST /recall call the LLM internally and can hang
 * for 2+ minutes. We test only GET endpoints which are fast.
 */
import { describe, it, expect, beforeAll } from "@jest/globals";
import { getAuthClient, CFG } from "../../lib/client.mjs";

describe("Cognee Search & Retrieval", () => {
  let client;
  beforeAll(async () => { client = await getAuthClient(); });

  describe("GET /api/v1/search", () => {
    it("should search with CHUNKS type", async () => {
      const params = new URLSearchParams({ query: "tech", searchType: "CHUNKS", topK: "3" });
      const { status, data } = await client.get(`/api/v1/search?${params}`, { timeout: CFG.longTimeout });
      expect(status).toBe(200);
      expect(data).toBeDefined();
    }, 120_000);
  });

  describe("GET /api/v1/recall", () => {
    it("should recall with query params", async () => {
      const params = new URLSearchParams({ query: "Vietnam", searchType: "CHUNKS", topK: "3" });
      const { status, data } = await client.get(`/api/v1/recall?${params}`, { timeout: CFG.longTimeout });
      expect(status).toBe(200);
      expect(data).toBeDefined();
    }, 120_000);
  });

  describe("GET /api/v1/datasets/:id/graph", () => {
    it("should return graph for existing dataset", async () => {
      const { data: datasets } = await client.get("/api/v1/datasets");
      if (!datasets || datasets.length === 0) return;
      const { status, data } = await client.get(`/api/v1/datasets/${datasets[0].id}/graph`, {
        timeout: CFG.longTimeout,
      });
      expect(status).toBe(200);
      if (data && typeof data === "object") {
        expect(data).toHaveProperty("nodes");
        expect(data).toHaveProperty("edges");
      }
    }, 120_000);
  });
});
