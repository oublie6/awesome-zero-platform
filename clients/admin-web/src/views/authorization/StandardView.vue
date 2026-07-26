<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageHeader from '@/components/PageHeader.vue'
import { adminApi } from '@/api/admin'
import type { Resource } from '@/types/admin'

const items = ref<Resource[]>([])
const dialog = ref(false)
const editing = ref<Resource | null>(null)
const form = reactive({ code: '', displayName: '', module: '', pattern: '', actionsText: 'GET', description: '' })

async function load() {
  items.value = await adminApi.resources()
}

function open(row?: Resource) {
  editing.value = row || null
  Object.assign(form, row ? { ...row, actionsText: row.actions.join(', ') } : { code: '', displayName: '', module: '', pattern: '', actionsText: 'GET', description: '' })
  dialog.value = true
}

async function save() {
  const body = {
    code: form.code,
    displayName: form.displayName,
    module: form.module,
    pattern: form.pattern,
    actions: form.actionsText.split(',').map(value => value.trim()).filter(Boolean),
    description: form.description,
  }
  if (editing.value) await adminApi.updateResource(editing.value.code, body)
  else await adminApi.createResource(body)
  dialog.value = false
  await load()
  ElMessage.success('资源目录已保存')
}

async function remove(row: Resource) {
  await ElMessageBox.confirm(`删除资源 ${row.displayName}？现有 Casbin 策略不会被隐式删除，请先检查专家模式。`, '确认')
  await adminApi.deleteResource(row.code)
  await load()
}

onMounted(load)
</script>

<template>
  <PageHeader title="权限配置" description="面向日常管理员的标准资源目录。资源定义页面模块、匹配路径和允许勾选的操作，角色页再完成授权。">
    <el-button type="primary" @click="open()">登记资源</el-button>
  </PageHeader>

  <div class="panel">
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
            <el-button link @click="open(scope.row)">编辑</el-button>
            <el-button v-if="!scope.row.system" link type="danger" @click="remove(scope.row)">删除</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>
  </div>

  <el-dialog v-model="dialog" :title="editing ? '编辑资源' : '登记资源'" width="650">
    <el-form label-position="top">
      <div class="grid grid-2">
        <el-form-item label="资源编码"><el-input v-model="form.code" :disabled="Boolean(editing)" placeholder="admin.account" /></el-form-item>
        <el-form-item label="所属模块"><el-input v-model="form.module" placeholder="identity" /></el-form-item>
        <el-form-item label="显示名称"><el-input v-model="form.displayName" /></el-form-item>
        <el-form-item label="Casbin 资源模式"><el-input v-model="form.pattern" placeholder="/admin/accounts/*" /></el-form-item>
      </div>
      <el-form-item label="可用操作（逗号分隔，可使用正则）"><el-input v-model="form.actionsText" placeholder="GET, POST, PATCH" /></el-form-item>
      <el-form-item label="说明"><el-input v-model="form.description" type="textarea" /></el-form-item>
    </el-form>
    <template #footer>
      <div class="button-group">
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </div>
    </template>
  </el-dialog>
</template>
