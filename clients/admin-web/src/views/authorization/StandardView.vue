<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageHeader from '@/components/PageHeader.vue'
import { adminApi } from '@/api/admin'
import { useAuthStore } from '@/stores/auth'
import type { Resource } from '@/types/admin'

const items = ref<Resource[]>([])
const loading = ref(false)
const saving = ref(false)
const loadError = ref('')
const dialog = ref(false)
const editing = ref<Resource | null>(null)
const form = reactive({ code: '', displayName: '', module: '', pattern: '', actionsText: 'GET', description: '' })
const formRef = ref()
const auth = useAuthStore()
const formRules = {
  code: [{ required: true, message: '请输入资源编码', trigger: 'blur' }],
  displayName: [{ required: true, message: '请输入显示名称', trigger: 'blur' }],
  module: [{ required: true, message: '请输入所属模块', trigger: 'blur' }],
  pattern: [{ required: true, message: '请输入资源匹配模式', trigger: 'blur' }],
}
let loadController: AbortController | undefined

async function load() {
  loadController?.abort()
  const controller = new AbortController()
  loadController = controller
  loading.value = true
  loadError.value = ''
  try {
    const next = await adminApi.resources(controller.signal)
    if (!controller.signal.aborted) items.value = next
  } catch (error) {
    if (!controller.signal.aborted) loadError.value = error instanceof Error ? error.message : '资源目录加载失败'
  } finally { if (!controller.signal.aborted) loading.value = false }
}

function open(row?: Resource) {
  editing.value = row || null
  Object.assign(form, row ? { ...row, actionsText: row.actions.join(', ') } : { code: '', displayName: '', module: '', pattern: '', actionsText: 'GET', description: '' })
  dialog.value = true
}

async function save() {
  if (saving.value) return
  const valid = await formRef.value?.validate().then(() => true).catch(() => false)
  if (!valid) return
  const body = {
    code: form.code,
    displayName: form.displayName,
    module: form.module,
    pattern: form.pattern,
    actions: form.actionsText.split(',').map(value => value.trim()).filter(Boolean),
    description: form.description,
  }
  saving.value = true
  try {
    if (editing.value) await adminApi.updateResource(editing.value.code, body)
    else await adminApi.createResource(body)
    dialog.value = false
    await load()
    ElMessage.success('资源目录已保存')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '资源保存失败')
  } finally { saving.value = false }
}

async function remove(row: Resource) {
  try {
    await ElMessageBox.confirm(`删除资源 ${row.displayName}？现有 Casbin 策略不会被隐式删除，请先检查专家模式。`, '确认删除资源', { type: 'warning', confirmButtonText: '删除资源', cancelButtonText: '取消' })
    await adminApi.deleteResource(row.code)
    await load()
    ElMessage.success('资源目录已删除')
  } catch (error) {
    if (error instanceof Error) ElMessage.error(error.message)
  }
}

onBeforeUnmount(() => loadController?.abort())
onMounted(load)
</script>

<template>
  <PageHeader title="权限配置" description="面向日常管理员的标准资源目录。资源定义页面模块、匹配路径和允许勾选的操作，角色页再完成授权。">
    <el-button v-if="auth.can('/admin/authorization/resources', 'POST')" type="primary" @click="open()">登记资源</el-button>
  </PageHeader>

  <el-alert v-if="loadError" type="error" :title="loadError" show-icon :closable="false" />
  <el-button v-if="loadError" style="margin: 12px 0" @click="load">重试加载</el-button>
  <div class="panel" v-loading="loading">
    <el-table :data="items">
      <el-table-column label="资源" min-width="240">
        <template #default="scope">
          <b>{{ scope.row.displayName }}</b>
          <div class="technical muted">{{ scope.row.code }}</div>
        </template>
      </el-table-column>
      <el-table-column prop="module" label="模块" width="150" />
      <el-table-column label="匹配模式" min-width="260">
        <template #default="scope"><span class="technical">{{ scope.row.pattern }}</span></template>
      </el-table-column>
      <el-table-column label="操作" min-width="250">
        <template #default="scope">
          <el-tag v-for="action in scope.row.actions" :key="action" size="small" style="margin: 3px">
            <span class="technical">{{ action }}</span>
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="说明" min-width="260" />
      <el-table-column label="操作" min-width="130">
        <template #default="scope">
          <div class="table-actions">
            <el-button v-if="auth.can('/admin/authorization/resources/_', 'PATCH')" link @click="open(scope.row as Resource)">编辑</el-button>
            <el-button v-if="!scope.row.system && auth.can('/admin/authorization/resources/_', 'DELETE')" link type="danger" @click="remove(scope.row as Resource)">删除</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>
  </div>

  <el-dialog v-model="dialog" :title="editing ? '编辑资源' : '登记资源'" width="650">
    <el-form ref="formRef" :model="form" :rules="formRules" label-position="top">
      <div class="grid grid-2">
        <el-form-item label="资源编码" prop="code"><el-input v-model="form.code" :disabled="Boolean(editing)" placeholder="admin.account" /></el-form-item>
        <el-form-item label="所属模块" prop="module"><el-input v-model="form.module" placeholder="identity" /></el-form-item>
        <el-form-item label="显示名称" prop="displayName"><el-input v-model="form.displayName" /></el-form-item>
        <el-form-item label="Casbin 资源模式" prop="pattern"><el-input v-model="form.pattern" placeholder="/admin/accounts/*" /></el-form-item>
      </div>
      <el-form-item label="可用操作（逗号分隔，可使用正则）"><el-input v-model="form.actionsText" placeholder="GET, POST, PATCH" /></el-form-item>
      <el-form-item label="说明"><el-input v-model="form.description" type="textarea" /></el-form-item>
    </el-form>
    <template #footer>
      <div class="button-group">
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </div>
    </template>
  </el-dialog>
</template>
