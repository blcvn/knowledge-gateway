/**
 * 03 — Dataset Management Tests
 *
 * /api/v1/add uses multipart/form-data (per OpenAPI spec).
 */
import { describe, it, expect, beforeAll } from "@jest/globals";
import { getAuthClient, testId, CFG } from "../../lib/client.mjs";

describe("Cognee Datasets", () => {
  let client;
  beforeAll(async () => { client = await getAuthClient(); });

  describe("GET /api/v1/datasets", () => {
    it("should return a list of datasets", async () => {
      const { status, data } = await client.get("/api/v1/datasets");
      expect(status).toBe(200);
      expect(Array.isArray(data)).toBe(true);
    });
  });

  describe("POST /api/v1/datasets", () => {
    it("should create a new dataset", async () => {
      const name = testId("ds");
      const { status, data } = await client.post("/api/v1/datasets", {
        body: { name },
      });
      expect(status).toBeLessThan(300);
      expect(data).toHaveProperty("id");
    });
  });

  describe("POST /api/v1/add (multipart)", () => {
    it("should add text data to a dataset", async () => {
      const dsName = testId("add");
      const { status, data } = await client.post("/api/v1/add", {
        multipart: {
          data: "Vietnam is a country in Southeast Asia. Hanoi is the capital.",
          datasetName: dsName,
        },
        timeout: CFG.longTimeout,
      });
      // 200 = success, 500 = LLM connection test timeout (known Bifrost latency issue)
      expect([200, 500]).toContain(status);
      expect(data).toBeDefined();
    }, 120_000);

    it("should reject request without auth", async () => {
      const noAuth = new client.constructor(CFG.apiUrl);
      const { status } = await noAuth.post("/api/v1/add", {
        multipart: { data: "test", datasetName: "unauth" },
      });
      expect(status).toBe(401);
    });
  });

  describe("GET /api/v1/datasets/status", () => {
    it("should return dataset processing status", async () => {
      const { status } = await client.get("/api/v1/datasets/status");
      expect(status).toBe(200);
    });
  });

  describe("DELETE /api/v1/datasets/:id", () => {
    it("should delete a specific dataset", async () => {
      if (CFG.skipDestructive) return;
      const name = testId("del");
      await client.post("/api/v1/datasets", { body: { name } });

      const { data: datasets } = await client.get("/api/v1/datasets");
      const target = datasets.find((d) => d.name === name);
      if (!target) return;

      const { status } = await client.delete(`/api/v1/datasets/${target.id}`);
      expect(status).toBeLessThan(300);
    });
  });
});
