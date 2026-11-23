<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ChevronLeft, ChevronRight, Download, ListPlus, LogIn, LogOut, Music2, QrCode, RefreshCw, Save, Search, X } from 'lucide-vue-next'
import { adminApi } from '@/api'
import { errorMessage } from '@/api/client'
import type { ImportResult, NeteaseConfig, NeteaseQrSession, NeteaseQrState, NeteaseSearchResult, NeteaseStatus, Playlist, Track } from '@/types'
import AlertMessage from '@/components/common/AlertMessage.vue'
import FormField from '@/components/common/FormField.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

const status = ref<NeteaseStatus>({ authenticated: false })
const statusLoading = ref(true)
const loggingOut = ref(false)
const qrSession = ref<NeteaseQrSession | null>(null)
const qrState = ref<NeteaseQrState['status']>('waiting')
const qrMessage = ref('')
const qrCreating = ref(false)
const error = ref('')
const success = ref('')
const config = ref<NeteaseConfig>({ base_url: '' })
const savedBaseURL = ref('')
const configLoading = ref(true)
const configSaving = ref(false)

const keywords = ref('')
const searchResult = ref<NeteaseSearchResult | null>(null)
const searching = ref(false)
const selectedTracks = ref<Track[]>([])
const createName = ref('')
const creating = ref(false)
const playlists = ref<Playlist[]>([])
const selectedPlaylistID = ref<number | null>(null)
const appending = ref(false)
const importInput = ref('')
const importName = ref('')
const importing = ref(false)

let pollTimer: ReturnType<typeof setTimeout> | null = null
let qrVersion = 0
let disposed = false

const configured = computed(() => savedBaseURL.value !== '')
const totalPages = computed(() => {
  if (!searchResult.value) return 1
  return Math.max(1, Math.ceil(searchResult.value.total / searchResult.value.page_size))
})
const currentPage = computed(() => searchResult.value?.page || 1)
const allCurrentSelected = computed(() => {
  const tracks = searchResult.value?.tracks || []
  return tracks.length > 0 && tracks.every((track) => isSelected(track))
})

function trackKey(track: Track) {
  return track.external_id || String(track.id)
}

function isSelected(track: Track) {
  return selectedTracks.value.some((item) => trackKey(item) === trackKey(track))
}

function toggleTrack(track: Track) {
  if (isSelected(track)) {
    selectedTracks.value = selectedTracks.value.filter((item) => trackKey(item) !== trackKey(track))
  } else {
    selectedTracks.value.push(track)
  }
}

function toggleAll() {
  const tracks = searchResult.value?.tracks || []
  if (allCurrentSelected.value) {
    const keys = new Set(tracks.map(trackKey))
    selectedTracks.value = selectedTracks.value.filter((track) => !keys.has(trackKey(track)))
    return
  }
  for (const track of tracks) {
    if (!isSelected(track)) selectedTracks.value.push(track)
  }
}

function stopPolling() {
  qrVersion += 1
  if (pollTimer !== null) clearTimeout(pollTimer)
  pollTimer = null
}

function closeQr() {
  stopPolling()
  qrSession.value = null
  qrState.value = 'waiting'
  qrMessage.value = ''
}

async function loadConfig() {
  configLoading.value = true
  error.value = ''
  try {
    const result = await adminApi.neteaseConfig()
    config.value = result
    savedBaseURL.value = result.base_url
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    configLoading.value = false
  }
}

async function saveConfig() {
  error.value = ''
  success.value = ''
  if (!config.value.base_url.trim()) {
    error.value = '请输入网易云 API 地址'
    return
  }
  configSaving.value = true
  try {
    const previousBaseURL = savedBaseURL.value
    const result = await adminApi.saveNeteaseConfig({ base_url: config.value.base_url.trim() })
    config.value = result
    savedBaseURL.value = result.base_url
    if (previousBaseURL !== result.base_url) {
      closeQr()
      status.value = { authenticated: false }
      searchResult.value = null
      selectedTracks.value = []
    }
    success.value = '网易云 API 地址已保存'
    await Promise.all([refreshStatus(true), loadPlaylists()])
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    configSaving.value = false
  }
}

async function loadPlaylists() {
  try {
    playlists.value = await adminApi.playlists()
    if (!playlists.value.some((item) => item.id === selectedPlaylistID.value)) {
      selectedPlaylistID.value = playlists.value[0]?.id || null
    }
  } catch (reason) {
    error.value = errorMessage(reason)
  }
}

