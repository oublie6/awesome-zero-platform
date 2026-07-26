<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageHeader from '@/components/PageHeader.vue';import CodePanel from '@/components/CodePanel.vue';import { adminApi } from '@/api/admin'
const overview=ref<Record<string,unknown>>({}),configuration=ref<Record<string,unknown>>({});const loading=ref(true)
onMounted(async()=>{try{[overview.value,configuration.value]=await Promise.all([adminApi.overview(),adminApi.configuration()])}finally{loading.value=false}})
</script>
<template><PageHeader title="系统状态" description="普通视图关注可用性，诊断视图展示配置来源和安全状态，但永不回显 Secret。"/><el-tabs v-loading="loading"><el-tab-pane label="运行状态"><div class="grid grid-2"><section class="panel" v-for="(value,key) in overview" :key="key"><div class="metric-label technical">{{key}}</div><div class="metric-value">{{value}}</div></section></div></el-tab-pane><el-tab-pane label="配置诊断"><div class="expert-banner">CONFIG DIAGNOSTICS / READ ONLY — 敏感配置只显示是否已配置，不显示原始值。</div><CodePanel title="EFFECTIVE CONFIGURATION" :value="JSON.stringify(configuration,null,2)"/></el-tab-pane></el-tabs></template>
