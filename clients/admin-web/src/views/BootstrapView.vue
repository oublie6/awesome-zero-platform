<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { adminApi } from '@/api/admin'

const router = useRouter()
const available = ref(false)
const checked = ref(false)
const checking = ref(false)
const creating = ref(false)
const statusError = ref('')
const formRef = ref()
const form = reactive({ token: '', username: 'admin', displayName: '平台管理员', password: '' })
const formRules = {
  token: [{ required: true, message: '请输入部署时提供的 Bootstrap Token', trigger: 'blur' }],
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  displayName: [{ required: true, message: '请输入显示名称', trigger: 'blur' }],
  password: [{ validator: (_rule: unknown, value: string, done: (error?: Error) => void) => {
    if (new TextEncoder().encode(value || '').length < 8) done(new Error('密码至少需要 8 个字节'))
    else done()
  }, trigger: 'blur' }],
}
let controller: AbortController | undefined

async function loadStatus() {
  controller?.abort()
  const next = new AbortController()
  controller = next
  checking.value = true
  statusError.value = ''
  try {
    const result = await adminApi.bootstrapStatus(next.signal)
    if (!next.signal.aborted) { available.value = result.available; checked.value = true }
  } catch (error) {
    if (!next.signal.aborted) statusError.value = error instanceof Error ? error.message : '初始化状态检查失败'
  } finally { if (!next.signal.aborted) checking.value = false }
}

async function submit() {
  if (creating.value) return
  const valid = await formRef.value?.validate().then(() => true).catch(() => false)
  if (!valid) return
  creating.value = true
  try {
    await adminApi.bootstrap(form.token, { username: form.username, displayName: form.displayName, password: form.password })
    ElMessage.success('超级管理员已创建，请登录')
    await router.replace('/login')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '初始化失败')
  } finally { creating.value = false }
}

onMounted(loadStatus)
onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <div class="bootstrap">
    <section class="panel card">
      <div class="technical muted">BOOTSTRAP / ONE-TIME OPERATION</div>
      <h1>初始化超级管理员</h1>
      <el-skeleton v-if="checking && !checked && !statusError" :rows="4" animated />
      <template v-else-if="statusError">
        <el-alert type="error" :title="statusError" :closable="false" show-icon />
        <el-button :loading="checking" style="margin-top: 12px" @click="loadStatus">重试检查</el-button>
      </template>
      <el-alert v-else-if="checked && !available" type="warning" :closable="false" title="初始化入口不可用：未配置 Bootstrap Token，或超级管理员已经存在。" />
      <template v-else-if="available">
        <p>Bootstrap Token 只能来自部署环境，不会在页面中显示。完成后该令牌应立即从生产环境移除。</p>
        <el-form ref="formRef" :model="form" :rules="formRules" label-position="top" @submit.prevent="submit">
          <el-form-item label="Bootstrap Token" prop="token"><el-input v-model="form.token" type="password" show-password autocomplete="off" /></el-form-item>
          <el-form-item label="用户名" prop="username"><el-input v-model="form.username" autocomplete="username" /></el-form-item>
          <el-form-item label="显示名称" prop="displayName"><el-input v-model="form.displayName" /></el-form-item>
          <el-form-item label="初始密码" prop="password"><el-input v-model="form.password" type="password" show-password autocomplete="new-password" /></el-form-item>
          <el-button native-type="submit" type="danger" :loading="creating">创建唯一初始超级管理员</el-button>
        </el-form>
      </template>
      <el-button link @click="router.push('/login')">返回登录</el-button>
    </section>
  </div>
</template>

<style scoped>
.bootstrap { min-height: 100vh; display: grid; place-items: center; padding: 24px; }
.card { width: min(620px, 100%); }
.card h1 { font-size: clamp(24px, 5vw, 30px); }
.card p { color: var(--az-muted); line-height: 1.7; }
.el-form { margin: 26px 0; }
</style>
