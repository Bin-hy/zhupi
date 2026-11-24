<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import type { CreateSiteInput, Site, UpdateSiteInput } from '../api/types'

const props = defineProps<{ modelValue: boolean; site?: Site | null }>()
const emit = defineEmits<{ 'update:modelValue': [boolean]; saved: [] }>()

const form = reactive({
  name: '',
  displayName: '',
  origins: [''],
  moderationMode: 'post' as 'post' | 'pre',
  themeColor: '#087f5b',
  defaultLocale: 'zh-CN',
  status: 'active' as 'active' | 'disabled',
})
const saving = ref(false)

const editing = computed(() => !!props.site)

function reset(): void {
  const site = props.site
  form.name = site?.name || ''
  form.displayName = site?.displayName || ''
  form.origins = site?.origins?.length ? [...site.origins] : ['']
  form.moderationMode = site?.moderationMode || 'post'
  form.themeColor = site?.themeColor || '#087f5b'
  form.defaultLocale = site?.defaultLocale || 'zh-CN'
  form.status = site?.status || 'active'
}

watch(
  () => [props.modelValue, props.site],
  () => {
    if (props.modelValue) reset()
  },
  { immediate: true },
)

function addOrigin(): void {
  form.origins.push('')
}

function removeOrigin(index: number): void {
  form.origins.splice(index, 1)
  if (form.origins.length === 0) form.origins.push('')
}

function close(): void {
  emit('update:modelValue', false)
}

async function submit(): Promise<void> {
  const origins = form.origins.map((o) => o.trim()).filter(Boolean)
  if (!editing.value && !form.name.trim()) {
    ElMessage.warning('请输入站点名称')
    return
  }
  if (origins.length === 0) {
    ElMessage.warning('至少填写一个允许来源')
    return
  }
  for (const origin of origins) {
    if (!/^https?:\/\/[^/\s]+$/i.test(origin)) {
      ElMessage.warning(`来源格式不正确：${origin}`)
      return
    }
  }

  saving.value = true
  try {
    if (props.site) {
      const input: UpdateSiteInput = {
        displayName: form.displayName.trim(),
        origins,
        moderationMode: form.moderationMode,
        themeColor: form.themeColor,
        defaultLocale: form.defaultLocale.trim() || 'zh-CN',
        status: form.status,
      }
      await api.updateSite(props.site.id, input)
    } else {
      const input: CreateSiteInput = {
        name: form.name.trim(),
        displayName: form.displayName.trim(),
        origins,
        moderationMode: form.moderationMode,
        themeColor: form.themeColor,
        defaultLocale: form.defaultLocale.trim() || 'zh-CN',
      }
      await api.createSite(input)
    }
    ElMessage.success(props.site ? '站点已更新' : '站点已创建')
    emit('saved')
    close()
  } catch (err: any) {
    ElMessage.error(err?.message || '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    :title="editing ? '编辑站点' : '创建站点'"
    width="640px"
    @update:model-value="close"
  >
    <el-form label-width="96px">
      <el-form-item label="内部名称" required>
        <el-input v-model="form.name" :disabled="editing" maxlength="64" placeholder="唯一名称" />
      </el-form-item>
      <el-form-item label="展示名">
        <el-input v-model="form.displayName" maxlength="100" placeholder="默认使用内部名称" />
      </el-form-item>
      <el-form-item label="允许来源" required>
        <div class="origins">
          <div v-for="(origin, index) in form.origins" :key="index" class="origin-row">
            <el-input v-model="form.origins[index]" placeholder="https://blog.example.com" />
            <el-button text type="danger" @click="removeOrigin(index)">删除</el-button>
          </div>
          <el-button text type="primary" @click="addOrigin">添加来源</el-button>
        </div>
      </el-form-item>
      <el-form-item label="审核模式">
        <el-radio-group v-model="form.moderationMode">
          <el-radio value="post">先发后审</el-radio>
          <el-radio value="pre">先审后发</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="主题色">
        <el-color-picker v-model="form.themeColor" />
      </el-form-item>
      <el-form-item label="默认语言">
        <el-input v-model="form.defaultLocale" placeholder="zh-CN" />
      </el-form-item>
      <el-form-item v-if="editing" label="状态">
        <el-radio-group v-model="form.status">
          <el-radio value="active">启用</el-radio>
          <el-radio value="disabled">停用</el-radio>
        </el-radio-group>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>


<style scoped>
.origins {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.origin-row {
  display: flex;
  gap: 8px;
}
</style>
