import { ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { publicApi } from '@/api'
import { createFastfetchData } from '@/components/terminal/fastfetch'
import { DEFAULT_SETTINGS } from '@/data/defaults'
import { detectSystem } from '@/utils/system'
import type { GuestbookEntry, SystemInfo, Visitor } from '@/types'
import { useTerminal } from './useTerminal'

vi.mock('@/utils/system', () => ({ detectSystem: vi.fn() }))

const systemInfo: SystemInfo = {
  os: 'Linux', distribution: 'linux', distributionKey: 'linux', browserName: 'Chrome', browserVersion: '130.0.0.0',
}

function setupTerminal() {
  const entries = ref<GuestbookEntry[]>([])
  const settings = ref(structuredClone(DEFAULT_SETTINGS))
  const visitor = ref<Visitor | null>({ ip_address: '127.0.0.1', visit_time: new Date().toISOString() })
  const terminalError = ref('')
  const ensureData = vi.fn().mockResolvedValue(undefined)
  const terminal = useTerminal({ entries, settings, visitor, terminalError, ensureData })
  terminal.phase.value = 'interactive'
  return { entries, settings, terminal, ensureData }
}

beforeEach(() => {
  vi.restoreAllMocks()
  vi.mocked(detectSystem).mockResolvedValue(systemInfo)
  vi.spyOn(console, 'warn').mockImplementation(() => {})
  vi.stubGlobal('window', {
    matchMedia: vi.fn().mockReturnValue({ matches: false }),
    screen: { width: 1920, height: 1080 }, innerWidth: 1280, innerHeight: 720,
  })
  vi.stubGlobal('navigator', { userAgent: 'Desktop', language: 'en-US' })
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('useTerminal guestbook submission', () => {
  it('keeps pending messages out of the public list and clears input after delivery', async () => {
    const { entries, terminal } = setupTerminal()
    const submit = vi.spyOn(publicApi, 'submitGuestbook').mockResolvedValue({
      id: 1,
      system_name: 'windows',
      distribution: 'windows',
      browser_name: 'Chrome',
      browser_version: '1',
      ip_address: '127.0.0.1',
      message: 'hello world',
      status: 'pending',
      created_at: new Date().toISOString(),
    })
    terminal.input.value = '  hello world  '

    await terminal.submit()

    expect(submit).toHaveBeenCalledWith(expect.objectContaining({ message: 'hello world' }))
    expect(terminal.input.value).toBe('')
    expect(entries.value).toEqual([])
    expect(terminal.lines.value.some((line) => line.text.includes('awaiting moderation'))).toBe(true)
  })

  it('keeps the original input when delivery fails', async () => {
    const { terminal } = setupTerminal()
    vi.spyOn(publicApi, 'submitGuestbook').mockRejectedValue(new Error('network offline'))
    terminal.input.value = 'keep this message'

    await terminal.submit()

    expect(terminal.input.value).toBe('keep this message')
    expect(terminal.submitting.value).toBe(false)
    expect(terminal.lines.value.some((line) => line.kind === 'error' && line.text.includes('input has been kept'))).toBe(true)
  })

  it('runs local commands without sending them as guestbook messages', async () => {
    const { terminal } = setupTerminal()
    const submit = vi.spyOn(publicApi, 'submitGuestbook')
    terminal.input.value = 'help'

    await terminal.submit()

    expect(submit).not.toHaveBeenCalled()
    expect(terminal.input.value).toBe('')
    expect(terminal.lines.value.some((line) => line.text.includes('Commands:'))).toBe(true)
  })

  it('rejects overlong messages before the API request and preserves input', async () => {
    const { settings, terminal } = setupTerminal()
    const submit = vi.spyOn(publicApi, 'submitGuestbook')
    settings.value.guestbook_max_length = 5
    terminal.input.value = '123456'

    await terminal.submit()

    expect(submit).not.toHaveBeenCalled()
    expect(terminal.input.value).toBe('123456')
    expect(terminal.lines.value.some((line) => line.kind === 'error' && line.text.includes('6/5'))).toBe(true)
  })

  it('uses the detected OS and visitor IP in command echoes and whoami', async () => {
    const { terminal } = setupTerminal()
    terminal.system.value = systemInfo
    terminal.input.value = 'whoami'

    await terminal.submit()

    expect(terminal.lines.value[0].text).toBe('Linux@127.0.0.1:~$ whoami')
    expect(terminal.lines.value[1].text).toBe('Linux · Chrome · 127.0.0.1')
  })

  it('renders manual fastfetch as one complete block with its color logo', async () => {
    const { terminal } = setupTerminal()
    terminal.system.value = systemInfo
    terminal.input.value = 'fastfetch'
    const expectedDetails = createFastfetchData(systemInfo, {
      ip_address: '127.0.0.1', visit_time: new Date().toISOString(),
    }).details.length

    await terminal.submit()

    const blocks = terminal.lines.value.filter((line) => line.kind === 'fastfetch')
    expect(blocks).toHaveLength(1)
    expect(blocks[0].fastfetch?.identity).toBe('Linux@127.0.0.1')
    expect(blocks[0].fastfetch?.logo).toContain('linux.png')
    expect(blocks[0].fastfetch?.details).toHaveLength(expectedDetails)
  })
})

describe('useTerminal startup', () => {
  it('holds fastfetch for the configured duration before clearing and printing messages', async () => {
    vi.useFakeTimers()
    const { settings, terminal, ensureData } = setupTerminal()
    settings.value.terminal.fastfetch_loading_ms = 2000

    const expectedDetails = createFastfetchData(systemInfo, {
      ip_address: '127.0.0.1', visit_time: new Date().toISOString(),
    }).details.length
    const running = terminal.start()
    await vi.advanceTimersByTimeAsync(460 + 360 + 260)
    expect(terminal.phase.value).toBe('fastfetch')
    const block = terminal.lines.value.find((line) => line.kind === 'fastfetch')
    const renderedDetails = block?.fastfetch?.details.length || 0
    expect(renderedDetails).toBeLessThan(expectedDetails)
    await vi.advanceTimersByTimeAsync(expectedDetails * 105)

    expect(terminal.phase.value).toBe('fastfetch_loading')
    expect(block?.fastfetch?.details).toHaveLength(expectedDetails)
    expect(terminal.lines.value.filter((line) => line.kind === 'fastfetch')).toHaveLength(1)
    expect(terminal.lines.value.some((line) => line.text === 'Linux@127.0.0.1:~$ ./fastfetch')).toBe(true)
    const heldLines = terminal.lines.value
    await vi.advanceTimersByTimeAsync(1999)
    expect(terminal.phase.value).toBe('fastfetch_loading')
    expect(terminal.lines.value).toEqual(heldLines)

    await vi.advanceTimersByTimeAsync(1)
    expect(terminal.phase.value).toBe('clearing')
    await vi.advanceTimersByTimeAsync(420)
    expect(terminal.phase.value).toBe('loading_guestbook')
    expect(terminal.lines.value.some((line) => line.kind === 'fastfetch')).toBe(false)

    await vi.advanceTimersByTimeAsync(1400)
    await running
    expect(terminal.phase.value).toBe('interactive')
    expect(terminal.lines.value.some((line) => line.text === 'Write a message and press Enter.')).toBe(true)
    expect(ensureData).toHaveBeenCalledTimes(1)
  })

  it('cancels the previous startup sequence when restarted', async () => {
    vi.useFakeTimers()
    const { terminal } = setupTerminal()

    const firstRun = terminal.start()
    await vi.advanceTimersByTimeAsync(100)
    const secondRun = terminal.start()
    await vi.advanceTimersByTimeAsync(10000)
    await Promise.all([firstRun, secondRun])

    expect(terminal.phase.value).toBe('interactive')
    expect(terminal.lines.value.filter((line) => line.text.includes('published message(s)'))).toHaveLength(1)
  })
})
