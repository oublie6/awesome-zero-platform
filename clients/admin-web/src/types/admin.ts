export interface Account { id: string; username?: string; email?: string; phone?: string; displayName: string; status: 'active' | 'disabled'; createdAt: string; updatedAt: string }
export interface AccountPage { items: Account[]; total: number; page: number; pageSize: number }
export interface Role { code: string; displayName: string; description: string; system: boolean; createdAt: string; updatedAt: string }
export interface Resource { code: string; displayName: string; module: string; pattern: string; actions: string[]; description: string; system: boolean; createdAt: string; updatedAt: string }
export interface Permission { resource: string; action: string }
export interface RawRule { ptype: string; values: string[] }
export interface EngineInfo { id: string; name: string; version: string; modelType: string; policyTypes: string[]; supportsRawPolicy: boolean; supportsModelInspection: boolean; supportsModelEditing: boolean; supportsPolicyExplanation: boolean; supportsBatchImport: boolean; supportsRoleHierarchy: boolean; loadedAt: string }
export interface Explanation { allowed: boolean; subject: string; resource: string; action: string; roles: string[]; matchedRules: RawRule[] }
export interface SessionView { id: string; accountId: string; createdAt: string; expiresAt: string }
export interface AuditEvent { id: string; actorId?: string; action: string; resourceType: string; resourceId?: string; outcome: string; requestId?: string; clientIp?: string; userAgent?: string; details?: Record<string, unknown>; createdAt: string }
export interface Me { account: Account; roles: string[]; permissions: Permission[] }
