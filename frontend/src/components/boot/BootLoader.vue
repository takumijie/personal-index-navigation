<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { LoaderSettings, Visitor } from '@/types'
import { DEFAULT_WATERMARK_CAPTION, DEFAULT_WATERMARK_SUBTITLE, DEFAULT_WATERMARK_TITLE } from '@/data/defaults'
import { detectSystem } from '@/utils/system'
import { imageToHalftone } from '@/utils/halftone'

const props = defineProps<{
  ready: boolean
  loader: LoaderSettings
  visitor: Visitor | null
}>()
const emit = defineEmits<{ leave: []; finish: [] }>()

const watermarkTitle = computed(() => props.loader.watermark_title || DEFAULT_WATERMARK_TITLE)
const watermarkSubtitle = computed(() => props.loader.watermark_subtitle || DEFAULT_WATERMARK_SUBTITLE)
const watermarkCaption = computed(() => props.loader.watermark_caption || DEFAULT_WATERMARK_CAPTION)

/* ---------------- 音效（Web Audio 合成，默认关闭） ---------------- */
class SpaceAudioSynth {
  private ctx: AudioContext | null = null
  enabled = false

  private init() {
    if (!this.ctx) {
      const AudioCtor = window.AudioContext || (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
      if (AudioCtor) this.ctx = new AudioCtor()
    }
    if (this.ctx?.state === 'suspended') void this.ctx.resume()
  }

  unlock() {
    this.init()
  }

  toggle() {
    this.enabled = !this.enabled
    if (this.enabled) {
      this.init()
      this.playChime()
    }
    return this.enabled
  }

  playTick(pitch = 1) {
    if (!this.enabled || !this.ctx) return
    try {
      const now = this.ctx.currentTime
      const osc = this.ctx.createOscillator()
      const gain = this.ctx.createGain()
      osc.type = 'sine'
      osc.frequency.setValueAtTime(1320 * pitch, now)
      osc.frequency.exponentialRampToValueAtTime(680, now + 0.028)
      gain.gain.setValueAtTime(0.03, now)
      gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.032)
      osc.connect(gain)
      gain.connect(this.ctx.destination)
      osc.start(now)
      osc.stop(now + 0.035)
    } catch { /* 忽略音频异常 */ }
  }

  playChime() {
    if (!this.enabled || !this.ctx) return
    try {
      const now = this.ctx.currentTime
      const notes = [587.33, 880, 1174.66, 1318.51]
      notes.forEach((freq, index) => {
        const osc = this.ctx!.createOscillator()
        const gain = this.ctx!.createGain()
        osc.type = 'sine'
        osc.frequency.setValueAtTime(freq, now + index * 0.045)
        gain.gain.setValueAtTime(0.038, now + index * 0.045)
        gain.gain.exponentialRampToValueAtTime(0.0001, now + index * 0.045 + 0.36)
        osc.connect(gain)
        gain.connect(this.ctx!.destination)
        osc.start(now + index * 0.045)
        osc.stop(now + index * 0.045 + 0.38)
      })
    } catch { /* 忽略音频异常 */ }
  }

  dispose() {
    void this.ctx?.close().catch(() => {})
    this.ctx = null
  }
}
const synth = new SpaceAudioSynth()
const soundEnabled = ref(false)
let soundTouched = false

watch(() => props.loader.sound_enabled, (enabled) => {
  if (soundTouched) return
  soundEnabled.value = enabled
  synth.enabled = enabled
}, { immediate: true })

function removeAudioUnlockListeners() {
  window.removeEventListener('pointerdown', unlockAudio)
  window.removeEventListener('keydown', unlockAudio)
}

function unlockAudio() {
  if (!soundEnabled.value) return
  synth.unlock()
  removeAudioUnlockListeners()
}

function toggleSound() {
  soundTouched = true
  removeAudioUnlockListeners()
  soundEnabled.value = synth.toggle()
}

