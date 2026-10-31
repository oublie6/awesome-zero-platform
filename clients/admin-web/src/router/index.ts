import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { hasToken } from '@/api/client'
import { useAuthStore } from '@/stores/auth'

declare module 'vue-router' {
  interface RouteMeta {
    public?: boolean
    title?: string
    permission?: { resource: string; action: string }
  }
}

const routes: RouteRecordRaw[] = [
  { path: '/login', component: () => import('@/views/LoginView.vue'), meta: { public: true, title: '登录' } },
  { path: '/bootstrap', component: () => import('@/views/BootstrapView.vue'), meta: { public: true, title: '初始化管理员' } },
  {
    path: '/', component: () => import('@/layouts/AdminLayout.vue'), redirect: '/dashboard', children: [
      { path: 'dashboard', component: () => import('@/views/DashboardView.vue'), meta: { title: '运行概览', permission: { resource: '/admin/system/overview', action: 'GET' } } },
      { path: 'accounts', component: () => import('@/views/accounts/AccountsView.vue'), meta: { title: '账号管理', permission: { resource: '/admin/accounts', action: 'GET' } } },
      { path: 'roles', component: () => import('@/views/roles/RolesView.vue'), meta: { title: '角色管理', permission: { resource: '/admin/roles', action: 'GET' } } },
      { path: 'authorization/standard', component: () => import('@/views/authorization/StandardView.vue'), meta: { title: '权限配置', permission: { resource: '/admin/authorization/resources', action: 'GET' } } },
      { path: 'authorization/engine', component: () => import('@/views/authorization/EngineView.vue'), meta: { title: '权限引擎 · 专家模式', expert: true, permission: { resource: '/admin/authorization/engine', action: 'GET' } } },
      { path: 'audit', component: () => import('@/views/AuditView.vue'), meta: { title: '操作审计', permission: { resource: '/admin/audit/events', action: 'GET' } } },
      { path: 'system', component: () => import('@/views/system/SystemView.vue'), meta: { title: '系统状态', permission: { resource: '/admin/system/overview', action: 'GET' } } },
      { path: 'profile', component: () => import('@/views/ProfileView.vue'), meta: { title: '个人中心', permission: { resource: '/admin/me', action: 'GET' } } },
      { path: '403', component: () => import('@/views/ForbiddenView.vue'), meta: { title: '无权访问' } }
    ]
  },
  { path: '/:pathMatch(.*)*', component: () => import('@/views/NotFoundView.vue'), meta: { public: true, title: '页面不存在' } }
]
const router = createRouter({ history: createWebHistory(), routes })
router.beforeEach(to => {
  document.title = `${String(to.meta.title || 'Admin')} · Awesome Zero Platform`
  if (!to.meta.public && !hasToken()) return { path: '/login', query: { redirect: to.fullPath } }
  const auth = useAuthStore()
  if (to.meta.public) {
    if (to.path !== '/login' || !hasToken()) return true
    return auth.loadMe().then(() => '/dashboard').catch(() => true)
  }
  return auth.loadMe().then(() => {
    const required = to.meta.permission
    if (required && !auth.can(required.resource, required.action)) return '/403'
    return true
  }).catch(() => ({ path: '/login', query: { redirect: to.fullPath } }))
})
window.addEventListener('admin-auth-expired', () => router.replace({ path: '/login', query: { expired: '1' } }))
export default router
