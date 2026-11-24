<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ status: string }>()

const map: Record<string, { label: string; type: 'success' | 'warning' | 'danger' | 'info' | 'primary' }> = {
  pending: { label: '待审', type: 'warning' },
  approved: { label: '已通过', type: 'success' },
  rejected: { label: '已拒绝', type: 'danger' },
  hidden: { label: '已隐藏', type: 'info' },
  deleted: { label: '已删除', type: 'info' },
  active: { label: '启用', type: 'success' },
  disabled: { label: '停用', type: 'info' },
  banned: { label: '已封禁', type: 'danger' },
  open: { label: '开放', type: 'primary' },
  resolved: { label: '已解决', type: 'info' },
  post: { label: '先发后审', type: 'primary' },
  pre: { label: '先审后发', type: 'warning' },
}

const view = computed(() => map[props.status] || { label: props.status, type: 'info' as const })
</script>

<template>
  <el-tag :type="view.type" size="small" disable-transitions>{{ view.label }}</el-tag>
</template>
