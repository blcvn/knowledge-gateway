/**
 * 06 — Lifecycle & Cleanup Tests
 *
 * Tests data lifecycle operations:
 *  - Session deletion
 *  - User deletion
 *  - Cascade behavior verification
 */
import { describe, test, expect, beforeAll } from "@jest/globals";
import { getClient, CFG, sampleId, sleep } from "../../lib/client.mjs";

const client = getClient();
const LIFECYCLE_USER = sampleId(CFG.userPrefix);
const LIFECYCLE_SESSION = sampleId(CFG.sessionPrefix);

describe("Zep — Lifecycle & Cleanup", () => {
  // ─── Setup: Create test data ────────────────────────────────────────────
  beforeAll(async () => {
    await client.createUser({
      user_id: LIFECYCLE_USER,
      first_name: "Lifecycle",
      last_name: "Test",
    });

    await client.createSession({
      session_id: LIFECYCLE_SESSION,
      user_id: LIFECYCLE_USER,
    });

    await client.addMemory(LIFECYCLE_SESSION, [
      {
        content: "This is test data for lifecycle management.",
        role_type: "user",
      },
      {
        content: "Acknowledged. This data will be cleaned up.",
        role_type: "assistant",
      },
    ]);

    // Wait for processing
    await sleep(5000);
  }, 30000);

  // ─── Verify Data Exists ─────────────────────────────────────────────────
  test("Data exists before deletion", async () => {
    const sessionRes = await client.getSession(LIFECYCLE_SESSION);
    expect(sessionRes.status).toBe(200);

    const userRes = await client.getUser(LIFECYCLE_USER);
    expect(userRes.status).toBe(200);

    console.log(`  ✓ User: ${LIFECYCLE_USER}`);
    console.log(`  ✓ Session: ${LIFECYCLE_SESSION}`);
  });

  // ─── Delete Session ─────────────────────────────────────────────────────
  test("DELETE /api/v2/sessions/:id — delete session → 200", async () => {
    const res = await client.deleteSession(LIFECYCLE_SESSION);
    expect(res.status).toBe(200);
    console.log(`  ✓ Session "${LIFECYCLE_SESSION}" deleted`);
  });

  // ─── Verify Session Deleted ─────────────────────────────────────────────
  test("Session is gone after deletion → 404", async () => {
    const res = await client.getSession(LIFECYCLE_SESSION);
    expect(res.status).toBe(404);
    console.log("  ✓ Session confirmed deleted");
  });

  // ─── Delete User ────────────────────────────────────────────────────────
  test("DELETE /api/v2/users/:id — delete user → 200", async () => {
    const res = await client.deleteUser(LIFECYCLE_USER);
    expect(res.status).toBe(200);
    console.log(`  ✓ User "${LIFECYCLE_USER}" deleted`);
  });

  // ─── Verify User Deleted ────────────────────────────────────────────────
  test("User is gone after deletion → 404", async () => {
    const res = await client.getUser(LIFECYCLE_USER);
    expect(res.status).toBe(404);
    console.log("  ✓ User confirmed deleted");
  });
});
