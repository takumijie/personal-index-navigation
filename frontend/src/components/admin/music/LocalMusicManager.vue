<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Link2Off, Music2, Pencil, Plus, Save, Trash2, Upload, X } from 'lucide-vue-next'
import { adminApi } from '@/api'
import { errorMessage } from '@/api/client'
import type { Playlist, Track } from '@/types'
import AlertMessage from '@/components/common/AlertMessage.vue'
import FormField from '@/components/common/FormField.vue'
import ImageUploadField from '@/components/common/ImageUploadField.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import ToggleSwitch from '@/components/common/ToggleSwitch.vue'

type TrackForm = Omit<Track, 'id' | 'playlist_ids'> & { playlist_ids: number[] }

const blank = (): TrackForm => ({
  title: '', artist: '', album: '', cover: '', url: '', source: 'local', external_id: '', object_key: '',
  lyric_path: '', sort: 0, enabled: true, playlist_ids: [],
})
const items = ref<Track[]>([])
const playlists = ref<Playlist[]>([])
const form = ref<TrackForm>(blank())
const editingId = ref<number | null>(null)
const formOpen = ref(false)
const loading = ref(true)
const saving = ref(false)
const uploadingAudio = ref(false)
const uploadingLyrics = ref(false)
const deletingId = ref<number | null>(null)
const audioInput = ref<HTMLInputElement | null>(null)
const lyricsInput = ref<HTMLInputElement | null>(null)
const error = ref('')
const success = ref('')
const fieldErrors = ref<Record<string, string>>({})

function sortTracks() {
  items.value.sort((a, b) => a.sort - b.sort || a.id - b.id)
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [trackData, playlistData] = await Promise.all([adminApi.tracks(), adminApi.playlists()])
    items.value = trackData.filter((track) => track.source === 'local' || track.source === 'remote')
    playlists.value = playlistData.sort((a, b) => a.sort - b.sort || a.id - b.id)
    sortTracks()
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    loading.value = false
  }
}
onMounted(load)

function create() {
  editingId.value = null
  form.value = { ...blank(), sort: items.value.length ? Math.max(...items.value.map((item) => item.sort)) + 1 : 1 }
  fieldErrors.value = {}
  formOpen.value = true
  error.value = ''
  success.value = ''
}

function edit(item: Track) {
  editingId.value = item.id
  const { id: _id, ...data } = item
  form.value = { ...structuredClone(data), playlist_ids: [...(item.playlist_ids ?? [])] }
  fieldErrors.value = {}
  formOpen.value = true
  error.value = ''
  success.value = ''
}

async function uploadAudio(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!/\.(mp3|flac|wav|ogg)$/i.test(file.name)) {
    error.value = '请选择 MP3、FLAC、WAV 或 OGG 音频文件'
    input.value = ''
    return
  }
  uploadingAudio.value = true
  error.value = ''
  success.value = ''
  try {
    const result = await adminApi.uploadMusic(file)
    form.value.url = result.url
    form.value.source = 'local'
    delete fieldErrors.value.url
    success.value = '音频已上传，保存曲目后生效'
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    uploadingAudio.value = false
    input.value = ''
  }
}

async function uploadLyrics(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!file.name.toLowerCase().endsWith('.lrc')) {
    error.value = '请选择 .lrc 歌词文件'
    input.value = ''
    return
  }
  uploadingLyrics.value = true
  error.value = ''
  success.value = ''
  try {
    const result = await adminApi.uploadLyrics(file)
    form.value.lyric_path = result.url
    success.value = '歌词已上传，保存曲目后生效'
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    uploadingLyrics.value = false
    input.value = ''
  }
}

function removeLyrics() {
  form.value.lyric_path = ''
  success.value = '已解除歌词关联，保存曲目后生效'
}

