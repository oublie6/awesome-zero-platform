<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const loading = ref(false)
const formRef = ref()
const form = reactive({ identifier: '', password: '' })
const formRules = {
  identifier: [{ required: true, message: '请输入用户名、邮箱或手机号', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function submit() {
  if (loading.value) return
  const valid = await formRef.value?.validate().then(() => true).catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    await auth.login(form.identifier, form.password)
    const requested = typeof route.query.redirect === 'string' ? route.query.redirect : ''
    const destination = requested.startsWith('/') && !requested.startsWith('//') ? requested : '/dashboard'
    await router.replace(destination)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <section class="auth-visual" aria-label="Awesome Zero Platform">
      <div class="scanline" />
      <div class="terminal">
        <span>AWESOME ZERO PLATFORM</span>
        <h1>ADMIN<br />CONTROL PLANE</h1>
        <p>IDENTITY · AUTHORIZATION · OPERATIONS</p>
        <div class="status">● SECURE CHANNEL READY</div>
      </div>
    </section>
    <section class="auth-form">
      <el-form ref="formRef" :model="form" :rules="formRules" label-position="top" @submit.prevent="submit">
        <div class="kicker">AUTHORIZED PERSONNEL ONLY</div>
        <h2>管理员登录</h2>
        <p class="muted">使用用户名、邮箱或手机号登录平台控制面。</p>
        <el-alert v-if="route.query.expired" type="warning" :closable="false" show-icon title="会话已过期，请重新登录。" />
        <el-form-item label="身份标识" prop="identifier"><el-input v-model="form.identifier" size="large" autocomplete="username" placeholder="username / email / phone" /></el-form-item>
        <el-form-item label="密码" prop="password"><el-input v-model="form.password" size="large" type="password" show-password autocomplete="current-password" /></el-form-item>
        <el-button native-type="submit" type="primary" size="large" :loading="loading" class="login-submit">建立管理会话</el-button>
        <el-button link class="bootstrap-link" @click="router.push('/bootstrap')">首次部署？初始化超级管理员</el-button>
      </el-form>
    </section>
  </div>
</template>

<style scoped>
.auth-page { min-height: 100vh; display: grid; grid-template-columns: minmax(0, 1.4fr) minmax(360px, 1fr); }
.auth-visual { position: relative; display: flex; align-items: flex-end; padding: 7vw; overflow: hidden; background: linear-gradient(135deg, #07101a, #0d1e2c 58%, #111827); }
.auth-visual::before { position: absolute; inset: 0; content: ''; background-image: linear-gradient(#38bdf811 1px, transparent 1px), linear-gradient(90deg, #38bdf811 1px, transparent 1px); background-size: 44px 44px; mask-image: linear-gradient(to bottom, transparent, #000); }
.terminal { position: relative; min-width: 0; }
.terminal span, .terminal p, .status { color: #7dd3fc; font: 12px var(--az-mono); letter-spacing: 2px; }
.terminal h1 { margin: 18px 0; color: #f1f7fd; font: 800 clamp(48px, 7vw, 104px)/.96 var(--az-mono); letter-spacing: -0.06em; overflow-wrap: anywhere; }
.status { margin-top: 50px; color: #4ade80; }
.auth-form { display: grid; place-items: center; padding: 50px; background: #0c1219; }
.auth-form :deep(.el-form) { width: min(420px, 100%); }
.kicker { color: #fb7185; font: 11px var(--az-mono); letter-spacing: 1.5px; }
h2 { margin: 12px 0 6px; font-size: 32px; }
.el-form-item { margin-top: 25px; }
.login-submit, .bootstrap-link { width: 100%; }
.bootstrap-link { margin: 14px 0 0; }
.scanline { position: absolute; inset: 0; background: linear-gradient(transparent 50%, #00000018 50%); background-size: 100% 4px; pointer-events: none; }

@media (max-width: 760px) {
  .auth-page { grid-template-columns: minmax(0, 1fr); }
  .auth-visual { min-height: 190px; align-items: center; padding: 22px 24px; }
  .terminal h1 { margin: 10px 0; font-size: clamp(26px, 8vw, 42px); letter-spacing: -0.04em; }
  .terminal span, .terminal p, .status { font-size: 10px; letter-spacing: 1px; }
  .status { margin-top: 12px; }
  .auth-form { min-height: 460px; padding: 28px 22px; }
  h2 { font-size: 28px; }
}
</style>
