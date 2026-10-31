<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { DataAnalysis, User, Key, Setting, Document, Lock, Fold, Expand, SwitchButton } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const collapsed = ref(typeof window !== 'undefined' && window.innerWidth <= 760)
const mobile = ref(typeof window !== 'undefined' && window.innerWidth <= 760)
const mobileMenuOpen = ref(false)
function syncViewport() {
  mobile.value = window.innerWidth <= 760
  if (!mobile.value) mobileMenuOpen.value = false
}
onMounted(() => { window.addEventListener('resize', syncViewport) })
onUnmounted(() => window.removeEventListener('resize', syncViewport))
watch(() => route.path, () => { mobileMenuOpen.value = false })
const title = computed(() => String(route.meta.title || 'Admin'))

const canView = (resource: string) => auth.can(resource, 'GET')
const canViewIdentity = computed(() => canView('/admin/accounts') || canView('/admin/roles'))
const canViewAuthorization = computed(() => canView('/admin/authorization/resources') || canView('/admin/authorization/engine'))

async function signOut() {
  try {
    await auth.logout()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '服务端退出失败，已清除此浏览器会话')
  } finally {
    await router.replace('/login')
  }
}
</script>

<template>
  <div class="shell">
    <div v-if="mobileMenuOpen" class="mobile-backdrop" @click="mobileMenuOpen = false" />
    <aside
      v-if="!mobile || mobileMenuOpen"
      :class="['sidebar', { collapsed: !mobile && collapsed, 'mobile-sidebar': mobile }]"
      aria-label="管理导航"
    >
      <div class="brand">
        <div class="brand-mark">AZ</div>
        <div v-show="!collapsed" class="brand-copy">
          <strong>Awesome Zero</strong>
          <small>PLATFORM ADMIN</small>
        </div>
      </div>
      <el-menu router :default-active="route.path" :collapse="!mobile && collapsed" background-color="transparent" text-color="#aebdca" active-text-color="#fff" @select="mobileMenuOpen = false">
        <el-menu-item v-if="canView('/admin/system/overview')" index="/dashboard"><el-icon><DataAnalysis /></el-icon><template #title>运行概览</template></el-menu-item>
        <el-sub-menu v-if="canViewIdentity" index="identity">
          <template #title><el-icon><User /></el-icon><span>身份管理</span></template>
          <el-menu-item v-if="canView('/admin/accounts')" index="/accounts">账号管理</el-menu-item>
          <el-menu-item v-if="canView('/admin/roles')" index="/roles">角色管理</el-menu-item>
        </el-sub-menu>
        <el-sub-menu v-if="canViewAuthorization" index="authorization">
          <template #title><el-icon><Key /></el-icon><span>权限管理</span></template>
          <el-menu-item v-if="canView('/admin/authorization/resources')" index="/authorization/standard">标准配置</el-menu-item>
          <el-menu-item v-if="canView('/admin/authorization/engine')" index="/authorization/engine">权限引擎 <span class="expert-dot">DEV</span></el-menu-item>
        </el-sub-menu>
        <el-menu-item v-if="canView('/admin/audit/events')" index="/audit"><el-icon><Document /></el-icon><template #title>操作审计</template></el-menu-item>
        <el-menu-item v-if="canView('/admin/system/overview')" index="/system"><el-icon><Setting /></el-icon><template #title>系统状态</template></el-menu-item>
      </el-menu>
    </aside>

    <main>
      <header class="topbar">
        <el-button
          text
          circle
          :aria-label="mobile ? '打开导航菜单' : collapsed ? '展开导航' : '收起导航'"
          @click="mobile ? mobileMenuOpen = !mobileMenuOpen : collapsed = !collapsed"
        >
          <el-icon><component :is="mobile ? (mobileMenuOpen ? Fold : Expand) : (collapsed ? Expand : Fold)" /></el-icon>
        </el-button>
        <div class="crumb"><span>ADMIN</span><b>/</b>{{ title }}</div>
        <div class="top-actions">
          <el-tag v-if="auth.roles.includes('platform_super_admin')" type="danger" effect="plain">SUPER ADMIN</el-tag>
          <el-dropdown>
            <span class="user-trigger"><el-icon><Lock /></el-icon><span class="user-name">{{ auth.displayName }}</span></span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-if="canView('/admin/me')" @click="router.push('/profile')">个人中心</el-dropdown-item>
                <el-dropdown-item divided @click="signOut"><el-icon><SwitchButton /></el-icon>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>
      <section class="content"><router-view /></section>
    </main>
  </div>
</template>

<style scoped>
.shell {
  min-height: 100vh;
  min-width: 0;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
}

.sidebar {
  width: 248px;
  overflow: hidden;
  border-right: 1px solid #24303d;
  background: linear-gradient(180deg, #101923, #0b1118);
  transition: width 0.2s;
}

.sidebar.collapsed {
  width: 64px;
}

.mobile-backdrop { display: none; }

.brand {
  height: 64px;
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 0 14px;
  border-bottom: 1px solid #24303d;
}

.brand-mark {
  min-width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  border: 1px solid #38bdf8;
  border-radius: 8px;
  box-shadow: 0 0 22px #0ea5e933;
  color: #7dd3fc;
  font: bold 14px var(--az-mono);
}

.brand-copy {
  min-width: 0;
  white-space: nowrap;
}

.brand strong {
  display: block;
  color: #eef6ff;
}

.brand small {
  display: block;
  margin-top: 2px;
  color: #64748b;
  font: 10px var(--az-mono);
  letter-spacing: 1.2px;
}

.el-menu {
  border: 0;
}

.topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  height: 64px;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 22px;
  border-bottom: 1px solid var(--az-border);
  background: color-mix(in srgb, var(--az-panel) 92%, transparent);
}

.crumb {
  min-width: 0;
  overflow: hidden;
  color: var(--az-muted);
  font: 12px var(--az-mono);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.crumb span {
  color: #38bdf8;
}

.crumb b {
  padding: 0 9px;
}

.top-actions {
  margin-left: auto;
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 12px 16px;
}

.user-trigger {
  max-width: 220px;
  display: flex;
  align-items: center;
  gap: 7px;
  cursor: pointer;
  color: var(--az-text);
  white-space: nowrap;
}

.user-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

main {
  min-width: 0;
}

.content {
  max-width: 1680px;
  min-width: 0;
  margin: auto;
  padding: 24px;
}

.expert-dot {
  margin-left: 8px;
  color: #fb7185;
  font: 9px var(--az-mono);
}

@media (max-width: 980px) {
  .sidebar:not(.collapsed) {
    width: 210px;
  }

  .content {
    padding: 18px;
  }

  .top-actions {
    gap: 10px;
  }
}

@media (max-width: 760px) {
  .shell { grid-template-columns: minmax(0, 1fr); }
  .sidebar.mobile-sidebar {
    position: fixed;
    inset: 0 auto 0 0;
    z-index: 31;
    width: min(280px, 84vw);
    overflow-y: auto;
    box-shadow: 16px 0 40px #0008;
  }
  .mobile-backdrop {
    display: block;
    position: fixed;
    inset: 0;
    z-index: 30;
    background: #0009;
  }
  .brand { padding: 0 14px; }
  .brand-copy { display: block; }
  .topbar { padding: 0 10px; }
  .content { padding: 12px; }
  .user-trigger { max-width: 110px; }
  .top-actions .el-tag { display: none; }
  .crumb b { padding: 0 3px; }
  .topbar > .el-button { min-width: 40px; min-height: 40px; }
}
</style>
