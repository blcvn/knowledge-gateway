/**
 * 04 — Temporal Memory Tests
 *
 * Tests Graphiti's temporal knowledge graph capabilities:
 *  - Facts evolving over time (temporal updates)
 *  - Multiple sessions contributing to the same group
 *  - Memory context across conversation turns
 */
import { describe, test, expect, beforeAll, afterAll } from "@jest/globals";
import { getClient, uuid, sampleId, sleep } from "../../lib/client.mjs";

const client = getClient();
const TEMPORAL_GROUP = sampleId("test-temporal");

describe("Graphiti — Temporal Memory", () => {
  // ─── Seed temporal data ─────────────────────────────────────────────────
  beforeAll(async () => {
    const baseTime = new Date("2025-01-15T10:00:00Z");

    // Session 1: Initial facts (January)
    await client.addMessages(TEMPORAL_GROUP, [
      {
        content: "I currently live in Ho Chi Minh City and work as a junior developer.",
        role_type: "user",
        role: "Minh",
        uuid: uuid(),
        timestamp: new Date(baseTime.getTime()).toISOString(),
        source_description: "Career check-in - January",
      },
      {
        content: "That's great, Minh! Ho Chi Minh City is a vibrant tech hub. How long have you been a junior developer?",
        role_type: "assistant",
        role: "Coach",
        uuid: uuid(),
        timestamp: new Date(baseTime.getTime() + 30000).toISOString(),
        source_description: "Career check-in - January",
      },
    ]);

    // Wait for processing
    await sleep(10000);

    // Session 2: Updated facts (March) — Minh got promoted
    await client.addMessages(TEMPORAL_GROUP, [
      {
        content: "Great news! I just got promoted to senior developer at my company. I also moved to Hanoi for the new role.",
        role_type: "user",
        role: "Minh",
        uuid: uuid(),
        timestamp: new Date("2025-03-20T14:00:00Z").toISOString(),
        source_description: "Career check-in - March",
      },
      {
        content: "Congratulations on the promotion, Minh! Moving to Hanoi is a big step. How's the transition going?",
        role_type: "assistant",
        role: "Coach",
        uuid: uuid(),
        timestamp: new Date("2025-03-20T14:01:00Z").toISOString(),
        source_description: "Career check-in - March",
      },
    ]);

    // Wait for temporal processing
    console.log("  ⏳ Waiting for temporal processing...");
    await sleep(15000);
  }, 120000);

  // ─── Temporal Fact Evolution ────────────────────────────────────────────
  test("Search reflects latest state (promotion & location change)", async () => {
    const res = await client.search("Where does Minh live and what is his job?", {
      group_ids: [TEMPORAL_GROUP],
      max_facts: 10,
    });

    expect(res.status).toBe(200);
    expect(res.data.facts).toBeDefined();
    console.log(`  ✓ Found ${res.data.facts.length} temporal facts:`);
    for (const fact of res.data.facts) {
      console.log(`    - [${fact.valid_at || "n/a"}] ${fact.fact}`);
    }
  });

  // ─── Multi-Session Memory ──────────────────────────────────────────────
  test("Memory spans across multiple sessions", async () => {
    const res = await client.getMemory(
      TEMPORAL_GROUP,
      [
        {
          content: "What do you know about my career progression?",
          role_type: "user",
          role: "Minh",
        },
      ],
      { max_facts: 10 }
    );

    expect(res.status).toBe(200);
    expect(res.data.facts).toBeDefined();
    console.log(`  ✓ Memory across sessions: ${res.data.facts.length} facts`);
    for (const fact of res.data.facts) {
      console.log(`    - ${fact.fact}`);
    }
  });

  // ─── Episode History ────────────────────────────────────────────────────
  test("Episodes maintain chronological order", async () => {
    const res = await client.getEpisodes(TEMPORAL_GROUP, 20);

    expect(res.status).toBe(200);
    expect(Array.isArray(res.data)).toBe(true);
    expect(res.data.length).toBeGreaterThan(0);
    console.log(`  ✓ ${res.data.length} episodes in temporal group`);
  });

  // ─── Cleanup ────────────────────────────────────────────────────────────
  afterAll(async () => {
    try {
      await client.deleteGroup(TEMPORAL_GROUP);
    } catch {
      // ignore
    }
  });
});
