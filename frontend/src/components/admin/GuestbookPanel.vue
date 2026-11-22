<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Check, MessageSquareText, RefreshCw, Trash2, X } from 'lucide-vue-next'
import { adminApi } from '@/api'
import { errorMessage } from '@/api/client'
import type { GuestbookEntry } from '@/types'
import AlertMessage from '@/components/common/AlertMessage.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import PanelShell from '@/components/common/PanelShell.vue'

const entries = ref<GuestbookEntry[]>([])
const filter = ref<'all' | GuestbookEntry['status']>('all')
const loading = ref(true)
const updatingId = ref<number | null>(null)
const error = ref('')
const success = ref('')
const filteredEntries = computed(() => filter.value === 'all' ? entries.value : entries.value.filter((entry) => entry.status === filter.value))
const counts = computed(() => ({
  all: entries.value.length,
  pending: entries.value.filter((entry) => entry.status === 'pending').length,
  published: entries.value.filter((entry) => entry.status === 'published').length,
  rejected: entries.value.filter((entry) => entry.status === 'rejected').length,
}))
const labels = { all: '全部', pending: '待审核', published: '已发布', rejected: '已拒绝' }

async function load() {
  loading.value = true
  error.value = ''
  try { entries.value = await adminApi.guestbook() } catch (reason) { error.value = errorMessage(reason) } finally { loading.value = false }
}
onMounted(load)

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}
function statusText(status: GuestbookEntry['status']) {
  return { pending: '待审核', published: '已发布', rejected: '已拒绝' }[status]
}
async function setStatus(entry: GuestbookEntry, status: GuestbookEntry['status']) {
  if (entry.status === status) return
  updatingId.value = entry.id
  error.value = ''
  success.value = ''
  try {
    const updated = await adminApi.setGuestbookStatus(entry.id, status)
    entries.value = entries.value.map((item) => item.id === entry.id ? updated : item)
    success.value = `留言 #${entry.id} 已${status === 'published' ? '发布' : status === 'rejected' ? '拒绝' : '设为待审核'}`
  } catch (reason) { error.value = errorMessage(reason) } finally { updatingId.value = null }
}
async function remove(entry: GuestbookEntry) {
  if (!window.confirm(`确定永久删除留言 #${entry.id} 吗？此操作无法撤销。`)) return
  updatingId.value = entry.id
  error.value = ''
  try {
    await adminApi.deleteGuestbook(entry.id)
    entries.value = entries.value.filter((item) => item.id !== entry.id)
    success.value = '留言已删除'
  } catch (reason) { error.value = errorMessage(reason) } finally { updatingId.value = null }
}
</script>

<template>
  <PanelShell title="留言审核" description="审核终端提交的访客留言，控制公开显示或永久删除。">
    <template #actions><button class="secondary-button" type="button" :disabled="loading" @click="load"><RefreshCw :size="14" :class="loading && 'animate-spin'" />刷新</button></template>
    <div class="mb-5 flex gap-1.5 overflow-x-auto rounded-xl bg-white/[0.025] p-1">
      <button v-for="key in (['all', 'pending', 'published', 'rejected'] as const)" :key="key" class="shrink-0 rounded-lg px-3 py-2 text-xs transition" :class="filter === key ? 'bg-violet-300/10 text-violet-100' : 'text-white/35 hover:text-white/65'" type="button" @click="filter = key">{{ labels[key] }} <span class="ml-1 font-mono text-[9px] opacity-55">{{ counts[key] }}</span></button>
    </div>
    <AlertMessage :message="error" type="error" /><AlertMessage :message="success" type="success" />
    <LoadingSpinner v-if="loading" label="正在加载留言…" class="py-10 text-white/40" />
    <div v-else-if="filteredEntries.length" class="space-y-3">
      <article v-for="entry in filteredEntries" :key="entry.id" class="rounded-2xl border border-white/[0.07] bg-white/[0.025] p-4 min-[640px]:p-5" :class="updatingId === entry.id && 'opacity-60'">
        <div class="flex flex-col gap-4 min-[760px]:flex-row min-[760px]:items-start">
          <div class="min-w-0 flex-1">
            <div class="mb-2.5 flex flex-wrap items-center gap-2">
              <span class="rounded-md px-2 py-1 text-[9px]" :class="entry.status === 'published' ? 'bg-emerald-300/10 text-emerald-200/70' : entry.status === 'pending' ? 'bg-amber-300/10 text-amber-200/70' : 'bg-white/[0.055] text-white/35'">{{ statusText(entry.status) }}</span>
              <span class="font-mono text-[9px] text-white/22">#{{ entry.id }} · {{ formatDate(entry.created_at) }}</span>
            </div>
            <p class="whitespace-pre-wrap break-words text-sm leading-6 text-white/78">{{ entry.message }}</p>
            <div class="mt-3 flex flex-wrap gap-x-4 gap-y-1 font-mono text-[10px] text-white/28">
              <span>{{ entry.system_name || 'Unknown' }}<template v-if="entry.distribution"> / {{ entry.distribution }}</template></span>
              <span>{{ entry.browser_name }} {{ entry.browser_version }}</span><span>IP: {{ entry.ip_address || 'unknown' }}</span>
            </div>
          </div>
          <div class="flex shrink-0 items-center gap-1.5 border-t border-white/[0.06] pt-3 min-[760px]:border-t-0 min-[760px]:pt-0">
            <button v-if="entry.status !== 'published'" class="secondary-button px-3 py-2 text-[11px] text-emerald-200/75" type="button" :disabled="updatingId === entry.id" @click="setStatus(entry, 'published')"><Check :size="13" />发布</button>
            <button v-if="entry.status !== 'rejected'" class="secondary-button px-3 py-2 text-[11px]" type="button" :disabled="updatingId === entry.id" @click="setStatus(entry, 'rejected')"><X :size="13" />拒绝</button>
            <button v-if="entry.status === 'rejected'" class="secondary-button px-3 py-2 text-[11px]" type="button" :disabled="updatingId === entry.id" @click="setStatus(entry, 'pending')">待审</button>
            <button class="icon-button hover:bg-rose-400/10 hover:text-rose-200" type="button" title="永久删除" :disabled="updatingId === entry.id" @click="remove(entry)"><Trash2 :size="15" /></button>
          </div>
        </div>
      </article>
    </div>
    <div v-else class="flex flex-col items-center py-14 text-white/25"><MessageSquareText :size="28" class="mb-3" /><p class="text-sm">当前筛选下没有留言</p></div>
  </PanelShell>
</template>
