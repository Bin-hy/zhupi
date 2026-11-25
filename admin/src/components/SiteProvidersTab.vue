<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import type { Provider } from '../api/types'
import ProviderFormDialog from './ProviderFormDialog.vue'
import StatusTag from './StatusTag.vue'
import { formatTime } from '../utils/format'

const props = defineProps<{ siteId: number }>()
const providers = ref<Provider[]>([])
const loading = ref(false)
const error = ref('')
const dialog = ref(false)
const editing = ref<Provider | null>(null)

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const res = await api.listProviders(props.siteId)
    providers.value = res.providers
  } catch (err: any) {
    error.value = err?.message || '加载提供商失败'
  } finally {
    loading.value = false
  }
}

function openCreate(): void {
  editing.value = null
  dialog.value = true
}

function openEdit(provider: any): void {
  editing.value = provider
  dialog.value = true
}

async function toggle(provider: any): Promise<void> {
  try {
    await api.updateProvider(props.siteId, provider.id, {
      name: provider.name,
      issuer: provider.issuer,
      clientId: provider.clientId,
      scopes: provider.scopes,
      allowedEmailDomains: provider.allowedEmailDomains,
      enabled: !provider.enabled,
    })
    ElMessage.success(provider.enabled ? '已停用' : '已启用')
    await load()
  } catch (err: any) {
    ElMessage.error(err?.message || '更新失败')
  }
}

async function remove(provider: any): Promise<void> {
  try {
    await ElMessageBox.confirm(`删除提供商「${provider.name}」？`, '确认', { type: 'warning' })
  } catch {
    return
  }
  try {
    await api.deleteProvider(props.siteId, provider.id)
    ElMessage.success('已删除')
    await load()
  } catch (err: any) {
    ElMessage.error(err?.message || '删除失败')
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <el-button type="primary" @click="openCreate">新增提供商</el-button>
      <el-button :loading="loading" @click="load">刷新</el-button>
    </div>
    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <el-table v-loading="loading" :data="providers" border>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="name" label="名称" min-width="130" />
      <el-table-column prop="issuer" label="Issuer" min-width="200" show-overflow-tooltip />
      <el-table-column prop="clientId" label="Client ID" min-width="150" show-overflow-tooltip />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <StatusTag :status="row.enabled ? 'active' : 'disabled'" />
        </template>
      </el-table-column>
      <el-table-column label="更新时间" width="170">
        <template #default="{ row }">{{ formatTime(row.updatedAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="190" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="warning" @click="toggle(row)">
            {{ row.enabled ? '停用' : '启用' }}
          </el-button>
          <el-button link type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="!loading && providers.length === 0" description="还没有 OIDC 提供商" />
    <ProviderFormDialog
      v-model="dialog"
      :site-id="siteId"
      :provider="editing"
      @saved="load"
    />
  </div>
</template>
