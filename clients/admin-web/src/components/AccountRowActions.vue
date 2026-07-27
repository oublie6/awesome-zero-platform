<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { MoreFilled } from '@element-plus/icons-vue'
import type { Account } from '@/types/admin'

type ActionKey = 'detail' | 'edit' | 'toggle' | 'reset'
type ActionType = 'primary' | 'success' | 'warning' | 'danger' | 'info'

type RowAction = {
  key: ActionKey
  label: string
  type: ActionType
}

const props = defineProps<{ status: Account['status'] }>()
const emit = defineEmits<{
  detail: []
  edit: []
  toggle: []
  reset: []
}>()

const GAP = 8
const container = ref<HTMLElement | null>(null)
const measureRoot = ref<HTMLElement | null>(null)
const visibleCount = ref(0)
let resizeObserver: ResizeObserver | undefined

const actions = computed<RowAction[]>(() => [
  { key: 'detail', label: '详情', type: 'info' },
  { key: 'edit', label: '编辑', type: 'primary' },
  { key: 'toggle', label: props.status === 'active' ? '禁用' : '启用', type: props.status === 'active' ? 'warning' : 'success' },
  { key: 'reset', label: '重置密码', type: 'danger' },
])

const visibleActions = computed(() => actions.value.slice(0, visibleCount.value))
const overflowActions = computed(() => actions.value.slice(visibleCount.value))

function actionWidth(elements: HTMLElement[], count: number) {
  if (count === 0) return 0
  return elements.slice(0, count).reduce((total, element) => total + element.offsetWidth, 0) + GAP * (count - 1)
}

async function recalculate() {
  await nextTick()
  const host = container.value
  const measurer = measureRoot.value
  if (!host || !measurer) return

  const availableWidth = host.clientWidth
  const actionElements = Array.from(measurer.querySelectorAll<HTMLElement>('[data-measure-action]'))
  const overflowElement = measurer.querySelector<HTMLElement>('[data-measure-overflow]')
  if (availableWidth <= 0 || actionElements.length !== actions.value.length || !overflowElement) return

  for (let count = actions.value.length; count >= 0; count -= 1) {
    const hasOverflow = count < actions.value.length
    const directWidth = actionWidth(actionElements, count)
    const overflowWidth = hasOverflow ? overflowElement.offsetWidth + (count > 0 ? GAP : 0) : 0
    if (directWidth + overflowWidth <= availableWidth) {
      visibleCount.value = count
      return
    }
  }

  visibleCount.value = 0
}

function execute(command: string | number) {
  switch (String(command) as ActionKey) {
    case 'detail':
      emit('detail')
      break
    case 'edit':
      emit('edit')
      break
    case 'toggle':
      emit('toggle')
      break
    case 'reset':
      emit('reset')
      break
  }
}

onMounted(() => {
  resizeObserver = new ResizeObserver(() => recalculate())
  if (container.value) resizeObserver.observe(container.value)
  recalculate()
})

watch(actions, () => recalculate(), { deep: true })

onBeforeUnmount(() => resizeObserver?.disconnect())
</script>

<template>
  <div ref="container" class="account-row-actions">
    <el-button
      v-for="action in visibleActions"
      :key="action.key"
      size="small"
      :type="action.type"
      plain
      @click="execute(action.key)"
    >
      {{ action.label }}
    </el-button>

    <el-dropdown v-if="overflowActions.length" trigger="click" placement="bottom-end" @command="execute">
      <el-button class="more-action" size="small" circle aria-label="更多账号操作">
        <el-icon><MoreFilled /></el-icon>
      </el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item v-for="action in overflowActions" :key="action.key" :command="action.key">
            <span :class="{ 'danger-text': action.type === 'danger' }">{{ action.label }}</span>
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>

    <div ref="measureRoot" class="action-measurer" aria-hidden="true">
      <el-button
        v-for="action in actions"
        :key="action.key"
        data-measure-action
        size="small"
        :type="action.type"
        plain
      >
        {{ action.label }}
      </el-button>
      <el-button data-measure-overflow size="small" circle>
        <el-icon><MoreFilled /></el-icon>
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.account-row-actions {
  position: relative;
  width: 100%;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
}

.account-row-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}

.account-row-actions :deep(.el-dropdown) {
  flex: 0 0 auto;
}

.more-action {
  flex: 0 0 auto;
}

.action-measurer {
  position: absolute;
  top: 0;
  left: -10000px;
  width: max-content;
  display: flex;
  align-items: center;
  gap: 8px;
  visibility: hidden;
  pointer-events: none;
}

.action-measurer :deep(.el-button + .el-button) {
  margin-left: 0;
}

.danger-text {
  color: var(--el-color-danger);
}
</style>
