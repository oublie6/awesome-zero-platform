<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageHeader from '@/components/PageHeader.vue'
import CodePanel from '@/components/CodePanel.vue'
import { adminApi } from '@/api/admin'
import type { Permission, Resource, Role } from '@/types/admin'

const roles = ref<Role[]>([])
const resources = ref<Resource[]>([])
const selected = ref<Role | null>(null)
const permissions = ref<Permission[]>([])
const dialog = ref(false)
const editing = ref<Role | null>(null)
const form = reactive({ code: '', displayName: '', description: '' })

async function load() {
  ;[roles.value, resources.value] = await Promise.all([adminApi.roles(), adminApi.resources()])
  if (!selected.value && roles.value[0]) await select(roles.value[0])
}

async function select(role: Role) {
  selected.value = role
  permissions.value = await adminApi.rolePermissions(role.code)
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
  if (!selected.value) return
  await adminApi.replaceRolePermissions(selected.value.code, permissions.value)
  ElMessage.success('权限已保存')
}

function open(role?: Role) {
  editing.value = role || null
  Object.assign(form, role || { code: '', displayName: '', description: '' })
  dialog.value = true
}

async function save() {
  if (editing.value) await adminApi.updateRole(editing.value.code, form)
  else await adminApi.createRole(form)
  dialog.value = false
  await load()
  ElMessage.success('角色已保存')
}

async function remove(role: Role) {
  await ElMessageBox.confirm(`删除角色 ${role.displayName}？该角色必须没有成员。`, '危险操作')
  await adminApi.deleteRole(role.code)
  selected.value = null
  await load()
}

const rawPreview = () => permissions.value.map(permission => `p, ${selected.value?.code}, ${permission.resource}, ${permission.action}`).join('\n')

onMounted(load)
</script>

<template>
  <PageHeader title="角色管理" description="角色信息、资源操作矩阵和最终生成的 Casbin p 策略放在同一个工作区。">
    <el-button type="primary" @click="open()">创建角色</el-button>
  </PageHeader>

  <div class="role-grid">
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
          <el-button @click="open(selected)">编辑</el-button>
          <el-button v-if="!selected.system" type="danger" plain @click="remove(selected)">删除</el-button>
        </div>
      </div>

      <el-tabs>
        <el-tab-pane label="权限矩阵">
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
                  <el-checkbox v-for="action in scope.row.actions" :key="action" :model-value="checked(scope.row.pattern, action)" @change="toggle(scope.row.pattern, action, $event)">
                    <span class="technical">{{ action }}</span>
                  </el-checkbox>
                </div>
              </template>
            </el-table-column>
          </el-table>
          <el-button type="primary" style="margin-top: 16px" @click="savePermissions">保存权限</el-button>
        </el-tab-pane>
        <el-tab-pane label="原始策略"><CodePanel title="GENERATED CASBIN POLICY" :value="rawPreview() || '# no policy'" /></el-tab-pane>
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
    <el-form label-position="top">
      <el-form-item label="角色编码"><el-input v-model="form.code" :disabled="Boolean(editing)" placeholder="platform_operator" /></el-form-item>
      <el-form-item label="显示名称"><el-input v-model="form.displayName" /></el-form-item>
      <el-form-item label="描述"><el-input v-model="form.description" type="textarea" /></el-form-item>
    </el-form>
    <template #footer>
      <div class="button-group">
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
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
