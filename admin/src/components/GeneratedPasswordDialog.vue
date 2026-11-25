<script setup lang="ts">
import { ElMessage } from 'element-plus'

defineProps<{
  modelValue: boolean
  password: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

async function copy(password: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(password)
    ElMessage.success('已复制')
  } catch {
    ElMessage.warning('复制失败，请手动选择文本')
  }
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    title="请保存初始密码（仅显示一次）"
    width="440px"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <el-input :model-value="password" readonly />
    <template #footer>
      <el-button @click="emit('update:modelValue', false)">我已保存</el-button>
      <el-button type="primary" @click="copy(password)">复制</el-button>
    </template>
  </el-dialog>
</template>
