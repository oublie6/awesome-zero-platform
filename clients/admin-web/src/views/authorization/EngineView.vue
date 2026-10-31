<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageHeader from '@/components/PageHeader.vue'
import CodePanel from '@/components/CodePanel.vue'
import { adminApi } from '@/api/admin'
import { useAuthStore } from '@/stores/auth'
import type { EngineInfo, Explanation, RawRule } from '@/types/admin'

const auth = useAuthStore()
const engine = ref<EngineInfo | null>(null)
const model = ref('')
const rules = ref<RawRule[]>([])
const text = ref('')
const loading = ref(true)
const saving = ref(false)
const validating = ref(false)
const explaining = ref(false)
const loadError = ref('')
const explanation = ref<Explanation | null>(null)
const probe = reactive({ subject: '', resource: '/admin/accounts', action: 'GET' })
let loadController: AbortController | undefined

const canValidate = computed(() => auth.can('/admin/authorization/engine/policies/validate', 'POST'))
const canReplace = computed(() => auth.can('/admin/authorization/engine/policies', 'PUT'))
const canExplain = computed(() => auth.can('/admin/authorization/engine/explain', 'POST'))
const canReadModel = computed(() => auth.can('/admin/authorization/engine/model', 'GET'))
const canReadPolicies = computed(() => auth.can('/admin/authorization/engine/policies', 'GET'))
const policyText = computed(() => rules.value.map(rule => [rule.ptype, ...rule.values].join(', ')).join('\n'))
const capabilities = computed(() => {
  if (!engine.value) return []
  const info = engine.value
  return [
    ['raw-policy', info.supportsRawPolicy], ['model-inspection', info.supportsModelInspection],
    ['model-editing', info.supportsModelEditing], ['policy-explanation', info.supportsPolicyExplanation],
    ['batch-import', info.supportsBatchImport], ['role-hierarchy', info.supportsRoleHierarchy],
  ].filter(([, enabled]) => enabled).map(([name]) => String(name))
})

function parsePolicy(input: string): RawRule[] {
  return input.split('\n').map(line => line.trim()).filter(line => line && !line.startsWith('#')).map(line => {
    const [ptype, ...values] = line.split(',').map(value => value.trim())
    return { ptype, values }
  })
}

async function load() {
  loadController?.abort()
  const controller = new AbortController()
  loadController = controller
  loading.value = true
  loadError.value = ''
  try {
    const [nextEngine, nextModel, nextRules] = await Promise.all([
      adminApi.engine(controller.signal),
      canReadModel.value ? adminApi.model(controller.signal) : Promise.resolve({ model: '' }),
      canReadPolicies.value ? adminApi.policies(controller.signal) : Promise.resolve([]),
    ])
    if (controller.signal.aborted) return
    engine.value = nextEngine
    model.value = nextModel.model
    rules.value = nextRules
    text.value = nextRules.map(rule => [rule.ptype, ...rule.values].join(', ')).join('\n')
  } catch (error) {
    if (!controller.signal.aborted) loadError.value = error instanceof Error ? error.message : '权限引擎信息加载失败'
  } finally { if (!controller.signal.aborted) loading.value = false }
}

async function validate() {
  if (validating.value) return
  validating.value = true
  try {
    await adminApi.validatePolicies(parsePolicy(text.value))
    ElMessage.success('策略格式和平台保护规则校验通过')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '校验失败')
  } finally { validating.value = false }
}

async function save() {
  if (!canReplace.value || saving.value) return
  try {
    await ElMessageBox.confirm('这会整体替换当前权限引擎策略。系统将拒绝删除最后一个超级管理员或超级管理员通配规则。', '高风险策略替换', {
      type: 'warning', confirmButtonText: '替换策略', cancelButtonText: '取消',
    })
  } catch {
    return
  }
  saving.value = true
  try {
    await adminApi.replacePolicies(parsePolicy(text.value))
    ElMessage.success('策略已原子替换')
    await load()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '策略替换失败')
  } finally { saving.value = false }
}

async function explain() {
  if (explaining.value) return
  explaining.value = true
  try {
    explanation.value = await adminApi.explain(probe)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'Enforce 解释失败')
  } finally { explaining.value = false }
}

