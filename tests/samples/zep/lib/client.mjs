/**
 * Zep API Client — Shared helpers for samples & tests
 *
 * Provides:
 *  - ZepClient: typed wrapper over fetch for Zep v2 REST API
 *  - Environment config loader (from .env)
 *  - UUID & test helper utilities
 *
 * Zep API Reference (v2 — ghcr.io/getzep/zep:latest):
 *  GET    /healthz                              — Health check
 *
 *  POST   /api/v2/users                         — Create user
 *  GET    /api/v2/users/{userId}                — Get user
 *  GET    /api/v2/users                         — List users
 *  DEL    /api/v2/users/{userId}                — Delete user
 *
 *  POST   /api/v2/sessions                      — Create session
 *  GET    /api/v2/sessions/{sessionId}          — Get session
 *  GET    /api/v2/sessions                      — List sessions
 *  DEL    /api/v2/sessions/{sessionId}          — Delete session
 *
 *  POST   /api/v2/sessions/{sessionId}/memory   — Add memory (messages)
 *  GET    /api/v2/sessions/{sessionId}/memory   — Get session memory
 *  GET    /api/v2/sessions/{sessionId}/messages — Get session messages
 *
 *  POST   /api/v2/graph/search                  — Search graph
 */
import { config } from "dotenv";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));
config({ path: resolve(__dirname, "../.env") });

// ─── Configuration ──────────────────────────────────────────────────────────
export const CFG = {
  apiUrl: process.env.ZEP_API_URL || "https://c6.openledger.vn/zep",
  apiKey: process.env.ZEP_API_KEY || "",
  timeout: Number(process.env.REQUEST_TIMEOUT || 30) * 1000,
  longTimeout: Number(process.env.LONG_TIMEOUT || 120) * 1000,
  userPrefix: process.env.SAMPLE_USER_PREFIX || "sample-zep",
  sessionPrefix: process.env.SAMPLE_SESSION_PREFIX || "sample-session",
};

// ─── HTTP Client ────────────────────────────────────────────────────────────
export class ZepClient {
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

    // Add API key if configured
    if (CFG.apiKey) {
      opts.headers["Authorization"] = CFG.apiKey;
    }

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
  put(path, opts) {
    return this.request("PUT", path, opts);
  }
  patch(path, opts) {
    return this.request("PATCH", path, opts);
  }
  delete(path, opts) {
    return this.request("DELETE", path, opts);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  //  HEALTH
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Check if the Zep service is healthy.
   * @returns {{ status: number, data: string }}
   */
  async healthcheck() {
    return this.get("/healthz");
  }

  // ═══════════════════════════════════════════════════════════════════════════
  //  USERS
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Create a new user.
   *
   * @param {{ user_id: string, email?: string, first_name?: string, last_name?: string, metadata?: object }} user
   * @returns {{ status: number, data: object }}
   */
  async createUser(user) {
    return this.post("/api/v2/users", { body: user });
  }

  /**
   * Get a user by ID.
   * @param {string} userId
   */
  async getUser(userId) {
    return this.get(`/api/v2/users/${encodeURIComponent(userId)}`);
  }

  /**
   * List all users.
   * @param {{ limit?: number, cursor?: number }} opts
   */
  async listUsers({ limit, cursor } = {}) {
    const query = {};
    if (limit) query.limit = String(limit);
    if (cursor) query.cursor = String(cursor);
    return this.get("/api/v2/users", { query });
  }

  /**
   * Delete a user.
   * @param {string} userId
   */
  async deleteUser(userId) {
    return this.delete(`/api/v2/users/${encodeURIComponent(userId)}`);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  //  SESSIONS
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Create a new session.
   *
   * @param {{ session_id: string, user_id: string, metadata?: object }} session
   * @returns {{ status: number, data: object }}
   */
  async createSession(session) {
    return this.post("/api/v2/sessions", { body: session });
  }

  /**
   * Get a session by ID.
   * @param {string} sessionId
   */
  async getSession(sessionId) {
    return this.get(`/api/v2/sessions/${encodeURIComponent(sessionId)}`);
  }

  /**
   * List sessions.
   * @param {{ limit?: number, cursor?: number }} opts
   */
  async listSessions({ limit, cursor } = {}) {
    const query = {};
    if (limit) query.limit = String(limit);
    if (cursor) query.cursor = String(cursor);
    return this.get("/api/v2/sessions", { query });
  }

  /**
   * Delete a session.
   * @param {string} sessionId
   */
  async deleteSession(sessionId) {
    return this.delete(`/api/v2/sessions/${encodeURIComponent(sessionId)}`);
  }

  // ═══════════════════════════════════════════════════════════════════════════
  //  MEMORY
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Add memory (messages) to a session.
   *
   * @param {string} sessionId
   * @param {Array<{
   *   content: string,
   *   role_type: 'user'|'assistant'|'system'|'norole',
   *   role?: string,
   *   metadata?: object,
   * }>} messages
   * @returns {{ status: number, data: object }}
   */
  async addMemory(sessionId, messages) {
    return this.post(`/api/v2/sessions/${encodeURIComponent(sessionId)}/memory`, {
      body: { messages },
      timeout: CFG.longTimeout,
    });
  }

  /**
   * Get memory for a session (includes context, facts, messages, summary).
   *
   * @param {string} sessionId
   * @param {{ lastn?: number, min_rating?: number }} opts
   * @returns {{ status: number, data: {
   *   context: string,
   *   facts: string[],
   *   messages: Array,
   *   relevant_facts: Array,
   *   summary: object,
   *   metadata: object,
   * } }}
   */
  async getMemory(sessionId, { lastn, min_rating } = {}) {
    const query = {};
    if (lastn) query.lastn = String(lastn);
    if (min_rating) query.min_rating = String(min_rating);
    return this.get(`/api/v2/sessions/${encodeURIComponent(sessionId)}/memory`, {
      query,
      timeout: CFG.longTimeout,
    });
  }

  /**
   * Get messages for a session.
   *
   * @param {string} sessionId
   * @param {{ limit?: number, cursor?: number }} opts
   */
  async getMessages(sessionId, { limit, cursor } = {}) {
    const query = {};
    if (limit) query.limit = String(limit);
    if (cursor) query.cursor = String(cursor);
    return this.get(`/api/v2/sessions/${encodeURIComponent(sessionId)}/messages`, { query });
  }

  // ═══════════════════════════════════════════════════════════════════════════
  //  GRAPH SEARCH
  // ═══════════════════════════════════════════════════════════════════════════

  /**
   * Search the knowledge graph.
   *
   * @param {string} queryText — Natural language search query
   * @param {{ user_id?: string, limit?: number, scope?: string }} opts
   * @returns {{ status: number, data: { edges: Array, nodes: Array, episodes: Array, context: string } }}
   */
  async searchGraph(queryText, { user_id, limit, scope } = {}) {
    const body = { query: queryText };
    if (user_id) body.user_id = user_id;
    if (limit) body.limit = limit;
    if (scope) body.scope = scope;
    return this.post("/api/v2/graph/search", {
      body,
      timeout: CFG.longTimeout,
    });
  }
}

// ─── Global singleton ───────────────────────────────────────────────────────
let _client;
export function getClient() {
  if (!_client) _client = new ZepClient();
  return _client;
}

// ─── Utility Helpers ────────────────────────────────────────────────────────

/**
 * Generate a unique sample identifier.
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
