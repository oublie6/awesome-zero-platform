<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import PageHeader from '@/components/PageHeader.vue'
import CodePanel from '@/components/CodePanel.vue'
import { adminApi } from '@/api/admin'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canReadConfiguration = auth.can('/admin/system/configuration', 'GET')
const overview = ref<Record<string, unknown>>({})
const configuration = ref<Record<string, unknown>>({})
const loading = ref(true)
const error = ref('')
let controller: AbortController | undefined

async function load() {
  controller?.abort()
  const next = new AbortController()
  controller = next
  loading.value = true
  error.value = ''
  try {
    const [nextOverview, nextConfiguration] = await Promise.all([
      adminApi.overview(next.signal),
      canReadConfiguration ? adminApi.configuration(next.signal) : Promise.resolve({}),
    ])
    if (!next.signal.aborted) {
      overview.value = nextOverview
      configuration.value = nextConfiguration
    }
  } catch (reason) {
    if (!next.signal.aborted) error.value = reason instanceof Error ? reason.message : '系统状态加载失败'
  } finally { if (!next.signal.aborted) loading.value = false }
}

onMounted(load)
onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <PageHeader title="系统状态" description="普通视图关注可用性，诊断视图展示配置来源和安全状态，但永不回显 Secret。" />
  <el-alert v-if="error" type="error" :title="error" show-icon :closable="false" />
  <el-button v-if="error" style="margin: 12px 0" @click="load">重试加载</el-button>
  <el-tabs v-loading="loading">
    <el-tab-pane label="运行状态">
      <el-empty v-if="!loading && !Object.keys(overview).length" description="暂无系统状态" />
      <div v-else class="grid grid-2">
        <section v-for="(value, key) in overview" :key="key" class="panel">
          <div class="metric-label technical">{{ key }}</div>
          <div class="metric-value">{{ value }}</div>
        </section>
      </div>
    </el-tab-pane>
    <el-tab-pane v-if="canReadConfiguration" label="配置诊断">
      <div class="expert-banner">CONFIG DIAGNOSTICS / READ ONLY — 敏感配置只显示是否已配置，不显示原始值。</div>
      <CodePanel title="EFFECTIVE CONFIGURATION" :value="JSON.stringify(configuration, null, 2)" />
    </el-tab-pane>
  </el-tabs>
</template>
