import type { Tenant, Policy, AuditLogEntry } from '../types/governance';

export const governanceMock = {
  tenants: [{ id: 't_1', name: 'Default Tenant', created_at: new Date().toISOString(), status: 'Active' }] as Tenant[],
  policies: [{ id: 'p_1', name: 'Admin Only', rego_code: 'package authz\nallow { input.role == "admin" }', scope: 'global', enabled: true }] as Policy[],
  auditLogs: [{ id: 'log_1', tenant_id: 't_1', actor_id: 'admin', action: 'CREATE', entity_type: 'Policy', created_at: new Date().toISOString(), result: 'success' }] as AuditLogEntry[],
};
