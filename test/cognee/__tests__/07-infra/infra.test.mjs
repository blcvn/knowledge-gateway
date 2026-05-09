/**
 * 07 — Infrastructure Integration Tests
 */
import { describe, it, expect, beforeAll } from "@jest/globals";
import { getAuthClient, CFG } from "../../lib/client.mjs";

describe("Cognee Infrastructure Integration", () => {
  let client;
  beforeAll(async () => { client = await getAuthClient(); });

  describe("GET /api/v1/activity/pipeline-runs", () => {
    it("should return pipeline run history", async () => {
      const { status, data } = await client.get("/api/v1/activity/pipeline-runs");
      expect(status).toBe(200);
      expect(data).toBeDefined();
    });
  });

  describe("GET /api/v1/activity/users", () => {
    it("should return activity by users", async () => {
      const { status } = await client.get("/api/v1/activity/users");
      expect(status).toBe(200);
    });
  });

  describe("GET /api/v1/sessions", () => {
    it("should return sessions list", async () => {
      const { status } = await client.get("/api/v1/sessions");
      expect(status).toBe(200);
    });
  });

  describe("GET /api/v1/sessions/stats", () => {
    it("should return session stats", async () => {
      const { status } = await client.get("/api/v1/sessions/stats");
      expect(status).toBe(200);
    });
  });

  describe("GET /api/v1/users/me", () => {
    it("should return current user details", async () => {
      const { status, data } = await client.get("/api/v1/users/me");
      expect(status).toBe(200);
      expect(data).toHaveProperty("id");
      expect(data).toHaveProperty("email");
    });
  });

  describe("Responses API (OpenAI-compatible)", () => {
    it("should accept a response request", async () => {
      const { status } = await client.post("/api/v1/responses/", {
        body: { model: "openai/gpt-4o-mini", input: "What is Cognee?" },
        timeout: CFG.longTimeout,
      });
      expect(status).toBeDefined();
    }, 120_000);
  });
});
