<script setup lang="ts">defineProps<{ title?: string; value: string }>()</script>
<template><section class="code-panel"><header v-if="title">{{ title }}</header><pre>{{ value }}</pre></section></template>
<style scoped>.code-panel{border:1px solid var(--az-border);border-radius:10px;background:#080d13;overflow:hidden}.code-panel header{padding:10px 14px;border-bottom:1px solid #26313d;color:#7ee787;font-family:var(--az-mono)}pre{margin:0;padding:16px;overflow:auto;color:#d7e1ec;line-height:1.6;font:13px/1.6 var(--az-mono);white-space:pre-wrap}</style>
