<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import AccountRowActions from '@/components/AccountRowActions.vue'
import PageHeader from '@/components/PageHeader.vue'
import { adminApi } from '@/api/admin'
import { useAuthStore } from '@/stores/auth'
import type { Account, Role, SessionView } from '@/types/admin'

const items = ref<Account[]>([])
const roles = ref<Role[]>([])
const total = ref(0)
const loading = ref(false)
const saving = ref(false)
const loadError = ref('')
const dialog = ref(false)
const detail = ref(false)
const editing = ref<Account | null>(null)
const sessions = ref<SessionView[]>([])
const detailLoading = ref(false)
const detailError = ref('')
const roleSaving = ref(false)
const revoking = ref(false)
const assigned = ref<string[]>([])
type PendingAccountAction = 'toggle' | 'reset'
const pendingActions = reactive<Record<string, PendingAccountAction>>({})
const query = reactive({ search: '', status: '', page: 1, pageSize: 20 })
const form = reactive({ username: '', email: '', phone: '', displayName: '', status: 'active', password: '', roles: [] as string[] })
const formRef = ref()
const auth = useAuthStore()
const canReadRoles = auth.can('/admin/roles', 'GET')
const canReadAccountRoles = auth.can('/admin/accounts/_/roles', 'GET')
const canReadSessions = auth.can('/admin/accounts/_/sessions', 'GET')
const canReplaceAccountRoles = auth.can('/admin/accounts/_/roles', 'PUT')
const canRevokeSessions = auth.can('/admin/accounts/_/revoke-sessions', 'POST')
let listController: AbortController | undefined
let detailController: AbortController | undefined

const formRules = {
  username: [{ validator: (_rule: unknown, _value: string, done: (error?: Error) => void) => {
    if (![form.username, form.email, form.phone].some(value => value.trim())) done(new Error('用户名、邮箱或手机号至少填写一项'))
    else done()
  }, trigger: 'blur' }],
  displayName: [{ required: true, whitespace: true, message: '请输入显示名称', trigger: 'blur' }],
  password: [{ validator: (_rule: unknown, value: string, done: (error?: Error) => void) => {
    if (!editing.value && new TextEncoder().encode(value || '').length < 8) done(new Error('密码至少需要 8 个字节'))
    else done()
  }, trigger: 'blur' }],
  email: [{ validator: (_rule: unknown, value: string, done: (error?: Error) => void) => {
    const email = (value || '').trim()
    // Keep basic client checks compatible with the server's net/mail parser,
    // including local domains and quoted local parts. The server validates RFC syntax.
    if (email && (email.length > 320 || /[^\x00-\x7f]/.test(email) || !/^[^@]+@[^@]+$/.test(email))) done(new Error('请输入有效邮箱地址'))
    else done()
  }, trigger: 'blur' }],
  phone: [{ validator: (_rule: unknown, value: string, done: (error?: Error) => void) => {
    const phone = (value || '').trim()
    if (phone && !/^\+[1-9]\d{1,14}$/.test(phone)) done(new Error('手机号须使用 E.164 格式'))
    else done()
  }, trigger: 'blur' }],
}
const allowedRowActions = [
  ...(auth.can('/admin/accounts/_', 'GET') ? ['detail' as const] : []),
  ...(auth.can('/admin/accounts/_', 'PATCH') ? ['edit' as const] : []),
  ...(auth.can('/admin/accounts/_/disable', 'POST') ? ['toggle' as const] : []),
  ...(auth.can('/admin/accounts/_/reset-password', 'POST') ? ['reset' as const] : []),
]

async function load() {
  listController?.abort()
  const controller = new AbortController()
  listController = controller
  loading.value = true
  loadError.value = ''
  try {
    const [page, roleList] = await Promise.all([
      adminApi.accounts(query, controller.signal),
      canReadRoles ? adminApi.roles(controller.signal) : Promise.resolve([]),
    ])
    if (controller.signal.aborted) return
    items.value = page.items
    total.value = page.total
    roles.value = roleList
  } catch (error) {
    if (controller.signal.aborted) return
    loadError.value = error instanceof Error ? error.message : '加载账号列表失败'
  } finally {
    if (!controller.signal.aborted) loading.value = false
  }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { username: '', email: '', phone: '', displayName: '', status: 'active', password: '', roles: [] })
  dialog.value = true
}

function openEdit(row: Account) {
  editing.value = row
  Object.assign(form, { username: row.username || '', email: row.email || '', phone: row.phone || '', displayName: row.displayName, status: row.status, password: '', roles: [] })
  dialog.value = true
}

