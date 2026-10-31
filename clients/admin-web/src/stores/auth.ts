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

let meRequest: Promise<Me> | null = null
let authGeneration = 0

export const useAuthStore = defineStore('auth', {
  state: () => ({ me: null as Me | null, loading: false }),
  getters: {
    authenticated: state => Boolean(state.me) || hasToken(),
    displayName: state => state.me?.account.displayName || '管理员',
    roles: state => state.me?.roles || [],
    permissions: state => state.me?.permissions || []
  },
  actions: {
    async login(identifier: string, password: string) {
      authGeneration += 1
      meRequest = null
      this.me = null
      this.loading = false
      await apiLogin(identifier, password)
      try {
        await this.loadMe()
      } catch (error) {
        await apiLogout().catch(() => undefined)
        this.me = null
        throw error
      }
    },
    async loadMe() {
      if (!hasToken()) { this.me = null; return }
      if (this.me) return
      if (!meRequest) {
        const generation = authGeneration
        this.loading = true
        let request: Promise<Me>
        request = adminApi.me().then(me => {
          if (generation === authGeneration && hasToken()) this.me = me
          return me
        }).finally(() => {
          if (meRequest === request) {
            if (generation === authGeneration) this.loading = false
            meRequest = null
          }
        })
        meRequest = request
      }
      await meRequest
    },
    async logout() {
      authGeneration += 1
      meRequest = null
      this.me = null
      this.loading = false
      await apiLogout()
    },
    can(path: string, action = 'GET') { return this.permissions.some((p: Permission) => pathMatch(path, p.resource) && actionMatch(action, p.action)) }
  }
})
