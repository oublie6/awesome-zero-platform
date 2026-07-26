<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import PageHeader from '@/components/PageHeader.vue'
import { adminApi } from '@/api/admin'
import { useAuthStore } from '@/stores/auth'
const auth=useAuthStore(),overview=ref<Record<string,unknown>>({}),loading=ref(true)
onMounted(async()=>{try{overview.value=await adminApi.overview()}finally{loading.value=false}})
const cards=computed(()=>{ const engine=overview.value.engine as {name?:string}|undefined; return [
  ['API SERVICE',overview.value.service||'unknown'],['MYSQL',overview.value.mysql||'unknown'],['REDIS',overview.value.redis||'unknown'],['AUTH ENGINE',engine?.name||'unknown']
]})
</script>
<template><PageHeader title="运行概览" description="管理控制面的实时状态、当前身份和基础依赖。默认视图只给结论，技术细节在系统状态页展开。"/><div class="grid grid-4" v-loading="loading"><div v-for="item in cards" :key="String(item[0])" class="panel"><div class="metric-label technical">{{item[0]}}</div><div class="metric-value">{{item[1]}}</div></div></div><div class="grid grid-2" style="margin-top:16px"><section class="panel"><h3>当前管理员</h3><el-descriptions :column="1" border><el-descriptions-item label="账号">{{auth.me?.account.displayName}}</el-descriptions-item><el-descriptions-item label="Account ID"><span class="technical">{{auth.me?.account.id}}</span></el-descriptions-item><el-descriptions-item label="角色"><el-tag v-for="role in auth.roles" :key="role" style="margin-right:6px">{{role}}</el-tag></el-descriptions-item></el-descriptions></section><section class="panel"><h3>管理面边界</h3><p class="muted">标准页面用于日常配置；带有 <b style="color:#fb7185">EXPERT MODE</b> 的页面展示权限插件、原始策略和运行配置诊断。两者共享同一后端服务，不存在第二套数据。</p></section></div></template>
