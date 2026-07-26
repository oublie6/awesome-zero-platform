<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { adminApi } from '@/api/admin'
const router=useRouter(),available=ref(false),checked=ref(false),loading=ref(false);const form=reactive({token:'',username:'admin',displayName:'平台管理员',password:''})
onMounted(async()=>{try{available.value=(await adminApi.bootstrapStatus()).available}finally{checked.value=true}})
async function submit(){loading.value=true;try{await adminApi.bootstrap(form.token,{username:form.username,displayName:form.displayName,password:form.password});ElMessage.success('超级管理员已创建，请登录');router.replace('/login')}catch(e){ElMessage.error(e instanceof Error?e.message:'初始化失败')}finally{loading.value=false}}
</script>
<template><div class="bootstrap"><div class="panel card"><div class="technical muted">BOOTSTRAP / ONE-TIME OPERATION</div><h1>初始化超级管理员</h1><el-alert v-if="checked&&!available" type="warning" :closable="false" title="初始化入口不可用：未配置 Bootstrap Token，或超级管理员已经存在。"/><template v-else><p>Bootstrap Token 只能来自部署环境，不会在页面中显示。完成后该令牌应立即从生产环境移除。</p><el-form label-position="top"><el-form-item label="Bootstrap Token"><el-input v-model="form.token" type="password" show-password/></el-form-item><el-form-item label="用户名"><el-input v-model="form.username"/></el-form-item><el-form-item label="显示名称"><el-input v-model="form.displayName"/></el-form-item><el-form-item label="初始密码"><el-input v-model="form.password" type="password" show-password/></el-form-item><el-button type="danger" :loading="loading" @click="submit">创建唯一初始超级管理员</el-button></el-form></template><el-button link @click="router.push('/login')">返回登录</el-button></div></div></template>
<style scoped>.bootstrap{min-height:100vh;display:grid;place-items:center;padding:40px}.card{width:620px}.card h1{font-size:30px}.card p{color:var(--az-muted);line-height:1.7}.el-form{margin:26px 0}</style>
