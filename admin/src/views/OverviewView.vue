<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import type { Overview, OverviewPending } from '../api/types'
import StatCard from '../components/StatCard.vue'
import EmptyState from '../components/EmptyState.vue'
import ErrorState from '../components/ErrorState.vue'
import { formatTime } from '../utils/format'

const router = useRouter()
const overview = ref<Overview | null>(null)
const loading = ref(true)
const error = ref('')

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    overview.value = await api.overview()
  } catch (err: any) {
    error.value = err?.message || '加载概览失败'
  } finally {
    loading.value = false
  }
}

function goModerate(item: any): void {
  void router.push({ path: '/moderation', query: { site: String(item.siteId), status: 'pending' } })
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
        <div v-if="overview" class="stat-grid">
          <StatCard label="站点" :value="overview.counts.sites" />
          <StatCard label="用户" :value="overview.counts.users" />
          <StatCard label="批注" :value="overview.counts.annotations" />
          <StatCard
            label="待审批注"
            :value="overview.counts.pendingAnnotations"
            hint="等待处理"
          />
          <StatCard label="评论" :value="overview.counts.comments" />
          <StatCard
            label="待审评论"
            :value="overview.counts.pendingComments"
            hint="等待处理"
          />
        </div>

        <div class="section">
          <h2 class="section-title">最近待审批注</h2>
          <el-table v-if="overview?.recentAnnotations.length" :data="overview.recentAnnotations" border>
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
          <el-table v-if="overview?.recentComments.length" :data="overview.recentComments" border>
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
