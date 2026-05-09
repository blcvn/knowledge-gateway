/**
 * 04 — Memory & Messages Tests
 *
 * Tests the core memory pipeline:
 *  - Add messages to session memory
 *  - Retrieve session memory (context, facts, summary)
 *  - Get raw messages
 *  - Multi-turn conversation memory
 */
import { describe, test, expect, beforeAll, afterAll } from "@jest/globals";
import { getClient, CFG, sampleId, sleep } from "../../lib/client.mjs";

const client = getClient();
const TEST_USER_ID = sampleId(CFG.userPrefix);
const TEST_SESSION_ID = sampleId(CFG.sessionPrefix);

describe("Zep — Memory & Messages", () => {
  // ─── Setup: Create user + session ───────────────────────────────────────
  beforeAll(async () => {
    await client.createUser({
      user_id: TEST_USER_ID,
      first_name: "Memory",
      last_name: "Tester",
    });
    await client.createSession({
      session_id: TEST_SESSION_ID,
      user_id: TEST_USER_ID,
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

  // ─── Add Single Message ─────────────────────────────────────────────────
  test("POST /sessions/:id/memory — add single message → 200", async () => {
    const res = await client.addMemory(TEST_SESSION_ID, [
      {
        content: "Hello! I'm looking for information about blockchain development.",
        role_type: "user",
        role: "Alice",
      },
    ]);

    expect(res.status).toBe(200);
    console.log(`  ✓ Message added to session`);
  });

  // ─── Add Multi-Turn Conversation ────────────────────────────────────────
  test("POST /sessions/:id/memory — multi-turn conversation → 200", async () => {
    const res = await client.addMemory(TEST_SESSION_ID, [
      {
        content: "What programming language should I learn for smart contract development?",
        role_type: "user",
        role: "Alice",
      },
      {
        content:
          "For smart contract development, Solidity is the most popular language for Ethereum. Rust is used for Solana, and Move for Aptos and Sui. I recommend starting with Solidity if you're targeting the Ethereum ecosystem.",
        role_type: "assistant",
        role: "Tutor",
      },
      {
        content: "I'll go with Solidity then. What tools do I need?",
        role_type: "user",
        role: "Alice",
      },
      {
        content:
          "Great choice! You'll need: 1) Hardhat or Foundry as your development framework, 2) MetaMask for wallet interaction, 3) An IDE like VS Code with the Solidity extension, and 4) Some test ETH from a faucet.",
        role_type: "assistant",
        role: "Tutor",
      },
    ]);

    expect(res.status).toBe(200);
    console.log(`  ✓ Multi-turn conversation added`);
  });

  // ─── Wait for processing ────────────────────────────────────────────────
  test("Wait for memory processing", async () => {
    // Zep processes messages asynchronously
    await sleep(10000);
    console.log("  ✓ Waited for processing");
  });

  // ─── Get Session Memory ─────────────────────────────────────────────────
  test("GET /sessions/:id/memory — retrieve memory → 200", async () => {
    const res = await client.getMemory(TEST_SESSION_ID);

    expect(res.status).toBe(200);
    expect(res.data).toBeDefined();

    // Memory response contains: context, facts, messages, relevant_facts, summary
    if (res.data.messages) {
      console.log(`  ✓ Messages: ${res.data.messages.length}`);
    }
    if (res.data.context) {
      console.log(`  ✓ Context: ${res.data.context.substring(0, 100)}...`);
    }
    if (res.data.relevant_facts) {
      console.log(`  ✓ Relevant facts: ${res.data.relevant_facts.length}`);
      for (const f of res.data.relevant_facts.slice(0, 3)) {
        console.log(`    → ${f.fact || f.content || JSON.stringify(f)}`);
      }
    }
    if (res.data.summary) {
      console.log(`  ✓ Summary available: ${!!res.data.summary.content}`);
    }
  });

  // ─── Get Messages with Limit ────────────────────────────────────────────
  test("GET /sessions/:id/memory — last n messages", async () => {
    const res = await client.getMemory(TEST_SESSION_ID, { lastn: 2 });

    expect(res.status).toBe(200);
    if (res.data.messages) {
      expect(res.data.messages.length).toBeLessThanOrEqual(2);
      console.log(`  ✓ Retrieved last ${res.data.messages.length} messages`);
    }
  });

  // ─── Get Raw Messages ──────────────────────────────────────────────────
  test("GET /sessions/:id/messages — raw message list → 200", async () => {
    const res = await client.getMessages(TEST_SESSION_ID, { limit: 20 });

    expect(res.status).toBe(200);
    expect(res.data).toBeDefined();

    if (Array.isArray(res.data.messages || res.data)) {
      const messages = res.data.messages || res.data;
      console.log(`  ✓ Raw messages: ${messages.length}`);
      for (const m of messages.slice(0, 3)) {
        console.log(`    → [${m.role_type}] ${m.content?.substring(0, 60)}...`);
      }
    }
  });

  // ─── System Message ─────────────────────────────────────────────────────
  test("POST /sessions/:id/memory — system message → 200", async () => {
    const res = await client.addMemory(TEST_SESSION_ID, [
      {
        content: "You are a blockchain education assistant. Always provide accurate technical information.",
        role_type: "system",
      },
    ]);

    expect(res.status).toBe(200);
  });
});