/* ---------------- 徽章：内置 SVG / 点阵化 / 原图降级 ---------------- */
const badgeMode = ref<'builtin' | 'halftone' | 'original'>('builtin')
const halftoneHost = ref<HTMLElement | null>(null)
let badgeRunId = 0

watch([() => props.loader.badge_image, () => props.loader.theme], ([url, theme]) => {
  const runId = ++badgeRunId
  badgeMode.value = 'builtin'
  if (!url) return
  const image = new Image()
  image.crossOrigin = 'anonymous'
  image.onload = () => {
    if (runId !== badgeRunId) return
    try {
      const canvas = imageToHalftone(image, {
        maxWidth: 150,
        cellSize: 4,
        color: theme === 'beige' ? '#3b3830' : '#e6e3d3',
      })
      canvas.style.width = '120px'
      canvas.style.height = 'auto'
      badgeMode.value = 'halftone'
      // 等待宿主元素渲染后挂载点阵画布
      requestAnimationFrame(() => {
        if (runId !== badgeRunId || !halftoneHost.value) return
        halftoneHost.value.replaceChildren(canvas)
      })
    } catch {
      // 跨域图片无法读取像素时降级为原图
      badgeMode.value = 'original'
    }
  }
  image.onerror = () => {
    if (runId !== badgeRunId) return
    if (image.crossOrigin === 'anonymous') {
      const fallback = new Image()
      fallback.onload = () => {
        if (runId === badgeRunId) badgeMode.value = 'original'
      }
      fallback.onerror = () => {
        if (runId === badgeRunId) badgeMode.value = 'builtin'
      }
      fallback.src = url
      return
    }
    badgeMode.value = 'builtin'
  }
  image.src = url
}, { immediate: true })

/* ---------------- 加载动画状态 ---------------- */
const logContainer = ref<HTMLElement | null>(null)
const progress = ref(0)
const phaseLabel = ref('SPATIAL LINK HANDSHAKE')
const subtitle = ref('正在同步个人空间')
const waitingData = ref(false)
const leaving = ref(false)
const topClock = ref('SYNC // --:--:--.---')
let cancelled = false
let leaveTimer: ReturnType<typeof setTimeout> | undefined
let clockTimer: ReturnType<typeof setInterval> | undefined
const scrambleFrames = new Set<number>()

const GLYPHS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/#*+=-:<>'
function scrambleRevealText(element: HTMLElement, finalText: string, duration = 200) {
  const start = performance.now()
  function frame(now: number) {
    if (cancelled) return
    const ratio = Math.min(1, (now - start) / duration)
    const revealed = Math.floor(ratio * finalText.length)
    let out = ''
    for (let i = 0; i < finalText.length; i++) {
      const ch = finalText[i]
      out += i < revealed || ch === ' ' || ch === ':' ? ch : GLYPHS[Math.floor(Math.random() * GLYPHS.length)]
    }
    element.textContent = out
    if (ratio < 1) {
      scrambleFrames.add(requestAnimationFrame(frame))
    } else {
      element.textContent = finalText
    }
  }
  scrambleFrames.add(requestAnimationFrame(frame))
}

function delay(ms: number) {
  return new Promise<void>((resolve) => setTimeout(resolve, ms))
}

function leave() {
  if (leaving.value) return
  leaving.value = true
  cancelled = true
  emit('leave')
  leaveTimer = setTimeout(() => emit('finish'), 680)
}

