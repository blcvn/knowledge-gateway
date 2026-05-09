/**
 * Cognee API Test Client — Shared helpers
 *
 * Provides:
 *  - CogneeClient: thin wrapper over fetch for Cognee REST API
 *  - Environment config loader (from .env)
 *  - Auth session management (login / register / token)
 */
import { config } from "dotenv";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));
config({ path: resolve(__dirname, "../.env") });

// ─── Configuration ──────────────────────────────────────────────────────────
export const CFG = {
  apiUrl: process.env.COGNEE_API_URL || "https://c6.openledger.vn/cognee",
  testEmail: process.env.TEST_USER_EMAIL || "test-cognee@vnpmemory.dev",
  testPassword: process.env.TEST_USER_PASSWORD || "TestCognee2026!",
  timeout: Number(process.env.TEST_TIMEOUT || 30) * 1000,
  longTimeout: Number(process.env.TEST_LONG_TIMEOUT || 120) * 1000,
  skipDestructive: process.env.SKIP_DESTRUCTIVE === "true",
};

// ─── HTTP Client ────────────────────────────────────────────────────────────
export class CogneeClient {
  constructor(baseUrl = CFG.apiUrl) {
    this.baseUrl = baseUrl.replace(/\/$/, "");
    this.token = null;
    this.apiKey = null;
  }

  /**
   * Set bearer token for subsequent requests
   */
  setToken(token) {
    this.token = token;
    return this;
  }

  /**
   * Set API key for subsequent requests (X-Api-Key header)
   */
  setApiKey(key) {
    this.apiKey = key;
    return this;
  }

  /**
   * Build auth headers based on current token/apiKey
   */
  _authHeaders() {
    const h = {};
    if (this.token) h["Authorization"] = `Bearer ${this.token}`;
    if (this.apiKey) h["X-Api-Key"] = this.apiKey;
    return h;
  }

  /**
   * Generic HTTP request
   */
  async request(method, path, { body, headers = {}, timeout, form, multipart } = {}) {
    const url = `${this.baseUrl}${path}`;
    const opts = {
      method,
      headers: {
        ...this._authHeaders(),
        ...headers,
      },
      signal: AbortSignal.timeout(timeout || CFG.timeout),
    };

    if (form) {
      opts.body = new URLSearchParams(form);
      opts.headers["Content-Type"] = "application/x-www-form-urlencoded";
    } else if (multipart) {
      // multipart/form-data — let fetch set the Content-Type with boundary
      const fd = new FormData();
      for (const [k, v] of Object.entries(multipart)) {
        if (v instanceof Blob) {
          fd.append(k, v);
        } else if (k === "data") {
          // Cognee expects 'data' as an UploadFile, not a plain string
          fd.append(k, new Blob([v], { type: "text/plain" }), "data.txt");
        } else {
          fd.append(k, String(v));
        }
      }
      opts.body = fd;
    } else if (body !== undefined) {
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

  // ── Auth helpers ─────────────────────────────────────────────────────────

  /**
   * Register a new user. Returns the response (may 400 if exists).
   */
  async register(email, password) {
    return this.post("/api/v1/auth/register", {
      body: { email, password },
    });
  }

  /**
   * Login and store the bearer token.
   * Returns { token, response }.
   */
  async login(email, password) {
    const res = await this.post("/api/v1/auth/login", {
      form: { username: email, password, grant_type: "password" },
    });
    if (res.status === 200 && res.data?.access_token) {
      this.token = res.data.access_token;
    }
    return { token: this.token, response: res };
  }

  /**
   * Ensure a test user exists and login. Idempotent.
   */
  async ensureAuth(email = CFG.testEmail, password = CFG.testPassword) {
    // Try login first
    const loginRes = await this.login(email, password);
    if (loginRes.token) return loginRes;

    // User doesn't exist → register + login
    await this.register(email, password);
    return this.login(email, password);
  }
}

// ─── Global singleton ───────────────────────────────────────────────────────
let _client;
export function getClient() {
  if (!_client) _client = new CogneeClient();
  return _client;
}

/**
 * Get an authenticated client (singleton, lazy init).
 * Call once in beforeAll().
 */
let _authed = false;
export async function getAuthClient() {
  const client = getClient();
  if (!_authed) {
    await client.ensureAuth();
    _authed = true;
  }
  return client;
}

// ─── Test Helpers ───────────────────────────────────────────────────────────

/**
 * Generate a unique test identifier to avoid collisions.
 */
export function testId(prefix = "test") {
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
