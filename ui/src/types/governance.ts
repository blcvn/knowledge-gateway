export interface Tenant {
  id: string;
  name: string;
  created_at: string;
  status: 'Active' | 'Suspended';
}

export interface Policy {
  id: string;
  name: string;
  description?: string;
  rego_code: string;
  scope: string;
  enabled: boolean;
}

export interface AuditLogEntry {
  id: string;
  tenant_id: string;
  actor_id: string;
  action: string;
  entity_type: string;
  created_at: string;
  result: string;
}

export interface GDPRRequest {
  id: string;
  user_id: string;
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
}

export interface RetentionRule {
  id: string;
  engine: string;
  ttl_days: number;
}
