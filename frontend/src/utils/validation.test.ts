import { describe, expect, it } from 'vitest'
import { validClockTime, validNavigationUrl, validTimezone, validWebUrl } from './validation'

describe('admin form validation', () => {
  it('accepts remote resources and local uploaded resources only over supported URLs', () => {
    expect(validWebUrl('https://example.com/image.webp')).toBe(true)
    expect(validWebUrl('/storage/images/upload.png')).toBe(true)
    expect(validWebUrl('javascript:alert(1)')).toBe(false)
    expect(validWebUrl('not a url')).toBe(false)
  })

  it('supports HTTP and email navigation but rejects script URLs', () => {
    expect(validNavigationUrl('https://example.com')).toBe(true)
    expect(validNavigationUrl('mailto:hello@example.com')).toBe(true)
    expect(validNavigationUrl('javascript:alert(1)')).toBe(false)
  })

  it('validates timezone names and strict 24-hour clock values', () => {
    expect(validTimezone('Asia/Shanghai')).toBe(true)
    expect(validTimezone('Invalid/Timezone')).toBe(false)
    expect(validClockTime('18:30')).toBe(true)
    expect(validClockTime('24:00')).toBe(false)
    expect(validClockTime('6:00')).toBe(false)
  })
})
