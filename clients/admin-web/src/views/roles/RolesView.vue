<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageHeader from '@/components/PageHeader.vue'
import CodePanel from '@/components/CodePanel.vue'
import { adminApi } from '@/api/admin'
import { useAuthStore } from '@/stores/auth'
import type { Permission, Resource, Role } from '@/types/admin'

const roles = ref<Role[]>([])
const loadError = ref('')
const loading = ref(false)
const permissionsLoading = ref(false)
const permissionsSaving = ref(false)
const saving = ref(false)
const resources = ref<Resource[]>([])
const selected = ref<Role | null>(null)
const permissions = ref<Permission[]>([])
const dialog = ref(false)
const editing = ref<Role | null>(null)
const form = reactive({ code: '', displayName: '', description: '' })
const auth = useAuthStore()
const canReadResources = auth.can('/admin/authorization/resources', 'GET')
const canReadRolePermissions = auth.can('/admin/roles/_/permissions', 'GET')
const formRef = ref()
const formRules = {
  code: [{ required: true, message: '请输入角色编码', trigger: 'blur' }, { pattern: /^[a-z][a-z0-9_]{1,63}$/, message: '使用小写字母开头的字母、数字或下划线', trigger: 'blur' }],
  displayName: [{ required: true, message: '请输入显示名称', trigger: 'blur' }],
}
let listController: AbortController | undefined
let permissionController: AbortController | undefined

async function load() {
  listController?.abort()
  const controller = new AbortController()
  listController = controller
  loading.value = true
  loadError.value = ''
  try {
    const [nextRoles, nextResources] = await Promise.all([
      adminApi.roles(controller.signal),
      canReadResources ? adminApi.resources(controller.signal) : Promise.resolve([]),
    ])
    if (controller.signal.aborted) return
    roles.value = nextRoles
    resources.value = nextResources
    const selectedRole = nextRoles.find(role => role.code === selected.value?.code) || nextRoles[0]
    if (selectedRole) await select(selectedRole)
    else { selected.value = null; permissions.value = [] }
  } catch (error) {
    if (controller.signal.aborted) return
    loadError.value = error instanceof Error ? error.message : '角色信息加载失败'
  } finally { if (!controller.signal.aborted) loading.value = false }
}

async function select(role: Role) {
  permissionController?.abort()
  const controller = new AbortController()
  permissionController = controller
  selected.value = role
  permissions.value = []
  if (!canReadRolePermissions) return
  permissionsLoading.value = true
  try {
    const nextPermissions = await adminApi.rolePermissions(role.code, controller.signal)
    if (!controller.signal.aborted && selected.value?.code === role.code) permissions.value = nextPermissions
  } catch (error) {
    if (!controller.signal.aborted) ElMessage.error(error instanceof Error ? error.message : '角色权限加载失败')
  } finally {
    if (!controller.signal.aborted) permissionsLoading.value = false
  }
}

function checked(resource: string, action: string) {
  return permissions.value.some(permission => permission.resource === resource && permission.action === action)
}

function toggle(resource: string, action: string, value: unknown) {
  const next = permissions.value.filter(permission => !(permission.resource === resource && permission.action === action))
  if (Boolean(value)) next.push({ resource, action })
  permissions.value = next
}

async function savePermissions() {
  if (!selected.value || permissionsSaving.value) return
  permissionsSaving.value = true
  try {
    await adminApi.replaceRolePermissions(selected.value.code, permissions.value)
    ElMessage.success('权限已保存')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '权限保存失败')
  } finally { permissionsSaving.value = false }
}

function open(role?: Role) {
  editing.value = role || null
  Object.assign(form, role || { code: '', displayName: '', description: '' })
  dialog.value = true
}

async function save() {
  if (saving.value) return
  const valid = await formRef.value?.validate().then(() => true).catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    if (editing.value) await adminApi.updateRole(editing.value.code, form)
    else await adminApi.createRole(form)
    dialog.value = false
    await load()
    ElMessage.success('角色已保存')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '角色保存失败')
  } finally { saving.value = false }
}

async function remove(role: Role) {
  try {
    await ElMessageBox.confirm(`删除角色 ${role.displayName}？该角色必须没有成员。`, '危险操作', { type: 'warning', confirmButtonText: '删除角色', cancelButtonText: '取消' })
    await adminApi.deleteRole(role.code)
    if (selected.value?.code === role.code) selected.value = null
    await load()
    ElMessage.success('角色已删除')
  } catch (error) {
    if (error instanceof Error) ElMessage.error(error.message)
  }
}

const rawPreview = () => permissions.value.map(permission => `p, ${selected.value?.code}, ${permission.resource}, ${permission.action}`).join('\n')

onBeforeUnmount(() => { listController?.abort(); permissionController?.abort() })
onMounted(load)
</script>

