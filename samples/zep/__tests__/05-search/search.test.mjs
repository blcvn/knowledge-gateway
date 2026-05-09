/**
 * 05 — Graph Search Tests
 *
 * Tests the knowledge graph search functionality.
 * NOTE: Requires data to have been ingested first (messages → graph).
 */
import { describe, test, expect, beforeAll, afterAll } from "@jest/globals";
import { getClient, CFG, sampleId, sleep } from "../../lib/client.mjs";

const client = getClient();
const TEST_USER_ID = sampleId(CFG.userPrefix);
const TEST_SESSION_ID = sampleId(CFG.sessionPrefix);

describe("Zep — Graph Search", () => {
  // ─── Seed data ──────────────────────────────────────────────────────────
  beforeAll(async () => {
    // Create user and session
    await client.createUser({
      user_id: TEST_USER_ID,
      first_name: "Graph",
      last_name: "Tester",
    });
    await client.createSession({
      session_id: TEST_SESSION_ID,
      user_id: TEST_USER_ID,
    });

    // Seed rich conversation for graph extraction
    await client.addMemory(TEST_SESSION_ID, [
      {
        content: "I'm Minh, a senior engineer at OpenLedger based in Ho Chi Minh City. I work on the VNP Memory platform.",
        role_type: "user",
        role: "Minh",
      },
      {
        content:
          "Hello Minh! Great to hear you're working on VNP Memory at OpenLedger. As a senior engineer in HCMC, you must be well-versed in the tech stack. VNP Memory uses Neo4j, PostgreSQL, and Qdrant for its infrastructure.",
        role_type: "assistant",
        role: "Assistant",
      },
      {
        content:
          "Yes! My team lead is Tran Duc. We use Go and Python primarily. Our deployment runs on Docker Compose with Nginx as the reverse proxy.",
        role_type: "user",
        role: "Minh",
      },
      {
        content:
          "That sounds like a solid setup! Go for the KGS Platform and Python for the AI services like Graphiti and Cognee. Docker Compose makes local development much easier.",
        role_type: "assistant",
        role: "Assistant",
      },
    ]);

    // Wait for graph construction
    console.log("  ⏳ Waiting for graph construction (20s)...");
    await sleep(20000);
  }, 120000);

  afterAll(async () => {
    try {
      await client.deleteSession(TEST_SESSION_ID);
      await client.deleteUser(TEST_USER_ID);
    } catch {
      // ignore
    }
  });

  // ─── User-scoped Graph Search ───────────────────────────────────────────
  test("POST /api/v2/graph/search — user-scoped search → 200", async () => {
    const res = await client.searchGraph("What does Minh work on?", {
      user_id: TEST_USER_ID,
    });

    expect(res.status).toBe(200);
    expect(res.data).toBeDefined();

    if (res.data.edges?.length > 0) {
      console.log(`  ✓ Found ${res.data.edges.length} edges`);
      for (const e of res.data.edges.slice(0, 3)) {
        console.log(`    → ${e.fact || e.name}`);
      }
    }
    if (res.data.nodes?.length > 0) {
      console.log(`  ✓ Found ${res.data.nodes.length} nodes`);
      for (const n of res.data.nodes.slice(0, 3)) {
        console.log(`    → ${n.name}: ${n.summary || "(no summary)"}`);
      }
    }
    if (res.data.context) {
      console.log(`  ✓ Context: ${res.data.context.substring(0, 150)}...`);
    }
  });

  // ─── Technology Stack Search ────────────────────────────────────────────
  test("POST /api/v2/graph/search — technology query", async () => {
    const res = await client.searchGraph("What technologies and databases are used?", {
      user_id: TEST_USER_ID,
    });

    expect(res.status).toBe(200);
    console.log(`  ✓ Tech search: ${res.data.edges?.length || 0} edges, ${res.data.nodes?.length || 0} nodes`);
  });

  // ─── People Search ─────────────────────────────────────────────────────
  test("POST /api/v2/graph/search — people and roles", async () => {
    const res = await client.searchGraph("Who is the team lead?", {
      user_id: TEST_USER_ID,
    });

    expect(res.status).toBe(200);
    console.log(`  ✓ People search: ${res.data.edges?.length || 0} edges`);
  });

  // ─── Search with Limit ─────────────────────────────────────────────────
  test("POST /api/v2/graph/search — with limit", async () => {
    const res = await client.searchGraph("engineering", {
      user_id: TEST_USER_ID,
      limit: 3,
    });

    expect(res.status).toBe(200);
  });
});