async function save() {
  error.value = ''
  success.value = ''
  const errors: Record<string, string> = {}
  if (!form.value.title.trim()) errors.title = '请输入曲目标题'
  if (!form.value.url.trim()) errors.url = form.value.source === 'local' ? '请先上传音频文件' : '请输入音频 URL'
  if (!Number.isFinite(form.value.sort) || form.value.sort < 0 || form.value.sort > 1000000) errors.sort = '排序值应为 0 至 1000000'
  fieldErrors.value = errors
  if (Object.keys(errors).length) return

  const id = editingId.value
  saving.value = true
  try {
    const payload: TrackForm = {
      ...form.value,
      title: form.value.title.trim(),
      artist: form.value.artist.trim(),
      album: form.value.album.trim(),
      cover: form.value.cover.trim(),
      url: form.value.url.trim(),
      lyric_path: form.value.lyric_path.trim(),
      source: id === null ? 'local' : form.value.source,
      playlist_ids: [...form.value.playlist_ids],
    }
    if (id === null) {
      const created = await adminApi.createTrack(payload)
      items.value.push({ ...created, playlist_ids: created.playlist_ids ?? [] })
    } else {
      const updated = await adminApi.updateTrack({ ...payload, id })
      items.value = items.value.map((item) => item.id === id ? { ...updated, playlist_ids: updated.playlist_ids ?? [] } : item)
    }
    sortTracks()
    formOpen.value = false
    success.value = id === null ? '本地曲目已创建' : '曲目已更新'
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    saving.value = false
  }
}

async function remove(item: Track) {
  if (!window.confirm(`确定删除曲目“${item.title}”吗？`)) return
  deletingId.value = item.id
  error.value = ''
  success.value = ''
  try {
    await adminApi.deleteTrack(item.id)
    items.value = items.value.filter((entry) => entry.id !== item.id)
    if (editingId.value === item.id) formOpen.value = false
    success.value = '曲目已删除'
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    deletingId.value = null
  }
}
</script>

