<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Save } from 'lucide-vue-next'
import { adminApi, publicApi } from '@/api'
import { errorMessage } from '@/api/client'
import { DEFAULT_SETTINGS } from '@/data/defaults'
import type { Settings } from '@/types'
import { validClockTime, validTimezone, validWebUrl } from '@/utils/validation'
import AlertMessage from '@/components/common/AlertMessage.vue'
import FormField from '@/components/common/FormField.vue'
import ImageUploadField from '@/components/common/ImageUploadField.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import PanelShell from '@/components/common/PanelShell.vue'
import ToggleSwitch from '@/components/common/ToggleSwitch.vue'

const form = ref<Settings>(structuredClone(DEFAULT_SETTINGS))
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const success = ref('')
const fieldErrors = ref<Record<string, string>>({})

onMounted(async () => {
  try { form.value = await publicApi.settings() } catch (reason) { error.value = errorMessage(reason) } finally { loading.value = false }
})

function validate() {
  const errors: Record<string, string> = {}
  if (!form.value.site_title.trim()) errors.site_title = '请输入站点标题'
  if (!validTimezone(form.value.timezone)) errors.timezone = '请输入有效的 IANA 时区'
  if (!validTimezone(form.value.background.timezone)) errors.background_timezone = '请输入有效的 IANA 时区'
  if (!validClockTime(form.value.background.day_start)) errors.day_start = '时间格式应为 HH:mm'
  if (!validClockTime(form.value.background.night_start)) errors.night_start = '时间格式应为 HH:mm'
  if (!validWebUrl(form.value.background.day_url)) errors.day_url = '请输入有效的日间背景地址'
  if (!validWebUrl(form.value.background.night_url)) errors.night_url = '请输入有效的夜间背景地址'
  if (form.value.background.duration_ms < 0 || form.value.background.duration_ms > 10000) errors.duration_ms = '过渡时长应在 0–10000 毫秒之间'
  if (form.value.guestbook_max_length < 1 || form.value.guestbook_max_length > 2000) errors.guestbook_max_length = '留言长度应在 1–2000 之间'
  fieldErrors.value = errors
  return !Object.keys(errors).length
}

async function save() {
  error.value = ''
  success.value = ''
  if (!validate()) return
  saving.value = true
  try {
    form.value = await adminApi.updateSettings(form.value)
    success.value = '站点设置已保存，主页将在下次加载时应用'
  } catch (reason) { error.value = errorMessage(reason) } finally { saving.value = false }
}
</script>

<template>
  <PanelShell title="站点设置" description="配置日夜背景调度、时钟显示与留言规则。">
    <template #actions><button class="primary-button" type="button" :disabled="loading || saving" @click="save"><Save :size="14" />{{ saving ? '保存中…' : '保存设置' }}</button></template>
    <LoadingSpinner v-if="loading" label="正在加载设置…" class="py-10 text-white/40" />
    <template v-else>
      <AlertMessage :message="error" type="error" /><AlertMessage :message="success" type="success" />
      <form class="space-y-8" @submit.prevent="save">
        <section>
          <h3 class="mb-4 text-xs font-semibold tracking-wider text-violet-200/65">基础与时钟</h3>
          <div class="grid gap-5 min-[760px]:grid-cols-2">
            <FormField label="站点标题" :error="fieldErrors.site_title"><input v-model="form.site_title" class="field" maxlength="80" /></FormField>
            <FormField label="时钟标签"><input v-model="form.clock_label" class="field" maxlength="50" placeholder="SHANGHAI · UTC+8" /></FormField>
            <FormField label="时钟时区" :error="fieldErrors.timezone" hint="IANA 时区"><input v-model="form.timezone" class="field" placeholder="Asia/Shanghai" /></FormField>
            <FormField label="时间格式"><select v-model="form.time_format" class="field"><option value="24">24 小时制</option><option value="12">12 小时制</option></select></FormField>
          </div>
        </section>

        <section>
          <h3 class="mb-4 text-xs font-semibold tracking-wider text-violet-200/65">背景与自动调度</h3>
          <div class="grid gap-5 min-[760px]:grid-cols-2">
            <ImageUploadField v-model="form.background.day_url" label="日间背景" required /><ImageUploadField v-model="form.background.night_url" label="夜间背景" required />
            <FormField label="背景调度时区" :error="fieldErrors.background_timezone"><input v-model="form.background.timezone" class="field" placeholder="Asia/Shanghai" /></FormField>
            <FormField label="淡入淡出时长" :error="fieldErrors.duration_ms" hint="毫秒"><input v-model.number="form.background.duration_ms" class="field" type="number" min="0" max="10000" step="100" /></FormField>
            <FormField label="日间开始时间" :error="fieldErrors.day_start"><input v-model="form.background.day_start" class="field" type="time" /></FormField>
            <FormField label="夜间开始时间" :error="fieldErrors.night_start"><input v-model="form.background.night_start" class="field" type="time" /></FormField>
          </div>
          <p v-if="fieldErrors.day_url || fieldErrors.night_url" class="mt-2 text-xs text-rose-300">{{ fieldErrors.day_url || fieldErrors.night_url }}</p>
        </section>

        <section>
          <h3 class="mb-4 text-xs font-semibold tracking-wider text-violet-200/65">留言规则</h3>
          <div class="grid items-center gap-5 rounded-xl border border-white/[0.07] bg-white/[0.025] p-4 min-[760px]:grid-cols-2">
            <ToggleSwitch v-model="form.guestbook_auto_publish" label="新留言自动发布" />
            <FormField label="留言最大长度" :error="fieldErrors.guestbook_max_length" hint="字符"><input v-model.number="form.guestbook_max_length" class="field" type="number" min="1" max="2000" /></FormField>
          </div>
        </section>
      </form>
    </template>
  </PanelShell>
</template>
