<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import type { LocalAdminResult, Site, User } from '../api/types'

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'open'
  user?: { id: number; displayName: string } | null
  users?: User[]
  sites: Site[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  success: [result: LocalAdminResult]
}>()

const form = reactive({
  userId: 0,
  username: '',
  displayName: '',
  password: '',
  isSuperadmin: false,
  siteIds: [] as number[],
})

const submitting = ref(false)

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    form.userId = props.user?.id ?? 0
    form.username = ''
    form.displayName = ''
    form.password = ''
    form.isSuperadmin = false
    form.siteIds = []
    submitting.value = false
  },
)

function close(): void {
  emit('update:modelValue', false)
}

function validate(): string | null {
  if (props.mode === 'open' && !form.userId) return '请选择用户'
  if (!/^[A-Za-z0-9._-]{3,32}$/.test(form.username.trim())) {
    return '用户名需为 3–32 个字符，仅字母、数字、.、_、-'
  }
  if (form.password && form.password.length < 8) return '密码至少需要 8 个字符'
  return null
}

async function submit(): Promise<void> {
  const problem = validate()
  if (problem) {
    ElMessage.warning(problem)
    return
  }
  submitting.value = true
  try {
    const result =
      props.mode === 'create'
        ? await api.createLocalAdmin({
            username: form.username.trim(),
            displayName: form.displayName.trim(),
            initialPassword: form.password || undefined,
            isSuperadmin: form.isSuperadmin,
            siteIds: form.siteIds,
          })
        : await api.openLocalCredential({
            userId: form.userId,
            username: form.username.trim(),
            initialPassword: form.password || undefined,
            isSuperadmin: form.isSuperadmin,
            siteIds: form.siteIds,
          })
    emit('success', result)
    close()
  } catch (err: any) {
    ElMessage.error(err?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    :title="mode === 'create' ? '新建管理员' : '为已有用户开通本地密码'"
    width="520px"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <el-form label-width="96px" @submit.prevent="submit">
      <el-form-item v-if="mode === 'open'" label="用户">
        <el-select
          v-model="form.userId"
          filterable
          :disabled="!!user"
          placeholder="选择已有用户"
          style="width: 100%"
        >
          <el-option
            v-for="item in users || []"
            :key="item.id"
            :value="item.id"
            :label="`${item.displayName || '未命名'} (#${item.id})`"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="用户名" required>
        <el-input v-model="form.username" placeholder="3–32 字符，字母/数字/._-" />
      </el-form-item>

      <el-form-item v-if="mode === 'create'" label="显示名称">
        <el-input v-model="form.displayName" placeholder="留空则与用户名相同" />
      </el-form-item>

      <el-form-item label="初始密码">
        <el-input
          v-model="form.password"
          type="password"
          show-password
          placeholder="留空自动生成，稍后只展示一次"
        />
      </el-form-item>

      <el-form-item label="超级管理员">
        <el-switch v-model="form.isSuperadmin" />
      </el-form-item>

      <el-form-item label="站点管理员">
        <el-select
          v-model="form.siteIds"
          multiple
          filterable
          placeholder="可不选"
          style="width: 100%"
        >
          <el-option
            v-for="site in sites"
            :key="site.id"
            :value="site.id"
            :label="site.displayName || site.name"
          />
        </el-select>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="submit">
        {{ mode === 'create' ? '创建' : '开通' }}
      </el-button>
    </template>
  </el-dialog>
</template>
