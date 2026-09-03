/**
 * 06 — Settings & Configuration Tests
 */
import { describe, it, expect, beforeAll } from "@jest/globals";
import { getAuthClient, testId, CFG } from "../../lib/client.mjs";

describe("Cognee Settings & Configuration", () => {
  let client;
  beforeAll(async () => { client = await getAuthClient(); });

  describe("GET /api/v1/settings", () => {
    it("should return current settings", async () => {
      const { status, data } = await client.get("/api/v1/settings");
      expect(status).toBe(200);
      expect(data).toHaveProperty("llm");
      expect(data).toHaveProperty("vectorDb");
    });
  });

  describe("GET /api/v1/ontologies", () => {
    it("should return ontologies list", async () => {
      const { status, data } = await client.get("/api/v1/ontologies");
      expect(status).toBe(200);
      expect(data).toBeDefined();
    });
  });

  describe("User Configuration", () => {
    it("should store user configuration", async () => {
      const { status } = await client.post("/api/v1/configuration/store_user_configuration", {
        body: { name: testId("cfg"), config: { theme: "dark" } },
      });
      expect(status).toBeLessThan(300);
    });

    it("should retrieve user configurations", async () => {
      const { status } = await client.get("/api/v1/configuration/get_user_configuration/");
      expect(status).toBe(200);
    });
  });
});
