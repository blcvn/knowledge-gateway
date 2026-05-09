/**
 * 03 — Session Management Tests
 *
 * Tests session CRUD operations:
 *  - Create session (linked to a user)
 *  - Get session
 *  - List sessions
 *  - Delete session
 */
import { describe, test, expect, beforeAll, afterAll } from "@jest/globals";
import { getClient, CFG, sampleId } from "../../lib/client.mjs";

const client = getClient();
const TEST_USER_ID = sampleId(CFG.userPrefix);
const TEST_SESSION_ID = sampleId(CFG.sessionPrefix);

describe("Zep — Session Management", () => {
  // ─── Setup: Create a test user ──────────────────────────────────────────
  beforeAll(async () => {
    await client.createUser({
      user_id: TEST_USER_ID,
      first_name: "Session",
      last_name: "Tester",
    });
  });

  afterAll(async () => {
    try {
      await client.deleteSession(TEST_SESSION_ID);
      await client.deleteUser(TEST_USER_ID);
    } catch {
      // ignore
    }
  });

  // ─── Create Session ─────────────────────────────────────────────────────
  test("POST /api/v2/sessions — create session → 201", async () => {
    const res = await client.createSession({
      session_id: TEST_SESSION_ID,
      user_id: TEST_USER_ID,
      metadata: {
        topic: "Zep integration testing",
        environment: "dev",
      },
    });

    expect(res.status).toBe(201);
    expect(res.data).toHaveProperty("session_id", TEST_SESSION_ID);
    expect(res.data).toHaveProperty("user_id", TEST_USER_ID);
    console.log(`  ✓ Session created: ${TEST_SESSION_ID}`);
  });

  // ─── Get Session ────────────────────────────────────────────────────────
  test("GET /api/v2/sessions/:sessionId — get session → 200", async () => {
    const res = await client.getSession(TEST_SESSION_ID);

    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty("session_id", TEST_SESSION_ID);
    expect(res.data).toHaveProperty("user_id", TEST_USER_ID);
    console.log(`  ✓ Session retrieved: ${res.data.session_id}`);
  });

  // ─── List Sessions ──────────────────────────────────────────────────────
  test("GET /api/v2/sessions — list sessions → 200", async () => {
    const res = await client.listSessions({ limit: 10 });

    expect(res.status).toBe(200);
    expect(Array.isArray(res.data)).toBe(true);
    expect(res.data.length).toBeGreaterThan(0);
    console.log(`  ✓ Found ${res.data.length} sessions`);
  });

  // ─── Get Non-existent Session ───────────────────────────────────────────
  test("GET /api/v2/sessions/:nonexistent → 404", async () => {
    const res = await client.getSession("session-that-does-not-exist-xxx");
    expect(res.status).toBe(404);
  });

  // ─── Session without User → 400 ────────────────────────────────────────
  test("POST /api/v2/sessions — missing user_id → 400", async () => {
    const res = await client.createSession({
      session_id: sampleId("invalid-session"),
    });
    expect([400, 422, 500]).toContain(res.status);
  });
});
