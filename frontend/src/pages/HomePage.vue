<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watchEffect } from 'vue'
import { ArrowUpRight, CloudOff, MoonStar, RefreshCw, SunMedium } from 'lucide-vue-next'
import BackgroundScene from '@/components/common/BackgroundScene.vue'
import HomeBrandLink from '@/components/common/HomeBrandLink.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import BootLoader from '@/components/boot/BootLoader.vue'
import ProfileCard from '@/components/profile/ProfileCard.vue'
import NavigationGrid from '@/components/navigation/NavigationGrid.vue'
import DeviceGrid from '@/components/devices/DeviceGrid.vue'
import TerminalCard from '@/components/terminal/TerminalCard.vue'
import ClockPanel from '@/components/clock/ClockPanel.vue'
import { useBackground } from '@/composables/useBackground'
import { useHomeData } from '@/composables/useHomeData'
import { updateFavicon } from '@/utils/favicon'

const { profile, navigation, devices, settings, guestbook, tracks, visitor, loading, error, terminalError, loadTerminalData, reload } = useHomeData()
const { mode, activeUrl, loaded, failed: backgroundFailed, duration, reducedMotion } = useBackground(settings)

watchEffect(() => {
  document.title = settings.value.site_title
  updateFavicon(settings.value.favicon_url)
})

const cards = computed(() => settings.value.cards)
const leftVisible = computed(() => cards.value.profile || cards.value.navigation || cards.value.devices)
const rightVisible = computed(() => cards.value.terminal || cards.value.clock)

// 加载页：每次进入主页播放，动画完成且首屏数据与当前背景就绪后自动过渡，8 秒超时兜底
const bootPhase = ref<'boot' | 'leaving' | 'done'>('boot')
const bootTimedOut = ref(false)
const bootReady = computed(() => bootTimedOut.value || (!loading.value && !!profile.value && (loaded.value.has(activeUrl.value) || backgroundFailed.value.has(activeUrl.value))))
let bootTimer: ReturnType<typeof setTimeout> | undefined
onMounted(() => {
  bootTimer = setTimeout(() => { bootTimedOut.value = true }, 8000)
})
onBeforeUnmount(() => {
  if (bootTimer) clearTimeout(bootTimer)
})
</script>

<template>
  <div
    class="relative min-h-screen text-white transition-all duration-[650ms] ease-out"
    :class="bootPhase === 'boot' ? 'translate-y-2 opacity-0 blur-[2px]' : 'translate-y-0 opacity-100 blur-0'"
  >
    <BackgroundScene
      :day-url="settings.background.day_url"
      :night-url="settings.background.night_url"
      :mode="mode"
      :loaded="loaded"
      :duration="duration"
      :reduced-motion="reducedMotion"
    />

    <header class="mx-auto flex h-[68px] w-full max-w-[1500px] items-center justify-between px-4 min-[640px]:px-6">
      <HomeBrandLink :title="settings.site_title" :home-button="settings.home_button" />
      <div class="flex items-center gap-2">
        <div class="flex items-center gap-1.5 rounded-full border border-white/[0.08] bg-black/10 px-3 py-1.5 text-[9px] tracking-wider text-white/45 backdrop-blur-lg">
          <SunMedium v-if="mode === 'day'" :size="12" class="text-amber-200/70" />
          <MoonStar v-else :size="12" class="text-violet-200/70" />
          {{ mode === 'day' ? 'DAYLIGHT' : 'AFTERGLOW' }}
        </div>
        <RouterLink to="/admin" class="icon-button border border-white/[0.08] bg-black/10 text-[10px] backdrop-blur-lg" title="站点管理" aria-label="站点管理"><ArrowUpRight :size="14" /></RouterLink>
      </div>
    </header>

    <div v-if="error && profile" class="mx-auto mb-2 flex w-full max-w-[1500px] items-center gap-2 px-5 text-[10px] text-amber-100/50 min-[640px]:px-7">
      <CloudOff :size="12" />{{ error }}
    </div>

    <main v-if="profile && (leftVisible || rightVisible)" class="mx-auto grid w-full max-w-[1500px] gap-4 px-4 pb-5 min-[640px]:px-6 lg:h-[calc(100vh-76px)] lg:min-h-[720px]" :class="leftVisible && rightVisible ? 'lg:grid-cols-[330px_minmax(0,1fr)] xl:grid-cols-[350px_minmax(0,1fr)]' : 'lg:grid-cols-1'">
      <div v-if="leftVisible" class="fine-scrollbar flex min-h-0 flex-col gap-4 lg:overflow-y-auto lg:pr-1">
        <ProfileCard
          v-if="cards.profile"
          :profile="profile"
          :active-background="activeUrl"
          :day-background="settings.background.day_url"
          :night-background="settings.background.night_url"
          class="shrink-0"
        />
        <NavigationGrid v-if="cards.navigation" :items="navigation" class="shrink-0" />
        <DeviceGrid v-if="cards.devices" :devices="devices" class="shrink-0" />
        <p class="mt-auto hidden px-2 pb-1 text-[9px] tracking-wider text-white/20 lg:block">DESIGNED FOR SLOW EVENINGS · {{ new Date().getFullYear() }}</p>
      </div>

      <div v-if="rightVisible" class="flex min-h-0 flex-col gap-4">
        <TerminalCard
          v-if="cards.terminal"
          v-model:entries="guestbook"
          :active="bootPhase === 'done'"
          :settings="settings"
          :visitor="visitor"
          :terminal-error="terminalError"
          :load-terminal-data="loadTerminalData"
        />
        <ClockPanel v-if="cards.clock" :settings="settings" :tracks="tracks" class="shrink-0" />
      </div>
    </main>

    <main v-else-if="profile" class="mx-auto flex min-h-[calc(100vh-100px)] max-w-lg items-center justify-center px-5 pb-16">
      <div class="glass-card w-full p-8 text-center">
        <p class="text-sm leading-6 text-white/45">所有卡片均已隐藏，可在后台「站点设置 → 卡片显示」中重新开启。</p>
      </div>
    </main>

    <main v-else class="mx-auto flex min-h-[calc(100vh-100px)] max-w-lg items-center justify-center px-5 pb-16">
      <div class="glass-card w-full p-8 text-center">
        <LoadingSpinner v-if="loading" label="正在唤醒这片数字角落…" class="text-sm text-white/55" />
        <template v-else>
          <CloudOff :size="30" class="mx-auto mb-4 text-white/30" />
          <h1 class="text-lg font-medium text-white/80">暂时无法加载主页</h1>
          <p class="mt-2 text-sm leading-6 text-white/45">{{ error || '服务器没有返回资料信息' }}</p>
          <button class="secondary-button mt-6" type="button" @click="reload"><RefreshCw :size="14" />重新加载</button>
        </template>
      </div>
    </main>
  </div>

  <BootLoader
    v-if="bootPhase !== 'done'"
    :ready="bootReady"
    :loader="settings.loader"
    :visitor="visitor"
    @leave="bootPhase = 'leaving'"
    @finish="bootPhase = 'done'"
  />
</template>
