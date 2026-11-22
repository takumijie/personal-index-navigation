import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { SystemInfo, Visitor } from '@/types'
import { createFastfetchData } from './fastfetch'

const system: SystemInfo = {
  os: 'Windows', distribution: 'windows', distributionKey: 'windows', browserName: 'Chrome', browserVersion: '130.0.0.0',
}
const visitor: Visitor = { ip_address: '192.168.1.109', visit_time: '2026-11-22T14:00:00+08:00' }

beforeEach(() => {
  vi.stubGlobal('navigator', { userAgent: 'Desktop', language: 'zh-CN' })
  vi.stubGlobal('window', { screen: { width: 1536, height: 864 }, innerWidth: 1280, innerHeight: 720 })
})

afterEach(() => vi.unstubAllGlobals())

describe('createFastfetchData', () => {
  it.each([
    ['windows', 'windows.png'],
    ['macos', 'macos.png'],
    ['ios', 'ios.png'],
    ['ipados', 'ios.png'],
    ['android', 'android.png'],
    ['linux', 'linux.png'],
    ['chromeos', 'chromeos.png'],
    ['freebsd', 'freebsd.png'],
  ])('maps the generic %s system to its color logo', (key, filename) => {
    const data = createFastfetchData({ ...system, distributionKey: key }, visitor)

    expect(data.logo).toContain(filename)
  })

  it('uses the visitor identity and structured details without distribution claims', () => {
    const data = createFastfetchData(system, visitor)

    expect(data.identity).toBe('Windows@192.168.1.109')
    expect(data.details).toContainEqual({ label: 'OS', value: 'Windows' })
    expect(data.details).toContainEqual({ label: 'IP Address', value: '192.168.1.109' })
    expect(data.details).toContainEqual({ label: 'Visit Time', value: visitor.visit_time })
    expect(data.details.some((detail) => detail.label === 'Distribution')).toBe(false)
  })

  it('does not guess a logo for an unknown system', () => {
    const data = createFastfetchData({ ...system, os: 'Unknown OS', distributionKey: 'unknown' })

    expect(data.logo).toBeNull()
    expect(data.identity).toBe('Unknown OS@unknown')
    expect(data.details).toContainEqual({ label: 'IP Address', value: 'unknown' })
  })
})
