#!/usr/bin/env node
/**
 * Demo: Advanced Search Patterns
 *
 * Demonstrates different search patterns available in Graphiti:
 *  - Semantic search with group filtering
 *  - Global search (cross-group)
 *  - Contextual memory retrieval
 *  - Episode retrieval
 *
 * Usage:
 *   node scripts/demo-search.mjs
 */
import { GraphitiClient, uuid, sampleId, sleep, pp, CFG } from "../lib/client.mjs";

const client = new GraphitiClient();
const GROUP_A = sampleId("demo-search-a");
const GROUP_B = sampleId("demo-search-b");

async function main() {
  console.log("╔══════════════════════════════════════════════════════════════╗");
  console.log("║       Graphiti — Advanced Search Demo                       ║");
  console.log("╚══════════════════════════════════════════════════════════════╝");
  console.log(`  API:     ${CFG.apiUrl}`);
  console.log(`  Group A: ${GROUP_A}`);
  console.log(`  Group B: ${GROUP_B}\n`);

  // ── Health check ────────────────────────────────────────────────────────
  const health = await client.healthcheck();
  if (health.status !== 200) {
    console.error("❌ Graphiti is not healthy");
    process.exit(1);
  }
  console.log("✅ Graphiti is healthy\n");

  // ── Seed Group A: Technical discussion ──────────────────────────────────
  console.log("📝 Seeding Group A (Technical discussion)...");
  await client.addMessages(GROUP_A, [
    {
      content: "Our API gateway handles 10,000 requests per second using Go and Redis caching.",
      role_type: "user",
      role: "DevOps Lead",
      uuid: uuid(),
      timestamp: new Date().toISOString(),
      source_description: "Architecture review",
    },
    {
      content: "We should consider adding rate limiting at the gateway level. Currently we use token bucket algorithm.",
      role_type: "assistant",
      role: "Senior Engineer",
      uuid: uuid(),
      timestamp: new Date().toISOString(),
      source_description: "Architecture review",
    },
  ]);

  // ── Seed Group B: Project management ────────────────────────────────────
  console.log("📝 Seeding Group B (Project management)...");
  await client.addMessages(GROUP_B, [
    {
      content: "The Q2 release is scheduled for June 15th. Key deliverables include the new authentication system and the dashboard redesign.",
      role_type: "user",
      role: "PM",
      uuid: uuid(),
      timestamp: new Date().toISOString(),
      source_description: "Sprint planning",
    },
    {
      content: "We need to allocate 3 engineers to the authentication work. The dashboard can be handled by the frontend team.",
      role_type: "assistant",
      role: "Tech Lead",
      uuid: uuid(),
      timestamp: new Date().toISOString(),
      source_description: "Sprint planning",
    },
  ]);

  // ── Wait ────────────────────────────────────────────────────────────────
  console.log("\n⏳ Waiting for processing (20s)...");
  await sleep(20000);

  // ── Pattern 1: Group-scoped search ──────────────────────────────────────
  console.log("\n" + "─".repeat(60));
  console.log("🔍 Pattern 1: Group-scoped search");
  console.log("─".repeat(60));

  const techSearch = await client.search("API performance and caching", {
    group_ids: [GROUP_A],
    max_facts: 5,
  });
  console.log(`\n  Group A results (${techSearch.data.facts?.length || 0} facts):`);
  for (const f of techSearch.data.facts || []) {
    console.log(`    → ${f.fact}`);
  }

  // ── Pattern 2: Cross-group search ───────────────────────────────────────
  console.log("\n" + "─".repeat(60));
  console.log("🔍 Pattern 2: Cross-group search");
  console.log("─".repeat(60));

  const crossSearch = await client.search("engineering team and project plans", {
    group_ids: [GROUP_A, GROUP_B],
    max_facts: 10,
  });
  console.log(`\n  Cross-group results (${crossSearch.data.facts?.length || 0} facts):`);
  for (const f of crossSearch.data.facts || []) {
    console.log(`    → ${f.fact}`);
  }

  // ── Pattern 3: Global search ────────────────────────────────────────────
  console.log("\n" + "─".repeat(60));
  console.log("🔍 Pattern 3: Global search (no group filter)");
  console.log("─".repeat(60));

  const globalSearch = await client.search("technology stack", { max_facts: 5 });
  console.log(`\n  Global results (${globalSearch.data.facts?.length || 0} facts):`);
  for (const f of globalSearch.data.facts || []) {
    console.log(`    → ${f.fact}`);
  }

  // ── Pattern 4: Contextual memory ────────────────────────────────────────
  console.log("\n" + "─".repeat(60));
  console.log("🧠 Pattern 4: Contextual memory retrieval");
  console.log("─".repeat(60));

  const memory = await client.getMemory(
    GROUP_B,
    [
      { content: "What are the deadlines for Q2?", role_type: "user", role: "Manager" },
    ],
    { max_facts: 5 }
  );
  console.log(`\n  Memory facts (${memory.data.facts?.length || 0}):`);
  for (const f of memory.data.facts || []) {
    console.log(`    → ${f.fact}`);
  }

  // ── Pattern 5: Episode retrieval ────────────────────────────────────────
  console.log("\n" + "─".repeat(60));
  console.log("📋 Pattern 5: Episode retrieval");
  console.log("─".repeat(60));

  const episodes = await client.getEpisodes(GROUP_A, 5);
  console.log(`\n  Episodes in Group A (${episodes.data.length}):`);
  for (const ep of episodes.data.slice(0, 3)) {
    console.log(`    → [${ep.uuid?.slice(0, 8)}...] ${ep.name || "(unnamed)"}`);
  }

  // ── Cleanup ─────────────────────────────────────────────────────────────
  console.log("\n\n🧹 Cleaning up...");
  await client.deleteGroup(GROUP_A);
  await client.deleteGroup(GROUP_B);
  console.log("✅ Search demo complete!\n");
}

main().catch((err) => {
  console.error("❌ Demo failed:", err.message);
  process.exit(1);
});
