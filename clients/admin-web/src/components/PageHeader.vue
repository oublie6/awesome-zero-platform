<script setup lang="ts">
defineProps<{ title: string; description?: string; expert?: boolean }>()
</script>

<template>
  <div class="page-header">
    <div class="page-header-copy">
      <div class="title-row">
        <h1>{{ title }}</h1>
        <el-tag v-if="expert" type="danger" effect="dark">EXPERT MODE</el-tag>
      </div>
      <p v-if="description">{{ description }}</p>
    </div>
    <div class="page-actions">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 18px 24px;
  margin-bottom: 22px;
}

.page-header-copy {
  min-width: 0;
}

.title-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 12px;
}

h1 {
  margin: 0;
  font-size: 24px;
}

p {
  max-width: 760px;
  margin: 8px 0 0;
  color: var(--az-muted);
  line-height: 1.6;
}

.page-actions {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  justify-content: flex-end;
  align-items: center;
  gap: 8px 10px;
}

.page-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}

@media (max-width: 760px) {
  .page-header {
    flex-direction: column;
  }

  .page-actions {
    width: 100%;
    justify-content: flex-start;
  }
}
</style>
