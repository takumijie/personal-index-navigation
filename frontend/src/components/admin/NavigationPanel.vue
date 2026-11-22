<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ArrowUpRight, Pencil, Plus, Save, Trash2, X } from 'lucide-vue-next'
import { adminApi } from '@/api'
import { errorMessage } from '@/api/client'
import type { NavigationItem } from '@/types'
import { validNavigationUrl, validWebUrl } from '@/utils/validation'
import AlertMessage from '@/components/common/AlertMessage.vue'
import AppIcon from '@/components/common/AppIcon.vue'
import FormField from '@/components/common/FormField.vue'
import ImageUploadField from '@/components/common/ImageUploadField.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import PanelShell from '@/components/common/PanelShell.vue'
import ToggleSwitch from '@/components/common/ToggleSwitch.vue'

type NavigationForm = Omit<NavigationItem, 'id'>
const blank = (): NavigationForm => ({ title: '', url: '', icon: 'Link2', icon_url: '', sort: 0, enabled: true, open_mode: 'blank' })
const items = ref<NavigationItem[]>([])
const form = ref<NavigationForm>(blank())
const editingId = ref<number | null>(null)
const formOpen = ref(false)
const loading = ref(true)
const saving = ref(false)
const deletingId = ref<number | null>(null)
const error = ref('')
const success = ref('')
const fieldErrors = ref<Record<string, string>>({})

async function load() {
  loading.value = true
  error.value = ''
  try { items.value = (await adminApi.navigation()).sort((a, b) => a.sort - b.sort) } catch (reason) { error.value = errorMessage(reason) } finally { loading.value = false }
}
onMounted(load)

function create() {
  editingId.value = null
  form.value = { ...blank(), sort: items.value.length ? Math.max(...items.value.map((item) => item.sort)) + 1 : 1 }
  fieldErrors.value = {}
  formOpen.value = true
}
function edit(item: NavigationItem) {
  editingId.value = item.id
  const { id: _id, ...data } = item
  form.value = structuredClone(data)
  fieldErrors.value = {}
  formOpen.value = true
}
function validate() {
  const errors: Record<string, string> = {}
  if (!form.value.title.trim()) errors.title = '请输入导航名称'
  if (!validNavigationUrl(form.value.url)) errors.url = '请输入有效的 http(s) 或 mailto 地址'
  if (!validWebUrl(form.value.icon_url, true)) errors.icon_url = '图标地址格式不正确'
  if (form.value.sort < 0) errors.sort = '排序值不能小于 0'
  fieldErrors.value = errors
  return !Object.keys(errors).length
}
async function save() {
  error.value = ''
  success.value = ''
  if (!validate()) return
  saving.value = true
  try {
    const payload = { ...form.value, title: form.value.title.trim(), url: form.value.url.trim() }
    if (editingId.value === null) {
      const created = await adminApi.createNavigation(payload)
      items.value.push(created)
    } else {
      const updated = await adminApi.updateNavigation({ ...payload, id: editingId.value })
      items.value = items.value.map((item) => item.id === updated.id ? updated : item)
    }
    items.value.sort((a, b) => a.sort - b.sort)
    formOpen.value = false
    success.value = editingId.value === null ? '导航已创建' : '导航已更新'
  } catch (reason) { error.value = errorMessage(reason) } finally { saving.value = false }
}
async function remove(item: NavigationItem) {
  if (!window.confirm(`确定删除导航“${item.title}”吗？`)) return
  deletingId.value = item.id
  error.value = ''
  try {
    await adminApi.deleteNavigation(item.id)
    items.value = items.value.filter((entry) => entry.id !== item.id)
    if (editingId.value === item.id) formOpen.value = false
    success.value = '导航已删除'
  } catch (reason) { error.value = errorMessage(reason) } finally { deletingId.value = null }
}
</script>

