import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { publicApi } from '@/api'
import { DEFAULT_SETTINGS } from '@/data/defaults'
import type { GuestbookEntry, Visitor } from '@/types'
import { useTerminal } from './useTerminal'

function setupTerminal() {
  const entries = ref<GuestbookEntry[]>([])
  const settings = ref(structuredClone(DEFAULT_SETTINGS))
  const visitor = ref<Visitor | null>({ ip: '127.0.0.1', visit_time: new Date().toISOString() })
  const terminal = useTerminal({ entries, settings, visitor })
  terminal.phase.value = 'interactive'
  return { entries, settings, terminal }
}

beforeEach(() => {
  vi.restoreAllMocks()
  vi.spyOn(console, 'warn').mockImplementation(() => {})
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
})