async function refreshStatus(initial = false) {
  if (initial) statusLoading.value = true
  try {
    status.value = await adminApi.neteaseStatus()
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    if (initial) statusLoading.value = false
  }
}

async function createQr() {
  stopPolling()
  qrCreating.value = true
  error.value = ''
  success.value = ''
  const version = qrVersion
  try {
    const session = await adminApi.neteaseQrCreate()
    if (disposed || version !== qrVersion) return
    qrSession.value = session
    qrState.value = 'waiting'
    qrMessage.value = '请使用网易云音乐 App 扫描二维码'
    pollTimer = setTimeout(() => pollQr(session.session_id, version), 2000)
  } catch (reason) {
    if (!disposed && version === qrVersion) error.value = errorMessage(reason)
  } finally {
    if (version === qrVersion) qrCreating.value = false
  }
}

async function pollQr(sessionID: string, version: number) {
  if (disposed || version !== qrVersion || !qrSession.value) return
  try {
    const state = await adminApi.neteaseQrCheck(sessionID)
    if (disposed || version !== qrVersion) return
    qrState.value = state.status
    qrMessage.value = state.message
    if (state.status === 'success') {
      stopPolling()
      qrSession.value = null
      success.value = '网易云音乐登录成功'
      await refreshStatus()
      return
    }
    if (state.status === 'expired') {
      stopPolling()
      return
    }
    pollTimer = setTimeout(() => pollQr(sessionID, version), 2000)
  } catch (reason) {
    if (disposed || version !== qrVersion) return
    stopPolling()
    error.value = errorMessage(reason)
  }
}

async function logout() {
  loggingOut.value = true
  error.value = ''
  success.value = ''
  try {
    await adminApi.neteaseLogout()
    closeQr()
    status.value = { authenticated: false }
    searchResult.value = null
    selectedTracks.value = []
    success.value = '已退出网易云音乐'
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    loggingOut.value = false
  }
}

async function search(page = 1) {
  const query = keywords.value.trim()
  error.value = ''
  success.value = ''
  if (!query) {
    error.value = '请输入搜索关键词'
    return
  }
  searching.value = true
  try {
    searchResult.value = await adminApi.neteaseSearch(query, page)
    if (page === 1) selectedTracks.value = []
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    searching.value = false
  }
}

function importSummary(result: ImportResult) {
  return `播放列表“${result.playlist.name}”已保存：导入 ${result.imported} 首，重复 ${result.duplicates} 首`
}

async function appendPlaylist() {
  error.value = ''
  success.value = ''
  if (!selectedPlaylistID.value) {
    error.value = '请选择要追加到的已有列表'
    return
  }
  if (!selectedTracks.value.length) {
    error.value = '请至少选择一首歌曲'
    return
  }
  appending.value = true
  try {
    const result = await adminApi.neteaseAppendPlaylist(selectedPlaylistID.value, selectedTracks.value)
    success.value = `播放列表“${result.playlist.name}”已更新：新增 ${result.imported} 首，重复 ${result.duplicates} 首`
    selectedTracks.value = []
    await loadPlaylists()
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    appending.value = false
  }
}

async function createPlaylist() {
  error.value = ''
  success.value = ''
  if (!createName.value.trim()) {
    error.value = '请输入新播放列表名称'
    return
  }
  if (!selectedTracks.value.length) {
    error.value = '请至少选择一首歌曲'
    return
  }
  creating.value = true
  try {
    const result = await adminApi.neteaseCreatePlaylist(createName.value.trim(), selectedTracks.value)
    success.value = importSummary(result)
    selectedTracks.value = []
    createName.value = ''
    selectedPlaylistID.value = result.playlist.id
    await loadPlaylists()
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    creating.value = false
  }
}

async function importPlaylist() {
  error.value = ''
  success.value = ''
  if (!importInput.value.trim()) {
    error.value = '请输入网易云歌单 ID 或分享链接'
    return
  }
  importing.value = true
  try {
    const result = await adminApi.neteaseImportPlaylist(importInput.value.trim(), importName.value.trim())
    success.value = importSummary(result)
    importInput.value = ''
    importName.value = ''
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    importing.value = false
  }
}

onMounted(async () => {
  await loadConfig()
  if (configured.value) {
    await refreshStatus(true)
  } else {
    statusLoading.value = false
  }
  await loadPlaylists()
})
onBeforeUnmount(() => {
  disposed = true
  stopPolling()
})
</script>

