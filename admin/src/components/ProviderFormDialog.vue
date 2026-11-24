<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import type { Provider } from '../api/types'

const props = defineProps<{ modelValue: boolean; siteId: number; provider?: Provider | null }>()
const emit = defineEmits<{ 'update:modelValue': [boolean]; saved: [] }>()

const form = reactive({
  name: '',
  issuer: '',
  clientId: '',
  clientSecret: '',
  scopes: 'openid, email, profile',
  allowedEmailDomains: '',
  enabled: true,
})
const saving = ref(false)
const editing = computed(() => !!props.provider)

watch(
  () => [props.modelValue, props.provider],
  () => {
    if (!props.modelValue) return
    const p = props.provider
    form.name = p?.name || ''
    form.issuer = p?.issuer || ''
    form.clientId = p?.clientId || ''
    form.clientSecret = ''
    form.scopes = p?.scopes?.join(', ') || 'openid, email, profile'
    form.allowedEmailDomains = p?.allowedEmailDomains?.join(', ') || ''
    form.enabled = p?.enabled ?? true
  },
  { immediate: true },
)

function splitCSV(raw: string): string[] {
  return raw
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

function close(): void {
  emit('update:modelValue', false)
}

async function submit(): Promise<void> {
  if (!form.name.trim() || !form.issuer.trim() || !form.clientId.trim()) {
    ElMessage.warning('名称、issuer、clientId 必填')
    return
  }
  if (!/^https?:\/\/[^/\s]+$/i.test(form.issuer.trim())) {
    ElMessage.warning('issuer 必须是绝对 URL')
    return
  }
  if (!editing.value && !form.clientSecret.trim()) {
    ElMessage.warning('新增提供商必须填写 clientSecret')
    return
  }

  const input = {
    name: form.name.trim(),
    issuer: form.issuer.trim(),
    clientId: form.clientId.trim(),
    ...(form.clientSecret.trim() ? { clientSecret: form.clientSecret.trim() } : {}),
    scopes: splitCSV(form.scopes),
    allowedEmailDomains: splitCSV(form.allowedEmailDomains),
    enabled: form.enabled,
  }

  saving.value = true
  try {
    if (props.provider) {
      await api.updateProvider(props.siteId, props.provider.id, input)
    } else {
      await api.createProvider(props.siteId, input)
    }
    ElMessage.success(editing.value ? '提供商已更新' : '提供商已创建')
    emit('saved')
    close()
  } catch (err: any) {
    ElMessage.error(err?.message || '保存失败（请检查 Discovery）')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    :title="editing ? '编辑 OIDC 提供商' : '新增 OIDC 提供商'"
    width="620px"
    @update:model-value="close"
  >
    <el-form label-width="130px">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" maxlength="64" />
      </el-form-item>
      <el-form-item label="Issuer" required>
        <el-input v-model="form.issuer" placeholder="https://sso.example.com" />
      </el-form-item>
      <el-form-item label="Client ID" required>
        <el-input v-model="form.clientId" />
      </el-form-item>
      <el-form-item label="Client Secret" :required="!editing">
        <el-input
          v-model="form.clientSecret"
          type="password"
          show-password
          :placeholder="editing ? '留空保持原值' : ''"
        />
      </el-form-item>
      <el-form-item label="Scopes">
        <el-input v-model="form.scopes" placeholder="openid, email, profile" />
      </el-form-item>
      <el-form-item label="允许邮箱域">
        <el-input v-model="form.allowedEmailDomains" placeholder="example.com, corp.example.com" />
      </el-form-item>
      <el-form-item label="启用">
        <el-switch v-model="form.enabled" />
      </el-form-item>
    </el-form>
    <el-alert
      title="保存时服务端会做一次 OIDC Discovery 校验，issuer 必须可访问。"
      type="info"
      :closable="false"
    />
    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>
