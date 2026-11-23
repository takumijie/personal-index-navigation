import { computed, nextTick, onBeforeUnmount, ref, type Ref } from 'vue'
import { publicApi } from '@/api'
import { errorMessage } from '@/api/client'
import { createFastfetchData, type FastfetchData } from '@/components/terminal/fastfetch'
import { isShellCommand } from '@/utils/commandRecognition'
import { detectSystem } from '@/utils/system'
import type { GuestbookEntry, Settings, SystemInfo, Visitor } from '@/types'

export type TerminalPhase = 'initializing' | 'fastfetch' | 'fastfetch_loading' | 'clearing' | 'loading_guestbook' | 'interactive'
export interface TerminalLine {
  id: number
  text: string
  kind: 'plain' | 'muted' | 'fastfetch' | 'success' | 'error' | 'command'
  fastfetch?: FastfetchData
}

interface TerminalOptions {
  entries: Ref<GuestbookEntry[]>
  settings: Ref<Settings>
  visitor: Ref<Visitor | null>
  terminalError: Ref<string>
  ensureData: () => Promise<void>
}

export function useTerminal({ entries, settings, visitor, terminalError, ensureData }: TerminalOptions) {
  const phase = ref<TerminalPhase>('initializing')
  const lines = ref<TerminalLine[]>([])
  const input = ref('')
  const submitting = ref(false)
  const system = ref<SystemInfo>({ os: 'Detecting…', distribution: 'unknown', distributionKey: 'unknown', browserName: 'Detecting…', browserVersion: '' })
  const prompt = computed(() => `${system.value.os}@${visitor.value?.ip_address || 'unknown'}:~$`)
  const output = ref<HTMLElement | null>(null)
  const inputElement = ref<HTMLInputElement | null>(null)
  const history: string[] = []
  let historyIndex = 0
  let lineId = 0
  let runId = 0

  function reducedMotion() {
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches
  }

  function wait(ms: number, id: number, respectReducedMotion = true) {
    const duration = respectReducedMotion && reducedMotion() ? 10 : ms
    return new Promise<void>((resolve) => setTimeout(resolve, duration)).then(() => {
      if (id !== runId) throw new Error('cancelled')
    })
  }

  function append(text = '', kind: TerminalLine['kind'] = 'plain') {
    lines.value.push({ id: ++lineId, text, kind })
    scrollToBottom()
  }

  function appendFastfetch(data: FastfetchData, details = data.details) {
    const fastfetch = { ...data, details }
    lines.value.push({ id: ++lineId, text: '', kind: 'fastfetch', fastfetch })
    scrollToBottom()
    return fastfetch
  }

  function scrollToBottom() {
    void nextTick(() => {
      if (output.value) output.value.scrollTop = output.value.scrollHeight
    })
  }

  function publishedEntries() {
    return entries.value.filter((entry) => entry.status === 'published')
  }

  function entryLine(entry: GuestbookEntry) {
    const name = entry.system_name || entry.distribution || 'unknown'
    return `${name}@${entry.ip_address || 'unknown'}:~$ ${entry.message}`
  }

  function printGuestbook() {
    const published = publishedEntries()
    if (!published.length) {
      append('No messages yet. Be the first to say hello.', 'muted')
      return
    }
    published.forEach((entry) => append(entryLine(entry)))
  }

  async function start() {
    const id = ++runId
    phase.value = 'initializing'
    lines.value = []
    input.value = ''
    submitting.value = false
    historyIndex = history.length
    try {
      append('Initializing visitor session...')
      const detection = detectSystem().then((result) => {
        if (id === runId) system.value = result
      })
      await wait(460, id)
      append('Detecting operating system...', 'muted')
      await detection
      if (id !== runId) throw new Error('cancelled')
      await ensureData()
      if (id !== runId) throw new Error('cancelled')
      if (terminalError.value) append(`Visitor context warning: ${terminalError.value}`, 'error')
      await wait(360, id)
      append(`${prompt.value} ./fastfetch`, 'command')
      await wait(260, id)
      phase.value = 'fastfetch'
      const data = createFastfetchData(system.value, visitor.value || undefined)
      const fastfetch = appendFastfetch(data, [])
      for (const detail of data.details) {
        fastfetch.details.push(detail)
        await wait(105, id)
      }
      phase.value = 'fastfetch_loading'
      await wait(settings.value.terminal.fastfetch_loading_ms, id, false)
      phase.value = 'clearing'
      append(`${prompt.value} clear`, 'command')
      await wait(420, id)
      lines.value = []
      phase.value = 'loading_guestbook'
      append(`${prompt.value} cat /etc/guestbook/entries`, 'command')
      await wait(200, id)
      append('Loading guestbook...')
      await wait(240, id)
      append('Fetching previous visitors...', 'muted')
      await wait(200, id)
      append('Connecting to guestbook server...', 'muted')
      await wait(240, id)
      append(`Fetched ${publishedEntries().length} published message(s).`, 'success')
      await wait(280, id)
      printGuestbook()
      await wait(240, id)
      append('')
      append('Write a message and press Enter.', 'muted')
      phase.value = 'interactive'
      scrollToBottom()
    } catch (reason) {
      if (reason instanceof Error && reason.message === 'cancelled') return
      phase.value = 'interactive'
      append(`Startup warning: ${errorMessage(reason)}`, 'error')
      append("Type 'help' to continue.", 'muted')
    }
  }

  function runCommand(command: string): boolean {
    switch (command) {
      case 'help':
        append('Commands:')
        append('  help       Show this help')
        append('  fastfetch  Display system information')
        append('  guestbook  Show published messages')
        append('  about      About this space')
        append('  whoami     Current guest identity')
        append('  date       Current local date and time')
        append('  uname      Detected operating system')
        append('  clear      Clear terminal output')
        append('Plain text is submitted to the guestbook; shell commands are not executed.', 'muted')
        return true
      case 'fastfetch':
        appendFastfetch(createFastfetchData(system.value, visitor.value || undefined))
        return true
      case 'guestbook':
        printGuestbook()
        return true
      case 'about':
        append('A quiet personal corner on the web, built with Vue and a little twilight.')
        append('This terminal only accepts guestbook messages; it never executes shell commands.', 'muted')
        return true
      case 'whoami':
        append(`${system.value.os} · ${system.value.browserName} · ${visitor.value?.ip_address || 'IP pending'}`)
        return true
      case 'date':
        append(new Date().toString())
        return true
      case 'uname':
        append(system.value.os)
        return true
      case 'clear':
        lines.value = []
        return true
      default:
        return false
    }
  }

  async function submit() {
    if (phase.value !== 'interactive' || submitting.value) return
    const raw = input.value
    const message = raw.trim()
    if (!message) return
    history.push(raw)
    historyIndex = history.length
    append(`${prompt.value} ${raw}`, 'command')
    const command = message.toLowerCase()
    if (runCommand(command)) {
      input.value = ''
      return
    }
    if (isShellCommand(message)) {
      append('Permission denied', 'error')
      input.value = ''
      return
    }
    const maxLength = settings.value.guestbook_max_length
    if (message.length > maxLength) {
      append(`Message is too long (${message.length}/${maxLength}). Your input has been kept.`, 'error')
      return
    }
    submitting.value = true
    try {
      const saved = await publicApi.submitGuestbook({
        message,
        system_name: system.value.distributionKey === 'android' ? system.value.os : system.value.distributionKey,
        distribution: system.value.distribution,
        browser_name: system.value.browserName,
        browser_version: system.value.browserVersion,
      })
      if (saved.status === 'published') {
        entries.value = [...entries.value.filter((entry) => entry.id !== saved.id), saved]
        append('Message delivered. Thanks for leaving a little light here.', 'success')
      } else {
        append('Message received and awaiting moderation. Thank you.', 'success')
      }
      input.value = ''
    } catch (reason) {
      append(`Delivery failed: ${errorMessage(reason)}. Your input has been kept.`, 'error')
    } finally {
      submitting.value = false
      scrollToBottom()
    }
  }

  function navigateHistory(direction: 1 | -1, event: KeyboardEvent) {
    if (!history.length) return
    event.preventDefault()
    historyIndex = Math.max(0, Math.min(history.length, historyIndex + direction))
    input.value = history[historyIndex] || ''
  }

  onBeforeUnmount(() => { runId += 1 })
  return { phase, lines, input, submitting, system, prompt, output, inputElement, start, submit, scrollToBottom, navigateHistory }
}
