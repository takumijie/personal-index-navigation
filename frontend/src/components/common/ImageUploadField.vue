<script setup lang="ts">
import { computed, ref } from 'vue'
import { Film, ImagePlus, LoaderCircle } from 'lucide-vue-next'
import { adminApi } from '@/api'
import { errorMessage } from '@/api/client'
import { isVideoUrl } from '@/utils/media'
import FormField from './FormField.vue'

const model = defineModel<string>({ required: true })
const isVideo = computed(() => isVideoUrl(model.value))
withDefaults(defineProps<{ label: string; required?: boolean }>(), { required: false })
const fileInput = ref<HTMLInputElement | null>(null)
const uploading = ref(false)
const error = ref('')

async function upload(event: Event) {
  const inputElement = event.target as HTMLInputElement
  const file = inputElement.files?.[0]
  if (!file) return
  error.value = ''
  if (!file.type.startsWith('image/')) {
    error.value = '请选择图片文件'
    inputElement.value = ''
    return
  }
  if (file.size > 8 * 1024 * 1024) {
    error.value = '图片不能超过 8 MB'
    inputElement.value = ''
    return
  }
  uploading.value = true
  try {
    const result = await adminApi.uploadImage(file)
    model.value = result.url
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    uploading.value = false
    inputElement.value = ''
  }
}
</script>

<template>
  <FormField :label="label" :error="error" hint="支持 URL 或本地上传">
    <div class="flex gap-2">
      <div v-if="model" class="flex size-[38px] shrink-0 items-center justify-center overflow-hidden rounded-[10px] border border-white/10 bg-white/5">
        <Film v-if="isVideo" :size="16" class="text-white/45" />
        <img v-else :src="model" alt="图片预览" class="size-full object-cover" />
      </div>
      <input v-model="model" class="field" type="text" :required="required" placeholder="https://… 或 /uploads/…" />
      <input ref="fileInput" type="file" accept="image/*" class="hidden" @change="upload" />
      <button class="secondary-button shrink-0 px-3" type="button" :disabled="uploading" :title="`上传${label}`" @click="fileInput?.click()">
        <LoaderCircle v-if="uploading" :size="15" class="animate-spin" /><ImagePlus v-else :size="15" />
        <span class="hidden min-[640px]:inline">{{ uploading ? '上传中' : '上传' }}</span>
      </button>
    </div>
  </FormField>
</template>
