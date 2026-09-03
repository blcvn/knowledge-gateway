#!/usr/bin/env node
/**
 * Demo: Entity Graph Construction
 *
 * Demonstrates how to manually create entity nodes and build
 * a structured knowledge graph with Graphiti.
 *
 * Usage:
 *   node scripts/demo-entity-graph.mjs
 */
import { GraphitiClient, uuid, sampleId, sleep, pp, CFG } from "../lib/client.mjs";

const client = new GraphitiClient();
const GROUP_ID = sampleId("demo-entity");

async function main() {
  console.log("╔══════════════════════════════════════════════════════════════╗");
  console.log("║       Graphiti — Entity Graph Demo                          ║");
  console.log("╚══════════════════════════════════════════════════════════════╝");
  console.log(`  API:   ${CFG.apiUrl}`);
  console.log(`  Group: ${GROUP_ID}\n`);

  // ── Health check ────────────────────────────────────────────────────────
  const health = await client.healthcheck();
  if (health.status !== 200) {
    console.error("❌ Graphiti is not healthy");
    process.exit(1);
  }
  console.log("✅ Graphiti is healthy\n");

  // ── Create entity nodes ─────────────────────────────────────────────────
  console.log("🏗️  Creating entity nodes...");

  const entities = [
    {
      uuid: uuid(),
      group_id: GROUP_ID,
      name: "VNP Memory",
      summary: "A multi-engine memory platform combining Cognee, Graphiti, Zep, and OpenViking for AI agent memory management.",
    },
    {
      uuid: uuid(),
      group_id: GROUP_ID,
      name: "Neo4j",
      summary: "Graph database used as the shared knowledge graph store for Cognee, Graphiti, and Zep services.",
    },
    {
      uuid: uuid(),
      group_id: GROUP_ID,
      name: "Bifrost AI Gateway",
      summary: "External LLM routing gateway that provides unified access to multiple AI model providers (OpenAI, Anthropic, Google).",
    },
    {
      uuid: uuid(),
      group_id: GROUP_ID,
      name: "KGS Platform",
      summary: "Knowledge Graph Service built in Go/Kratos that provides CRUD, search, and policy-based access control for knowledge graphs.",
    },
  ];

  for (const entity of entities) {
    const res = await client.addEntityNode(entity);
    console.log(`  ✓ Created: ${entity.name} (${res.status})`);
  }

  // ── Ingest relationships via messages ───────────────────────────────────
  console.log("\n📝 Ingesting relationship messages...");

  const messages = [
    {
      content: "VNP Memory uses Neo4j as its primary graph database for storing knowledge graph data.",
      role_type: "system",
      uuid: uuid(),
      timestamp: new Date().toISOString(),
      source_description: "Architecture documentation",
    },
    {
      content: "The Bifrost AI Gateway routes all LLM calls for VNP Memory services, supporting OpenAI, Anthropic, and Google providers.",
      role_type: "system",
      uuid: uuid(),
      timestamp: new Date().toISOString(),
      source_description: "Architecture documentation",
    },
    {
      content: "KGS Platform is part of the VNP Memory ecosystem and uses Neo4j for graph storage along with PostgreSQL, Qdrant, and Redis.",
      role_type: "system",
      uuid: uuid(),
      timestamp: new Date().toISOString(),
      source_description: "Architecture documentation",
    },
  ];

  const ingestRes = await client.addMessages(GROUP_ID, messages);
  pp("Ingest Response", ingestRes.data);

  // ── Wait for graph construction ─────────────────────────────────────────
  console.log("\n⏳ Waiting for graph construction (20s)...");
  await sleep(20000);

  // ── Query the entity graph ──────────────────────────────────────────────
  const queries = [
    "What databases does VNP Memory use?",
    "What is the Bifrost AI Gateway?",
    "How are the services connected?",
  ];

  for (const q of queries) {
    console.log(`\n🔍 Query: "${q}"`);
    const res = await client.search(q, {
      group_ids: [GROUP_ID],
      max_facts: 5,
    });
    if (res.data.facts?.length > 0) {
      for (const fact of res.data.facts) {
        console.log(`   → ${fact.fact}`);
      }
    } else {
      console.log("   → No facts found (graph may still be processing)");
    }
  }

  // ── Cleanup ─────────────────────────────────────────────────────────────
  console.log("\n🧹 Cleaning up...");
  await client.deleteGroup(GROUP_ID);
  console.log("✅ Entity graph demo complete!\n");
}

main().catch((err) => {
  console.error("❌ Demo failed:", err.message);
  process.exit(1);
});