<template>
  <div>
    <div class="mb-5 flex items-center justify-between gap-3">
      <p class="text-xs text-white/35">上传本地音频与 LRC 歌词；历史 Remote 曲目可继续编辑。</p>
      <button class="primary-button shrink-0" type="button" @click="create"><Plus :size="15" />新增本地曲目</button>
    </div>
    <AlertMessage :message="error" type="error" />
    <AlertMessage :message="success" type="success" />

    <div v-if="formOpen" class="mb-6 rounded-2xl border border-violet-200/10 bg-violet-200/[0.025] p-4 min-[640px]:p-5">
      <div class="mb-5 flex items-center justify-between gap-3">
        <div><h3 class="text-sm font-medium text-white/80">{{ editingId === null ? '新增本地曲目' : '编辑曲目' }}</h3><p class="mt-1 text-[10px] text-white/30">{{ form.source === 'remote' ? '历史 Remote 来源将保留，不会转为本地曲目' : '本地音频 URL 仅通过上传生成' }}</p></div>
        <button class="icon-button size-7 shrink-0" type="button" aria-label="关闭表单" @click="formOpen = false"><X :size="14" /></button>
      </div>
      <form class="grid gap-4 min-[760px]:grid-cols-2" @submit.prevent="save">
        <FormField label="曲目标题" :error="fieldErrors.title"><input v-model="form.title" class="field" maxlength="200" placeholder="歌曲名称" /></FormField>
        <FormField label="艺术家"><input v-model="form.artist" class="field" maxlength="200" placeholder="艺术家" /></FormField>
        <FormField label="专辑"><input v-model="form.album" class="field" maxlength="200" placeholder="专辑名称" /></FormField>
        <FormField label="排序" :error="fieldErrors.sort"><input v-model.number="form.sort" class="field" type="number" min="0" max="1000000" /></FormField>
        <div class="min-[760px]:col-span-2"><ImageUploadField v-model="form.cover" label="专辑封面（可选）" /></div>

        <FormField v-if="form.source === 'local'" label="音频文件" :hint="form.url ? '重新上传可替换' : '支持 MP3 / FLAC / WAV / OGG'" :error="fieldErrors.url">
          <input ref="audioInput" class="hidden" type="file" accept=".mp3,.flac,.wav,.ogg,audio/mpeg,audio/flac,audio/wav,audio/ogg" @change="uploadAudio" />
          <div class="flex items-center gap-2">
            <button class="secondary-button shrink-0" type="button" :disabled="uploadingAudio" @click="audioInput?.click()"><Upload :size="14" />{{ uploadingAudio ? '上传中…' : form.url ? '替换音频' : '上传音频' }}</button>
            <span v-if="form.url" class="min-w-0 flex-1 truncate text-[11px] text-emerald-200/60" :title="form.url">{{ form.url }}</span>
          </div>
        </FormField>
        <FormField v-else label="Remote 音频 URL" :error="fieldErrors.url" hint="仅历史曲目可编辑"><input v-model="form.url" class="field" type="url" placeholder="https://example.com/music.mp3" /></FormField>

        <FormField label="LRC 歌词" :hint="form.lyric_path ? '可替换或解除关联' : 'UTF-8 文本'">
          <input ref="lyricsInput" class="hidden" type="file" accept=".lrc,text/plain" @change="uploadLyrics" />
          <div class="flex flex-wrap items-center gap-2">
            <button class="secondary-button" type="button" :disabled="uploadingLyrics" @click="lyricsInput?.click()"><Upload :size="14" />{{ uploadingLyrics ? '上传中…' : form.lyric_path ? '替换歌词' : '上传歌词' }}</button>
            <button v-if="form.lyric_path" class="secondary-button" type="button" @click="removeLyrics"><Link2Off :size="14" />解除关联</button>
            <span v-if="form.lyric_path" class="min-w-0 flex-1 truncate text-[11px] text-emerald-200/60" :title="form.lyric_path">{{ form.lyric_path }}</span>
          </div>
        </FormField>

        <FormField label="所属播放列表" hint="按住 Ctrl / Command 可多选" class="min-[760px]:col-span-2">
          <select v-model="form.playlist_ids" class="field min-h-28 py-2" multiple>
            <option v-for="playlist in playlists" :key="playlist.id" :value="playlist.id">{{ playlist.name }}{{ playlist.enabled ? '' : '（已停用）' }}</option>
          </select>
        </FormField>
        <div class="min-[760px]:col-span-2"><ToggleSwitch v-model="form.enabled" label="启用曲目" /></div>
        <div class="flex justify-end gap-2 min-[760px]:col-span-2">
          <button class="secondary-button" type="button" @click="formOpen = false">取消</button>
          <button class="primary-button" type="submit" :disabled="saving || uploadingAudio || uploadingLyrics"><Save :size="14" />{{ saving ? '保存中…' : '保存曲目' }}</button>
        </div>
      </form>
    </div>

    <LoadingSpinner v-if="loading" label="正在加载本地音乐…" class="py-10 text-white/40" />
    <div v-else-if="items.length" class="space-y-2">
      <article v-for="item in items" :key="item.id" class="flex items-center gap-3 rounded-xl border border-white/[0.065] bg-white/[0.025] p-3.5">
        <span class="flex size-11 shrink-0 items-center justify-center overflow-hidden rounded-xl bg-white/[0.05] text-violet-200/60">
          <img v-if="item.cover" :src="item.cover" alt="" class="size-full object-cover" /><Music2 v-else :size="18" />
        </span>
        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-2">
            <h3 class="truncate text-sm font-medium text-white/75">{{ item.title }}</h3>
            <span class="rounded bg-white/5 px-1.5 py-0.5 text-[9px] text-white/35">{{ item.source === 'local' ? '本地' : '历史 Remote' }}</span>
            <span class="rounded px-1.5 py-0.5 text-[9px]" :class="item.lyric_path ? 'bg-emerald-300/[0.07] text-emerald-200/55' : 'bg-white/5 text-white/25'">{{ item.lyric_path ? '有歌词' : '无歌词' }}</span>
            <span v-if="!item.enabled" class="rounded bg-white/5 px-1.5 py-0.5 text-[9px] text-white/30">已停用</span>
          </div>
          <p class="mt-1 truncate text-[11px] text-white/30">{{ item.artist || '未知艺术家' }}<span v-if="item.album"> · {{ item.album }}</span> · {{ item.playlist_ids?.length ?? 0 }} 个列表</p>
        </div>
        <span class="hidden w-12 text-center font-mono text-[10px] text-white/25 min-[640px]:block">#{{ item.sort }}</span>
        <button class="icon-button size-8 shrink-0" type="button" title="编辑" @click="edit(item)"><Pencil :size="14" /></button>
        <button class="icon-button size-8 shrink-0 hover:bg-rose-400/10 hover:text-rose-200" type="button" title="删除" :disabled="deletingId === item.id" @click="remove(item)"><LoadingSpinner v-if="deletingId === item.id" label="" /><Trash2 v-else :size="14" /></button>
      </article>
    </div>
    <p v-else class="py-12 text-center text-sm text-white/30">还没有本地曲目，点击“新增本地曲目”上传音乐。</p>
  </div>
</template>