<template>
  <PageHeader title="角色管理" description="角色信息、资源操作矩阵和最终生成的 Casbin p 策略放在同一个工作区。">
    <el-button v-if="auth.can('/admin/roles', 'POST')" type="primary" @click="open()">创建角色</el-button>
  </PageHeader>

  <el-alert v-if="loadError" type="error" :title="loadError" :closable="false" show-icon style="margin-bottom:12px" />
  <el-button v-if="loadError" @click="load" style="margin-bottom:12px">重试加载</el-button>
  <div class="role-grid" v-loading="loading">
    <section class="panel role-list">
      <div v-for="role in roles" :key="role.code" :class="['role-item', { active: selected?.code === role.code }]" @click="select(role)">
        <div class="role-copy">
          <b>{{ role.displayName }}</b>
          <small>{{ role.code }}</small>
        </div>
        <el-tag v-if="role.system" size="small">SYSTEM</el-tag>
      </div>
    </section>

    <section v-if="selected" class="panel role-workspace">
      <div class="role-title">
        <div class="role-copy">
          <h2>{{ selected.displayName }}</h2>
          <span class="technical muted">{{ selected.code }}</span>
        </div>
        <div class="button-group role-actions">
          <el-button v-if="auth.can('/admin/roles/_', 'PATCH')" @click="open(selected)">编辑</el-button>
          <el-button v-if="!selected.system && auth.can('/admin/roles/_', 'DELETE')" type="danger" plain @click="remove(selected)">删除</el-button>
        </div>
      </div>

      <el-tabs v-loading="permissionsLoading">
        <el-tab-pane v-if="canReadRolePermissions && canReadResources" label="权限矩阵">
          <el-table :data="resources">
            <el-table-column label="资源" min-width="230">
              <template #default="scope">
                <b>{{ scope.row.displayName }}</b>
                <div class="technical muted">{{ scope.row.pattern }}</div>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="420">
              <template #default="scope">
                <div class="permission-actions">
                <el-checkbox v-for="action in scope.row.actions" :key="action" :disabled="!auth.can('/admin/roles/_/permissions', 'PUT')" :model-value="checked(scope.row.pattern, action)" @change="toggle(scope.row.pattern, action, $event)">
                    <span class="technical">{{ action }}</span>
                  </el-checkbox>
                </div>
              </template>
            </el-table-column>
          </el-table>
          <el-button v-if="auth.can('/admin/roles/_/permissions', 'PUT')" type="primary" :loading="permissionsSaving" style="margin-top: 16px" @click="savePermissions">保存权限</el-button>
        </el-tab-pane>
        <el-tab-pane v-if="canReadRolePermissions" label="原始策略"><CodePanel title="GENERATED CASBIN POLICY" :value="rawPreview() || '# no policy'" /></el-tab-pane>
        <el-tab-pane label="技术信息">
          <el-descriptions :column="1" border>
            <el-descriptions-item label="Role Code"><span class="technical">{{ selected.code }}</span></el-descriptions-item>
            <el-descriptions-item label="System Role">{{ selected.system }}</el-descriptions-item>
            <el-descriptions-item label="Created">{{ selected.createdAt }}</el-descriptions-item>
          </el-descriptions>
        </el-tab-pane>
      </el-tabs>
    </section>
  </div>

  <el-dialog v-model="dialog" :title="editing ? '编辑角色' : '创建角色'">
    <el-form ref="formRef" :model="form" :rules="formRules" label-position="top">
      <el-form-item label="角色编码" prop="code"><el-input v-model="form.code" :disabled="Boolean(editing)" placeholder="platform_operator" /></el-form-item>
      <el-form-item label="显示名称" prop="displayName"><el-input v-model="form.displayName" /></el-form-item>
      <el-form-item label="描述"><el-input v-model="form.description" type="textarea" /></el-form-item>
    </el-form>
    <template #footer>
      <div class="button-group">
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.role-grid {
  display: grid;
  grid-template-columns: minmax(240px, 300px) minmax(0, 1fr);
  gap: 16px;
}

.role-list {
  padding: 8px;
}

.role-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 14px;
  border-radius: 8px;
  cursor: pointer;
}

.role-item:hover,
.role-item.active {
  background: #1b2b39;
}

.role-copy {
  min-width: 0;
}

.role-item b,
.role-item small,
.role-title h2,
.role-title .technical {
  overflow-wrap: anywhere;
}

.role-item small {
  display: block;
  margin-top: 5px;
  color: var(--az-muted);
  font: 11px var(--az-mono);
}

.role-workspace {
  min-width: 0;
}

.role-title {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px 18px;
  margin-bottom: 12px;
}

.role-title h2 {
  margin: 0;
}

.role-actions {
  justify-content: flex-end;
}

.permission-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 16px;
}

.permission-actions :deep(.el-checkbox) {
  margin-right: 0;
}

@media (max-width: 980px) {
  .role-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .role-list {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
    gap: 6px;
  }
}
</style>
