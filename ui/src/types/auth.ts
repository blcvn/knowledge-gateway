/**
 * Authentication Types
 * Matches backend JWT RS256 response schema
 */

export interface AuthUser {
  id:          string;
  name:        string;
  email:       string;
  role:        string;
  tenant_id:   string;
  avatar_url?: string;
}

export interface LoginRequest {
  email:    string;
  password: string;
}

export interface LoginResponse {
  access_token:  string;
  refresh_token: string;
  expires_in:    number;
  token_type:    'Bearer';
  user:          AuthUser;
}

export interface RefreshResponse {
  access_token: string;
  expires_in:   number;
}
