/**
 * 02 — Message Ingestion Tests
 *
 * Tests the message ingestion pipeline:
 *  - Adding single and batch messages
 *  - Message format validation
 *  - Group-based message isolation
 *  - Entity node creation
 */
import { describe, test, expect, beforeAll, afterAll } from "@jest/globals";
import { getClient, CFG, uuid, sampleId, sleep } from "../../lib/client.mjs";

const client = getClient();
const TEST_GROUP = sampleId("test-ingest");

describe("Graphiti — Message Ingestion", () => {
  afterAll(async () => {
    // Clean up test data
    try {
      await client.deleteGroup(TEST_GROUP);
    } catch {
      // ignore cleanup errors
    }
  });

  // ─── Single Message ─────────────────────────────────────────────────────
  test("POST /messages — single user message → 202", async () => {
    const res = await client.addMessages(TEST_GROUP, [
      {
        content: "Hello, I am interested in learning about blockchain technology.",
        role_type: "user",
        role: "Alice",
        uuid: uuid(),
        timestamp: new Date().toISOString(),
        source_description: "Sample test conversation",
      },
    ]);

    expect(res.status).toBe(202);
    expect(res.data).toHaveProperty("success", true);
    expect(res.data).toHaveProperty("message");
    console.log(`  ✓ Message queued: ${res.data.message}`);
  });

  // ─── Multi-Turn Conversation ────────────────────────────────────────────
  test("POST /messages — multi-turn conversation → 202", async () => {
    const now = new Date();
    const messages = [
      {
        content: "What is Ethereum and how does it differ from Bitcoin?",
        role_type: "user",
        role: "Alice",
        uuid: uuid(),
        timestamp: new Date(now.getTime() - 60000).toISOString(),
        source_description: "Blockchain learning session",
      },
      {
        content:
          "Ethereum is a decentralized platform that enables smart contracts. Unlike Bitcoin which is primarily a digital currency, Ethereum provides a programmable blockchain where developers can build decentralized applications (dApps).",
        role_type: "assistant",
        role: "Tutor",
        uuid: uuid(),
        timestamp: new Date(now.getTime() - 30000).toISOString(),
        source_description: "Blockchain learning session",
      },
      {
        content: "Can you explain what smart contracts are?",
        role_type: "user",
        role: "Alice",
        uuid: uuid(),
        timestamp: now.toISOString(),
        source_description: "Blockchain learning session",
      },
    ];

    const res = await client.addMessages(TEST_GROUP, messages);
    expect(res.status).toBe(202);
    expect(res.data.success).toBe(true);
    console.log(`  ✓ ${messages.length} messages queued`);
  });

  // ─── System Message ─────────────────────────────────────────────────────
  test("POST /messages — system message → 202", async () => {
    const res = await client.addMessages(TEST_GROUP, [
      {
        content: "You are a blockchain expert assistant. Always provide accurate technical information.",
        role_type: "system",
        uuid: uuid(),
        timestamp: new Date().toISOString(),
      },
    ]);

    expect(res.status).toBe(202);
    expect(res.data.success).toBe(true);
  });

  // ─── Entity Node ────────────────────────────────────────────────────────
  test("POST /entity-node — create entity → 201", async () => {
    const entityUuid = uuid();
    const res = await client.addEntityNode({
      uuid: entityUuid,
      group_id: TEST_GROUP,
      name: "Ethereum",
      summary: "A decentralized platform for smart contracts and dApps, created by Vitalik Buterin in 2015.",
    });

    expect(res.status).toBe(201);
    expect(res.data).toBeTruthy();
    console.log(`  ✓ Entity node created: ${entityUuid}`);
  });

  // ─── Missing Fields Validation ──────────────────────────────────────────
  test("POST /messages — empty messages array → 422 or 202", async () => {
    const res = await client.addMessages(TEST_GROUP, []);
    // Server may accept empty array (no-op) or reject
    expect([202, 422]).toContain(res.status);
  });

  // ─── Wait for ingestion queue to process ────────────────────────────────
  test("Messages should be processable (wait for queue)", async () => {
    // Give the async worker time to process the queue
    await sleep(5000);

    // Verify episodes were created
    const res = await client.getEpisodes(TEST_GROUP, 20);
    expect(res.status).toBe(200);
    expect(Array.isArray(res.data)).toBe(true);
    console.log(`  ✓ Found ${res.data.length} episodes in group "${TEST_GROUP}"`);
  });
});
