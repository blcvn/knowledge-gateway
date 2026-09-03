/**
 * Governance Types — extends with missing AuditFilters and GDPRPreviewResponse
 * TASK-API-009
 */

export interface Tenant {
  id:         string;
  name:       string;
  slug?:      string;
  plan?:      'free' | 'pro' | 'enterprise';
  created_at: string;
  status:     'Active' | 'Suspended' | 'active' | 'suspended';
}

export interface Policy {
  id:          string;
  name:        string;
  description?: string;
  rego_code:   string;
  scope:       string;
  enabled:     boolean;
  tenant_id?:  string;
  created_at?: string;
}

export interface AuditLogEntry {
  id:          string;
  tenant_id:   string;
  actor_id:    string;
  action:      string;
  entity_type: string;
  entity_id?:  string;
  result:      string;
  created_at:  string;
}

export interface AuditFilters {
  action?:      string;
  actor_id?:    string;
  entity_type?: string;
  from?:        string;
  to?:          string;
}

export interface GDPRPreviewResponse {
  user_id:             string;
  estimated_items:     number;
  breakdown_by_engine: Record<string, number>;
  warnings:            string[];
}

export interface GDPRForgetResponse {
  success:       boolean;
  deleted_count: number;
}

export interface GDPRRequest {
  id:      string;
  user_id: string;
  status:  'pending' | 'in_progress' | 'completed' | 'failed';
}

export interface RetentionRule {
  id:       string;
  engine:   string;
  ttl_days: number;
}
