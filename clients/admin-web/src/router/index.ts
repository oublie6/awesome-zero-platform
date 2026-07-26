import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { hasToken } from '@/api/client'

const routes: RouteRecordRaw[] = [
  { path: '/login', component: () => import('@/views/LoginView.vue'), meta: { public: true, title: '登录' } },
  { path: '/bootstrap', component: () => import('@/views/BootstrapView.vue'), meta: { public: true, title: '初始化管理员' } },
  {
    path: '/', component: () => import('@/layouts/AdminLayout.vue'), redirect: '/dashboard', children: [
      { path: 'dashboard', component: () => import('@/views/DashboardView.vue'), meta: { title: '运行概览' } },
      { path: 'accounts', component: () => import('@/views/accounts/AccountsView.vue'), meta: { title: '账号管理' } },
      { path: 'roles', component: () => import('@/views/roles/RolesView.vue'), meta: { title: '角色管理' } },
      { path: 'authorization/standard', component: () => import('@/views/authorization/StandardView.vue'), meta: { title: '权限配置' } },
      { path: 'authorization/engine', component: () => import('@/views/authorization/EngineView.vue'), meta: { title: '权限引擎 · 专家模式', expert: true } },
      { path: 'audit', component: () => import('@/views/AuditView.vue'), meta: { title: '操作审计' } },
      { path: 'system', component: () => import('@/views/system/SystemView.vue'), meta: { title: '系统状态' } },
      { path: 'profile', component: () => import('@/views/ProfileView.vue'), meta: { title: '个人中心' } }
    ]
  },
  { path: '/:pathMatch(.*)*', component: () => import('@/views/NotFoundView.vue'), meta: { public: true, title: '页面不存在' } }
]
const router = createRouter({ history: createWebHistory(), routes })
router.beforeEach(to => {
  document.title = `${String(to.meta.title || 'Admin')} · Awesome Zero Platform`
  if (!to.meta.public && !hasToken()) return { path: '/login', query: { redirect: to.fullPath } }
  if (to.path === '/login' && hasToken()) return '/dashboard'
})
window.addEventListener('admin-auth-expired', () => router.replace({ path: '/login', query: { expired: '1' } }))
export default router
