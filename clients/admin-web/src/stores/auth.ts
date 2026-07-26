import { defineStore } from 'pinia'
import { adminApi } from '@/api/admin'
import { hasToken, login as apiLogin, logout as apiLogout } from '@/api/client'
import type { Me, Permission } from '@/types/admin'

function pathMatch(path: string, pattern: string): boolean {
  if (pattern === '/*' || pattern === '*') return true
  const escaped = pattern.replace(/[.+?^${}()|[\]\\]/g, '\\$&').replace(/\*/g, '.*').replace(/:[^/]+/g, '[^/]+')
  return new RegExp(`^${escaped}$`).test(path)
}
function actionMatch(action: string, pattern: string): boolean { try { return new RegExp(pattern).test(action) } catch { return action === pattern } }

export const useAuthStore = defineStore('auth', {
  state: () => ({ me: null as Me | null, loading: false }),
  getters: {
    authenticated: state => Boolean(state.me) || hasToken(),
    displayName: state => state.me?.account.displayName || '管理员',
    roles: state => state.me?.roles || [],
    permissions: state => state.me?.permissions || []
  },
  actions: {
    async login(identifier: string, password: string) { await apiLogin(identifier, password); await this.loadMe() },
    async loadMe() { if (!hasToken()) { this.me = null; return }; this.loading = true; try { this.me = await adminApi.me() } finally { this.loading = false } },
    async logout() { await apiLogout(); this.me = null },
    can(path: string, action = 'GET') { return this.permissions.some((p: Permission) => pathMatch(path, p.resource) && actionMatch(action, p.action)) }
  }
})
