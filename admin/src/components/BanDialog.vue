<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'

const props = defineProps<{ modelValue: boolean; siteId: number }>()
const emit = defineEmits<{ 'update:modelValue': [boolean]; saved: [] }>()

const userId = ref<number | undefined>()
const reason = ref('')
const saving = ref(false)

function close(): void {
  emit('update:modelValue', false)
}

async function submit(): Promise<void> {
  if (!userId.value || userId.value <= 0) {
    ElMessage.warning('请输入用户 ID')
    return
  }
  saving.value = true
  try {
    await api.banSiteUser(props.siteId, userId.value, reason.value.trim())
    ElMessage.success('已封禁')
    userId.value = undefined
    reason.value = ''
    emit('saved')
    close()
  } catch (err: any) {
    ElMessage.error(err?.message || '封禁失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    title="封禁站点用户"
    width="460px"
    @update:model-value="close"
  >
    <el-form label-width="80px">
      <el-form-item label="用户 ID" required>
        <el-input-number v-model="userId" :min="1" controls-position="right" />
      </el-form-item>
      <el-form-item label="原因">
        <el-input v-model="reason" maxlength="200" placeholder="可选" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="danger" :loading="saving" @click="submit">封禁</el-button>
    </template>
  </el-dialog>
</template>
