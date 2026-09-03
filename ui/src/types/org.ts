/**
 * Org & SDK Types — for Organization Settings and API Key management
 * TASK-API-013
 */

// ─── Org Settings ─────────────────────────────────────────────────────────────

export interface OrgSettings {
  name:                   string;
  slug:                   string;
  domain?:                string;
  timezone:               string;
  max_agents:             number;
  max_memories_per_user:  number;
  plan?:                  'free' | 'pro' | 'enterprise';
}

// ─── Members & Roles ─────────────────────────────────────────────────────────

export interface OrgMember {
  id:        string;
  name:      string;
  email:     string;
  role:      string;
  status:    'active' | 'inactive';
  joined_at: string;
}

export interface OrgRole {
  id:          string;
  name:        string;
  permissions: string[];
}

// ─── API Keys ─────────────────────────────────────────────────────────────────

export interface APIKey {
  id:         string;
  name:       string;
  prefix:     string;    // visible prefix e.g. "vnp_prod_sk_3f9a..."
  scopes:     string[];
  created_at: string;
  last_used?: string;
  expires_at?: string;
  status:     'active' | 'revoked' | 'expired';
}

/**
 * Response from POST /v1/console/sdk/keys
 * IMPORTANT: raw_key is shown ONCE — UI must capture and display immediately
 */
export interface CreateKeyResponse {
  key:     APIKey;
  raw_key: string;  // Full key, never stored/shown again after initial display
}

export interface CreateKeyPayload {
  name:              string;
  permissions:       string[];
  expires_in_days?:  number;
}

// ─── Rate Limits ──────────────────────────────────────────────────────────────

export interface RateLimitConfig {
  scope:   string;
  rps:     number;
  rpm:     number;
  burst:   number;
  tier:    'enterprise' | 'standard' | 'restricted';
}

// ─── Webhooks ─────────────────────────────────────────────────────────────────

export interface Webhook {
  id:           string;
  url:          string;
  events:       string[];
  status:       'active' | 'paused' | 'failed';
  secret?:      string;
  success_rate: number;
  created_at:   string;
}

export interface CreateWebhookPayload {
  url:     string;
  events:  string[];
  secret?: string;
}
