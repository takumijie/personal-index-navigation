<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { CheckCircle2, ChevronLeft, ChevronRight, Cloud, Download, FileAudio, RefreshCw, Save, Trash2 } from 'lucide-vue-next'
import { adminApi } from '@/api'
import { errorMessage } from '@/api/client'
import type { ImportResult, Playlist, S3Config, S3Object } from '@/types'
import AlertMessage from '@/components/common/AlertMessage.vue'
import FormField from '@/components/common/FormField.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import ToggleSwitch from '@/components/common/ToggleSwitch.vue'

const emptyConfig = (): S3Config => ({
  endpoint: '',
  region: '',
  bucket: '',
  access_key_id: '',
  secret_access_key: '',
  prefix: '',
  force_path_style: false,
  public_base_url: '',
  access_mode: 'presigned',
  has_secret: false,
})

const form = ref<S3Config>(emptyConfig())
const loading = ref(true)
const saving = ref(false)
const testing = ref(false)
const error = ref('')
const success = ref('')

const objectPrefix = ref('')
const objects = ref<S3Object[]>([])
const objectsLoading = ref(false)
const objectsLoaded = ref(false)
const objectError = ref('')
const objectSuccess = ref('')
const tokens = ref<string[]>([''])
const tokenIndex = ref(0)
const nextToken = ref('')
const selectedObjects = ref<S3Object[]>([])
const importName = ref('')
const importing = ref(false)
const playlists = ref<Playlist[]>([])
const appendPlaylistID = ref<number | null>(null)
const appending = ref(false)
const deletingKey = ref('')

const allCurrentSelected = computed(() => objects.value.length > 0 && objects.value.every((object) => isSelected(object)))
const objectActionBusy = computed(() => importing.value || appending.value || deletingKey.value !== '')

function applyConfig(config: S3Config) {
  form.value = { ...emptyConfig(), ...config, secret_access_key: '' }
}

async function loadPlaylists() {
  playlists.value = (await adminApi.playlists()).sort((a, b) => a.sort - b.sort || a.id - b.id)
  if (!playlists.value.some((playlist) => playlist.id === appendPlaylistID.value)) {
    appendPlaylistID.value = playlists.value[0]?.id ?? null
  }
}

async function refreshPlaylists() {
  try {
    await loadPlaylists()
  } catch (reason) {
    objectError.value = `播放列表刷新失败：${errorMessage(reason)}`
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [config] = await Promise.all([adminApi.s3Config(), loadPlaylists()])
    applyConfig(config)
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    loading.value = false
  }
}

async function save() {
  error.value = ''
  success.value = ''
  if (!form.value.endpoint.trim() || !form.value.region.trim() || !form.value.bucket.trim() || !form.value.access_key_id.trim()) {
    error.value = '请填写 Endpoint、Region、Bucket 和 Access Key ID'
    return
  }
  if (!form.value.has_secret && !form.value.secret_access_key) {
    error.value = '请输入 Secret Access Key'
    return
  }
  if (form.value.access_mode === 'public' && !form.value.public_base_url.trim()) {
    error.value = '公开访问模式需要填写 Public Base URL'
    return
  }
  saving.value = true
  try {
    const payload: S3Config = {
      ...form.value,
      endpoint: form.value.endpoint.trim(),
      region: form.value.region.trim(),
      bucket: form.value.bucket.trim(),
      access_key_id: form.value.access_key_id.trim(),
      prefix: form.value.prefix.trim(),
      public_base_url: form.value.public_base_url.trim(),
      secret_access_key: form.value.secret_access_key || '',
    }
    applyConfig(await adminApi.saveS3Config(payload))
    resetObjects()
    success.value = 'S3 配置已保存；连接测试与对象列表将使用已保存配置'
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    saving.value = false
  }
}

async function testConnection() {
  testing.value = true
  error.value = ''
  success.value = ''
  try {
    const result = await adminApi.testS3()
    if (!result.connected) {
      error.value = 'S3 连接测试未通过'
      return
    }
    success.value = 'S3 连接测试通过'
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    testing.value = false
  }
}

function isSelected(object: S3Object) {
  return selectedObjects.value.some((item) => item.key === object.key)
}

