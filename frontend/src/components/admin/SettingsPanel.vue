<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Save } from 'lucide-vue-next'
import { adminApi, publicApi } from '@/api'
import { errorMessage } from '@/api/client'
import { DEFAULT_SETTINGS, DEFAULT_WATERMARK_CAPTION, DEFAULT_WATERMARK_SUBTITLE, DEFAULT_WATERMARK_TITLE } from '@/data/defaults'
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
  if (form.value.loader.watermark_title.length > 60) errors.watermark_title = '水印主标题不能超过 60 字符'
  if (form.value.loader.watermark_subtitle.length > 80) errors.watermark_subtitle = '水印副标题不能超过 80 字符'
  if (form.value.loader.watermark_caption.length > 120) errors.watermark_caption = '水印小字不能超过 120 字符'
  if (form.value.loader.badge_image && !validWebUrl(form.value.loader.badge_image) && !form.value.loader.badge_image.startsWith('/')) errors.badge_image = '请输入有效的徽章图片地址'
  if (form.value.background.duration_ms < 0 || form.value.background.duration_ms > 60000) errors.duration_ms = '过渡时长应在 0–60000 毫秒之间'
  if (form.value.terminal.fastfetch_loading_ms < 0 || form.value.terminal.fastfetch_loading_ms > 10000) errors.fastfetch_loading_ms = 'Fastfetch 停留时长应在 0–10000 毫秒之间'
  if (form.value.guestbook_max_length < 1 || form.value.guestbook_max_length > 10000) errors.guestbook_max_length = '留言长度应在 1–10000 之间'
  fieldErrors.value = errors
  return !Object.keys(errors).length
}

async function save() {
  error.value = ''
  success.value = ''
  if (!validate()) return
  saving.value = true
  try {
    const latest = await publicApi.settings()
    form.value = await adminApi.updateSettings({ ...form.value, music: latest.music })
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
            <FormField label="站点标题" :error="fieldErrors.site_title"><input v-model="form.site_title" class="field" maxlength="100" /></FormField>
            <FormField label="时钟标签"><input v-model="form.clock_label" class="field" maxlength="100" placeholder="SHANGHAI · UTC+8" /></FormField>
            <FormField label="时钟时区" :error="fieldErrors.timezone" hint="IANA 时区"><input v-model="form.timezone" class="field" placeholder="Asia/Shanghai" /></FormField>
            <FormField label="时间格式"><select v-model="form.time_format" class="field"><option value="24">24 小时制</option><option value="12">12 小时制</option></select></FormField>
          </div>
        </section>

        <section>
          <h3 class="mb-4 text-xs font-semibold tracking-wider text-violet-200/65">背景与自动调度</h3>
          <div class="grid gap-5 min-[760px]:grid-cols-2">
            <ImageUploadField v-model="form.background.day_url" label="日间背景" required /><ImageUploadField v-model="form.background.night_url" label="夜间背景" required />
            <FormField label="背景调度时区" :error="fieldErrors.background_timezone"><input v-model="form.background.timezone" class="field" placeholder="Asia/Shanghai" /></FormField>
            <FormField label="淡入淡出时长" :error="fieldErrors.duration_ms" hint="毫秒"><input v-model.number="form.background.duration_ms" class="field" type="number" min="0" max="60000" step="100" /></FormField>
            <FormField label="日间开始时间" :error="fieldErrors.day_start"><input v-model="form.background.day_start" class="field" type="time" /></FormField>
            <FormField label="夜间开始时间" :error="fieldErrors.night_start"><input v-model="form.background.night_start" class="field" type="time" /></FormField>
          </div>
          <p class="mt-3 text-[11px] text-white/35">背景支持图片或视频直链（.mp4 / .webm / .mov 等，自动识别，静音循环播放）。</p>
          <p v-if="fieldErrors.day_url || fieldErrors.night_url" class="mt-2 text-xs text-rose-300">{{ fieldErrors.day_url || fieldErrors.night_url }}</p>
        </section>

        <section>
          <h3 class="mb-4 text-xs font-semibold tracking-wider text-violet-200/65">卡片显示</h3>
          <div class="grid items-center gap-5 rounded-xl border border-white/[0.07] bg-white/[0.025] p-4 min-[760px]:grid-cols-2">
            <ToggleSwitch v-model="form.cards.profile" label="个人资料卡片" />
            <ToggleSwitch v-model="form.cards.navigation" label="站点导航卡片" />
            <ToggleSwitch v-model="form.cards.devices" label="设备展示卡片" />
            <ToggleSwitch v-model="form.cards.terminal" label="终端留言卡片" />
            <ToggleSwitch v-model="form.cards.clock" label="时钟与音乐卡片" />
          </div>
        </section>

        <section>
          <h3 class="mb-4 text-xs font-semibold tracking-wider text-violet-200/65">加载页</h3>
          <div class="grid gap-5 min-[760px]:grid-cols-2">
            <FormField label="主题"><select v-model="form.loader.theme" class="field"><option value="dark">经典暗色</option><option value="beige">米白</option></select></FormField>
            <FormField label="水印主标题" :error="fieldErrors.watermark_title" hint="留空使用默认文案"><input v-model="form.loader.watermark_title" class="field" maxlength="60" :placeholder="DEFAULT_WATERMARK_TITLE" /></FormField>
            <FormField label="水印副标题" :error="fieldErrors.watermark_subtitle" hint="留空使用默认文案"><input v-model="form.loader.watermark_subtitle" class="field" maxlength="80" :placeholder="DEFAULT_WATERMARK_SUBTITLE" /></FormField>
            <FormField label="水印小字" :error="fieldErrors.watermark_caption" hint="留空使用默认文案"><input v-model="form.loader.watermark_caption" class="field" maxlength="120" :placeholder="DEFAULT_WATERMARK_CAPTION" /></FormField>
            <ImageUploadField v-model="form.loader.badge_image" label="徽章图片（可选，点阵化显示）" />
          </div>
          <div class="mt-5 rounded-xl border border-white/[0.07] bg-white/[0.025] p-4">
            <ToggleSwitch v-model="form.loader.sound_enabled" label="加载页音效" />
          </div>
          <p v-if="fieldErrors.badge_image" class="mt-2 text-xs text-rose-300">{{ fieldErrors.badge_image }}</p>
          <p class="mt-3 text-[11px] text-white/35">徽章图片支持链接或上传，加载页会将其转换为半调点阵风格；留空使用内置几何徽章。</p>
          <p class="mt-2 text-[11px] text-white/35">音效开关控制初始状态，访客仍可在加载页调整；默认开启时将在首次点击或按键后启用声音。</p>
        </section>

        <section>
          <h3 class="mb-4 text-xs font-semibold tracking-wider text-violet-200/65">终端与留言</h3>
          <div class="grid items-center gap-5 rounded-xl border border-white/[0.07] bg-white/[0.025] p-4 min-[760px]:grid-cols-2">
            <FormField label="Fastfetch 停留时长" :error="fieldErrors.fastfetch_loading_ms" hint="毫秒"><input v-model.number="form.terminal.fastfetch_loading_ms" class="field" type="number" min="0" max="10000" step="100" /></FormField>
            <FormField label="留言最大长度" :error="fieldErrors.guestbook_max_length" hint="字符"><input v-model.number="form.guestbook_max_length" class="field" type="number" min="1" max="10000" /></FormField>
            <ToggleSwitch v-model="form.guestbook_auto_publish" label="新留言自动发布" />
          </div>
        </section>
      </form>
    </template>
  </PanelShell>
</template>
