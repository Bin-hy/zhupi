<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import type { Overview, OverviewPending } from '../api/types'
import StatCard from '../components/StatCard.vue'
import EmptyState from '../components/EmptyState.vue'
import ErrorState from '../components/ErrorState.vue'
import { formatTime } from '../utils/format'
import { moderationBadge } from '../stores/moderation'

const router = useRouter()
const overview = ref<Overview | null>(null)
const loading = ref(true)
const error = ref('')

const pendingTotal = computed(
  () => (overview.value?.counts.pendingAnnotations || 0) + (overview.value?.counts.pendingComments || 0),
)

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    overview.value = await api.overview()
    moderationBadge.pending = pendingTotal.value
  } catch (err: any) {
    error.value = err?.message || '加载概览失败'
  } finally {
    loading.value = false
  }
}

function goModerate(raw: Record<PropertyKey, any>): void {
  const item = raw as OverviewPending
  void router.push({
    path: '/moderation',
    query: { site: String(item.siteId), status: 'pending', type: item.kind },
  })
}

function goPending(type?: 'annotation' | 'comment'): void {
  void router.push({ path: '/moderation', query: { status: 'pending', ...(type ? { type } : {}) } })
}

onMounted(load)
</script>

<template>
  <div class="admin-page">
    <div class="page-header">
      <div>
        <h1>概览</h1>
        <p>待审积压与站点状态一览</p>
      </div>
      <el-button :loading="loading" @click="load">刷新</el-button>
    </div>

    <ErrorState v-if="error" :message="error" @retry="load" />
    <template v-else>
      <div v-loading="loading">
        <div v-if="pendingTotal > 0" class="pending-banner">
          <div>
            <strong>{{ pendingTotal }}</strong>
            <span> 条内容等待审核</span>
            <span class="muted">批注 {{ overview?.counts.pendingAnnotations || 0 }} · 评论 {{ overview?.counts.pendingComments || 0 }}</span>
          </div>
          <el-button type="primary" @click="goPending()">立即处理</el-button>
        </div>

        <div v-if="overview" class="stat-grid">
          <StatCard label="站点" :value="overview.counts.sites" />
          <StatCard label="用户" :value="overview.counts.users" />
          <StatCard label="批注" :value="overview.counts.annotations" />
          <div class="stat-click" @click="goPending('annotation')">
            <StatCard label="待审批注" :value="overview.counts.pendingAnnotations" hint="点击处理" />
          </div>
          <StatCard label="评论" :value="overview.counts.comments" />
          <div class="stat-click" @click="goPending('comment')">
            <StatCard label="待审评论" :value="overview.counts.pendingComments" hint="点击处理" />
          </div>
        </div>

        <div class="section">
          <h2 class="section-title">最近待审批注</h2>
          <el-table
            v-if="overview?.recentAnnotations?.length"
            :data="overview.recentAnnotations"
            border
          >
            <el-table-column prop="siteName" label="站点" width="140" />
            <el-table-column label="作者" width="140">
              <template #default="{ row }">{{ row.author.name }}</template>
            </el-table-column>
            <el-table-column prop="quoteExact" label="引用" min-width="220" show-overflow-tooltip />
            <el-table-column label="时间" width="170">
              <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button link type="primary" @click="goModerate(row)">去审核</el-button>
              </template>
            </el-table-column>
          </el-table>
          <EmptyState v-else description="没有待审批注" />
        </div>

        <div class="section">
          <h2 class="section-title">最近待审评论</h2>
          <el-table
            v-if="overview?.recentComments?.length"
            :data="overview.recentComments"
            border
          >
            <el-table-column prop="siteName" label="站点" width="140" />
            <el-table-column label="作者" width="140">
              <template #default="{ row }">{{ row.author.name }}</template>
            </el-table-column>
            <el-table-column prop="content" label="内容" min-width="220" show-overflow-tooltip />
            <el-table-column label="时间" width="170">
              <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button link type="primary" @click="goModerate(row)">去审核</el-button>
              </template>
            </el-table-column>
          </el-table>
          <EmptyState v-else description="没有待审评论" />
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.pending-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
  padding: 12px 16px;
  border: 1px solid var(--el-color-warning-light-5);
  border-radius: 10px;
  background: var(--el-color-warning-light-9);
}
.pending-banner strong {
  margin-right: 4px;
  font-size: 20px;
  color: var(--el-color-warning-dark-2);
}
.pending-banner .muted {
  margin-left: 10px;
}
.stat-click {
  cursor: pointer;
  transition: transform 0.15s ease;
}
.stat-click:hover {
  transform: translateY(-2px);
}
</style>
