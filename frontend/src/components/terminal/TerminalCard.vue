<script setup lang="ts">
import { computed, onMounted, type Ref } from 'vue'
import { RotateCcw, TerminalSquare } from 'lucide-vue-next'
import { useTerminal } from '@/composables/useTerminal'
import type { GuestbookEntry, Settings, Visitor } from '@/types'

const props = defineProps<{ entries: GuestbookEntry[]; settings: Settings; visitor: Visitor | null }>()
const emit = defineEmits<{ 'update:entries': [entries: GuestbookEntry[]] }>()
const entriesProxy = computed({
  get: () => props.entries,
  set: (value: GuestbookEntry[]) => emit('update:entries', value),
})
const {
  phase, lines, input, submitting, output, inputElement, start, submit, navigateHistory,
} = useTerminal({
  entries: entriesProxy as Ref<GuestbookEntry[]>,
  settings: computed(() => props.settings),
  visitor: computed(() => props.visitor),
})

const phaseLabel = computed(() => ({
  initializing: 'INITIALIZING', fastfetch: 'FASTFETCH', clearing: 'CLEARING',
  loading_guestbook: 'LOADING GUESTBOOK', interactive: submitting.value ? 'SENDING' : 'INTERACTIVE',
}[phase.value]))

function lineClass(kind: string) {
  return {
    plain: 'text-[#c7d4d9]', muted: 'text-[#71848c]', ascii: 'text-[#a9d9cd]',
    success: 'text-[#80d7ac]', error: 'text-[#f0a0aa]', command: 'text-[#ddd3ff]',
  }[kind] || 'text-[#c7d4d9]'
}

function focusInput() {
  if (phase.value === 'interactive') inputElement.value?.focus()
}

onMounted(start)
</script>

<template>
  <section class="glass-card animate-enter flex min-h-[430px] flex-col overflow-hidden [animation-delay:80ms] lg:min-h-0 lg:flex-1" aria-label="交互终端">
    <header class="flex h-12 shrink-0 items-center justify-between border-b border-white/[0.07] bg-black/10 px-4">
      <div class="flex items-center gap-3">
        <div class="flex gap-1.5" aria-hidden="true">
          <span class="size-2.5 rounded-full bg-[#ff7b72]/75" />
          <span class="size-2.5 rounded-full bg-[#f2cc60]/75" />
          <span class="size-2.5 rounded-full bg-[#7ee2a8]/75" />
        </div>
        <div class="flex items-center gap-2 text-[11px] text-white/45"><TerminalSquare :size="13" /> visitor@personal-space: ~</div>
      </div>
      <div class="flex items-center gap-2">
        <span class="hidden items-center gap-1.5 font-mono text-[8px] tracking-[0.13em] text-white/28 min-[420px]:flex">
          <span class="size-1 rounded-full bg-emerald-300/70" :class="phase !== 'interactive' && 'animate-pulse'" />{{ phaseLabel }}
        </span>
        <button class="icon-button size-7" type="button" title="重新启动终端" aria-label="重新启动终端" @click="start"><RotateCcw :size="12" /></button>
      </div>
    </header>

    <div ref="output" class="terminal-body fine-scrollbar relative flex-1 overflow-auto px-4 py-4 font-mono text-[11px] leading-[1.65] min-[640px]:px-5 min-[640px]:text-[12px]" @click="focusInput">
      <div class="contents">
        <div v-for="line in lines" :key="line.id" class="min-h-[1.65em] whitespace-pre-wrap break-words" :class="lineClass(line.kind)">{{ line.text || ' ' }}</div>
      </div>

      <div v-if="phase === 'interactive'" class="flex min-h-[1.7em] items-start text-[#ddd3ff]">
        <span class="mr-2 shrink-0 text-[#a99ad8]">guest@homepage:~$</span>
        <div class="relative min-w-0 flex-1">
          <input
            ref="inputElement"
            v-model="input"
            type="text"
            class="relative z-10 w-full border-0 bg-transparent p-0 font-mono text-transparent caret-transparent outline-none disabled:opacity-60"
            :maxlength="Math.max(settings.guestbook_max_length, 1000)"
            :disabled="submitting"
            autocomplete="off"
            autocapitalize="off"
            spellcheck="false"
            aria-label="终端命令或留言"
            @keydown.enter.exact="!$event.isComposing && submit()"
            @keydown.up="navigateHistory(-1, $event)"
            @keydown.down="navigateHistory(1, $event)"
          />
          <div class="pointer-events-none absolute inset-0 whitespace-pre-wrap break-all text-[#e5defc]" aria-hidden="true">{{ input }}<span v-if="!submitting" class="terminal-cursor -mb-0.5 ml-0.5 inline-block h-[1.05em] w-[0.62em] translate-y-[0.16em] bg-[#cbbdf5]/85" /><span v-else class="ml-2 text-white/35">sending…</span></div>
        </div>
      </div>
      <div v-else class="mt-2 flex items-center gap-2 text-[#7b9199]"><span class="terminal-cursor inline-block h-[1em] w-[0.6em] bg-[#8ca79e]/70" /><span class="text-[9px] tracking-wider">{{ phaseLabel }}</span></div>
    </div>

    <footer class="flex h-8 shrink-0 items-center justify-between border-t border-white/[0.055] px-4 font-mono text-[8px] tracking-wider text-white/22">
      <span>UTF-8 · NO SHELL ACCESS</span>
      <span>↑↓ HISTORY · ENTER SUBMIT</span>
    </footer>
  </section>
</template>

<style scoped>
.terminal-body {
  background: linear-gradient(145deg, rgba(4, 12, 18, 0.53), rgba(8, 10, 22, 0.28));
  text-shadow: 0 0 12px rgba(169, 217, 205, 0.08);
}
</style>
