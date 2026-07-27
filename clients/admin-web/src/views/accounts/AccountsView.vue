<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import AccountRowActions from '@/components/AccountRowActions.vue'
import PageHeader from '@/components/PageHeader.vue'
import { adminApi } from '@/api/admin'
import type { Account, Role, SessionView } from '@/types/admin'

const items = ref<Account[]>([])
const roles = ref<Role[]>([])
const total = ref(0)
const loading = ref(false)
const dialog = ref(false)
const detail = ref(false)
const editing = ref<Account | null>(null)
const sessions = ref<SessionView[]>([])
const assigned = ref<string[]>([])
const query = reactive({ search: '', status: '', page: 1, pageSize: 20 })
const form = reactive({ username: '', email: '', phone: '', displayName: '', status: 'active', password: '', roles: [] as string[] })

async function load() {
  loading.value = true
  try {
    const [page, roleList] = await Promise.all([adminApi.accounts(query), adminApi.roles()])
    items.value = page.items
    total.value = page.total
    roles.value = roleList
  } finally {
    loading.value = false
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
  try {
    if (editing.value) {
      await adminApi.updateAccount(editing.value.id, { username: form.username, email: form.email, phone: form.phone, displayName: form.displayName })
    } else {
      await adminApi.createAccount(form)
    }
    dialog.value = false
    ElMessage.success('账号已保存')
    load()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '保存失败')
  }
}

async function toggle(row: Account) {
  try {
    await ElMessageBox.confirm(`${row.status === 'active' ? '禁用' : '启用'}账号 ${row.displayName}？`, '确认')
    await adminApi.setAccount(row.id, row.status !== 'active')
    load()
  } catch (error) {
    if (error instanceof Error) ElMessage.error(error.message)
  }
}

async function showDetail(row: Account) {
  editing.value = row
  detail.value = true
  ;[assigned.value, sessions.value] = await Promise.all([adminApi.accountRoles(row.id), adminApi.sessions(row.id)])
}

async function saveRoles() {
  if (!editing.value) return
  await adminApi.replaceAccountRoles(editing.value.id, assigned.value)
  ElMessage.success('角色已更新')
}

async function revoke() {
  if (!editing.value) return
  const result = await adminApi.revokeSessions(editing.value.id)
  ElMessage.success(`已撤销 ${result.revoked} 个会话`)
  sessions.value = []
}

async function reset(row: Account) {
  const { value } = await ElMessageBox.prompt('输入新的临时密码，重置后该账号全部会话会被撤销。', '重置密码', {
    inputType: 'password',
    inputPattern: /.{8,}/,
    inputErrorMessage: '至少 8 个字符',
  })
  await adminApi.resetPassword(row.id, value)
  ElMessage.success('密码已重置')
}

onMounted(load)
</script>

<template>
  <PageHeader title="账号管理" description="默认视图完成账号生命周期；详情抽屉提供 UUID、角色原始编码和 Redis 会话等技术信息。">
    <el-button type="primary" @click="openCreate">创建账号</el-button>
  </PageHeader>

  <div class="panel">
    <div class="toolbar">
      <el-input v-model="query.search" clearable placeholder="用户名 / 邮箱 / 手机号 / 显示名" style="width: 320px" />
      <el-select v-model="query.status" clearable placeholder="状态" style="width: 130px">
        <el-option label="启用" value="active" />
        <el-option label="禁用" value="disabled" />
      </el-select>
      <el-button @click="query.page = 1; load()">查询</el-button>
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
            @detail="showDetail(scope.row)"
            @edit="openEdit(scope.row)"
            @toggle="toggle(scope.row)"
            @reset="reset(scope.row)"
          />
        </template>
      </el-table-column>
    </el-table>

    <el-pagination v-model:current-page="query.page" :total="total" :page-size="query.pageSize" layout="total,prev,pager,next" style="margin-top: 16px" @current-change="load" />
  </div>

  <el-dialog v-model="dialog" :title="editing ? '编辑账号' : '创建账号'" width="650">
    <el-form label-position="top">
      <div class="grid grid-2">
        <el-form-item label="用户名"><el-input v-model="form.username" /></el-form-item>
        <el-form-item label="显示名称"><el-input v-model="form.displayName" /></el-form-item>
        <el-form-item label="邮箱"><el-input v-model="form.email" /></el-form-item>
        <el-form-item label="手机号（E.164）"><el-input v-model="form.phone" /></el-form-item>
        <el-form-item v-if="!editing" label="初始密码"><el-input v-model="form.password" type="password" show-password /></el-form-item>
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
        <el-button type="primary" @click="save">保存</el-button>
      </div>
    </template>
  </el-dialog>

  <el-drawer v-model="detail" size="720px" title="账号详情">
    <template v-if="editing">
      <el-tabs>
        <el-tab-pane label="角色权限">
          <el-checkbox-group v-model="assigned">
            <div v-for="role in roles" :key="role.code" style="margin: 12px 0">
              <el-checkbox :value="role.code"><b>{{ role.displayName }}</b> <span class="technical muted">{{ role.code }}</span></el-checkbox>
            </div>
          </el-checkbox-group>
          <el-button type="primary" @click="saveRoles">保存角色</el-button>
        </el-tab-pane>
        <el-tab-pane label="活跃会话">
          <el-button type="danger" plain style="margin-bottom: 12px" @click="revoke">撤销全部会话</el-button>
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
  </el-drawer>
</template>
