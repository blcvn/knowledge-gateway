/**
 * Graphiti API Client — Shared helpers for samples & tests
 *
 * Provides:
 *  - GraphitiClient: typed wrapper over fetch for Graphiti REST API
 *  - Environment config loader (from .env)
 *  - UUID & test helper utilities
 *
 * API Reference (Graphiti Server — getzep/graphiti):
 *  POST   /messages           — Ingest conversation messages
 *  POST   /entity-node        — Create an entity node
 *  POST   /search             — Semantic search across facts
 *  POST   /get-memory         — Retrieve contextual memory
 *  GET    /episodes/:group_id — Get recent episodes
 *  GET    /entity-edge/:uuid  — Get a specific edge
 *  DELETE /entity-edge/:uuid  — Delete an entity edge
 *  DELETE /group/:group_id    — Delete all data for a group
 *  DELETE /episode/:uuid      — Delete an episode
 *  POST   /clear              — Clear entire graph (DESTRUCTIVE)
 *  GET    /healthcheck        — Health check
 */
import { config } from "dotenv";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));
config({ path: resolve(__dirname, "../.env") });

// ─── Configuration ──────────────────────────────────────────────────────────
export const CFG = {
  apiUrl: process.env.GRAPHITI_API_URL || "https://c6.openledger.vn/graphiti",
  timeout: Number(process.env.REQUEST_TIMEOUT || 30) * 1000,
  longTimeout: Number(process.env.LONG_TIMEOUT || 120) * 1000,
  sampleGroupId: process.env.SAMPLE_GROUP_ID || "sample-graphiti-demo",
};

// ─── HTTP Client ────────────────────────────────────────────────────────────
export class GraphitiClient {
  constructor(baseUrl = CFG.apiUrl) {
    this.baseUrl = baseUrl.replace(/\/$/, "");
  }

  /**
   * Generic HTTP request
   */
  async request(method, path, { body, headers = {}, timeout, query } = {}) {
    let url = `${this.baseUrl}${path}`;
    if (query) {
      const params = new URLSearchParams(query);
      url += `?${params.toString()}`;
    }

    const opts = {
      method,
      headers: {
        ...headers,
      },
      signal: AbortSignal.timeout(timeout || CFG.timeout),
    };

    if (body !== undefined) {
      opts.body = JSON.stringify(body);
      opts.headers["Content-Type"] = "application/json";
    }

    const res = await fetch(url, opts);
    const text = await res.text();
    let data;
    try {
      data = JSON.parse(text);
    } catch {
      data = text;
    }
    return { status: res.status, data, headers: res.headers };
  }

