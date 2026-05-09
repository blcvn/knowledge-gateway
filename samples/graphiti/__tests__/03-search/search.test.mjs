/**
 * 03 — Search & Retrieval Tests
 *
 * Tests semantic search and memory retrieval from the knowledge graph.
 * NOTE: These tests require data to have been ingested first.
 *       Run test:ingest before running these.
 */
import { describe, test, expect, beforeAll } from "@jest/globals";
import { getClient, CFG, uuid, sampleId, sleep } from "../../lib/client.mjs";

const client = getClient();
const SEARCH_GROUP = sampleId("test-search");

describe("Graphiti — Search & Retrieval", () => {
  // ─── Seed data for search tests ─────────────────────────────────────────
  beforeAll(async () => {
    const now = new Date();

    // Seed a rich conversation for search
    const messages = [
      {
        content: "I just started working at OpenLedger as a software engineer.",
        role_type: "user",
        role: "Bob",
        uuid: uuid(),
        timestamp: new Date(now.getTime() - 300000).toISOString(),
        source_description: "Onboarding conversation",
      },
      {
        content:
          "Welcome to OpenLedger, Bob! As a software engineer, you'll be working on our blockchain infrastructure. Your tech lead is Sarah Chen, and you'll be on the Platform team.",
        role_type: "assistant",
        role: "HR Assistant",
        uuid: uuid(),
        timestamp: new Date(now.getTime() - 240000).toISOString(),
        source_description: "Onboarding conversation",
      },
      {
        content:
          "Thanks! I heard we use Neo4j for our knowledge graph. Is that correct?",
        role_type: "user",
        role: "Bob",
        uuid: uuid(),
        timestamp: new Date(now.getTime() - 180000).toISOString(),
        source_description: "Onboarding conversation",
      },
      {
        content:
          "Yes! We use Neo4j as our primary graph database, along with PostgreSQL for relational data and Qdrant for vector search. The entire stack runs on Docker Compose in our dev environment.",
        role_type: "assistant",
        role: "HR Assistant",
        uuid: uuid(),
        timestamp: new Date(now.getTime() - 120000).toISOString(),
        source_description: "Onboarding conversation",
      },
      {
        content:
          "Our next team meeting is scheduled for Friday at 2 PM. Sarah will present the Q2 roadmap.",
        role_type: "assistant",
        role: "HR Assistant",
        uuid: uuid(),
        timestamp: new Date(now.getTime() - 60000).toISOString(),
        source_description: "Onboarding conversation",
      },
    ];

    await client.addMessages(SEARCH_GROUP, messages);

    // Wait for async processing
    console.log("  ⏳ Waiting for message processing...");
    await sleep(15000);
  }, 120000);

  // ─── Basic Search ───────────────────────────────────────────────────────
  test("POST /search — keyword search → returns facts", async () => {
    const res = await client.search("OpenLedger software engineer", {
      group_ids: [SEARCH_GROUP],
      max_facts: 5,
    });

    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty("facts");
    expect(Array.isArray(res.data.facts)).toBe(true);
    console.log(`  ✓ Found ${res.data.facts.length} facts`);

    if (res.data.facts.length > 0) {
      const fact = res.data.facts[0];
      expect(fact).toHaveProperty("uuid");
      expect(fact).toHaveProperty("fact");
      expect(fact).toHaveProperty("created_at");
      console.log(`  ✓ Top fact: "${fact.fact}"`);
    }
  });

  // ─── Semantic Search ────────────────────────────────────────────────────
  test("POST /search — semantic query → finds related facts", async () => {
    const res = await client.search("What databases does the company use?", {
      group_ids: [SEARCH_GROUP],
      max_facts: 10,
    });

    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty("facts");
    console.log(`  ✓ Semantic search found ${res.data.facts.length} facts`);
  });

  // ─── Search without Group Filter ────────────────────────────────────────
  test("POST /search — global search (no group filter)", async () => {
    const res = await client.search("blockchain technology", {
      max_facts: 5,
    });

    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty("facts");
    console.log(`  ✓ Global search found ${res.data.facts.length} facts`);
  });

  // ─── Max Facts Limit ────────────────────────────────────────────────────
  test("POST /search — respects max_facts limit", async () => {
    const res = await client.search("technology", {
      group_ids: [SEARCH_GROUP],
      max_facts: 2,
    });

    expect(res.status).toBe(200);
    expect(res.data.facts.length).toBeLessThanOrEqual(2);
  });

  // ─── Get Episodes ──────────────────────────────────────────────────────
  test("GET /episodes/:group_id — retrieve recent episodes", async () => {
    const res = await client.getEpisodes(SEARCH_GROUP, 10);

    expect(res.status).toBe(200);
    expect(Array.isArray(res.data)).toBe(true);
    console.log(`  ✓ Retrieved ${res.data.length} episodes`);

    if (res.data.length > 0) {
      const ep = res.data[0];
      expect(ep).toHaveProperty("uuid");
      console.log(`  ✓ Latest episode UUID: ${ep.uuid}`);
    }
  });

  // ─── Get Memory ─────────────────────────────────────────────────────────
  test("POST /get-memory — contextual memory retrieval", async () => {
    const res = await client.getMemory(
      SEARCH_GROUP,
      [
        {
          content: "Who is the tech lead?",
          role_type: "user",
          role: "Bob",
        },
      ],
      { max_facts: 5 }
    );

    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty("facts");
    expect(Array.isArray(res.data.facts)).toBe(true);
    console.log(`  ✓ Memory retrieval returned ${res.data.facts.length} facts`);
  });

  // ─── Cleanup ────────────────────────────────────────────────────────────
  afterAll(async () => {
    try {
      await client.deleteGroup(SEARCH_GROUP);
    } catch {
      // ignore
    }
  });
});
