<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Pencil, Plus, Save, Trash2, X } from 'lucide-vue-next'
import { adminApi } from '@/api'
import { errorMessage } from '@/api/client'
import type { Device } from '@/types'
import AlertMessage from '@/components/common/AlertMessage.vue'
import AppIcon from '@/components/common/AppIcon.vue'
import FormField from '@/components/common/FormField.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import PanelShell from '@/components/common/PanelShell.vue'
import ToggleSwitch from '@/components/common/ToggleSwitch.vue'

type DeviceForm = Omit<Device, 'id'>
const blank = (): DeviceForm => ({ name: '', description: '', icon: 'Laptop', category: '', sort: 0, enabled: true })
const items = ref<Device[]>([])
const form = ref<DeviceForm>(blank())
const editingId = ref<number | null>(null)
const formOpen = ref(false)
const loading = ref(true)
const saving = ref(false)
const deletingId = ref<number | null>(null)
const error = ref('')
const success = ref('')
const fieldErrors = ref<Record<string, string>>({})

onMounted(async () => {
  try { items.value = (await adminApi.devices()).sort((a, b) => a.sort - b.sort) } catch (reason) { error.value = errorMessage(reason) } finally { loading.value = false }
})
function create() {
  editingId.value = null
  form.value = { ...blank(), sort: items.value.length ? Math.max(...items.value.map((item) => item.sort)) + 1 : 1 }
  fieldErrors.value = {}
  formOpen.value = true
}
function edit(item: Device) {
  editingId.value = item.id
  const { id: _id, ...data } = item
  form.value = structuredClone(data)
  fieldErrors.value = {}
  formOpen.value = true
}
async function save() {
  error.value = ''
  success.value = ''
  const errors: Record<string, string> = {}
  if (!form.value.name.trim()) errors.name = '请输入设备名称'
  if (!form.value.description.trim()) errors.description = '请输入设备描述'
  if (!form.value.category.trim()) errors.category = '请输入设备分类'
  if (!Number.isFinite(form.value.sort) || form.value.sort < 0 || form.value.sort > 1000000) errors.sort = '排序值应为 0 至 1000000'
  fieldErrors.value = errors
  if (Object.keys(errors).length) return
  saving.value = true
  try {
    const payload = { ...form.value, name: form.value.name.trim(), description: form.value.description.trim(), category: form.value.category.trim() }
    if (editingId.value === null) {
      items.value.push(await adminApi.createDevice(payload))
    } else {
      const updated = await adminApi.updateDevice({ ...payload, id: editingId.value })
      items.value = items.value.map((item) => item.id === updated.id ? updated : item)
    }
    items.value.sort((a, b) => a.sort - b.sort)
    formOpen.value = false
    success.value = editingId.value === null ? '设备已创建' : '设备已更新'
  } catch (reason) { error.value = errorMessage(reason) } finally { saving.value = false }
}
async function remove(item: Device) {
  if (!window.confirm(`确定删除设备“${item.name}”吗？`)) return
  deletingId.value = item.id
  error.value = ''
  try {
    await adminApi.deleteDevice(item.id)
    items.value = items.value.filter((entry) => entry.id !== item.id)
    if (editingId.value === item.id) formOpen.value = false
    success.value = '设备已删除'
  } catch (reason) { error.value = errorMessage(reason) } finally { deletingId.value = null }
}
</script>

<template>
  <PanelShell title="设备管理" description="维护主页设备网格中的名称、分类、描述与图标。">
    <template #actions><button class="primary-button" type="button" @click="create"><Plus :size="15" />新增设备</button></template>
    <AlertMessage :message="error" type="error" /><AlertMessage :message="success" type="success" />
    <div v-if="formOpen" class="mb-6 rounded-2xl border border-violet-200/10 bg-violet-200/[0.025] p-4 min-[640px]:p-5">
      <div class="mb-5 flex items-center justify-between"><h3 class="text-sm font-medium text-white/80">{{ editingId === null ? '新增设备' : '编辑设备' }}</h3><button class="icon-button size-7" type="button" aria-label="关闭表单" @click="formOpen = false"><X :size="14" /></button></div>
      <form class="grid gap-4 min-[760px]:grid-cols-2" @submit.prevent="save">
        <FormField label="设备名称" :error="fieldErrors.name"><input v-model="form.name" class="field" maxlength="100" placeholder="MacBook Pro" /></FormField>
        <FormField label="设备分类" :error="fieldErrors.category"><input v-model="form.category" class="field" maxlength="50" placeholder="日常主力" /></FormField>
        <div class="min-[760px]:col-span-2"><FormField label="设备描述" :error="fieldErrors.description"><input v-model="form.description" class="field" maxlength="500" placeholder="配置、用途或一句描述" /></FormField></div>
        <FormField label="Lucide 图标名" hint="如 Laptop"><input v-model="form.icon" class="field" maxlength="100" placeholder="Laptop" /></FormField>
        <FormField label="排序" :error="fieldErrors.sort"><input v-model.number="form.sort" class="field" type="number" min="0" max="1000000" /></FormField>
        <div class="min-[760px]:col-span-2"><ToggleSwitch v-model="form.enabled" label="在主页显示" /></div>
        <div class="flex justify-end gap-2 min-[760px]:col-span-2"><button class="secondary-button" type="button" @click="formOpen = false">取消</button><button class="primary-button" type="submit" :disabled="saving"><Save :size="14" />{{ saving ? '保存中…' : '保存设备' }}</button></div>
      </form>
    </div>
    <LoadingSpinner v-if="loading" label="正在加载设备…" class="py-10 text-white/40" />
    <div v-else-if="items.length" class="grid gap-2 min-[760px]:grid-cols-2">
      <article v-for="item in items" :key="item.id" class="flex items-center gap-3 rounded-xl border border-white/[0.065] bg-white/[0.025] p-3.5">
        <span class="flex size-10 shrink-0 items-center justify-center rounded-xl bg-white/[0.05] text-indigo-200/60"><AppIcon :name="item.icon" :size="17" /></span>
        <div class="min-w-0 flex-1"><p class="text-[9px] uppercase tracking-wider text-white/25">{{ item.category }} · #{{ item.sort }}</p><h3 class="mt-0.5 truncate text-sm text-white/75">{{ item.name }}<span v-if="!item.enabled" class="ml-2 text-[9px] text-white/25">已隐藏</span></h3><p class="mt-1 truncate text-[11px] text-white/30">{{ item.description }}</p></div>
        <div class="flex shrink-0 flex-col min-[420px]:flex-row"><button class="icon-button size-8" type="button" title="编辑" @click="edit(item)"><Pencil :size="14" /></button><button class="icon-button size-8 hover:text-rose-200" type="button" title="删除" :disabled="deletingId === item.id" @click="remove(item)"><Trash2 :size="14" /></button></div>
      </article>
    </div>
    <p v-else class="py-12 text-center text-sm text-white/30">还没有设备，点击“新增设备”开始配置。</p>
  </PanelShell>
</template>
