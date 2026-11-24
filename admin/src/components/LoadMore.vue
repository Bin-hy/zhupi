<script setup lang="ts">
defineProps<{ hasMore: boolean; loading?: boolean }>()
const emit = defineEmits<{ load: [] }>()
</script>

<template>
  <div v-if="hasMore" class="load-more">
    <el-button :loading="loading" @click="emit('load')">加载更多</el-button>
  </div>
</template>

<style scoped>
.load-more {
  display: flex;
  justify-content: center;
  padding: 14px 0 4px;
}
</style>