async function save() {
  if (saving.value) return
  const valid = await formRef.value?.validate().then(() => true).catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    if (editing.value) {
      await adminApi.updateAccount(editing.value.id, { username: form.username, email: form.email, phone: form.phone, displayName: form.displayName })
    } else {
      await adminApi.createAccount(form)
    }
    dialog.value = false
    ElMessage.success('账号已保存')
    await load()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function toggle(row: Account) {
  if (pendingActions[row.id]) return
  pendingActions[row.id] = 'toggle'
  try {
    await ElMessageBox.confirm(`${row.status === 'active' ? '禁用' : '启用'}账号 ${row.displayName}？`, '确认', {
      confirmButtonText: row.status === 'active' ? '禁用' : '启用', cancelButtonText: '取消',
    })
    await adminApi.setAccount(row.id, row.status !== 'active')
    ElMessage.success(row.status === 'active' ? '账号已禁用' : '账号已启用')
    await load()
  } catch (error) {
    if (error instanceof Error) ElMessage.error(error.message)
  } finally {
    delete pendingActions[row.id]
  }
}

async function showDetail(row: Account) {
  detailController?.abort()
  const controller = new AbortController()
  detailController = controller
  editing.value = row
  detail.value = true
  detailError.value = ''
  assigned.value = []
  sessions.value = []
  detailLoading.value = true
  try {
    const [roleList, sessionList] = await Promise.all([
      canReadAccountRoles ? adminApi.accountRoles(row.id, controller.signal) : Promise.resolve([]),
      canReadSessions ? adminApi.sessions(row.id, controller.signal) : Promise.resolve([]),
    ])
    if (controller.signal.aborted) return
    assigned.value = roleList
    sessions.value = sessionList
  } catch (error) {
    if (!controller.signal.aborted) detailError.value = error instanceof Error ? error.message : '加载账号详情失败'
  } finally {
    if (!controller.signal.aborted) detailLoading.value = false
  }
}

async function saveRoles() {
  if (!editing.value || roleSaving.value) return
  roleSaving.value = true
  try {
    await adminApi.replaceAccountRoles(editing.value.id, assigned.value)
    ElMessage.success('角色已更新')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '角色更新失败')
  } finally {
    roleSaving.value = false
  }
}

async function revoke() {
  if (!editing.value || revoking.value) return
  try {
    await ElMessageBox.confirm(`确认撤销 ${editing.value.displayName} 的全部活动会话？该用户需要重新登录。`, '撤销全部会话', { type: 'warning', confirmButtonText: '撤销会话', cancelButtonText: '取消' })
    revoking.value = true
    const result = await adminApi.revokeSessions(editing.value.id)
    ElMessage.success(`已撤销 ${result.revoked} 个会话`)
    sessions.value = []
  } catch (error) {
    if (error instanceof Error) ElMessage.error(error.message)
  } finally {
    revoking.value = false
  }
}

async function reset(row: Account) {
  if (pendingActions[row.id]) return
  pendingActions[row.id] = 'reset'
  try {
    const { value } = await ElMessageBox.prompt('输入新的临时密码，重置后该账号全部会话会被撤销。', '重置密码', {
      inputType: 'password',
      confirmButtonText: '确定', cancelButtonText: '取消',
      inputValidator: value => new TextEncoder().encode(value || '').length >= 8 || '密码至少需要 8 个字节',
    })
    await adminApi.resetPassword(row.id, value)
    ElMessage.success('密码已重置，原有会话已撤销')
  } catch (error) {
    if (error instanceof Error) ElMessage.error(error.message)
  } finally {
    delete pendingActions[row.id]
  }
}

onBeforeUnmount(() => { listController?.abort(); detailController?.abort() })
onMounted(load)
</script>

