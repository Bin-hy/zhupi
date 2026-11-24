<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import type { SiteBanView } from '../api/types'
import BanDialog from './BanDialog.vue'
import { formatTime } from '../utils/format'

const props = defineProps<{ siteId: number }>()
const bans = ref<SiteBanView[]>([])
const loading = ref(false)
const error = ref('')
const dialog = ref(false)

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const res = await api.listSiteBans(props.siteId)
    bans.value = res.bans
  } catch (err: any) {
    error.value = err?.message || '加载封禁列表失败'
  } finally {
    loading.value = false
  }
}

async function unban(ban: any): Promise<void> {
  try {
    await ElMessageBox.confirm(`解除对用户 ${ban.userId} 的封禁？`, '确认', { type: 'warning' })
  } catch {
    return
  }
  try {
    await api.unbanSiteUser(props.siteId, ban.userId)
    ElMessage.success('已解封')
    await load()
  } catch (err: any) {
    ElMessage.error(err?.message || '解封失败')
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <el-button type="danger" @click="dialog = true">封禁用户</el-button>
      <el-button :loading="loading" @click="load">刷新</el-button>
    </div>
    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <el-table v-loading="loading" :data="bans" border>
      <el-table-column prop="userId" label="用户 ID" width="100" />
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column prop="reason" label="原因" min-width="180" show-overflow-tooltip />
      <el-table-column prop="bannedBy" label="操作者" width="100" />
      <el-table-column label="时间" width="170">
        <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-button link type="primary" @click="unban(row)">解封</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="!loading && bans.length === 0" description="没有封禁用户" />
    <BanDialog v-model="dialog" :site-id="siteId" @saved="load" />
  </div>
</template>
