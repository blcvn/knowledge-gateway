/**
 * 01 — Health & Connectivity Tests
 *
 * Verifies that the Zep service is reachable and responding correctly.
 */
import { describe, test, expect } from "@jest/globals";
import { getClient, CFG } from "../../lib/client.mjs";

const client = getClient();

describe("Zep — Health & Connectivity", () => {
  test("GET /healthz → 200", async () => {
    const res = await client.healthcheck();
    expect(res.status).toBe(200);
    console.log(`  ✓ Zep health response: ${JSON.stringify(res.data)}`);
  });

  test("API URL is configured", () => {
    expect(CFG.apiUrl).toBeTruthy();
    expect(CFG.apiUrl).toMatch(/^https?:\/\//);
    console.log(`  ✓ Zep API URL: ${CFG.apiUrl}`);
  });

  test("Unknown route → 404", async () => {
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