async function runSequence() {
  const system = await detectSystem().catch(() => null)
  if (cancelled) return

  const now = new Date()
  const pad = (n: number, len = 2) => String(n).padStart(len, '0')
  const timestamp = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())} [${Intl.DateTimeFormat().resolvedOptions().timeZone || 'LOCAL'}]`
  const ip = props.visitor?.ip_address || 'LOCAL-NODE'
  const os = system ? `${system.os} // ${system.browserName} ${system.browserVersion}`.trim() : 'Unknown OS'
  const resolution = `${window.screen.width}×${window.screen.height}`
  const cores = navigator.hardwareConcurrency ? `${navigator.hardwareConcurrency} 线程` : '未知算力'

  const sequence: { text: string; delay: number; phase?: string; highlight?: boolean; bold?: boolean; final?: boolean }[] = [
    { text: '正在初始化空间链路协议...', delay: 130, phase: 'INITIALIZING SPATIAL PROTOCOL' },
    { text: `终端节点识别: ${ip}`, delay: 190, highlight: true },
    { text: `纪元时间戳校验: ${timestamp}`, delay: 160, highlight: true },
    { text: `访客系统环境: ${os}`, delay: 170, highlight: true },
    { text: `视口与算力矩阵: ${resolution} (${cores})`, delay: 120 },
    { text: '记忆体扇区校验: 正常', delay: 110, phase: 'SYNCHRONIZING ARCHIVE SECTORS' },
    { text: '空间次元坐标锁定: 稳定', delay: 120 },
    { text: '灵感与观测数据流: 已就绪', delay: 120 },
    { text: '个人归档库索引加载: 100%', delay: 130 },
    { text: '情感共鸣波段同步: 正常', delay: 120, phase: 'VERIFYING RESONANCE BAND' },
    { text: '数字精神角落屏障: 已激活', delay: 110 },
    { text: '空间门扉协议认证: 完成', delay: 140, bold: true },
    { text: '个人空间连接就绪', delay: 200, bold: true, final: true },
  ]

  for (let i = 0; i < sequence.length; i++) {
    if (cancelled) return
    const step = sequence[i]
    await delay(step.delay)
    if (cancelled) return
    if (step.phase) phaseLabel.value = step.phase

    const container = logContainer.value
    if (!container) return
    const row = document.createElement('div')
    row.className = 'log-line flex items-start gap-2'
    const textSpan = document.createElement('span')
    if (step.highlight) {
      const prefix = document.createElement('span')
      prefix.className = 'select-none font-mono-tech opacity-60'
      prefix.textContent = '>'
      textSpan.className = 'underline decoration-current/35 underline-offset-4'
      row.appendChild(prefix)
    } else if (step.bold) {
      textSpan.className = 'font-medium tracking-[0.15em]'
    }
    row.appendChild(textSpan)
    container.appendChild(row)
    scrambleRevealText(textSpan, step.text, step.highlight ? 240 : 170)
    synth.playTick(step.final ? 1.25 : 0.94 + (i % 5) * 0.04)
    progress.value = Math.round(((i + 1) / sequence.length) * 100)
  }

  subtitle.value = '个人空间同步完成'
  phaseLabel.value = 'CONNECTION ESTABLISHED // 100%'
  synth.playChime()
  if (props.ready) {
    leaveTimer = setTimeout(leave, 450)
  } else {
    waitingData.value = true
  }
}

watch(() => props.ready, (ready) => {
  if (ready && progress.value >= 100 && !leaving.value) {
    waitingData.value = false
    leaveTimer = setTimeout(leave, 350)
  }
})

onMounted(() => {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    emit('finish')
    return
  }
  window.addEventListener('pointerdown', unlockAudio)
  window.addEventListener('keydown', unlockAudio)
  clockTimer = setInterval(() => {
    const now = new Date()
    const pad = (n: number, len = 2) => String(n).padStart(len, '0')
    topClock.value = `SYNC // ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}.${pad(now.getMilliseconds(), 3)}`
  }, 83)
  void runSequence()
})

onBeforeUnmount(() => {
  cancelled = true
  if (leaveTimer) clearTimeout(leaveTimer)
  if (clockTimer) clearInterval(clockTimer)
  scrambleFrames.forEach((id) => cancelAnimationFrame(id))
  scrambleFrames.clear()
  removeAudioUnlockListeners()
  synth.dispose()
})
</script>

