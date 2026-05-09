/**
 * 02 — User Management Tests
 *
 * Tests user CRUD operations:
 *  - Create user
 *  - Get user
 *  - List users
 *  - Delete user
 */
import { describe, test, expect, afterAll } from "@jest/globals";
import { getClient, CFG, sampleId } from "../../lib/client.mjs";

const client = getClient();
const TEST_USER_ID = sampleId(CFG.userPrefix);

describe("Zep — User Management", () => {
  afterAll(async () => {
    try {
      await client.deleteUser(TEST_USER_ID);
    } catch {
      // ignore cleanup errors
    }
  });

  // ─── Create User ────────────────────────────────────────────────────────
  test("POST /api/v2/users — create user → 201", async () => {
    const res = await client.createUser({
      user_id: TEST_USER_ID,
      email: `${TEST_USER_ID}@vnpmemory.dev`,
      first_name: "Test",
      last_name: "User",
      metadata: {
        department: "Engineering",
        role: "Developer",
      },
    });

    expect(res.status).toBe(201);
    expect(res.data).toHaveProperty("user_id", TEST_USER_ID);
    console.log(`  ✓ User created: ${TEST_USER_ID}`);
  });

  // ─── Get User ───────────────────────────────────────────────────────────
  test("GET /api/v2/users/:userId — get user → 200", async () => {
    const res = await client.getUser(TEST_USER_ID);

    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty("user_id", TEST_USER_ID);
    expect(res.data).toHaveProperty("email");
    console.log(`  ✓ User retrieved: ${res.data.user_id}`);
  });

  // ─── Duplicate User ─────────────────────────────────────────────────────
  test("POST /api/v2/users — duplicate user → 400 or 409", async () => {
    const res = await client.createUser({
      user_id: TEST_USER_ID,
    });
    expect([400, 409]).toContain(res.status);
  });

  // ─── List Users ─────────────────────────────────────────────────────────
  test("GET /api/v2/users — list users → 200", async () => {
    const res = await client.listUsers({ limit: 10 });

    expect(res.status).toBe(200);
    // Response is an array of users
    expect(Array.isArray(res.data)).toBe(true);
    console.log(`  ✓ Found ${res.data.length} users`);
  });

  // ─── Get Non-existent User ──────────────────────────────────────────────
  test("GET /api/v2/users/:nonexistent → 404", async () => {
    const res = await client.getUser("user-that-does-not-exist-xxx");
    expect(res.status).toBe(404);
  });
});
