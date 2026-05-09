/**
 * 01 — Health & Connectivity Tests
 *
 * Verifies that the Graphiti service is reachable and responding correctly.
 */
import { describe, test, expect } from "@jest/globals";
import { getClient, CFG } from "../../lib/client.mjs";

const client = getClient();

describe("Graphiti — Health & Connectivity", () => {
  test("GET /healthcheck → 200 with status=healthy", async () => {
    const res = await client.healthcheck();
    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty("status", "healthy");
  });

  test("API URL is configured", () => {
    expect(CFG.apiUrl).toBeTruthy();
    expect(CFG.apiUrl).toMatch(/^https?:\/\//);
    console.log(`  ✓ Graphiti API URL: ${CFG.apiUrl}`);
  });

  test("Unknown route → 404 or 405", async () => {
    const res = await client.get("/this-route-does-not-exist");
    expect([404, 405]).toContain(res.status);
  });

  test("Service responds within timeout", async () => {
    const start = Date.now();
    await client.healthcheck();
    const elapsed = Date.now() - start;
    expect(elapsed).toBeLessThan(CFG.timeout);
    console.log(`  ✓ Response time: ${elapsed}ms`);
  });
});