<template>
  <div class="boot-root yorha-grid-bg fixed inset-0 z-50 antialiased" :class="{ 'boot-leaving': leaving, 'theme-beige': loader.theme === 'beige' }">
    <div class="crt-scanlines pointer-events-none fixed inset-0 z-40" />
    <div class="crt-vignette pointer-events-none fixed inset-0 z-40" />

    <!-- 居中水印：徽章 + 三行可配置文字 -->
    <div class="space-watermark flex w-[92vw] max-w-[820px] select-none flex-col items-center justify-center">
      <div class="flex flex-col items-center justify-center gap-5 sm:flex-row sm:gap-8">
        <div class="flex items-center justify-center text-current">
          <svg v-if="badgeMode === 'builtin'" class="h-28 w-24 sm:h-36 sm:w-32" viewBox="0 0 160 190" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
            <circle cx="80" cy="92" r="54" fill="none" stroke="currentColor" stroke-width="2" stroke-dasharray="10 5 2 5" />
            <circle cx="80" cy="92" r="42" fill="none" stroke="currentColor" stroke-width="1.2" opacity="0.75" />
            <line x1="80" y1="12" x2="80" y2="172" stroke="currentColor" stroke-width="2" />
            <line x1="18" y1="92" x2="142" y2="92" stroke="currentColor" stroke-width="1.5" />
            <path d="M80,38 L94,76 L132,92 L94,108 L80,146 L66,108 L28,92 L66,76 Z" fill="none" stroke="currentColor" stroke-width="2.2" />
            <path d="M80,54 L89,81 L116,92 L89,103 L80,130 L71,103 L44,92 L71,81 Z" fill="currentColor" opacity="0.85" />
            <circle cx="80" cy="92" r="6" fill="var(--boot-bg)" stroke="currentColor" stroke-width="2.5" />
            <polygon points="80,4 85,12 80,20 75,12" fill="currentColor" />
            <polygon points="80,164 85,172 80,180 75,172" fill="currentColor" />
          </svg>
          <div v-else-if="badgeMode === 'halftone'" ref="halftoneHost" class="flex items-center justify-center" />
          <img v-else :src="loader.badge_image" alt="空间徽章" class="max-h-36 w-auto max-w-[140px] object-contain" />
        </div>
        <div class="flex flex-col items-center text-center sm:items-start sm:text-left">
          <div class="text-3xl font-normal uppercase leading-tight tracking-[0.16em] sm:text-5xl md:text-6xl">{{ watermarkTitle }}</div>
          <div class="font-mono-tech mt-1.5 text-xs uppercase tracking-[0.26em] opacity-90 sm:mt-2 sm:text-base md:text-lg">{{ watermarkSubtitle }}</div>
          <div class="mt-1 text-[11px] font-light tracking-[0.2em] opacity-75 sm:text-xs">{{ watermarkCaption }}</div>
        </div>
      </div>
    </div>

    <!-- 右上角控制栏 -->
    <header class="fixed right-4 top-4 z-50 flex items-center gap-2.5 sm:right-8 sm:top-7 sm:gap-3.5">
      <div class="font-mono-tech mr-1 hidden text-xs tracking-widest opacity-65 lg:block">{{ topClock }}</div>
      <button type="button" title="开启/关闭空间终端电子提示音" class="yorha-btn font-mono-tech flex cursor-pointer items-center gap-1.5 px-2.5 py-1.5 text-[11px] tracking-wider sm:px-3 sm:text-xs" @click="toggleSound">
        <span class="diamond-marker" /><span>{{ soundEnabled ? 'AUDIO: ON' : 'AUDIO: OFF' }}</span>
      </button>
      <button type="button" title="跳过自检动画直接进入个人空间" class="yorha-btn font-mono-tech flex cursor-pointer items-center gap-1.5 px-2.5 py-1.5 text-[11px] tracking-wider sm:px-3 sm:text-xs" @click="leave">
        <span>SKIP &gt;&gt;</span>
      </button>
      <div class="relative ml-0.5 flex size-8 items-center justify-center sm:size-9" title="空间观测节点同步中">
        <svg class="size-full" viewBox="0 0 40 40" aria-hidden="true">
          <circle cx="20" cy="20" r="16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-dasharray="22 8 6 8" class="radar-ring-outer opacity-70" />
          <circle cx="20" cy="20" r="10.5" fill="none" stroke="currentColor" stroke-width="2" stroke-dasharray="18 12" class="radar-ring-inner opacity-90" />
          <circle cx="20" cy="20" r="3.8" fill="currentColor" class="opacity-85" />
        </svg>
      </div>
    </header>

    <!-- 加载主视图 -->
    <main class="relative z-10 flex min-h-screen flex-col justify-between px-6 py-8 sm:px-14 sm:py-12 md:px-20 md:py-14">
      <div>
        <div class="chromatic-ghost mb-6 flex flex-wrap items-baseline gap-x-3 gap-y-1 sm:mb-8">
          <h1 class="text-2xl font-normal tracking-[0.26em] sm:text-3xl md:text-[34px]">正在连接</h1>
          <span class="text-sm font-light tracking-[0.14em] opacity-90 sm:text-base md:text-lg">
            . <span>{{ subtitle }}</span><span class="loading-dot">.</span><span class="loading-dot">.</span><span class="loading-dot">.</span><span class="cursor-block" />
          </span>
        </div>
        <div ref="logContainer" class="chromatic-ghost w-full max-w-3xl space-y-1.5 text-xs font-normal leading-relaxed tracking-[0.09em] sm:space-y-2 sm:text-sm md:text-[15px]" />
      </div>

      <div class="mt-8 max-w-3xl pt-4">
        <div class="mb-4">
          <div class="font-mono-tech mb-1.5 flex justify-between text-[11px] tracking-widest opacity-75">
            <span>{{ phaseLabel }}</span><span>{{ progress }}%</span>
          </div>
          <div class="h-[2px] w-full overflow-hidden bg-current/15">
            <div class="h-full bg-current transition-all duration-150 ease-out" :style="{ width: `${progress}%` }" />
          </div>
        </div>
        <p v-if="waitingData" class="font-mono-tech text-[11px] tracking-widest opacity-60">WAITING FOR HOMEPAGE DATA // 正在等待主页数据就绪...</p>
      </div>
    </main>
  </div>
