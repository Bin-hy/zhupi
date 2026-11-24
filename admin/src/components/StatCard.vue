<script setup lang="ts">
defineProps<{ label: string; value: number | string; hint?: string }>()
</script>

<template>
  <el-card shadow="never" class="stat-card">
    <div class="label">{{ label }}</div>
    <div class="value">{{ value }}</div>
    <div v-if="hint" class="muted">{{ hint }}</div>
  </el-card>
</template>

<style scoped>
.stat-card {
  border-radius: 10px;
}
.label {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.value {
  margin-top: 6px;
  font-size: 26px;
  font-weight: 700;
  line-height: 1.2;
}
</style>