function toggleObject(object: S3Object) {
  if (isSelected(object)) {
    selectedObjects.value = selectedObjects.value.filter((item) => item.key !== object.key)
  } else {
    selectedObjects.value.push(object)
  }
}

function toggleAll() {
  if (allCurrentSelected.value) {
    const keys = new Set(objects.value.map((object) => object.key))
    selectedObjects.value = selectedObjects.value.filter((object) => !keys.has(object.key))
    return
  }
  for (const object of objects.value) {
    if (!isSelected(object)) selectedObjects.value.push(object)
  }
}

function resetObjects() {
  objects.value = []
  objectsLoaded.value = false
  tokens.value = ['']
  tokenIndex.value = 0
  nextToken.value = ''
  selectedObjects.value = []
  objectError.value = ''
  objectSuccess.value = ''
}

async function loadObjects(index = 0, reset = false) {
  if (reset) resetObjects()
  objectsLoading.value = true
  objectError.value = ''
  objectSuccess.value = ''
  try {
    const result = await adminApi.s3Objects(objectPrefix.value.trim(), tokens.value[index] || '')
    objects.value = result.objects
    tokenIndex.value = index
    nextToken.value = result.next_token || ''
    objectsLoaded.value = true
    tokens.value = tokens.value.slice(0, index + 1)
    if (nextToken.value) tokens.value.push(nextToken.value)
  } catch (reason) {
    objectError.value = errorMessage(reason)
  } finally {
    objectsLoading.value = false
  }
}

function fileName(key: string) {
  return key.split('/').pop() || key
}

