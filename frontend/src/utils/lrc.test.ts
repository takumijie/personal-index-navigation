import { describe, expect, it } from 'vitest'
import { findCurrentLrcLine, parseLrc } from './lrc'

describe('parseLrc', () => {
  it('expands multiple timestamps and normalizes fractional seconds', () => {
    expect(parseLrc('[00:10.00][00:02.5]重复歌词\n[00:03.25][00:04.125]下一句')).toEqual([
      { time: 2.5, text: '重复歌词' },
      { time: 3.25, text: '下一句' },
      { time: 4.125, text: '下一句' },
      { time: 10, text: '重复歌词' },
    ])
  })

  it('ignores metadata, malformed timestamps, invalid seconds and plain text', () => {
    expect(parseLrc('[ar:歌手]\n无效行\n[00:60.00]错误秒数\n[00:01.]错误格式\n[00:05.00]有效')).toEqual([
      { time: 5, text: '有效' },
    ])
  })

  it('sorts out-of-order lines and supports CRLF and BOM', () => {
    expect(parseLrc('\uFEFF[00:08]后来\r\n[00:01]首先\r\n[00:03]然后')).toEqual([
      { time: 1, text: '首先' },
      { time: 3, text: '然后' },
      { time: 8, text: '后来' },
    ])
  })
})

describe('findCurrentLrcLine', () => {
  const lines = parseLrc('[00:02]第二句\n[00:08]第三句\n[00:00]第一句')

  it('finds the latest line at or before the playback time', () => {
    expect(findCurrentLrcLine(lines, -1)).toBe(-1)
    expect(findCurrentLrcLine(lines, 0)).toBe(0)
    expect(findCurrentLrcLine(lines, 7.99)).toBe(1)
    expect(findCurrentLrcLine(lines, 8)).toBe(2)
    expect(findCurrentLrcLine(lines, 20)).toBe(2)
  })

  it('handles empty lyrics and invalid playback times', () => {
    expect(findCurrentLrcLine([], 0)).toBe(-1)
    expect(findCurrentLrcLine(lines, Number.NaN)).toBe(-1)
  })
})
