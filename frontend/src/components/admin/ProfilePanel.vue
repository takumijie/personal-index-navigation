<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Save } from 'lucide-vue-next'
import { adminApi, publicApi } from '@/api'
import { errorMessage } from '@/api/client'
import type { Profile } from '@/types'
import { validWebUrl } from '@/utils/validation'
import AlertMessage from '@/components/common/AlertMessage.vue'
import FormField from '@/components/common/FormField.vue'
import ImageUploadField from '@/components/common/ImageUploadField.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import PanelShell from '@/components/common/PanelShell.vue'

const emptyProfile: Profile = { name: '', username: '', description: '', avatar: '', cover_mode: 'current', cover_url: '', tags: [] }
const form = ref<Profile>(structuredClone(emptyProfile))
const tagsText = ref('')
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const success = ref('')
const fieldErrors = ref<Record<string, string>>({})

onMounted(async () => {
  try {
    form.value = await publicApi.profile()
    tagsText.value = form.value.tags.join('，')
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    loading.value = false
  }
})

function validate() {
  const errors: Record<string, string> = {}
  if (!form.value.name.trim()) errors.name = '请输入显示名称'
  if (!form.value.username.trim()) errors.username = '请输入用户名'
  if (!form.value.description.trim()) errors.description = '请输入个人简介'
  if (!validWebUrl(form.value.avatar, true)) errors.avatar = '头像地址格式不正确'
  if (form.value.cover_mode === 'custom' && !validWebUrl(form.value.cover_url)) errors.cover_url = '自定义封面需要有效的图片地址'
  fieldErrors.value = errors
  return !Object.keys(errors).length
}

async function save() {
  success.value = ''
  error.value = ''
  if (!validate()) return
  saving.value = true
  try {
    const payload: Profile = {
      ...form.value,
      name: form.value.name.trim(), username: form.value.username.trim(), description: form.value.description.trim(),
      tags: tagsText.value.split(/[,，]/).map((tag) => tag.trim()).filter(Boolean).slice(0, 12),
    }
    form.value = await adminApi.updateProfile(payload)
    tagsText.value = form.value.tags.join('，')
    success.value = '个人资料已保存'
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <PanelShell title="个人资料" description="管理主页展示的名称、简介、标签与视觉图片。">
    <template #actions><button class="primary-button" type="button" :disabled="loading || saving" @click="save"><Save :size="14" />{{ saving ? '保存中…' : '保存资料' }}</button></template>
    <LoadingSpinner v-if="loading" label="正在加载资料…" class="py-10 text-white/40" />
    <template v-else>
      <AlertMessage :message="error" type="error" /><AlertMessage :message="success" type="success" />
      <form class="grid gap-5 min-[760px]:grid-cols-2" @submit.prevent="save">
        <FormField label="显示名称" :error="fieldErrors.name"><input v-model="form.name" class="field" maxlength="50" placeholder="你的名字" /></FormField>
        <FormField label="用户名" :error="fieldErrors.username"><input v-model="form.username" class="field" maxlength="50" placeholder="@username" /></FormField>
        <div class="min-[760px]:col-span-2"><FormField label="个人简介" :error="fieldErrors.description" :hint="`${form.description.length}/240`"><textarea v-model="form.description" class="field min-h-24 resize-y" maxlength="240" placeholder="介绍一下自己…" /></FormField></div>
        <div class="min-[760px]:col-span-2"><ImageUploadField v-model="form.avatar" label="头像图片" /><p v-if="fieldErrors.avatar" class="mt-1 text-xs text-rose-300">{{ fieldErrors.avatar }}</p></div>
        <FormField label="封面模式">
          <select v-model="form.cover_mode" class="field"><option value="current">跟随当前背景</option><option value="day">始终使用日间背景</option><option value="night">始终使用夜间背景</option><option value="custom">自定义封面</option></select>
        </FormField>
        <FormField label="个人标签" hint="使用逗号分隔"><input v-model="tagsText" class="field" placeholder="开发，摄影，咖啡" /></FormField>
        <div v-if="form.cover_mode === 'custom'" class="min-[760px]:col-span-2"><ImageUploadField v-model="form.cover_url" label="自定义封面" required /><p v-if="fieldErrors.cover_url" class="mt-1 text-xs text-rose-300">{{ fieldErrors.cover_url }}</p></div>
      </form>
    </template>
  </PanelShell>
</template>
