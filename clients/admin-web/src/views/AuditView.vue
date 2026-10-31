<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import PageHeader from '@/components/PageHeader.vue'
import { adminApi } from '@/api/admin'
import type { AuditEvent } from '@/types/admin'

const items = ref<AuditEvent[]>([])
const total = ref(0)
const loading = ref(false)
const loadError = ref('')
const query = reactive({ search: '', action: '', outcome: '', page: 1, pageSize: 20 })
let controller: AbortController | undefined

async function load() {
  controller?.abort()
  const next = new AbortController()
  controller = next
  loading.value = true
  loadError.value = ''
  try {
    const result = await adminApi.audit(query, next.signal)
    if (!next.signal.aborted) {
      items.value = result.items
      total.value = result.total
    }
  } catch (error) {
    if (!next.signal.aborted) loadError.value = error instanceof Error ? error.message : '审计事件加载失败'
  } finally { if (!next.signal.aborted) loading.value = false }
}

function search() { query.page = 1; void load() }
onMounted(load)
onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <PageHeader title="操作审计" description="普通描述与请求级技术字段放在同一条不可变审计事件中。" />
  <el-alert v-if="loadError" type="error" :title="loadError" show-icon :closable="false" />
  <el-button v-if="loadError" style="margin: 12px 0" @click="load">重试加载</el-button>
  <div class="panel" v-loading="loading">
    <div class="toolbar">
      <el-input v-model="query.search" clearable placeholder="账号 / 资源 / Request ID" style="width: 280px" @keyup.enter="search" @clear="search" />
      <el-input v-model="query.action" clearable placeholder="action_name" style="width: 210px" @keyup.enter="search" @clear="search" />
      <el-select v-model="query.outcome" clearable placeholder="结果" style="width: 130px" @change="search" @clear="search">
        <el-option label="成功" value="success" />
        <el-option label="失败" value="failure" />
      </el-select>
      <el-button type="primary" :loading="loading" @click="search">查询</el-button>
    </div>
    <el-table :data="items">
      <el-table-column prop="createdAt" label="时间" width="190" />
      <el-table-column prop="action" label="操作" width="220"><template #default="scope"><span class="technical">{{ scope.row.action }}</span></template></el-table-column>
      <el-table-column prop="actorId" label="操作者" min-width="250" />
      <el-table-column label="资源" min-width="230"><template #default="scope">{{ scope.row.resourceType }} / <span class="technical">{{ scope.row.resourceId }}</span></template></el-table-column>
      <el-table-column prop="outcome" label="结果" width="90"><template #default="scope"><el-tag :type="scope.row.outcome === 'success' ? 'success' : 'danger'">{{ scope.row.outcome }}</el-tag></template></el-table-column>
      <el-table-column prop="requestId" label="Request ID" min-width="220" />
      <template #empty><el-empty description="没有匹配的审计事件" /></template>
    </el-table>
    <el-pagination
      v-model:current-page="query.page"
      v-model:page-size="query.pageSize"
      :total="total"
      :page-sizes="[10, 20, 50]"
      layout="total, sizes, prev, pager, next"
      style="margin-top: 16px"
      @current-change="load"
      @size-change="query.page = 1; load()"
    />
  </div>
</template>
