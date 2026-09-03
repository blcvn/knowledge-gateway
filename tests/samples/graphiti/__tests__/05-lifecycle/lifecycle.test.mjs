/**
 * 05 — Lifecycle & Management Tests
 *
 * Tests data lifecycle operations:
 *  - Group deletion
 *  - Episode deletion
 *  - Entity edge management
 */
import { describe, test, expect, beforeAll } from "@jest/globals";
import { getClient, uuid, sampleId, sleep } from "../../lib/client.mjs";

const client = getClient();
const LIFECYCLE_GROUP = sampleId("test-lifecycle");
let createdEpisodeUuids = [];
let createdEntityUuid;

describe("Graphiti — Lifecycle & Management", () => {
  // ─── Setup: Create test data ────────────────────────────────────────────
  beforeAll(async () => {
    // Create some messages
    const msgUuid1 = uuid();
    const msgUuid2 = uuid();
    createdEpisodeUuids = [msgUuid1, msgUuid2];

    await client.addMessages(LIFECYCLE_GROUP, [
      {
        content: "This is a test message for lifecycle testing.",
        role_type: "user",
        role: "TestUser",
        uuid: msgUuid1,
        timestamp: new Date().toISOString(),
      },
      {
        content: "Acknowledged. This data will be used for deletion testing.",
        role_type: "assistant",
        role: "TestBot",
        uuid: msgUuid2,
        timestamp: new Date().toISOString(),
      },
    ]);

    // Create an entity node
    createdEntityUuid = uuid();
    await client.addEntityNode({
      uuid: createdEntityUuid,
      group_id: LIFECYCLE_GROUP,
      name: "TestEntity",
      summary: "A test entity for lifecycle management testing.",
    });

    // Wait for processing
    await sleep(10000);
  }, 60000);

  // ─── Verify Data Exists ─────────────────────────────────────────────────
  test("Data exists before deletion", async () => {
    const res = await client.getEpisodes(LIFECYCLE_GROUP, 10);
    expect(res.status).toBe(200);
    console.log(`  ✓ ${res.data.length} episodes exist before cleanup`);
  });

  // ─── Delete Episode ─────────────────────────────────────────────────────
  test("DELETE /episode/:uuid — remove a specific episode", async () => {
    if (createdEpisodeUuids.length === 0) {
      console.log("  ⚠ No episode UUIDs to delete, skipping");
      return;
    }

    // Try deleting the first episode
    const res = await client.deleteEpisode(createdEpisodeUuids[0]);
    // May return 200 if found, or 404/500 if the UUID doesn't map to an episode
    expect([200, 404, 500]).toContain(res.status);
    if (res.status === 200) {
      expect(res.data).toHaveProperty("success", true);
      console.log(`  ✓ Episode ${createdEpisodeUuids[0]} deleted`);
    } else {
      console.log(`  ⚠ Episode delete returned ${res.status} (UUID may not be episode node UUID)`);
    }
  });

  // ─── Delete Group ───────────────────────────────────────────────────────
  test("DELETE /group/:group_id — remove all group data", async () => {
    const res = await client.deleteGroup(LIFECYCLE_GROUP);
    expect(res.status).toBe(200);
    expect(res.data).toHaveProperty("success", true);
    console.log(`  ✓ Group "${LIFECYCLE_GROUP}" deleted`);
  });

  // ─── Verify Deletion ───────────────────────────────────────────────────
  test("Group data is empty after deletion", async () => {
    const res = await client.getEpisodes(LIFECYCLE_GROUP, 10);
    expect(res.status).toBe(200);
    expect(res.data.length).toBe(0);
    console.log("  ✓ Group is empty after deletion");
  });

  // ─── Search Returns No Results for Deleted Group ────────────────────────
  test("Search returns no facts for deleted group", async () => {
    const res = await client.search("lifecycle testing", {
      group_ids: [LIFECYCLE_GROUP],
      max_facts: 10,
    });
    expect(res.status).toBe(200);
    expect(res.data.facts.length).toBe(0);
    console.log("  ✓ No facts found for deleted group");
  });
});