onBeforeUnmount(() => loadController?.abort())
onMounted(load)
</script>

<template>
  <PageHeader title="权限引擎" expert description="面向后端开发与安全管理员的插件原生工作台：查看 Casbin Model、编辑 p/g 策略并执行 Enforce 解释。" />
  <div class="expert-banner">HIGH IMPACT AREA — 所有写操作仍通过 Authz 插件接口执行，不直接修改 authorization_casbin_rules 表。</div>
  <el-alert v-if="loadError" type="error" :title="loadError" show-icon :closable="false" />
  <el-button v-if="loadError" style="margin: 12px 0" @click="load">重试加载</el-button>
  <template v-if="engine">
    <section class="grid grid-4 engine-metrics" v-loading="loading">
      <div class="panel"><div class="metric-label">ENGINE</div><div class="metric-value">{{ engine.name }}</div></div>
      <div class="panel"><div class="metric-label">ADAPTER</div><div class="metric-value metric-small">{{ engine.version }}</div></div>
      <div class="panel"><div class="metric-label">MODEL</div><div class="metric-value metric-small">{{ engine.modelType }}</div></div>
      <div class="panel"><div class="metric-label">RULES</div><div class="metric-value">{{ rules.length }}</div></div>
    </section>
    <el-tabs v-loading="loading">
      <el-tab-pane v-if="canReadModel" label="模型">
        <CodePanel title="CASBIN MODEL (READ ONLY)" :value="model" />
        <div class="panel capability-panel"><b>Capabilities</b><div><el-tag v-for="capability in capabilities" :key="capability"><span class="technical">{{ capability }}</span></el-tag></div></div>
      </el-tab-pane>
      <el-tab-pane v-if="canReadPolicies" label="原始策略">
        <div class="toolbar">
          <el-button @click="text = policyText">恢复已加载内容</el-button>
          <el-button v-if="canValidate" type="success" plain :loading="validating" @click="validate">校验</el-button>
          <el-button v-if="canReplace" type="danger" :loading="saving" @click="save">整体替换</el-button>
          <span class="muted">支持 p、g；逗号分隔，每行一条。</span>
        </div>
        <el-input v-model="text" type="textarea" :rows="24" class="policy-editor" spellcheck="false" />
      </el-tab-pane>
      <el-tab-pane label="Enforce 调试器">
        <div class="grid grid-2">
          <section class="panel">
            <el-form label-position="top">
              <el-form-item label="Subject"><el-input v-model="probe.subject" placeholder="account UUID" /></el-form-item>
              <el-form-item label="Object"><el-input v-model="probe.resource" /></el-form-item>
              <el-form-item label="Action"><el-input v-model="probe.action" /></el-form-item>
              <el-button v-if="canExplain" type="primary" :loading="explaining" @click="explain">执行并解释</el-button>
            </el-form>
          </section>
          <section class="panel">
            <template v-if="explanation">
              <el-result :icon="explanation.allowed ? 'success' : 'error'" :title="explanation.allowed ? 'ALLOW' : 'DENY'" :sub-title="`${explanation.subject} → ${explanation.resource} [${explanation.action}]`" />
              <div><b>Roles</b><p class="technical">{{ explanation.roles.join(', ') || '(none)' }}</p><b>Matched Rules</b><CodePanel :value="explanation.matchedRules.map(rule => [rule.ptype, ...rule.values].join(', ')).join('\n') || '# no matching policy'" /></div>
            </template>
            <el-empty v-else description="输入请求三元组后执行" />
          </section>
        </div>
      </el-tab-pane>
    </el-tabs>
  </template>
</template>

<style scoped>
.engine-metrics { margin-bottom: 16px; }
.metric-small { font-size: 17px; overflow-wrap: anywhere; }
.capability-panel { margin-top: 14px; }
.capability-panel > div { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 10px; }
.policy-editor { min-width: 0; }
.policy-editor :deep(textarea) { font: 13px/1.65 var(--az-mono); background: #080d13; color: #d7e1ec; overflow-wrap: anywhere; }
</style>
