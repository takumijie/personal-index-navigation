<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { FileAudio, Music2, Pencil, Plus, Save, Trash2, Upload, X } from 'lucide-vue-next'
import { adminApi } from '@/api'
import { errorMessage } from '@/api/client'
import type { Track } from '@/types'
import { validWebUrl } from '@/utils/validation'
import AlertMessage from '@/components/common/AlertMessage.vue'
import FormField from '@/components/common/FormField.vue'
import ImageUploadField from '@/components/common/ImageUploadField.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import PanelShell from '@/components/common/PanelShell.vue'
import ToggleSwitch from '@/components/common/ToggleSwitch.vue'

type TrackForm = Omit<Track, 'id'>
const blank = (): TrackForm => ({ title: '', artist: '', album: '', cover: '', url: '', source: 'remote', sort: 0, enabled: true })
const tracks = ref<Track[]>([])
const form = ref<TrackForm>(blank())
const editingId = ref<number | null>(null)
const formOpen = ref(false)
const loading = ref(true)
const saving = ref(false)
const uploading = ref(false)
const deletingId = ref<number | null>(null)
const musicInput = ref<HTMLInputElement | null>(null)
const error = ref('')
const success = ref('')
const fieldErrors = ref<Record<string, string>>({})

onMounted(async () => {
  try { tracks.value = (await adminApi.tracks()).sort((a, b) => a.sort - b.sort) } catch (reason) { error.value = errorMessage(reason) } finally { loading.value = false }
})
function create() {
  editingId.value = null
  form.value = { ...blank(), sort: tracks.value.length ? Math.max(...tracks.value.map((track) => track.sort)) + 1 : 1 }
  fieldErrors.value = {}
  formOpen.value = true
}
function edit(track: Track) {
  editingId.value = track.id
  const { id: _id, ...data } = track
  form.value = structuredClone(data)
  fieldErrors.value = {}
  formOpen.value = true
}
async function uploadMusic(event: Event) {
  const inputElement = event.target as HTMLInputElement
  const file = inputElement.files?.[0]
  if (!file) return
  error.value = ''
  if (!file.type.startsWith('audio/') && !/\.(mp3|wav|ogg|flac)$/i.test(file.name)) {
    error.value = '请选择有效的音频文件'
    inputElement.value = ''
    return
  }
  if (file.size > 50 * 1024 * 1024) {
    error.value = '音频文件不能超过 50 MB'
    inputElement.value = ''
    return
  }
  uploading.value = true
  try {
    const result = await adminApi.uploadMusic(file)
    form.value.url = result.url
    form.value.source = 'local'
    if (!form.value.title) form.value.title = file.name.replace(/\.[^.]+$/, '')
    success.value = '音频上传成功，请补全曲目信息后保存'
  } catch (reason) { error.value = errorMessage(reason) } finally { uploading.value = false; inputElement.value = '' }
}
async function save() {
  error.value = ''
  success.value = ''
  const errors: Record<string, string> = {}
  if (!form.value.title.trim()) errors.title = '请输入曲目标题'
  if (!form.value.artist.trim()) errors.artist = '请输入艺术家'
  if (!validWebUrl(form.value.url)) errors.url = '请上传音频或输入有效的音频地址'
  if (!validWebUrl(form.value.cover, true)) errors.cover = '封面地址格式不正确'
  if (form.value.sort < 0) errors.sort = '排序值不能小于 0'
  fieldErrors.value = errors
  if (Object.keys(errors).length) return
  saving.value = true
  try {
    const payload = { ...form.value, title: form.value.title.trim(), artist: form.value.artist.trim(), album: form.value.album.trim() }
    if (editingId.value === null) {
      tracks.value.push(await adminApi.createTrack(payload))
    } else {
      const updated = await adminApi.updateTrack({ ...payload, id: editingId.value })
      tracks.value = tracks.value.map((track) => track.id === updated.id ? updated : track)
    }
    tracks.value.sort((a, b) => a.sort - b.sort)
    formOpen.value = false
    success.value = editingId.value === null ? '曲目已创建' : '曲目已更新'
  } catch (reason) { error.value = errorMessage(reason) } finally { saving.value = false }
}
async function remove(track: Track) {
  if (!window.confirm(`确定删除曲目“${track.title}”吗？`)) return
  deletingId.value = track.id
  error.value = ''
  try {
    await adminApi.deleteTrack(track.id)
    tracks.value = tracks.value.filter((item) => item.id !== track.id)
    if (editingId.value === track.id) formOpen.value = false
    success.value = '曲目已删除'
  } catch (reason) { error.value = errorMessage(reason) } finally { deletingId.value = null }
}
</script>