</template>

<style scoped>
.boot-root {
  --boot-bg: #0a0a09;
  --boot-text: #e6e3d3;
  --boot-muted: #8f8b7a;
  --boot-accent-dim: rgba(218, 212, 187, 0.12);
  --boot-border: rgba(218, 212, 187, 0.28);
  --boot-grid: rgba(255, 255, 255, 0.026);
  --boot-glow: 0 0 8px rgba(230, 227, 211, 0.22);
  --boot-watermark-opacity: 0.08;
  background-color: var(--boot-bg);
  color: var(--boot-text);
  font-family: var(--font-sans);
  transition: opacity 0.65s ease, transform 0.65s ease;
}
.boot-root.theme-beige {
  --boot-bg: #dad4bb;
  --boot-text: #3b3830;
  --boot-muted: #6e695a;
  --boot-accent-dim: rgba(59, 56, 48, 0.1);
  --boot-border: rgba(59, 56, 48, 0.35);
  --boot-grid: rgba(59, 56, 48, 0.05);
  --boot-glow: none;
  --boot-watermark-opacity: 0.065;
}
.boot-root.theme-beige .crt-vignette {
  background: radial-gradient(circle at center, transparent 64%, rgba(59, 56, 48, 0.22) 100%);
}
.boot-root.theme-beige .chromatic-ghost {
  text-shadow: 0.8px 0.8px 0 rgba(59, 56, 48, 0.12);
}
.boot-root.boot-leaving {
  opacity: 0;
  transform: scale(1.015);
  pointer-events: none;
}
.font-mono-tech {
  font-family: var(--font-mono);
}
.yorha-grid-bg {
  background-image:
    radial-gradient(var(--boot-grid) 1px, transparent 1px),
    linear-gradient(to right, var(--boot-grid) 1px, transparent 1px),
    linear-gradient(to bottom, var(--boot-grid) 1px, transparent 1px);
  background-size: 16px 16px, 4px 4px, 4px 4px;
}
.crt-scanlines {
  background: linear-gradient(rgba(18, 16, 16, 0) 50%, rgba(0, 0, 0, 0.24) 50%);
  background-size: 100% 3px;
}
.crt-vignette {
  background: radial-gradient(circle at center, transparent 56%, rgba(0, 0, 0, 0.68) 100%);
}
.chromatic-ghost {
  text-shadow:
    -1.2px 0 0 rgba(255, 95, 95, 0.24),
    1.2px 0 0 rgba(95, 220, 255, 0.24),
    0 -0.6px 0 rgba(230, 227, 211, 0.16),
    var(--boot-glow);
}
@keyframes watermarkPulse {
  0%, 100% { opacity: var(--boot-watermark-opacity); transform: translate(-50%, -50%) scale(1); }
  50% { opacity: calc(var(--boot-watermark-opacity) * 1.35); transform: translate(-50%, -50%) scale(1.01); }
}
.space-watermark {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  opacity: var(--boot-watermark-opacity);
  animation: watermarkPulse 6.5s ease-in-out infinite;
  pointer-events: none;
  z-index: 1;
}
@keyframes spinSlow { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
@keyframes spinReverse { from { transform: rotate(360deg); } to { transform: rotate(0deg); } }
.radar-ring-outer { animation: spinSlow 9s linear infinite; transform-origin: center; }
.radar-ring-inner { animation: spinReverse 4.5s linear infinite; transform-origin: center; }
@keyframes dotBlink {
  0%, 20% { opacity: 0.2; }
  50% { opacity: 1; }
  80%, 100% { opacity: 0.2; }
}
.loading-dot:nth-child(1) { animation: dotBlink 1.4s infinite 0s; }
.loading-dot:nth-child(2) { animation: dotBlink 1.4s infinite 0.25s; }
.loading-dot:nth-child(3) { animation: dotBlink 1.4s infinite 0.5s; }
@keyframes cursorBlink {
  0%, 49% { opacity: 1; }
  50%, 100% { opacity: 0; }
}
.cursor-block {
  display: inline-block;
  width: 0.55em;
  height: 1.05em;
  background-color: currentColor;
  vertical-align: -0.15em;
  margin-left: 6px;
  animation: cursorBlink 0.9s infinite;
}
@keyframes logReveal {
  0% { opacity: 0; transform: translateX(-8px); filter: blur(1.2px); }
  100% { opacity: 1; transform: translateX(0); filter: blur(0); }
}
.log-line { animation: logReveal 0.18s cubic-bezier(0.22, 1, 0.36, 1) forwards; }
.yorha-btn {
  position: relative;
  border: 1px solid var(--boot-border);
  background: var(--boot-accent-dim);
  color: var(--boot-text);
  transition: all 0.22s ease;
  overflow: hidden;
  user-select: none;
}
.yorha-btn::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: var(--boot-text);
  transition: left 0.24s cubic-bezier(0.25, 1, 0.5, 1);
  z-index: 0;
}
.yorha-btn:hover::before, .yorha-btn:focus-visible::before { left: 0; }
.yorha-btn:hover, .yorha-btn:focus-visible { color: var(--boot-bg); border-color: var(--boot-text); }
.yorha-btn > span { position: relative; z-index: 1; }
.diamond-marker {
  display: inline-block;
  width: 7px;
  height: 7px;
  background-color: currentColor;
  transform: rotate(45deg);
  transition: transform 0.25s ease;
}
.yorha-btn:hover .diamond-marker { transform: rotate(225deg) scale(1.15); }
</style>