function formatSize(size: number) {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`
  return `${(size / 1024 / 1024 / 1024).toFixed(1)} GB`
}

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : ''
}

function validSelection() {
  if (!selectedObjects.value.length || selectedObjects.value.length > 100) {
    objectError.value = '请选择 1 至 100 个音频对象'
    return false
  }
  return true
}

async function importSelected() {
  objectError.value = ''
  objectSuccess.value = ''
  if (!importName.value.trim()) {
    objectError.value = '请输入播放列表名称'
    return
  }
  if (!validSelection()) return
  importing.value = true
  try {
    const result: ImportResult = await adminApi.importS3(importName.value.trim(), selectedObjects.value.map((object) => object.key))
    objectSuccess.value = `播放列表“${result.playlist.name}”已保存：导入 ${result.imported} 首，重复 ${result.duplicates} 首`
    selectedObjects.value = []
    importName.value = ''
    await refreshPlaylists()
  } catch (reason) {
    objectError.value = errorMessage(reason)
  } finally {
    importing.value = false
  }
}

async function appendSelected() {
  objectError.value = ''
  objectSuccess.value = ''
  if (!appendPlaylistID.value) {
    objectError.value = '请选择目标播放列表'
    return
  }
  if (!validSelection()) return
  appending.value = true
  try {
    const result = await adminApi.appendS3(appendPlaylistID.value, selectedObjects.value.map((object) => object.key))
    objectSuccess.value = `已追加到“${result.playlist.name}”：新增 ${result.imported} 首，重复 ${result.duplicates} 首`
    selectedObjects.value = []
    await refreshPlaylists()
  } catch (reason) {
    objectError.value = errorMessage(reason)
  } finally {
    appending.value = false
  }
}

async function removeObject(object: S3Object) {
  if (deletingKey.value) return
  if (!window.confirm(`确定删除 OSS 对象“${object.key}”吗？将同时删除音频对象、对应曲目及所有播放列表关联，且无法恢复。`)) return
  deletingKey.value = object.key
  objectError.value = ''
  objectSuccess.value = ''
  try {
    await adminApi.deleteS3Object(object.key)
    objects.value = objects.value.filter((item) => item.key !== object.key)
    selectedObjects.value = selectedObjects.value.filter((item) => item.key !== object.key)
    objectSuccess.value = `OSS 对象“${fileName(object.key)}”及对应曲目关联已删除`
    await refreshPlaylists()
  } catch (reason) {
    objectError.value = errorMessage(reason)
    await refreshPlaylists()
  } finally {
    deletingKey.value = ''
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-6">
    <section class="rounded-2xl border border-white/[0.065] bg-white/[0.025] p-4 min-[640px]:p-5">
      <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 class="flex items-center gap-2 text-sm font-medium text-white/80"><Cloud :size="16" />S3 兼容对象存储</h3>
          <p class="mt-1 text-xs text-white/35">保存配置后再进行连接测试或读取对象；错误信息由后端实际请求返回。</p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button class="secondary-button" type="button" :disabled="loading || testing || saving" @click="testConnection"><CheckCircle2 :size="14" />{{ testing ? '测试中…' : '测试已保存配置' }}</button>
          <button class="primary-button" type="button" :disabled="loading || saving || testing" @click="save"><Save :size="14" />{{ saving ? '保存中…' : '保存配置' }}</button>
        </div>
      </div>

      <AlertMessage :message="error" type="error" />
      <AlertMessage :message="success" type="success" />
      <LoadingSpinner v-if="loading" label="正在加载 S3 配置…" class="py-10 text-white/40" />
      <form v-else class="grid gap-4 min-[760px]:grid-cols-2" @submit.prevent="save">
        <div class="min-[760px]:col-span-2"><FormField label="Endpoint"><input v-model="form.endpoint" class="field" type="url" placeholder="https://s3.example.com" autocomplete="off" /></FormField></div>
        <FormField label="Region"><input v-model="form.region" class="field" placeholder="us-east-1" autocomplete="off" /></FormField>
        <FormField label="Bucket"><input v-model="form.bucket" class="field" placeholder="my-music" autocomplete="off" /></FormField>
        <FormField label="Access Key ID"><input v-model="form.access_key_id" class="field" autocomplete="off" /></FormField>
        <FormField label="Secret Access Key" :hint="form.has_secret ? '已保存 Secret；留空保留原值' : '尚未保存 Secret'">
          <input v-model="form.secret_access_key" class="field" type="password" :placeholder="form.has_secret ? '留空以保留原值' : '请输入 Secret Access Key'" autocomplete="new-password" />
        </FormField>
        <FormField label="配置前缀" hint="可选，自动附加到对象查询前缀前"><input v-model="form.prefix" class="field" placeholder="music/" /></FormField>
        <FormField label="访问模式"><select v-model="form.access_mode" class="field"><option value="presigned">预签名 URL</option><option value="public">公开 URL</option></select></FormField>
        <div class="min-[760px]:col-span-2"><FormField label="Public Base URL" :hint="form.access_mode === 'public' ? '公开访问模式必填' : '仅公开访问模式使用'"><input v-model="form.public_base_url" class="field" type="url" placeholder="https://cdn.example.com" /></FormField></div>
        <div class="rounded-xl border border-white/[0.06] bg-black/10 px-4 py-3 min-[760px]:col-span-2"><ToggleSwitch v-model="form.force_path_style" label="使用 Path-Style 地址" /></div>
      </form>
    </section>

    <section class="rounded-2xl border border-white/[0.065] bg-white/[0.025] p-4 min-[640px]:p-5">
      <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 class="flex items-center gap-2 text-sm font-medium text-white/80"><FileAudio :size="16" />音频对象</h3>
          <p class="mt-1 text-xs text-white/35">对象查询前缀相对于配置前缀，分页使用 continuation_token。</p>
        </div>
        <span v-if="selectedObjects.length" class="text-xs text-violet-200/60">已选 {{ selectedObjects.length }} 个</span>
      </div>
      <AlertMessage :message="objectError" type="error" />
      <AlertMessage :message="objectSuccess" type="success" />

      <form class="flex flex-col gap-2 min-[560px]:flex-row" @submit.prevent="loadObjects(0, true)">
        <input v-model="objectPrefix" class="field" placeholder="对象查询前缀，例如 albums/（可选）" />
        <button class="secondary-button shrink-0" type="submit" :disabled="objectsLoading"><RefreshCw :size="14" />{{ objectsLoading ? '读取中…' : '读取对象' }}</button>
      </form>

      <LoadingSpinner v-if="objectsLoading" label="正在读取音频对象…" class="mt-6 py-8 text-white/40" />
      <template v-else-if="objectsLoaded">
        <div v-if="objects.length" class="mt-5">
          <label class="mb-3 flex cursor-pointer items-center gap-2 text-xs text-white/40">
            <input type="checkbox" class="accent-violet-500" :checked="allCurrentSelected" :disabled="objectActionBusy" @change="toggleAll" />选择本页
          </label>
          <div class="fine-scrollbar max-h-[28rem] space-y-2 overflow-y-auto pr-1">
            <div v-for="object in objects" :key="object.key" class="flex items-center gap-2 rounded-xl border border-white/[0.055] bg-black/10 p-3 transition-colors hover:bg-white/[0.035]">
              <label class="flex min-w-0 flex-1 cursor-pointer items-center gap-3">
                <input type="checkbox" class="shrink-0 accent-violet-500" :checked="isSelected(object)" :disabled="objectActionBusy" @change="toggleObject(object)" />
                <span class="flex size-10 shrink-0 items-center justify-center rounded-lg bg-white/[0.04] text-indigo-200/50"><FileAudio :size="17" /></span>
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-sm text-white/75" :title="object.key">{{ fileName(object.key) }}</span>
                  <span class="mt-1 block truncate text-[11px] text-white/30" :title="object.key">{{ object.key }}</span>
                  <span class="mt-1 block text-[10px] text-white/25">{{ formatSize(object.size) }}<template v-if="object.last_modified"> · {{ formatDate(object.last_modified) }}</template></span>
                </span>
              </label>
              <button class="icon-button size-9 shrink-0 hover:bg-rose-400/10 hover:text-rose-200" type="button" :title="`删除 OSS 对象 ${object.key}`" :aria-label="`删除 OSS 对象 ${object.key}`" :disabled="objectActionBusy" @click="removeObject(object)">
                <LoadingSpinner v-if="deletingKey === object.key" label="" /><Trash2 v-else :size="15" />
              </button>
            </div>
          </div>
        </div>
        <p v-else class="py-10 text-center text-sm text-white/30">此前缀下没有可展示的音频对象</p>

        <div v-if="tokenIndex > 0 || nextToken" class="mt-4 flex items-center justify-center gap-3">
          <button class="icon-button" type="button" aria-label="上一页" :disabled="tokenIndex === 0" @click="loadObjects(tokenIndex - 1)"><ChevronLeft :size="16" /></button>
          <span class="text-xs text-white/40">第 {{ tokenIndex + 1 }} 页</span>
          <button class="icon-button" type="button" aria-label="下一页" :disabled="!nextToken" @click="loadObjects(tokenIndex + 1)"><ChevronRight :size="16" /></button>
        </div>

        <div class="mt-5 grid gap-5 border-t border-white/[0.06] pt-4 min-[900px]:grid-cols-2">
          <form class="space-y-2" @submit.prevent="importSelected">
            <label class="block text-xs text-white/45" for="s3-import-name">创建新列表导入</label>
            <div class="flex flex-col gap-2 min-[560px]:flex-row min-[900px]:flex-col min-[1100px]:flex-row">
              <input id="s3-import-name" v-model="importName" class="field" placeholder="新播放列表名称" maxlength="80" />
              <button class="primary-button shrink-0" type="submit" :disabled="objectActionBusy || !selectedObjects.length"><Download :size="14" />{{ importing ? '导入中…' : `创建列表（${selectedObjects.length}）` }}</button>
            </div>
          </form>
          <form class="space-y-2" @submit.prevent="appendSelected">
            <label class="block text-xs text-white/45" for="s3-append-playlist">追加到已有列表</label>
            <div class="flex flex-col gap-2 min-[560px]:flex-row min-[900px]:flex-col min-[1100px]:flex-row">
              <select id="s3-append-playlist" v-model.number="appendPlaylistID" class="field" :disabled="!playlists.length">
                <option v-for="playlist in playlists" :key="playlist.id" :value="playlist.id">{{ playlist.name }}</option>
              </select>
              <button class="secondary-button shrink-0" type="submit" :disabled="objectActionBusy || !selectedObjects.length || !appendPlaylistID"><Download :size="14" />{{ appending ? '追加中…' : `追加歌曲（${selectedObjects.length}）` }}</button>
            </div>
            <p v-if="!playlists.length" class="text-[10px] text-white/30">暂无已有列表，可先创建新列表导入。</p>
          </form>
        </div>
      </template>
      <p v-else-if="!objectError" class="py-10 text-center text-sm text-white/30">保存配置后，按前缀读取音频对象。</p>
    </section>
  </div>
</template>
