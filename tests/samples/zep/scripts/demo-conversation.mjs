#!/usr/bin/env node
/**
 * Demo: Conversation Memory with Zep
 *
 * Demonstrates the full Zep memory pipeline:
 *   User → Session → Messages → Memory (context + facts + summary)
 *
 * Usage:
 *   node scripts/demo-conversation.mjs
 */
import { ZepClient, sampleId, sleep, pp, CFG } from "../lib/client.mjs";

const client = new ZepClient();
const USER_ID = sampleId("demo-user");
const SESSION_ID = sampleId("demo-session");

async function main() {
  console.log("╔══════════════════════════════════════════════════════════════╗");
  console.log("║       Zep — Conversation Memory Demo                        ║");
  console.log("╚══════════════════════════════════════════════════════════════╝");
  console.log(`  API:     ${CFG.apiUrl}`);
  console.log(`  User:    ${USER_ID}`);
  console.log(`  Session: ${SESSION_ID}\n`);

  // ── Step 1: Health check ────────────────────────────────────────────────
  const health = await client.healthcheck();
  if (health.status !== 200) {
    console.error("❌ Zep is not healthy:", health.data);
    process.exit(1);
  }
  console.log("✅ Zep is healthy\n");

  // ── Step 2: Create user ─────────────────────────────────────────────────
  console.log("👤 Creating user...");
  const userRes = await client.createUser({
    user_id: USER_ID,
    email: `${USER_ID}@demo.vnpmemory.dev`,
    first_name: "Nguyen",
    last_name: "Van An",
    metadata: { department: "Engineering", team: "Platform" },
  });
  pp("User Created", userRes.data);

  // ── Step 3: Create session ──────────────────────────────────────────────
  console.log("\n💬 Creating session...");
  const sessionRes = await client.createSession({
    session_id: SESSION_ID,
    user_id: USER_ID,
  });
  pp("Session Created", sessionRes.data);

  // ── Step 4: Add conversation messages ───────────────────────────────────
  console.log("\n📝 Adding conversation...");
  await client.addMemory(SESSION_ID, [
    {
      content: "Hi! I'm An, a backend developer at OpenLedger. I specialize in Go and Kubernetes.",
      role_type: "user",
      role: "An",
    },
    {
      content:
        "Welcome An! Great to hear you work with Go and Kubernetes at OpenLedger. Those are excellent skills for backend infrastructure. What project are you working on?",
      role_type: "assistant",
      role: "Assistant",
    },
    {
      content:
        "I'm building the KGS Platform — a Knowledge Graph Service. It uses Neo4j for graph storage, PostgreSQL for relational data, and Qdrant for vector search.",
      role_type: "user",
      role: "An",
    },
    {
      content:
        "The KGS Platform sounds impressive! Using Neo4j + PostgreSQL + Qdrant gives you a powerful multi-model data architecture. Are you using the Kratos framework for the Go service?",
      role_type: "assistant",
      role: "Assistant",
    },
    {
      content:
        "Yes, exactly! Kratos with gRPC and HTTP transport. Our team lead Tran Duc has been guiding the architecture. We deploy everything via Docker Compose.",
      role_type: "user",
      role: "An",
    },
  ]);
  console.log("  ✓ 5 messages added\n");

  // ── Step 5: Wait for processing ─────────────────────────────────────────
  console.log("⏳ Waiting for Zep to process messages (15s)...\n");
  await sleep(15000);

  // ── Step 6: Retrieve memory ─────────────────────────────────────────────
  console.log("🧠 Retrieving session memory...");
  const memoryRes = await client.getMemory(SESSION_ID);
  pp("Memory Response", {
    context: memoryRes.data.context?.substring(0, 300) + "...",
    facts_count: memoryRes.data.facts?.length,
    messages_count: memoryRes.data.messages?.length,
    relevant_facts_count: memoryRes.data.relevant_facts?.length,
    has_summary: !!memoryRes.data.summary?.content,
  });

  if (memoryRes.data.relevant_facts?.length > 0) {
    console.log("\n📋 Relevant Facts:");
    for (const f of memoryRes.data.relevant_facts) {
      console.log(`   → ${f.fact || f.content}`);
    }
  }

  // ── Step 7: Get raw messages ────────────────────────────────────────────
  console.log("\n📨 Raw messages in session:");
  const msgsRes = await client.getMessages(SESSION_ID, { limit: 10 });
  const messages = msgsRes.data.messages || msgsRes.data || [];
  for (const m of messages.slice(0, 5)) {
    console.log(`   [${m.role_type}] ${m.content?.substring(0, 70)}...`);
  }

  // ── Cleanup ─────────────────────────────────────────────────────────────
  console.log("\n🧹 Cleaning up...");
  await client.deleteSession(SESSION_ID);
  await client.deleteUser(USER_ID);
  console.log("✅ Demo complete!\n");
}

main().catch((err) => {
  console.error("❌ Demo failed:", err.message);
  // Try cleanup on failure
  client.deleteSession(SESSION_ID).catch(() => {});
  client.deleteUser(USER_ID).catch(() => {});
  process.exit(1);
});
