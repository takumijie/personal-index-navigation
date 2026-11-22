<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ArrowDown, ArrowUp, ListMusic, Pencil, Plus, Save, Trash2, X } from 'lucide-vue-next'
import { adminApi, publicApi } from '@/api'
import { errorMessage } from '@/api/client'
import type { Playlist, Settings, Track } from '@/types'
import AlertMessage from '@/components/common/AlertMessage.vue'
import FormField from '@/components/common/FormField.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import ToggleSwitch from '@/components/common/ToggleSwitch.vue'

type PlaylistForm = Pick<Playlist, 'name' | 'sort' | 'enabled'>

const blank = (): PlaylistForm => ({ name: '', sort: 0, enabled: true })
const playlists = ref<Playlist[]>([])
const tracks = ref<Track[]>([])
const settings = ref<Settings | null>(null)
const selectedPlaylistId = ref<number | null>(null)
const orderedTracks = ref<Track[]>([])
const form = ref<PlaylistForm>(blank())
const editingId = ref<number | null>(null)
const formOpen = ref(false)
const loading = ref(true)
const saving = ref(false)
const savingPlayback = ref(false)
const savingTracks = ref(false)
const deletingId = ref<number | null>(null)
const error = ref('')
const success = ref('')
const fieldErrors = ref<Record<string, string>>({})

const selectedPlaylist = computed(() => playlists.value.find((item) => item.id === selectedPlaylistId.value) ?? null)
const enabledPlaylists = computed(() => playlists.value.filter((item) => item.enabled))
const selectedTrackIds = computed(() => new Set(orderedTracks.value.map((track) => track.id)))
const hasTrackChanges = computed(() => {
  const savedIds = (selectedPlaylist.value?.tracks ?? []).map((track) => track.id).join(',')
  return orderedTracks.value.map((track) => track.id).join(',') !== savedIds
})

function sortPlaylists() {
  playlists.value.sort((a, b) => a.sort - b.sort || a.id - b.id)
}

