<script setup lang="ts">
import { watchEffect } from 'vue'
import { ArrowUpRight, CloudOff, MoonStar, RefreshCw, SunMedium } from 'lucide-vue-next'
import BackgroundScene from '@/components/common/BackgroundScene.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import ProfileCard from '@/components/profile/ProfileCard.vue'
import NavigationGrid from '@/components/navigation/NavigationGrid.vue'
import DeviceGrid from '@/components/devices/DeviceGrid.vue'
import TerminalCard from '@/components/terminal/TerminalCard.vue'
import ClockPanel from '@/components/clock/ClockPanel.vue'
import { useBackground } from '@/composables/useBackground'
import { useHomeData } from '@/composables/useHomeData'

const { profile, navigation, devices, settings, guestbook, tracks, visitor, loading, error, reload } = useHomeData()
const { mode, activeUrl, loaded, duration } = useBackground(settings)

watchEffect(() => { document.title = settings.value.site_title })
</script>

<template>
  <div class="relative min-h-screen text-white">
    <BackgroundScene
      :day-url="settings.background.day_url"
      :night-url="settings.background.night_url"
      :mode="mode"
      :loaded="loaded"
      :duration="duration"
    />

    <header class="mx-auto flex h-[68px] w-full max-w-[1500px] items-center justify-between px-4 min-[640px]:px-6">
      <a href="/" class="group flex items-center gap-2.5" aria-label="返回主页">
        <span class="flex size-8 items-center justify-center rounded-xl border border-white/10 bg-white/[0.07] shadow-lg backdrop-blur-xl">
          <span class="size-2.5 rotate-45 rounded-[3px] border border-violet-200/70 bg-violet-300/20 transition group-hover:rotate-[135deg]" />
        </span>
        <div>
          <p class="max-w-[220px] truncate text-[12px] font-medium tracking-wide text-white/75">{{ settings.site_title }}</p>
          <p class="text-[8px] tracking-[0.22em] text-white/30">A QUIET DIGITAL CORNER</p>
        </div>
      </a>
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

    <main v-if="profile" class="mx-auto grid w-full max-w-[1500px] gap-4 px-4 pb-5 min-[640px]:px-6 lg:h-[calc(100vh-76px)] lg:min-h-[720px] lg:grid-cols-[330px_minmax(0,1fr)] xl:grid-cols-[350px_minmax(0,1fr)]">
      <div class="fine-scrollbar flex min-h-0 flex-col gap-4 lg:overflow-y-auto lg:pr-1">
        <ProfileCard
          :profile="profile"
          :active-background="activeUrl"
          :day-background="settings.background.day_url"
          :night-background="settings.background.night_url"
          class="shrink-0"
        />
        <NavigationGrid :items="navigation" class="shrink-0" />
        <DeviceGrid :devices="devices" class="shrink-0" />
        <p class="mt-auto hidden px-2 pb-1 text-[9px] tracking-wider text-white/20 lg:block">DESIGNED FOR SLOW EVENINGS · {{ new Date().getFullYear() }}</p>
      </div>

      <div class="flex min-h-0 flex-col gap-4">
        <TerminalCard v-model:entries="guestbook" :settings="settings" :visitor="visitor" />
        <ClockPanel :settings="settings" :tracks="tracks" class="shrink-0" />
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
</template>