<template>
  <div class="space-y-6">
    <AlertMessage :message="error" type="error" />
    <AlertMessage :message="success" type="success" />

    <section class="rounded-2xl border border-white/[0.065] bg-white/[0.025] p-4 min-[640px]:p-5">
      <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
        <h3 class="text-sm font-medium text-white/80">网易云 API 配置</h3>
        <a href="https://github.com/neteasecloudmusicapienhanced/api-enhanced" target="_blank" rel="noopener noreferrer" class="text-xs text-violet-300 transition-colors hover:text-violet-200">API 开源项目</a>
      </div>
      <form class="flex flex-col gap-2 min-[560px]:flex-row" @submit.prevent="saveConfig">
        <input v-model="config.base_url" class="field" type="url" placeholder="https://your-netease-api.example.com" :disabled="configLoading" />
        <button class="primary-button shrink-0" type="submit" :disabled="configSaving || configLoading"><Save :size="14" />{{ configSaving ? '保存中…' : '保存地址' }}</button>
      </form>
      <p class="mt-3 text-xs leading-5 text-white/35">请填写自行部署的 NeteaseCloudMusicApiEnhanced 服务地址，项目不内置公共或私人 API。</p>
      <p v-if="!configured && !configLoading" class="mt-2 text-xs text-amber-200/70">保存 API 地址后才能使用网易云登录、搜索和导入功能。</p>
    </section>

    <section v-if="configured" class="rounded-2xl border border-white/[0.065] bg-white/[0.025] p-4 min-[640px]:p-5">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div class="flex items-center gap-3">
          <img v-if="status.authenticated && status.avatar" :src="status.avatar" alt="" class="size-11 rounded-full object-cover" />
          <span v-else class="flex size-11 items-center justify-center rounded-full bg-white/[0.05] text-white/35"><Music2 :size="20" /></span>
          <div>
            <h3 class="text-sm font-medium text-white/80">网易云账号</h3>
            <p class="mt-1 text-xs text-white/35">
              <LoadingSpinner v-if="statusLoading" label="正在读取登录状态…" />
              <template v-else>{{ status.authenticated ? `已登录${status.nickname ? ` · ${status.nickname}` : ''}` : '尚未登录，请使用二维码授权' }}</template>
            </p>
          </div>
        </div>
        <button v-if="status.authenticated" class="secondary-button" type="button" :disabled="loggingOut" @click="logout">
          <LogOut :size="14" />{{ loggingOut ? '退出中…' : '退出登录' }}
        </button>
        <button v-else class="primary-button" type="button" :disabled="qrCreating || statusLoading" @click="createQr">
          <QrCode :size="15" />{{ qrCreating ? '生成中…' : qrSession ? '重新生成二维码' : '二维码登录' }}
        </button>
      </div>

      <div v-if="qrSession" class="mt-5 flex flex-col items-center gap-4 border-t border-white/[0.06] pt-5 min-[520px]:flex-row min-[520px]:items-start">
        <img :src="qrSession.qr_image" alt="网易云音乐登录二维码" class="size-44 shrink-0 rounded-xl bg-white p-2" />
        <div class="min-w-0 flex-1 text-center min-[520px]:text-left">
          <p class="text-sm" :class="qrState === 'expired' ? 'text-rose-200' : qrState === 'scanned' ? 'text-amber-200' : 'text-white/70'">{{ qrMessage }}</p>
          <p class="mt-2 text-xs text-white/30">有效期至 {{ new Date(qrSession.expires_at).toLocaleString() }}</p>
          <p v-if="qrState === 'waiting' || qrState === 'scanned'" class="mt-3 text-xs text-white/35"><LoadingSpinner label="每 2 秒自动检查登录状态" /></p>
          <button class="secondary-button mt-4" type="button" @click="closeQr"><X :size="14" />关闭二维码</button>
        </div>
      </div>
    </section>

    <template v-if="configured && status.authenticated">
      <section class="rounded-2xl border border-white/[0.065] bg-white/[0.025] p-4 min-[640px]:p-5">
        <h3 class="mb-4 flex items-center gap-2 text-sm font-medium text-white/80"><Search :size="16" />搜索歌曲</h3>
        <form class="flex flex-col gap-2 min-[560px]:flex-row" @submit.prevent="search(1)">
          <input v-model="keywords" class="field" placeholder="歌曲名、歌手或专辑" maxlength="200" />
          <button class="primary-button shrink-0" type="submit" :disabled="searching"><Search :size="14" />{{ searching ? '搜索中…' : '搜索' }}</button>
        </form>

        <div v-if="searchResult" class="mt-5">
          <div class="mb-3 flex flex-wrap items-center justify-between gap-2 text-xs text-white/35">
            <label class="flex cursor-pointer items-center gap-2">
              <input type="checkbox" class="accent-violet-500" :checked="allCurrentSelected" :disabled="!searchResult.tracks.length" @change="toggleAll" />
              选择本页
            </label>
            <span>共 {{ searchResult.total }} 首 · 已选 {{ selectedTracks.length }} 首</span>
          </div>
          <div v-if="searchResult.tracks.length" class="fine-scrollbar max-h-[26rem] space-y-2 overflow-y-auto pr-1">
            <label v-for="track in searchResult.tracks" :key="trackKey(track)" class="flex cursor-pointer items-center gap-3 rounded-xl border border-white/[0.055] bg-black/10 p-3 transition-colors hover:bg-white/[0.035]">
              <input type="checkbox" class="shrink-0 accent-violet-500" :checked="isSelected(track)" @change="toggleTrack(track)" />
              <img v-if="track.cover" :src="track.cover" alt="" class="size-10 shrink-0 rounded-lg object-cover" />
              <span v-else class="flex size-10 shrink-0 items-center justify-center rounded-lg bg-white/[0.04] text-white/25"><Music2 :size="16" /></span>
              <span class="min-w-0 flex-1">
                <span class="block truncate text-sm text-white/75">{{ track.title }}</span>
                <span class="mt-1 block truncate text-[11px] text-white/30">{{ track.artist }}<template v-if="track.album"> · {{ track.album }}</template></span>
              </span>
            </label>
          </div>
          <p v-else class="py-8 text-center text-sm text-white/30">没有找到匹配的歌曲</p>

          <div v-if="totalPages > 1" class="mt-4 flex items-center justify-center gap-3">
            <button class="icon-button" type="button" aria-label="上一页" :disabled="searching || currentPage <= 1" @click="search(currentPage - 1)"><ChevronLeft :size="16" /></button>
            <span class="text-xs text-white/40">{{ currentPage }} / {{ totalPages }}</span>
            <button class="icon-button" type="button" aria-label="下一页" :disabled="searching || currentPage >= totalPages" @click="search(currentPage + 1)"><ChevronRight :size="16" /></button>
          </div>

          <div class="mt-5 space-y-3 border-t border-white/[0.06] pt-4">
            <form class="flex flex-col gap-2 min-[560px]:flex-row" @submit.prevent="appendPlaylist">
              <select v-model.number="selectedPlaylistID" class="field" :disabled="!playlists.length">
                <option :value="null" disabled>{{ playlists.length ? '选择已有列表' : '暂无已有列表' }}</option>
                <option v-for="playlist in playlists" :key="playlist.id" :value="playlist.id">{{ playlist.name }}（{{ playlist.tracks.length }} 首）</option>
              </select>
              <button class="secondary-button shrink-0" type="submit" :disabled="appending || !selectedPlaylistID || !selectedTracks.length"><ListPlus :size="14" />{{ appending ? '追加中…' : `追加到已有列表（${selectedTracks.length}）` }}</button>
            </form>
            <form class="flex flex-col gap-2 min-[560px]:flex-row" @submit.prevent="createPlaylist">
              <input v-model="createName" class="field" placeholder="新播放列表名称" maxlength="100" />
              <button class="primary-button shrink-0" type="submit" :disabled="creating || !selectedTracks.length"><ListPlus :size="14" />{{ creating ? '创建中…' : `创建列表（${selectedTracks.length}）` }}</button>
            </form>
          </div>
        </div>
      </section>

      <section class="rounded-2xl border border-white/[0.065] bg-white/[0.025] p-4 min-[640px]:p-5">
        <h3 class="mb-4 flex items-center gap-2 text-sm font-medium text-white/80"><Download :size="16" />导入网易云歌单</h3>
        <form class="grid gap-4" @submit.prevent="importPlaylist">
          <FormField label="歌单 ID 或分享链接"><input v-model="importInput" class="field" placeholder="例如 123456789 或网易云歌单分享链接" /></FormField>
          <FormField label="本站播放列表名称" hint="可选"><input v-model="importName" class="field" placeholder="留空使用网易云歌单名称" maxlength="100" /></FormField>
          <div><button class="primary-button" type="submit" :disabled="importing"><LogIn :size="14" />{{ importing ? '导入中…' : '导入歌单' }}</button></div>
        </form>
      </section>
    </template>

    <button v-if="configured" class="secondary-button" type="button" :disabled="statusLoading" @click="refreshStatus(true)"><RefreshCw :size="14" />刷新登录状态</button>
  </div>
</template>
