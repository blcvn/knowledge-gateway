#!/usr/bin/env node
/**
 * Demo: Memory Retrieval Patterns
 *
 * Demonstrates different memory retrieval options:
 *  - Full memory (context + facts + summary)
 *  - Last N messages
 *  - Multi-session user memory
 *
 * Usage:
 *   node scripts/demo-memory-retrieval.mjs
 */
import { ZepClient, sampleId, sleep, pp, CFG } from "../lib/client.mjs";

const client = new ZepClient();
const USER_ID = sampleId("demo-mem-user");
const SESSION_A = sampleId("demo-session-a");
const SESSION_B = sampleId("demo-session-b");

async function main() {
  console.log("╔══════════════════════════════════════════════════════════════╗");
  console.log("║       Zep — Memory Retrieval Patterns Demo                  ║");
  console.log("╚══════════════════════════════════════════════════════════════╝");
  console.log(`  API:       ${CFG.apiUrl}`);
  console.log(`  User:      ${USER_ID}`);
  console.log(`  Session A: ${SESSION_A}`);
  console.log(`  Session B: ${SESSION_B}\n`);

  // ── Health check ────────────────────────────────────────────────────────
  const health = await client.healthcheck();
  if (health.status !== 200) {
    console.error("❌ Zep is not healthy");
    process.exit(1);
  }
  console.log("✅ Zep is healthy\n");

  // ── Setup ───────────────────────────────────────────────────────────────
  console.log("🏗️  Setting up user and sessions...");
  await client.createUser({
    user_id: USER_ID,
    first_name: "Demo",
    last_name: "User",
  });

  await client.createSession({ session_id: SESSION_A, user_id: USER_ID });
  await client.createSession({ session_id: SESSION_B, user_id: USER_ID });

  // ── Session A: Technical discussion ─────────────────────────────────────
  console.log("\n📝 Session A — Technical discussion...");
  await client.addMemory(SESSION_A, [
    {
      content: "I've been coding in Python for 5 years and recently started learning Rust.",
      role_type: "user",
    },
    {
      content: "That's a great combination! Python is excellent for rapid prototyping and AI/ML, while Rust provides memory safety and performance for systems programming.",
      role_type: "assistant",
    },
    {
      content: "I'm particularly interested in building high-performance web servers in Rust with Actix-web.",
      role_type: "user",
    },
  ]);

  // ── Session B: Personal preferences ─────────────────────────────────────
  console.log("📝 Session B — Personal preferences...");
  await client.addMemory(SESSION_B, [
    {
      content: "I prefer working remotely and I'm based in Da Nang. I usually work from 9 AM to 6 PM.",
      role_type: "user",
    },
    {
      content: "Da Nang is a great city for remote work! It has a growing tech scene. Your regular hours should make collaboration easy.",
      role_type: "assistant",
    },
    {
      content: "My favorite IDE is Neovim with LazyVim configuration. I also use tmux for terminal management.",
      role_type: "user",
    },
  ]);

  // ── Wait for processing ─────────────────────────────────────────────────
  console.log("\n⏳ Waiting for processing (15s)...\n");
  await sleep(15000);

  // ── Pattern 1: Full Session Memory ──────────────────────────────────────
  console.log("─".repeat(60));
  console.log("🧠 Pattern 1: Full Session Memory (Session A)");
  console.log("─".repeat(60));
  const memA = await client.getMemory(SESSION_A);
  if (memA.data.context) {
    console.log(`\n  Context:\n    ${memA.data.context.substring(0, 300)}...`);
  }
  if (memA.data.relevant_facts?.length > 0) {
    console.log(`\n  Facts (${memA.data.relevant_facts.length}):`);
    for (const f of memA.data.relevant_facts) {
      console.log(`    → ${f.fact || f.content}`);
    }
  }

  // ── Pattern 2: Last N Messages ──────────────────────────────────────────
  console.log("\n" + "─".repeat(60));
  console.log("🧠 Pattern 2: Last 2 Messages Only (Session B)");
  console.log("─".repeat(60));
  const memB = await client.getMemory(SESSION_B, { lastn: 2 });
  if (memB.data.messages) {
    for (const m of memB.data.messages) {
      console.log(`  [${m.role_type}] ${m.content?.substring(0, 80)}...`);
    }
  }

  // ── Pattern 3: Cross-session Graph Search ───────────────────────────────
  console.log("\n" + "─".repeat(60));
  console.log("🔍 Pattern 3: Cross-Session Graph Search");
  console.log("─".repeat(60));
  const searchRes = await client.searchGraph("What programming languages does the user know?", {
    user_id: USER_ID,
  });
  if (searchRes.data.edges?.length > 0) {
    console.log(`\n  Graph edges (${searchRes.data.edges.length}):`);
    for (const e of searchRes.data.edges.slice(0, 5)) {
      console.log(`    → ${e.fact || e.name}`);
    }
  }
  if (searchRes.data.context) {
    console.log(`\n  Graph context:\n    ${searchRes.data.context.substring(0, 200)}...`);
  }

  // ── Cleanup ─────────────────────────────────────────────────────────────
  console.log("\n\n🧹 Cleaning up...");
  await client.deleteSession(SESSION_A);
  await client.deleteSession(SESSION_B);
  await client.deleteUser(USER_ID);
  console.log("✅ Memory retrieval demo complete!\n");
}

main().catch((err) => {
  console.error("❌ Demo failed:", err.message);
  client.deleteSession(SESSION_A).catch(() => {});
  client.deleteSession(SESSION_B).catch(() => {});
  client.deleteUser(USER_ID).catch(() => {});
  process.exit(1);
});
