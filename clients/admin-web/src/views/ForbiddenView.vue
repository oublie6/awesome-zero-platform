<template>
  <main class="forbidden-page">
    <div class="technical">HTTP 403</div>
    <h1>当前账号无权访问此页面</h1>
    <p>页面访问控制仅用于界面导航；具体数据和操作仍由服务端逐次授权。</p>
    <el-button type="primary" @click="$router.replace('/dashboard')">返回运行概览</el-button>
  </main>
</template>

<style scoped>
.forbidden-page {
  min-height: 100vh;
  display: grid;
  place-content: center;
  justify-items: center;
  gap: 12px;
  padding: 24px;
  text-align: center;
}

.technical { color: #fb7185; }
h1 { margin: 0; font-size: clamp(22px, 4vw, 32px); }
p { max-width: 560px; color: var(--az-muted); line-height: 1.6; }
</style>
