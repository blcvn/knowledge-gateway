#!/usr/bin/env node
/**
 * Demo: Graph Search Capabilities
 *
 * Demonstrates Zep's knowledge graph search:
 *  - Semantic search across user's knowledge graph
 *  - Entity and relationship discovery
 *  - Cross-session knowledge retrieval
 *
 * Usage:
 *   node scripts/demo-graph-search.mjs
 */
import { ZepClient, sampleId, sleep, pp, CFG } from "../lib/client.mjs";

const client = new ZepClient();
const USER_ID = sampleId("demo-graph-user");
const SESSION_ID = sampleId("demo-graph-session");

async function main() {
  console.log("╔══════════════════════════════════════════════════════════════╗");
  console.log("║       Zep — Graph Search Demo                               ║");
  console.log("╚══════════════════════════════════════════════════════════════╝");
  console.log(`  API:     ${CFG.apiUrl}`);
  console.log(`  User:    ${USER_ID}`);
  console.log(`  Session: ${SESSION_ID}\n`);

  // ── Health check ────────────────────────────────────────────────────────
  const health = await client.healthcheck();
  if (health.status !== 200) {
    console.error("❌ Zep is not healthy");
    process.exit(1);
  }
  console.log("✅ Zep is healthy\n");

  // ── Setup ───────────────────────────────────────────────────────────────
  await client.createUser({
    user_id: USER_ID,
    first_name: "Graph",
    last_name: "Demo",
  });
  await client.createSession({
    session_id: SESSION_ID,
    user_id: USER_ID,
  });

  // ── Seed rich domain knowledge ──────────────────────────────────────────
  console.log("📝 Seeding domain knowledge...");
  await client.addMemory(SESSION_ID, [
    {
      content:
        "Our company OpenLedger has three main products: VNP Memory for AI agent memory, Bifrost for AI gateway routing, and KGS Platform for knowledge graph services.",
      role_type: "user",
    },
    {
      content:
        "That's an impressive product portfolio! VNP Memory, Bifrost, and KGS Platform cover the full AI infrastructure stack from memory to routing to knowledge management.",
      role_type: "assistant",
    },
    {
      content:
        "The CTO is Dr. Tran Van Nam. He designed the overall architecture. The engineering team is split into Platform (Go), AI Services (Python), and Infrastructure (Docker/K8s).",
      role_type: "user",
    },
    {
      content:
        "It sounds like a well-organized engineering structure. Having dedicated teams for Platform, AI Services, and Infrastructure ensures each layer gets proper attention.",
      role_type: "assistant",
    },
    {
      content:
        "We recently migrated from AWS to our own bare-metal servers in Ho Chi Minh City data center. Saves about 60% in cloud costs.",
      role_type: "user",
    },
    {
      content:
        "A 60% cost reduction from moving to bare-metal is significant! Managing your own infrastructure in an HCMC data center gives you more control too. What's your uptime target?",
      role_type: "assistant",
    },
    {
      content: "We target 99.95% uptime. We use Prometheus + Grafana for monitoring and PagerDuty for incident management.",
      role_type: "user",
    },
  ]);

  // ── Wait for graph construction ─────────────────────────────────────────
  console.log("\n⏳ Waiting for graph construction (25s)...\n");
  await sleep(25000);

  // ── Search Queries ──────────────────────────────────────────────────────
  const queries = [
    "What products does OpenLedger build?",
    "Who is the CTO and what did they design?",
    "What is the infrastructure setup?",
    "How are the engineering teams organized?",
    "What monitoring tools are used?",
  ];

  for (const q of queries) {
    console.log("─".repeat(60));
    console.log(`🔍 Query: "${q}"`);
    console.log("─".repeat(60));

    const res = await client.searchGraph(q, {
      user_id: USER_ID,
    });

    if (res.status === 200 && res.data) {
      // Show edges (facts/relationships)
      if (res.data.edges?.length > 0) {
        console.log(`  📊 Edges (${res.data.edges.length}):`);
        for (const e of res.data.edges.slice(0, 3)) {
          console.log(`    → ${e.fact || e.name}`);
        }
      }

      // Show nodes (entities)
      if (res.data.nodes?.length > 0) {
        console.log(`  🔵 Nodes (${res.data.nodes.length}):`);
        for (const n of res.data.nodes.slice(0, 3)) {
          console.log(`    → ${n.name}: ${(n.summary || "").substring(0, 80)}`);
        }
      }

      // Show assembled context
      if (res.data.context) {
        console.log(`  📄 Context: ${res.data.context.substring(0, 200)}...`);
      }
    } else {
      console.log(`  ⚠ No results (status: ${res.status})`);
    }
    console.log();
  }

  // ── Cleanup ─────────────────────────────────────────────────────────────
  console.log("🧹 Cleaning up...");
  await client.deleteSession(SESSION_ID);
  await client.deleteUser(USER_ID);
  console.log("✅ Graph search demo complete!\n");
}

main().catch((err) => {
  console.error("❌ Demo failed:", err.message);
  client.deleteSession(SESSION_ID).catch(() => {});
  client.deleteUser(USER_ID).catch(() => {});
  process.exit(1);
});