<template>
  <PanelShell title="音乐管理" description="上传本地音频或添加远程曲目，管理主页播放器列表。">
    <template #actions><button class="primary-button" type="button" @click="create"><Plus :size="15" />新增曲目</button></template>
    <AlertMessage :message="error" type="error" /><AlertMessage :message="success" type="success" />
    <div v-if="formOpen" class="mb-6 rounded-2xl border border-violet-200/10 bg-violet-200/[0.025] p-4 min-[640px]:p-5">
      <div class="mb-5 flex items-center justify-between"><h3 class="text-sm font-medium text-white/80">{{ editingId === null ? '新增曲目' : '编辑曲目' }}</h3><button class="icon-button size-7" type="button" aria-label="关闭表单" @click="formOpen = false"><X :size="14" /></button></div>
      <form class="grid gap-4 min-[760px]:grid-cols-2" @submit.prevent="save">
        <FormField label="曲目标题" :error="fieldErrors.title"><input v-model="form.title" class="field" maxlength="100" placeholder="Evening Walk" /></FormField>
        <FormField label="艺术家" :error="fieldErrors.artist"><input v-model="form.artist" class="field" maxlength="100" placeholder="艺术家名称" /></FormField>
        <FormField label="专辑"><input v-model="form.album" class="field" maxlength="100" placeholder="专辑名称（可选）" /></FormField>
        <FormField label="排序" :error="fieldErrors.sort"><input v-model.number="form.sort" class="field" type="number" min="0" /></FormField>
        <div class="min-[760px]:col-span-2">
          <FormField label="音频地址" :error="fieldErrors.url" hint="远程 URL 或本地上传">
            <div class="flex gap-2"><input v-model="form.url" class="field" placeholder="https://…/audio.mp3" /><input ref="musicInput" type="file" accept="audio/*,.mp3,.wav,.ogg,.flac" class="hidden" @change="uploadMusic" /><button class="secondary-button shrink-0" type="button" :disabled="uploading" @click="musicInput?.click()"><LoadingSpinner v-if="uploading" label="上传中" /><template v-else><Upload :size="14" />上传音频</template></button></div>
          </FormField>
        </div>
        <div class="min-[760px]:col-span-2"><ImageUploadField v-model="form.cover" label="曲目封面（可选）" /><p v-if="fieldErrors.cover" class="mt-1 text-xs text-rose-300">{{ fieldErrors.cover }}</p></div>
        <FormField label="来源"><select v-model="form.source" class="field"><option value="remote">远程地址</option><option value="local">本地上传</option></select></FormField>
        <div class="flex items-end pb-1"><ToggleSwitch v-model="form.enabled" label="加入主页播放列表" class="w-full" /></div>
        <div class="flex justify-end gap-2 min-[760px]:col-span-2"><button class="secondary-button" type="button" @click="formOpen = false">取消</button><button class="primary-button" type="submit" :disabled="saving || uploading"><Save :size="14" />{{ saving ? '保存中…' : '保存曲目' }}</button></div>
      </form>
    </div>
    <LoadingSpinner v-if="loading" label="正在加载曲目…" class="py-10 text-white/40" />
    <div v-else-if="tracks.length" class="space-y-2">
      <article v-for="track in tracks" :key="track.id" class="flex items-center gap-3 rounded-xl border border-white/[0.065] bg-white/[0.025] p-3.5">
        <div class="flex size-11 shrink-0 items-center justify-center overflow-hidden rounded-xl bg-violet-300/[0.06] text-violet-200/55"><img v-if="track.cover" :src="track.cover" alt="" class="size-full object-cover" /><Music2 v-else :size="18" /></div>
        <div class="min-w-0 flex-1"><div class="flex items-center gap-2"><h3 class="truncate text-sm text-white/75">{{ track.title }}</h3><span v-if="!track.enabled" class="text-[9px] text-white/25">已停用</span></div><p class="mt-1 truncate text-[11px] text-white/30">{{ track.artist }}<template v-if="track.album"> · {{ track.album }}</template></p><p class="mt-1 flex items-center gap-1 text-[9px] text-white/20"><FileAudio :size="10" />{{ track.source }} · #{{ track.sort }}</p></div>
        <button class="icon-button size-8" type="button" title="编辑" @click="edit(track)"><Pencil :size="14" /></button><button class="icon-button size-8 hover:text-rose-200" type="button" title="删除" :disabled="deletingId === track.id" @click="remove(track)"><Trash2 :size="14" /></button>
      </article>
    </div>
    <p v-else class="py-12 text-center text-sm text-white/30">播放列表为空，点击“新增曲目”上传音乐。</p>
  </PanelShell>
</template>
