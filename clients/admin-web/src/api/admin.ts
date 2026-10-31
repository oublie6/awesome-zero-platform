import { request } from './client'
import type { Account, AccountPage, AuditEvent, EngineInfo, Explanation, Me, Permission, RawRule, Resource, Role, SessionView } from '@/types/admin'

const qs = (params: Record<string, string | number | undefined>) => { const search = new URLSearchParams(); Object.entries(params).forEach(([k,v]) => { if (v !== undefined && v !== '') search.set(k, String(v)) }); return search.toString() }
export const adminApi = {
  bootstrapStatus: (signal?: AbortSignal) => request<{ available: boolean }>('/admin/bootstrap/status', { signal }),
  bootstrap: (token: string, body: { username: string; displayName: string; password: string }) => request<Account>('/admin/bootstrap', { method: 'POST', headers: { 'X-Admin-Bootstrap-Token': token }, body: JSON.stringify(body) }),
  me: () => request<Me>('/admin/me'),
  accounts: (params: { search?: string; status?: string; page: number; pageSize: number }, signal?: AbortSignal) => request<AccountPage>(`/admin/accounts?${qs(params)}`, { signal }),
  account: (id: string) => request<{ account: Account; roles: string[] }>(`/admin/accounts/${encodeURIComponent(id)}`),
  createAccount: (body: Record<string, unknown>) => request<Account>('/admin/accounts', { method: 'POST', body: JSON.stringify(body) }),
  updateAccount: (id: string, body: Record<string, unknown>) => request<Account>(`/admin/accounts/${encodeURIComponent(id)}`, { method: 'PATCH', body: JSON.stringify(body) }),
  setAccount: (id: string, enabled: boolean) => request<Account>(`/admin/accounts/${encodeURIComponent(id)}/${enabled ? 'enable' : 'disable'}`, { method: 'POST' }),
  resetPassword: (id: string, password: string) => request(`/admin/accounts/${encodeURIComponent(id)}/reset-password`, { method: 'POST', body: JSON.stringify({ password }) }),
  sessions: (id: string, signal?: AbortSignal) => request<SessionView[]>(`/admin/accounts/${encodeURIComponent(id)}/sessions`, { signal }),
  revokeSessions: (id: string) => request<{ revoked: number }>(`/admin/accounts/${encodeURIComponent(id)}/revoke-sessions`, { method: 'POST' }),
  accountRoles: (id: string, signal?: AbortSignal) => request<string[]>(`/admin/accounts/${encodeURIComponent(id)}/roles`, { signal }),
  replaceAccountRoles: (id: string, roles: string[]) => request(`/admin/accounts/${encodeURIComponent(id)}/roles`, { method: 'PUT', body: JSON.stringify({ roles }) }),
  roles: (signal?: AbortSignal) => request<Role[]>('/admin/roles', { signal }),
  createRole: (body: Partial<Role>) => request<Role>('/admin/roles', { method: 'POST', body: JSON.stringify(body) }),
  updateRole: (code: string, body: Partial<Role>) => request<Role>(`/admin/roles/${encodeURIComponent(code)}`, { method: 'PATCH', body: JSON.stringify(body) }),
  deleteRole: (code: string) => request(`/admin/roles/${encodeURIComponent(code)}`, { method: 'DELETE' }),
  rolePermissions: (code: string, signal?: AbortSignal) => request<Permission[]>(`/admin/roles/${encodeURIComponent(code)}/permissions`, { signal }),
  replaceRolePermissions: (code: string, permissions: Permission[]) => request(`/admin/roles/${encodeURIComponent(code)}/permissions`, { method: 'PUT', body: JSON.stringify({ permissions }) }),
  resources: (signal?: AbortSignal) => request<Resource[]>('/admin/authorization/resources', { signal }),
  createResource: (body: Partial<Resource>) => request<Resource>('/admin/authorization/resources', { method: 'POST', body: JSON.stringify(body) }),
  updateResource: (code: string, body: Partial<Resource>) => request<Resource>(`/admin/authorization/resources/${encodeURIComponent(code)}`, { method: 'PATCH', body: JSON.stringify(body) }),
  deleteResource: (code: string) => request(`/admin/authorization/resources/${encodeURIComponent(code)}`, { method: 'DELETE' }),
  engine: (signal?: AbortSignal) => request<EngineInfo>('/admin/authorization/engine', { signal }),
  model: (signal?: AbortSignal) => request<{ model: string }>('/admin/authorization/engine/model', { signal }),
  policies: (signal?: AbortSignal) => request<RawRule[]>('/admin/authorization/engine/policies', { signal }),
  validatePolicies: (rules: RawRule[]) => request<{ valid: boolean }>('/admin/authorization/engine/policies/validate', { method: 'POST', body: JSON.stringify({ rules }) }),
  replacePolicies: (rules: RawRule[]) => request('/admin/authorization/engine/policies', { method: 'PUT', body: JSON.stringify({ rules }) }),
  explain: (body: { subject: string; resource: string; action: string }) => request<Explanation>('/admin/authorization/engine/explain', { method: 'POST', body: JSON.stringify(body) }),
  audit: (params: { search?: string; action?: string; outcome?: string; page: number; pageSize: number }, signal?: AbortSignal) => request<{ items: AuditEvent[]; total: number; page: number; pageSize: number }>(`/admin/audit/events?${qs(params)}`, { signal }),
  overview: (signal?: AbortSignal) => request<Record<string, unknown>>('/admin/system/overview', { signal }),
  configuration: (signal?: AbortSignal) => request<Record<string, unknown>>('/admin/system/configuration', { signal })
}
