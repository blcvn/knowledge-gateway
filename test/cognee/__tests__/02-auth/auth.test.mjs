/**
 * 02 — Authentication & Authorization Tests
 *
 * Covers: register, login, token validation, logout, API key management.
 * Since ENABLE_BACKEND_ACCESS_CONTROL=false, most data endpoints don't
 * require auth — but the auth system itself must work correctly.
 */
import { describe, it, expect, beforeAll } from "@jest/globals";
import { getClient, testId, CFG } from "../../lib/client.mjs";

describe("Cognee Authentication", () => {
  let client;

  beforeAll(() => {
    client = new (getClient().constructor)(CFG.apiUrl);
  });

  // ── Registration ───────────────────────────────────────────────────────
  describe("POST /api/v1/auth/register", () => {
    it("should register a new user", async () => {
      const email = `${testId("reg")}@vnpmemory.dev`;
      const { status, data } = await client.register(email, CFG.testPassword);
      expect(status).toBe(201);
      expect(data).toHaveProperty("id");
      expect(data).toHaveProperty("email", email);
      expect(data.is_active).toBe(true);
    });

    it("should reject duplicate email registration", async () => {
      const email = `${testId("dup")}@vnpmemory.dev`;
      await client.register(email, CFG.testPassword);
      const { status, data } = await client.register(email, CFG.testPassword);
      expect(status).toBe(400);
      expect(data.detail).toBe("REGISTER_USER_ALREADY_EXISTS");
    });

    it("should handle very short passwords", async () => {
      const email = `${testId("short")}@vnpmemory.dev`;
      const { status } = await client.register(email, "ab");
      // Cognee v1.0.3 may accept short passwords; test the endpoint responds
      expect([201, 400]).toContain(status);
    });

    it("should reject invalid email format", async () => {
      const { status } = await client.register("not-an-email", CFG.testPassword);
      expect(status).toBeGreaterThanOrEqual(400);
    });
  });

  // ── Login ──────────────────────────────────────────────────────────────
  describe("POST /api/v1/auth/login", () => {
    const email = `${testId("login")}@vnpmemory.dev`;

    beforeAll(async () => {
      await client.register(email, CFG.testPassword);
    });

    it("should login and return an access token", async () => {
      const { token, response } = await client.login(email, CFG.testPassword);
      expect(response.status).toBe(200);
      expect(token).toBeDefined();
      expect(typeof token).toBe("string");
      expect(token.length).toBeGreaterThan(10);
    });

    it("should reject wrong password", async () => {
      const fresh = new (client.constructor)(CFG.apiUrl);
      const { response } = await fresh.login(email, "wrongpassword");
      expect(response.status).toBe(400);
      expect(response.data.detail).toBe("LOGIN_BAD_CREDENTIALS");
    });

    it("should reject non-existent user", async () => {
      const fresh = new (client.constructor)(CFG.apiUrl);
      const { response } = await fresh.login("nobody@example.com", "pass123");
      expect(response.status).toBe(400);
    });
  });

  // ── Token-Protected Endpoints ──────────────────────────────────────────
  describe("GET /api/v1/auth/me", () => {
    const email = `${testId("me")}@vnpmemory.dev`;

    beforeAll(async () => {
      await client.register(email, CFG.testPassword);
      await client.login(email, CFG.testPassword);
    });

    it("should return the current user profile", async () => {
      const { status, data } = await client.get("/api/v1/auth/me");
      expect(status).toBe(200);
      expect(data).toHaveProperty("email", email);
    });

    it("should reject request without token", async () => {
      const fresh = new (client.constructor)(CFG.apiUrl);
      const { status } = await fresh.get("/api/v1/auth/me");
      expect(status).toBe(401);
    });
  });

  // ── Logout ─────────────────────────────────────────────────────────────
  describe("POST /api/v1/auth/logout", () => {
    it("should logout successfully", async () => {
      const email = `${testId("logout")}@vnpmemory.dev`;
      const logoutClient = new (client.constructor)(CFG.apiUrl);
      await logoutClient.register(email, CFG.testPassword);
      await logoutClient.login(email, CFG.testPassword);

      const { status } = await logoutClient.post("/api/v1/auth/logout");
      expect(status).toBe(200);
    });
  });

  // ── API Key Management ────────────────────────────────────────────────
  describe("API Keys", () => {
    const email = `${testId("apikey")}@vnpmemory.dev`;
    let apiKeyClient;

    beforeAll(async () => {
      apiKeyClient = new (client.constructor)(CFG.apiUrl);
      await apiKeyClient.register(email, CFG.testPassword);
      await apiKeyClient.login(email, CFG.testPassword);
    });

    it("should create an API key", async () => {
      const { status, data } = await apiKeyClient.post("/api/v1/auth/api-keys", {
        body: { name: testId("key") },
      });
      expect(status).toBe(200);
      expect(data).toHaveProperty("key");
      expect(typeof data.key).toBe("string");
    });

    it("should list API keys", async () => {
      const { status, data } = await apiKeyClient.get("/api/v1/auth/api-keys");
      expect(status).toBe(200);
      expect(Array.isArray(data)).toBe(true);
    });

    it("should delete an API key", async () => {
      // Create key
      const createRes = await apiKeyClient.post("/api/v1/auth/api-keys", {
        body: { name: testId("del") },
      });
      const keyId = createRes.data.id;

      // Delete key
      const { status } = await apiKeyClient.delete(`/api/v1/auth/api-keys/${keyId}`);
      expect(status).toBeLessThan(300);
    });
  });
});
