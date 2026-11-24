<script setup lang="ts">
import type { Site } from '../api/types'

const props = defineProps<{ modelValue: number | undefined; sites: Site[] }>()
const emit = defineEmits<{ 'update:modelValue': [number] }>()

function onChange(value: number): void {
  emit('update:modelValue', value)
}
</script>

<template>
  <el-select
    :model-value="modelValue"
    placeholder="选择站点"
    style="width: 220px"
    @update:model-value="onChange"
  >
    <el-option
      v-for="site in props.sites"
      :key="site.id"
      :label="`${site.displayName || site.name} (#${site.id})`"
      :value="site.id"
    />
  </el-select>
</template>