  // ── Convenience methods ──────────────────────────────────────────────────
  get(path, opts) {
    return this.request("GET", path, opts);
  }
  post(path, opts) {
    return this.request("POST", path, opts);
  }
  delete(path, opts) {
    return this.request("DELETE", path, opts);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  //  HEALTHCHECK
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Check if the Graphiti service is healthy.
   * @returns {{ status: number, data: { status: string } }}
   */
  async healthcheck() {
    return this.get("/healthcheck");
  }

  // ═══════════════════════════════════════════════════════════════════════════
  //  INGEST
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Ingest conversation messages into the knowledge graph.
   *
   * @param {string} groupId    — Isolates messages into a conversation/session
   * @param {Array<{
   *   content: string,
   *   role_type: 'user'|'assistant'|'system',
   *   role?: string,
   *   uuid?: string,
   *   name?: string,
   *   timestamp?: string,
   *   source_description?: string,
   * }>} messages — Array of messages
   * @returns {{ status: number, data: { message: string, success: boolean } }}
   */
  async addMessages(groupId, messages) {
    return this.post("/messages", {
      body: {
        group_id: groupId,
        messages,
      },
      timeout: CFG.longTimeout,
    });
  }

  /**
   * Create an entity node in the knowledge graph.
   *
   * @param {{ uuid: string, group_id: string, name: string, summary?: string }} entity
   * @returns {{ status: number, data: object }}
   */
  async addEntityNode(entity) {
    return this.post("/entity-node", {
      body: entity,
      timeout: CFG.longTimeout,
    });
  }

  /**
   * Clear the entire graph. ⚠️ DESTRUCTIVE!
   * @returns {{ status: number, data: { message: string, success: boolean } }}
   */
  async clearGraph() {
    return this.post("/clear", {
      timeout: CFG.longTimeout,
    });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  //  RETRIEVE
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Semantic search across facts in the knowledge graph.
   *
   * @param {string} queryText        — Natural language query
   * @param {{
   *   group_ids?: string[],
   *   max_facts?: number,
   * }} opts
   * @returns {{ status: number, data: { facts: Array<FactResult> } }}
   */
  async search(queryText, { group_ids, max_facts = 10 } = {}) {
    return this.post("/search", {
      body: {
        query: queryText,
        group_ids: group_ids || null,
        max_facts,
      },
      timeout: CFG.longTimeout,
    });
  }

  /**
   * Get contextual memory based on recent messages.
   *
   * @param {string} groupId          — Conversation group
   * @param {Array<{
   *   content: string,
   *   role_type: 'user'|'assistant'|'system',
   *   role?: string,
   * }>} messages                     — Recent messages for context
   * @param {{ max_facts?: number, center_node_uuid?: string }} opts
   * @returns {{ status: number, data: { facts: Array<FactResult> } }}
   */
  async getMemory(groupId, messages, { max_facts = 10, center_node_uuid = null } = {}) {
    return this.post("/get-memory", {
      body: {
        group_id: groupId,
        messages,
        max_facts,
        center_node_uuid,
      },
      timeout: CFG.longTimeout,
    });
  }

  /**
   * Get recent episodes for a group.
   *
   * @param {string} groupId  — Group ID
   * @param {number} lastN    — Number of recent episodes
   * @returns {{ status: number, data: Array }}
   */
  async getEpisodes(groupId, lastN = 10) {
    return this.get(`/episodes/${encodeURIComponent(groupId)}`, {
      query: { last_n: String(lastN) },
    });
  }

  /**
   * Get a specific entity edge by UUID.
   *
   * @param {string} uuid — Edge UUID
   * @returns {{ status: number, data: FactResult }}
   */
  async getEntityEdge(uuid) {
    return this.get(`/entity-edge/${encodeURIComponent(uuid)}`);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  //  DELETE
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Delete an entity edge.
   * @param {string} uuid — Edge UUID
   */
  async deleteEntityEdge(uuid) {
    return this.delete(`/entity-edge/${encodeURIComponent(uuid)}`);
  }

  /**
   * Delete all data for a group.
   * @param {string} groupId — Group ID
   */
  async deleteGroup(groupId) {
    return this.delete(`/group/${encodeURIComponent(groupId)}`);
  }

  /**
   * Delete a specific episode.
   * @param {string} uuid — Episode UUID
   */
  async deleteEpisode(uuid) {
    return this.delete(`/episode/${encodeURIComponent(uuid)}`);
  }
}

// ─── Global singleton ───────────────────────────────────────────────────────
let _client;
export function getClient() {
  if (!_client) _client = new GraphitiClient();
  return _client;
}

// ─── Utility Helpers ────────────────────────────────────────────────────────

/**
 * Generate a UUID v4-like string.
 */
export function uuid() {
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === "x" ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

/**
 * Generate a unique test/sample identifier.
 */
export function sampleId(prefix = "sample") {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
}

/**
 * Sleep for ms.
 */
export function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

/**
 * Poll a condition fn until it returns truthy or timeout.
 */
export async function waitFor(fn, { interval = 2000, timeout = CFG.longTimeout } = {}) {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    const result = await fn();
    if (result) return result;
    await sleep(interval);
  }
  throw new Error(`waitFor timed out after ${timeout}ms`);
}

/**
 * Pretty print JSON for demo output.
 */
export function pp(label, obj) {
  console.log(`\n${"═".repeat(60)}`);
  console.log(`  ${label}`);
  console.log(`${"═".repeat(60)}`);
  console.log(JSON.stringify(obj, null, 2));
}
