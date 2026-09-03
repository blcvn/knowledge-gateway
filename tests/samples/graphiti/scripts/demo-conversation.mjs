#!/usr/bin/env node
/**
 * Demo: Conversation Memory with Graphiti
 *
 * Demonstrates how Graphiti builds a temporal knowledge graph from
 * multi-turn conversations and allows semantic retrieval of facts.
 *
 * Usage:
 *   node scripts/demo-conversation.mjs
 */
import { GraphitiClient, uuid, sampleId, sleep, pp, CFG } from "../lib/client.mjs";

const client = new GraphitiClient();
const GROUP_ID = sampleId("demo-conversation");

async function main() {
  console.log("╔══════════════════════════════════════════════════════════════╗");
  console.log("║       Graphiti — Conversation Memory Demo                   ║");
  console.log("╚══════════════════════════════════════════════════════════════╝");
  console.log(`  API:   ${CFG.apiUrl}`);
  console.log(`  Group: ${GROUP_ID}\n`);

  // ── Step 1: Check health ────────────────────────────────────────────────
  const health = await client.healthcheck();
  if (health.status !== 200) {
    console.error("❌ Graphiti is not healthy:", health.data);
    process.exit(1);
  }
  console.log("✅ Graphiti is healthy\n");

  // ── Step 2: Ingest a conversation ───────────────────────────────────────
  console.log("📝 Ingesting conversation...");
  const now = new Date();

  const conversation = [
    {
      content: "Hi! My name is Nguyen Van An. I just joined VNP as a backend engineer.",
      role_type: "user",
      role: "An",
      uuid: uuid(),
      timestamp: new Date(now.getTime() - 5 * 60000).toISOString(),
      source_description: "Onboarding chat",
    },
    {
      content:
        "Welcome to VNP, An! Great to have you on the backend team. You'll be working with our Knowledge Graph Service (KGS) which uses Neo4j, PostgreSQL, and Qdrant. Your team lead is Tran Minh Duc.",
      role_type: "assistant",
      role: "Onboarding Bot",
      uuid: uuid(),
      timestamp: new Date(now.getTime() - 4 * 60000).toISOString(),
      source_description: "Onboarding chat",
    },
    {
      content: "What programming languages does the team primarily use?",
      role_type: "user",
      role: "An",
      uuid: uuid(),
      timestamp: new Date(now.getTime() - 3 * 60000).toISOString(),
      source_description: "Onboarding chat",
    },
    {
      content:
        "The KGS Platform is built in Go using the Kratos framework. We also have Python services like Graphiti and Cognee for AI/ML workloads. For infrastructure, we use Docker Compose and deploy on Ubuntu 22.04 servers.",
      role_type: "assistant",
      role: "Onboarding Bot",
      uuid: uuid(),
      timestamp: new Date(now.getTime() - 2 * 60000).toISOString(),
      source_description: "Onboarding chat",
    },
    {
      content: "When is the next team standup?",
      role_type: "user",
      role: "An",
      uuid: uuid(),
      timestamp: new Date(now.getTime() - 1 * 60000).toISOString(),
      source_description: "Onboarding chat",
    },
    {
      content:
        "The daily standup is every weekday at 9:30 AM Vietnam time. The weekly sprint review is on Fridays at 3 PM.",
      role_type: "assistant",
      role: "Onboarding Bot",
      uuid: uuid(),
      timestamp: now.toISOString(),
      source_description: "Onboarding chat",
    },
  ];

  const ingestRes = await client.addMessages(GROUP_ID, conversation);
  pp("Ingest Response", ingestRes.data);

  // ── Step 3: Wait for processing ─────────────────────────────────────────
  console.log("\n⏳ Waiting for knowledge graph construction (20s)...");
  await sleep(20000);

  // ── Step 4: Retrieve episodes ───────────────────────────────────────────
  console.log("\n📋 Retrieving episodes...");
  const episodes = await client.getEpisodes(GROUP_ID, 10);
  pp(`Episodes (${episodes.data.length} found)`, episodes.data.slice(0, 3));

  // ── Step 5: Semantic search ─────────────────────────────────────────────
  const queries = [
    "Who is the team lead?",
    "What tech stack does VNP use?",
    "When is the standup?",
  ];

  for (const q of queries) {
    console.log(`\n🔍 Searching: "${q}"`);
    const searchRes = await client.search(q, {
      group_ids: [GROUP_ID],
      max_facts: 3,
    });

    if (searchRes.data.facts?.length > 0) {
      for (const fact of searchRes.data.facts) {
        console.log(`   → ${fact.fact}`);
      }
    } else {
      console.log("   → No facts found");
    }
  }

  // ── Step 6: Contextual memory ───────────────────────────────────────────
  console.log("\n🧠 Getting contextual memory...");
  const memoryRes = await client.getMemory(
    GROUP_ID,
    [
      {
        content: "Can you summarize what I need to know for my first day?",
        role_type: "user",
        role: "An",
      },
    ],
    { max_facts: 5 }
  );
  pp("Memory Response", memoryRes.data);

  // ── Cleanup ─────────────────────────────────────────────────────────────
  console.log("\n🧹 Cleaning up demo data...");
  await client.deleteGroup(GROUP_ID);
  console.log("✅ Demo complete!\n");
}

main().catch((err) => {
  console.error("❌ Demo failed:", err.message);
  process.exit(1);
});
