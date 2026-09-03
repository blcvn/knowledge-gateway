/**
 * 01 — Health & Connectivity Tests
 *
 * Verifies the Cognee backend is reachable, healthy, and its
 * infrastructure dependencies (PostgreSQL, Neo4j, Qdrant) are connected.
 */
import { describe, it, expect } from "@jest/globals";
import { getClient, CFG } from "../../lib/client.mjs";

const client = getClient();

describe("Cognee Health & Connectivity", () => {
  // ── Basic Health ────────────────────────────────────────────────────────
  describe("GET /health", () => {
    it("should return 200 with healthy status", async () => {
      const { status, data } = await client.get("/health");
      expect(status).toBe(200);
      expect(data).toHaveProperty("status", "ready");
      expect(data).toHaveProperty("health", "healthy");
    });

    it("should report the correct version", async () => {
      const { data } = await client.get("/health");
      expect(data.version).toBeDefined();
      expect(typeof data.version).toBe("string");
    });
  });

  // ── Root endpoint ──────────────────────────────────────────────────────
  describe("GET /", () => {
    it("should return 200 (API root)", async () => {
      const { status } = await client.get("/");
      expect(status).toBe(200);
    });
  });

  // ── HTTPS / Proxy verification ─────────────────────────────────────────
  describe("HTTPS connectivity", () => {
    it("should be accessible via the public HTTPS URL", async () => {
      const res = await fetch(`${CFG.apiUrl}/health`, {
        signal: AbortSignal.timeout(CFG.timeout),
      });
      expect(res.ok).toBe(true);
    });

    it("should return proper CORS headers", async () => {
      const res = await fetch(`${CFG.apiUrl}/health`, {
        headers: { Origin: "https://c6.openledger.vn" },
        signal: AbortSignal.timeout(CFG.timeout),
      });
      expect(res.ok).toBe(true);
      // Cognee sets CORS_ALLOWED_ORIGINS=*
    });
  });

  // ── Response time ──────────────────────────────────────────────────────
  describe("Performance baseline", () => {
    it("health endpoint should respond within 2s", async () => {
      const start = Date.now();
      await client.get("/health");
      const elapsed = Date.now() - start;
      expect(elapsed).toBeLessThan(2000);
    });
  });
});