<template>
  <PanelShell title="导航管理" description="配置主页动态导航卡片、图标、排序及打开方式。">
    <template #actions><button class="primary-button" type="button" @click="create"><Plus :size="15" />新增导航</button></template>
    <AlertMessage :message="error" type="error" /><AlertMessage :message="success" type="success" />

    <div v-if="formOpen" class="mb-6 rounded-2xl border border-violet-200/10 bg-violet-200/[0.025] p-4 min-[640px]:p-5">
      <div class="mb-5 flex items-center justify-between"><h3 class="text-sm font-medium text-white/80">{{ editingId === null ? '新增导航' : '编辑导航' }}</h3><button class="icon-button size-7" type="button" aria-label="关闭表单" @click="formOpen = false"><X :size="14" /></button></div>
      <form class="grid gap-4 min-[760px]:grid-cols-2" @submit.prevent="save">
        <FormField label="导航名称" :error="fieldErrors.title"><input v-model="form.title" class="field" maxlength="50" placeholder="我的博客" /></FormField>
        <FormField label="链接地址" :error="fieldErrors.url"><input v-model="form.url" class="field" placeholder="https://example.com" /></FormField>
        <FormField label="Lucide 图标名" hint="如 Github"><input v-model="form.icon" class="field" placeholder="Link2" /></FormField>
        <FormField label="排序" :error="fieldErrors.sort"><input v-model.number="form.sort" class="field" type="number" min="0" /></FormField>
        <div class="min-[760px]:col-span-2"><ImageUploadField v-model="form.icon_url" label="自定义图标（可选）" /><p v-if="fieldErrors.icon_url" class="mt-1 text-xs text-rose-300">{{ fieldErrors.icon_url }}</p></div>
        <FormField label="打开方式"><select v-model="form.open_mode" class="field"><option value="blank">新标签页打开</option><option value="self">当前页面打开</option></select></FormField>
        <div class="flex items-end pb-1"><ToggleSwitch v-model="form.enabled" label="在主页显示" class="w-full" /></div>
        <div class="flex justify-end gap-2 min-[760px]:col-span-2"><button class="secondary-button" type="button" @click="formOpen = false">取消</button><button class="primary-button" type="submit" :disabled="saving"><Save :size="14" />{{ saving ? '保存中…' : '保存导航' }}</button></div>
      </form>
    </div>

    <LoadingSpinner v-if="loading" label="正在加载导航…" class="py-10 text-white/40" />
    <div v-else-if="items.length" class="space-y-2">
      <article v-for="item in items" :key="item.id" class="flex items-center gap-3 rounded-xl border border-white/[0.065] bg-white/[0.025] p-3.5">
        <span class="flex size-10 shrink-0 items-center justify-center rounded-xl bg-white/[0.05] text-violet-200/60"><AppIcon :name="item.icon" :size="17" /></span>
        <div class="min-w-0 flex-1"><div class="flex items-center gap-2"><h3 class="truncate text-sm font-medium text-white/75">{{ item.title }}</h3><span v-if="!item.enabled" class="rounded bg-white/5 px-1.5 py-0.5 text-[9px] text-white/30">已隐藏</span></div><a :href="item.url" target="_blank" rel="noopener" class="mt-1 flex items-center gap-1 truncate text-[11px] text-white/30 hover:text-white/55"><span class="truncate">{{ item.url }}</span><ArrowUpRight :size="10" /></a></div>
        <span class="hidden w-12 text-center font-mono text-[10px] text-white/25 min-[640px]:block">#{{ item.sort }}</span>
        <button class="icon-button size-8" type="button" title="编辑" @click="edit(item)"><Pencil :size="14" /></button>
        <button class="icon-button size-8 hover:bg-rose-400/10 hover:text-rose-200" type="button" title="删除" :disabled="deletingId === item.id" @click="remove(item)"><LoadingSpinner v-if="deletingId === item.id" label="" /><Trash2 v-else :size="14" /></button>
      </article>
    </div>
    <p v-else class="py-12 text-center text-sm text-white/30">还没有导航，点击“新增导航”开始配置。</p>
  </PanelShell>
</template>
