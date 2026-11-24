<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import type { Site } from '../api/types'
import { auth, isSuperadmin, loadSession } from '../stores/auth'
import SiteFormDialog from '../components/SiteFormDialog.vue'
import SiteAdminsTab from '../components/SiteAdminsTab.vue'
import SiteProvidersTab from '../components/SiteProvidersTab.vue'
import SiteBansTab from '../components/SiteBansTab.vue'
import StatusTag from '../components/StatusTag.vue'
import { formatTime } from '../utils/format'

const route = useRoute()
const router = useRouter()
const siteId = computed(() => Number(route.params.id))
const site = computed<Site | undefined>(() => auth.sites.find((item) => item.id === siteId.value))
const superadmin = computed(() => isSuperadmin())
const dialog = ref(false)
const activeTab = ref('')

const tabs = computed(() => {
  const list: Array<{ name: string; label: string }> = []
  if (superadmin.value) {
    list.push({ name: 'settings', label: '设置' }, { name: 'admins', label: '管理员' })
  }
  list.push({ name: 'providers', label: 'OIDC 提供商' }, { name: 'bans', label: '封禁' })
  return list
})

function syncTab(): void {
  const fromQuery = String(route.query.tab || '')
  const allowed = tabs.value.map((t) => t.name)
  activeTab.value = allowed.includes(fromQuery) ? fromQuery : tabs.value[0]?.name || 'providers'
}

watch(() => [route.query.tab, tabs.value.length], syncTab, { immediate: true })

async function onSaved(): Promise<void> {
  await loadSession().catch(() => {})
}

async function rotate(): Promise<void> {
  const current = site.value
  if (!current) return
  try {
    await ElMessageBox.confirm('轮换 siteKey 后旧 key 立即失效，确定继续？', '确认', {
      type: 'warning',
    })
  } catch {
    return
  }
  try {
    const result = await api.rotateSiteKey(current.id)
    await ElMessageBox.alert(result.siteKey, '新的 siteKey', { confirmButtonText: '知道了' })
    await loadSession().catch(() => {})
  } catch (err: any) {
    ElMessage.error(err?.message || '轮换失败')
  }
}

async function copy(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

function goModeration(): void {
  void router.push({ path: '/moderation', query: { site: String(siteId.value) } })
}
</script>

<template>
  <div class="admin-page">
    <div class="page-header">
      <div>
        <h1>{{ site?.displayName || '站点详情' }}</h1>
        <p v-if="site">
          <span class="muted">{{ site.name }}</span>
          <StatusTag :status="site.status" />
        </p>
      </div>
      <el-button @click="router.push('/sites')">返回列表</el-button>
    </div>

    <el-result v-if="!site" icon="warning" title="站点不存在或无权访问" />

    <template v-else>
      <el-tabs v-model="activeTab">
        <el-tab-pane v-if="superadmin" name="settings" label="设置">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="siteKey">
              <span class="mono break">{{ site.key }}</span>
              <el-button link type="primary" @click="copy(site.key)">复制</el-button>
            </el-descriptions-item>
            <el-descriptions-item label="审核模式">
              <StatusTag :status="site.moderationMode" />
            </el-descriptions-item>
            <el-descriptions-item label="来源">
              {{ site.origins.join(', ') || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="默认语言">{{ site.defaultLocale }}</el-descriptions-item>
            <el-descriptions-item label="创建时间">{{ formatTime(site.createdAt) }}</el-descriptions-item>
            <el-descriptions-item label="更新时间">{{ formatTime(site.updatedAt) }}</el-descriptions-item>
          </el-descriptions>
          <div class="actions" style="margin-top: 16px">
            <el-button type="primary" @click="dialog = true">编辑站点</el-button>
            <el-button type="warning" @click="rotate">轮换 siteKey</el-button>
            <el-button @click="goModeration">去审核</el-button>
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="superadmin" name="admins" label="管理员">
          <SiteAdminsTab :site-id="siteId" />
        </el-tab-pane>

        <el-tab-pane name="providers" label="OIDC 提供商">
          <SiteProvidersTab :site-id="siteId" />
        </el-tab-pane>

        <el-tab-pane name="bans" label="封禁">
          <SiteBansTab :site-id="siteId" />
        </el-tab-pane>
      </el-tabs>

      <SiteFormDialog v-model="dialog" :site="site" @saved="onSaved" />
    </template>
  </div>
</template>
