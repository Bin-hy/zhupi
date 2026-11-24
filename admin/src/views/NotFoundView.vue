<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <div class="admin-page">
    <el-result icon="warning" title="页面不存在" sub-title="请检查地址是否正确">
      <template #extra>
        <el-button type="primary" @click="router.replace('/')">返回概览</el-button>
      </template>
    </el-result>
  </div>
</template>