<template>
  <PageHeader title="账号管理" description="默认视图完成账号生命周期；详情抽屉提供 UUID、角色原始编码和 Redis 会话等技术信息。">
    <el-button v-if="auth.can('/admin/accounts', 'POST')" type="primary" @click="openCreate">创建账号</el-button>
  </PageHeader>

  <div class="panel">
    <el-alert v-if="loadError" type="error" :title="loadError" :closable="false" show-icon style="margin-bottom:12px" />
    <el-button v-if="loadError" @click="load" style="margin-bottom:12px">重试加载</el-button>
    <div class="toolbar">
      <el-input v-model="query.search" clearable placeholder="用户名 / 邮箱 / 手机号 / 显示名" style="width: 320px" @keyup.enter="query.page = 1; load()" @clear="query.page = 1; load()" />
      <el-select v-model="query.status" clearable placeholder="状态" style="width: 130px">
        <el-option label="启用" value="active" />
        <el-option label="禁用" value="disabled" />
      </el-select>
      <el-button :loading="loading" @click="query.page = 1; load()">查询</el-button>
    </div>

    <el-table :data="items" v-loading="loading">
      <el-table-column prop="displayName" label="账号" min-width="170">
        <template #default="scope">
          <b>{{ scope.row.displayName }}</b>
          <div class="muted technical">{{ scope.row.username || scope.row.email || scope.row.phone }}</div>
        </template>
      </el-table-column>
      <el-table-column prop="id" label="Account UUID" min-width="280">
        <template #default="scope"><span class="technical">{{ scope.row.id }}</span></template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="90">
        <template #default="scope"><el-tag :type="scope.row.status === 'active' ? 'success' : 'info'">{{ scope.row.status }}</el-tag></template>
      </el-table-column>
      <el-table-column prop="updatedAt" label="更新时间" width="190" />
      <el-table-column label="操作" min-width="190" fixed="right">
        <template #default="scope">
          <AccountRowActions
            :status="scope.row.status"
            :allowed="allowedRowActions"
            :loading-action="pendingActions[scope.row.id]"
            @detail="showDetail(scope.row as Account)"
            @edit="openEdit(scope.row as Account)"
            @toggle="toggle(scope.row as Account)"
            @reset="reset(scope.row as Account)"
          />
        </template>
      </el-table-column>
    </el-table>

    <el-pagination v-model:current-page="query.page" v-model:page-size="query.pageSize" :total="total" :page-sizes="[10, 20, 50]" layout="total, sizes, prev, pager, next" style="margin-top: 16px" @current-change="load" @size-change="query.page = 1; load()" />
  </div>

  <el-dialog v-model="dialog" :title="editing ? '编辑账号' : '创建账号'" width="650">
    <el-form ref="formRef" :model="form" :rules="formRules" label-position="top">
      <div class="grid grid-2">
        <el-form-item label="用户名" prop="username"><el-input v-model="form.username" /></el-form-item>
        <el-form-item label="显示名称" prop="displayName"><el-input v-model="form.displayName" /></el-form-item>
        <el-form-item label="邮箱" prop="email"><el-input v-model="form.email" /></el-form-item>
        <el-form-item label="手机号（E.164）" prop="phone"><el-input v-model="form.phone" /></el-form-item>
        <el-form-item v-if="!editing" label="初始密码" prop="password"><el-input v-model="form.password" type="password" show-password /></el-form-item>
        <el-form-item v-if="!editing" label="初始角色">
          <el-select v-model="form.roles" multiple style="width: 100%">
            <el-option v-for="role in roles" :key="role.code" :label="role.displayName" :value="role.code" />
          </el-select>
        </el-form-item>
      </div>
    </el-form>
    <template #footer>
      <div class="button-group">
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </div>
    </template>
  </el-dialog>

  <el-drawer v-model="detail" size="720px" title="账号详情" @closed="detailController?.abort()">
    <el-alert v-if="detailError" type="error" :title="detailError" show-icon :closable="false" />
    <el-button v-if="detailError" style="margin-top: 12px" @click="editing && showDetail(editing)">重试加载</el-button>
    <div v-loading="detailLoading" class="detail-content">
    <template v-if="editing">
      <el-tabs>
        <el-tab-pane v-if="canReadAccountRoles" label="角色权限">
          <el-checkbox-group v-model="assigned">
            <div v-for="role in roles" :key="role.code" style="margin: 12px 0">
              <el-checkbox :value="role.code"><b>{{ role.displayName }}</b> <span class="technical muted">{{ role.code }}</span></el-checkbox>
            </div>
          </el-checkbox-group>
          <el-button v-if="canReplaceAccountRoles" type="primary" :loading="roleSaving" @click="saveRoles">保存角色</el-button>
        </el-tab-pane>
        <el-tab-pane v-if="canReadSessions" label="活跃会话">
          <el-button v-if="canRevokeSessions" type="danger" plain :loading="revoking" style="margin-bottom: 12px" @click="revoke">撤销全部会话</el-button>
          <el-table :data="sessions">
            <el-table-column prop="id" label="Session UUID" min-width="280" />
            <el-table-column prop="createdAt" label="创建时间" />
            <el-table-column prop="expiresAt" label="到期时间" />
          </el-table>
        </el-tab-pane>
        <el-tab-pane label="技术信息">
          <el-descriptions :column="1" border>
            <el-descriptions-item label="Account UUID"><span class="technical">{{ editing.id }}</span></el-descriptions-item>
            <el-descriptions-item label="Identity">{{ editing.username || '-' }} / {{ editing.email || '-' }} / {{ editing.phone || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Created">{{ editing.createdAt }}</el-descriptions-item>
            <el-descriptions-item label="Updated">{{ editing.updatedAt }}</el-descriptions-item>
          </el-descriptions>
        </el-tab-pane>
      </el-tabs>
    </template>
    </div>
  </el-drawer>
</template>