function syncOrderedTracks() {
  orderedTracks.value = [...(selectedPlaylist.value?.tracks ?? [])]
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [playlistData, trackData, settingsData] = await Promise.all([adminApi.playlists(), adminApi.tracks(), publicApi.settings()])
    playlists.value = playlistData.map((item) => ({ ...item, tracks: item.tracks ?? [] }))
    tracks.value = trackData.sort((a, b) => a.sort - b.sort || a.id - b.id)
    settings.value = settingsData
    sortPlaylists()
    selectedPlaylistId.value = playlists.value[0]?.id ?? null
    syncOrderedTracks()
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function savePlaybackSettings() {
  if (!settings.value) return
  savingPlayback.value = true
  error.value = ''
  success.value = ''
  try {
    settings.value = await adminApi.updateSettings(settings.value)
    success.value = '前台播放设置已保存'
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    savingPlayback.value = false
  }
}

function selectPlaylist(id: number) {
  selectedPlaylistId.value = id
  syncOrderedTracks()
  error.value = ''
  success.value = ''
}

function create() {
  editingId.value = null
  form.value = { ...blank(), sort: playlists.value.length ? Math.max(...playlists.value.map((item) => item.sort)) + 1 : 1 }
  fieldErrors.value = {}
  formOpen.value = true
}

function edit(item: Playlist) {
  editingId.value = item.id
  form.value = { name: item.name, sort: item.sort, enabled: item.enabled }
  fieldErrors.value = {}
  formOpen.value = true
}

async function save() {
  error.value = ''
  success.value = ''
  const errors: Record<string, string> = {}
  if (!form.value.name.trim()) errors.name = '请输入播放列表名称'
  if (!Number.isFinite(form.value.sort) || form.value.sort < 0 || form.value.sort > 1000000) errors.sort = '排序值应为 0 至 1000000'
  fieldErrors.value = errors
  if (Object.keys(errors).length) return

  const id = editingId.value
  saving.value = true
  try {
    const payload = { ...form.value, name: form.value.name.trim() }
    if (id === null) {
      const created = await adminApi.createPlaylist(payload)
      playlists.value.push({ ...created, tracks: created.tracks ?? [] })
      selectedPlaylistId.value = created.id
      orderedTracks.value = []
    } else {
      const updated = await adminApi.updatePlaylist({ ...payload, id })
      playlists.value = playlists.value.map((item) => item.id === id ? { ...updated, tracks: updated.tracks ?? item.tracks } : item)
    }
    sortPlaylists()
    formOpen.value = false
    success.value = id === null ? '播放列表已创建' : '播放列表已更新'
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    saving.value = false
  }
}

async function remove(item: Playlist) {
  if (!window.confirm(`确定删除播放列表“${item.name}”吗？其中的曲目不会被删除。`)) return
  deletingId.value = item.id
  error.value = ''
  success.value = ''
  try {
    await adminApi.deletePlaylist(item.id)
    playlists.value = playlists.value.filter((entry) => entry.id !== item.id)
    if (selectedPlaylistId.value === item.id) {
      selectedPlaylistId.value = playlists.value[0]?.id ?? null
      syncOrderedTracks()
    }
    if (editingId.value === item.id) formOpen.value = false
    success.value = '播放列表已删除'
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    deletingId.value = null
  }
}

function toggleTrack(track: Track, event: Event) {
  const checked = (event.target as HTMLInputElement).checked
  if (checked && !selectedTrackIds.value.has(track.id)) orderedTracks.value.push(track)
  if (!checked) orderedTracks.value = orderedTracks.value.filter((item) => item.id !== track.id)
  success.value = ''
}

function moveTrack(index: number, offset: number) {
  const target = index + offset
  if (target < 0 || target >= orderedTracks.value.length) return
  const next = [...orderedTracks.value]
  const [track] = next.splice(index, 1)
  next.splice(target, 0, track)
  orderedTracks.value = next
  success.value = ''
}

async function saveTracks() {
  const playlist = selectedPlaylist.value
  if (!playlist) return
  savingTracks.value = true
  error.value = ''
  success.value = ''
  try {
    const updated = await adminApi.setPlaylistTracks(playlist.id, orderedTracks.value.map((track) => track.id))
    const nextTracks = updated.tracks ?? []
    playlists.value = playlists.value.map((item) => item.id === playlist.id ? { ...updated, tracks: nextTracks } : item)
    orderedTracks.value = [...nextTracks]
    success.value = '曲目与顺序已保存'
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    savingTracks.value = false
  }
}
</script>

<template>
  <div>
    <div class="mb-5 flex items-center justify-between gap-3">
      <p class="text-xs text-white/35">创建播放列表，勾选曲目并调整播放顺序。</p>
      <button class="primary-button shrink-0" type="button" @click="create"><Plus :size="15" />新增列表</button>
    </div>
    <AlertMessage :message="error" type="error" />
    <AlertMessage :message="success" type="success" />

    <section v-if="settings" class="mb-6 rounded-2xl border border-white/[0.065] bg-white/[0.025] p-4">
      <div class="mb-4">
        <h3 class="text-sm font-medium text-white/80">前台播放</h3>
        <p class="mt-1 text-[10px] text-white/30">配置主页固定播放的列表，以及是否显示逐句歌词。</p>
      </div>
      <div class="grid items-end gap-4 min-[640px]:grid-cols-[1fr_1.4fr_auto]">
        <ToggleSwitch v-model="settings.music.lyrics_enabled" label="显示逐句歌词" />
        <label class="block text-xs text-white/45">
          <span class="mb-2 block">前台播放列表</span>
          <select v-model.number="settings.music.playlist_id" class="field">
            <option :value="0">自动选择首个启用列表</option>
            <option v-for="item in enabledPlaylists" :key="item.id" :value="item.id">{{ item.name }}</option>
          </select>
        </label>
        <button class="primary-button" type="button" :disabled="savingPlayback" @click="savePlaybackSettings"><Save :size="14" />{{ savingPlayback ? '保存中…' : '保存设置' }}</button>
      </div>
    </section>

    <div v-if="formOpen" class="mb-6 rounded-2xl border border-violet-200/10 bg-violet-200/[0.025] p-4 min-[640px]:p-5">
      <div class="mb-5 flex items-center justify-between">
        <h3 class="text-sm font-medium text-white/80">{{ editingId === null ? '新增播放列表' : '编辑播放列表' }}</h3>
        <button class="icon-button size-7" type="button" aria-label="关闭表单" @click="formOpen = false"><X :size="14" /></button>
      </div>
      <form class="grid gap-4 min-[640px]:grid-cols-2" @submit.prevent="save">
        <FormField label="列表名称" :error="fieldErrors.name"><input v-model="form.name" class="field" maxlength="100" placeholder="我的收藏" /></FormField>
        <FormField label="排序" :error="fieldErrors.sort"><input v-model.number="form.sort" class="field" type="number" min="0" max="1000000" /></FormField>
        <div class="min-[640px]:col-span-2"><ToggleSwitch v-model="form.enabled" label="启用播放列表" /></div>
        <div class="flex justify-end gap-2 min-[640px]:col-span-2">
          <button class="secondary-button" type="button" @click="formOpen = false">取消</button>
          <button class="primary-button" type="submit" :disabled="saving"><Save :size="14" />{{ saving ? '保存中…' : '保存列表' }}</button>
        </div>
      </form>
    </div>

    <LoadingSpinner v-if="loading" label="正在加载播放列表…" class="py-10 text-white/40" />
    <div v-else class="grid items-start gap-5 min-[1000px]:grid-cols-[minmax(240px,0.8fr)_minmax(360px,1.4fr)]">
      <section>
        <h3 class="mb-3 text-xs font-semibold tracking-wider text-violet-200/65">全部列表 · {{ playlists.length }}</h3>
        <div v-if="playlists.length" class="space-y-2">
          <article
            v-for="item in playlists"
            :key="item.id"
            class="flex items-center gap-2 rounded-xl border p-2.5 transition-colors"
            :class="selectedPlaylistId === item.id ? 'border-violet-300/20 bg-violet-300/[0.07]' : 'border-white/[0.065] bg-white/[0.025]'"
          >
            <button class="flex min-w-0 flex-1 items-center gap-3 text-left" type="button" :disabled="savingTracks" @click="selectPlaylist(item.id)">
              <span class="flex size-9 shrink-0 items-center justify-center rounded-lg bg-white/[0.05] text-violet-200/60"><ListMusic :size="16" /></span>
              <span class="min-w-0 flex-1">
                <span class="flex items-center gap-2"><span class="truncate text-sm text-white/75">{{ item.name }}</span><span v-if="!item.enabled" class="shrink-0 text-[9px] text-white/30">已停用</span></span>
                <span class="mt-0.5 block text-[10px] text-white/30">{{ item.tracks.length }} 首曲目 · 排序 {{ item.sort }}</span>
              </span>
            </button>
            <button class="icon-button size-8 shrink-0" type="button" title="编辑" @click="edit(item)"><Pencil :size="14" /></button>
            <button class="icon-button size-8 shrink-0 hover:bg-rose-400/10 hover:text-rose-200" type="button" title="删除" :disabled="deletingId === item.id" @click="remove(item)">
              <LoadingSpinner v-if="deletingId === item.id" label="" /><Trash2 v-else :size="14" />
            </button>
          </article>
        </div>
        <p v-else class="rounded-xl border border-dashed border-white/10 py-10 text-center text-xs text-white/30">还没有播放列表。</p>
      </section>

      <section v-if="selectedPlaylist" class="rounded-2xl border border-white/[0.065] bg-black/10 p-4">
        <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
          <div><h3 class="text-sm font-medium text-white/80">{{ selectedPlaylist.name }}</h3><p class="mt-1 text-[10px] text-white/30">已选 {{ orderedTracks.length }} 首，使用箭头调整顺序</p></div>
          <button class="primary-button" type="button" :disabled="savingTracks || !hasTrackChanges" @click="saveTracks"><Save :size="14" />{{ savingTracks ? '保存中…' : '保存曲目顺序' }}</button>
        </div>

        <div v-if="orderedTracks.length" class="mb-5 space-y-1.5">
          <article v-for="(track, index) in orderedTracks" :key="track.id" class="flex items-center gap-2 rounded-lg bg-white/[0.025] px-2.5 py-2">
            <span class="w-5 shrink-0 text-center font-mono text-[10px] text-white/25">{{ index + 1 }}</span>
            <div class="min-w-0 flex-1"><p class="truncate text-xs text-white/70">{{ track.title }}</p><p class="mt-0.5 truncate text-[10px] text-white/30">{{ track.artist || '未知艺术家' }}<span v-if="!track.enabled"> · 已停用</span></p></div>
            <button class="icon-button size-7" type="button" title="上移" aria-label="上移曲目" :disabled="index === 0 || savingTracks" @click="moveTrack(index, -1)"><ArrowUp :size="13" /></button>
            <button class="icon-button size-7" type="button" title="下移" aria-label="下移曲目" :disabled="index === orderedTracks.length - 1 || savingTracks" @click="moveTrack(index, 1)"><ArrowDown :size="13" /></button>
          </article>
        </div>
        <p v-else class="mb-5 rounded-lg border border-dashed border-white/10 py-6 text-center text-xs text-white/30">从下方勾选曲目加入此列表。</p>

        <h4 class="mb-2 text-xs font-medium text-white/55">加入或移出曲目</h4>
        <div v-if="tracks.length" class="max-h-72 space-y-1 overflow-y-auto pr-1">
          <label v-for="track in tracks" :key="track.id" class="flex cursor-pointer items-center gap-2.5 rounded-lg px-2 py-2 text-xs transition-colors hover:bg-white/[0.035]">
            <input class="size-3.5 shrink-0 accent-violet-500" type="checkbox" :checked="selectedTrackIds.has(track.id)" :disabled="savingTracks" @change="toggleTrack(track, $event)" />
            <span class="min-w-0 flex-1 truncate text-white/60">{{ track.title }}<span class="text-white/25"> · {{ track.artist || '未知艺术家' }}</span></span>
            <span class="shrink-0 text-[9px] uppercase text-white/25">{{ track.source }}</span>
          </label>
        </div>
        <p v-else class="py-5 text-center text-xs text-white/30">暂无可选曲目，请先添加音乐。</p>
      </section>
      <section v-else class="rounded-2xl border border-dashed border-white/10 py-16 text-center text-sm text-white/30">选择或新建一个播放列表以管理曲目。</section>
    </div>
  </div>
</template>
