<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Menu, Moon, Setting, Sunny, SwitchButton, User } from '@element-plus/icons-vue'
import { auth, isSuperadmin, logout } from '../stores/auth'
import { moderationBadge, refreshModerationBadge } from '../stores/moderation'

const route = useRoute()
const router = useRouter()
const collapsed = ref(window.innerWidth < 900)
const dark = ref(document.documentElement.classList.contains('dark'))

const active = computed(() => {
  if (route.path.startsWith('/sites')) return '/sites'
  if (route.path.startsWith('/moderation')) return '/moderation'
  if (route.path.startsWith('/users')) return '/users'
  return '/'
})

function onSelect(index: string): void {
  void router.push(index)
}

function toggleDark(): void {
  dark.value = !dark.value
  document.documentElement.classList.toggle('dark', dark.value)
  localStorage.setItem('annotation-admin:dark', dark.value ? '1' : '0')
}

async function onLogout(): Promise<void> {
  try {
    await ElMessageBox.confirm('确定退出登录？', '提示', { type: 'warning' })
  } catch {
    return
  }
  await logout()
  ElMessage.success('已退出')
  await router.replace('/login')
}

onMounted(() => {
  const saved = localStorage.getItem('annotation-admin:dark')
  if (saved !== null) {
    dark.value = saved === '1'
    document.documentElement.classList.toggle('dark', dark.value)
  }
  void refreshModerationBadge()
})
</script>

<template>
  <el-container class="layout">
    <el-aside :width="collapsed ? '64px' : '220px'" class="aside">
      <div class="brand" :class="{ collapsed }">
        <span v-if="!collapsed">朱批 · Zhupi</span>
        <span v-else>朱</span>
      </div>
      <el-menu :collapse="collapsed" :default-active="active" @select="onSelect">
        <el-menu-item index="/">概览</el-menu-item>
        <el-menu-item index="/sites">站点</el-menu-item>
        <el-menu-item index="/moderation">
          内容审核
          <span v-if="!collapsed && moderationBadge.pending" class="nav-badge">
            {{ moderationBadge.pending > 99 ? '99+' : moderationBadge.pending }}
          </span>
        </el-menu-item>
        <el-menu-item v-if="isSuperadmin()" index="/users">用户管理</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <el-button text circle @click="collapsed = !collapsed">
          <el-icon><Menu /></el-icon>
        </el-button>
        <div class="spacer" />
        <el-tag v-if="auth.me?.isSuperadmin" type="success">超级管理员</el-tag>
        <el-tag v-else-if="auth.me" type="info">站点管理员</el-tag>
        <span class="muted">{{ auth.sites.length }} 个可管理站点</span>
        <el-button text circle @click="toggleDark">
          <el-icon><Sunny v-if="dark" /><Moon v-else /></el-icon>
        </el-button>
        <el-dropdown>
          <span class="user">
            <el-icon><User /></el-icon>
            {{ auth.me?.displayName || '未登录' }}
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item :icon="Setting" @click="router.push('/account')">
                账号设置
              </el-dropdown-item>
              <el-dropdown-item :icon="SwitchButton" @click="onLogout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </el-header>
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<style scoped>
.layout {
  height: 100%;
}
.aside {
  border-right: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  transition: width 0.2s ease;
  overflow: hidden;
}
.brand {
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 18px;
  font-weight: 700;
  font-size: 15px;
  color: var(--el-color-primary);
  white-space: nowrap;
}
.brand.collapsed {
  justify-content: center;
  padding: 0;
}
.header {
  display: flex;
  align-items: center;
  gap: 12px;
  border-bottom: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.spacer {
  flex: 1;
}
.user {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  font-size: 14px;
}
.nav-badge {
  min-width: 18px;
  margin-left: 6px;
  padding: 0 5px;
  border-radius: 9px;
  background: var(--el-color-danger);
  color: #fff;
  font-size: 11px;
  line-height: 18px;
  text-align: center;
}
.main {
  background: var(--el-bg-color-page);
  padding: 0;
}
</style>
