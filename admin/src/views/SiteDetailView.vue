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
const apiBase = computed(() => window.location.origin)
const snippet = computed(() => {
  const key = site.value?.key || 'YOUR_SITE_KEY'
  return `<script src="${apiBase.value}/sdk/v1/zhupi.js"><\/script>\n<script>\n  window.ZhupiSDK.init({\n    siteKey: '${key}',\n    apiBase: '${apiBase.value}',\n  })\n<\/script>`
})

const tabs = computed(() => {
  const list: Array<{ name: string; label: string }> = []
  if (superadmin.value) {
    list.push({ name: 'settings', label: '设置' }, { name: 'admins', label: '管理员' })
  }
  list.push(
    { name: 'install', label: '接入' },
    { name: 'providers', label: 'OIDC 提供商' },
    { name: 'bans', label: '封禁' },
  )
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

        <el-tab-pane name="install" label="接入">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="siteKey">
              <span class="mono break">{{ site.key }}</span>
              <el-button link type="primary" @click="copy(site.key)">复制</el-button>
            </el-descriptions-item>
            <el-descriptions-item label="API 地址">
              <span class="mono break">{{ apiBase }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="审核模式">
              <StatusTag :status="site.moderationMode" />
            </el-descriptions-item>
            <el-descriptions-item label="允许来源">
              <span class="break">{{ site.origins.join(', ') || '-' }}</span>
            </el-descriptions-item>
          </el-descriptions>

          <h3 class="install-title">页面接入代码</h3>
          <p class="muted">
            把下面的代码放到文档站模板中（VitePress 可放在 <code>theme/index.ts</code> 的
            <code>setup()</code> 里动态加载）。SPA 切换页面后调用 <code>setPage()</code>。
          </p>
          <pre class="snippet"><code>{{ snippet }}</code></pre>
          <div class="actions" style="margin-top: 12px">
            <el-button type="primary" @click="copy(snippet)">复制接入代码</el-button>
            <el-button @click="goModeration">去审核</el-button>
          </div>

          <el-alert
            type="info"
            :closable="false"
            show-icon
            style="margin-top: 16px"
            title="上线前检查"
          >
            <ul class="checklist">
              <li>把正式域名和所有预览域名加入「允许来源」，否则 SDK 会收到 origin_not_allowed。</li>
              <li>VITE_* 等前端环境变量在构建期注入，修改后必须重新部署。</li>
              <li>先发后审模式下内容立即可见；先审后发模式下作者本人仍能看到自己的待审内容。</li>
              <li>轮换 siteKey 会让旧 key 立即失效，请同步更新部署环境变量。</li>
            </ul>
          </el-alert>
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

<style scoped>
.install-title {
  margin: 20px 0 6px;
  font-size: 15px;
}
.snippet {
  margin: 10px 0 0;
  padding: 14px 16px;
  overflow-x: auto;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-fill-color-light);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  line-height: 1.7;
  white-space: pre;
}
.checklist {
  margin: 6px 0 0;
  padding-left: 18px;
  line-height: 1.8;
}
</style>
