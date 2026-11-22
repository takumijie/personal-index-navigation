export interface LrcLine {
  time: number
  text: string
}

const timestampPattern = /\[(\d+):(\d{1,2})(?:[.:](\d{1,3}))?\]/g

export function parseLrc(text: string): LrcLine[] {
  const lines: LrcLine[] = []

  for (const source of text.replace(/^\uFEFF/, '').split(/\r?\n/)) {
    const matches = [...source.matchAll(timestampPattern)]
    if (!matches.length) continue

    const content = source.replace(timestampPattern, '').trim()
    for (const match of matches) {
      const minutes = Number(match[1])
      const seconds = Number(match[2])
      if (seconds >= 60) continue

      const milliseconds = Number((match[3] || '').padEnd(3, '0') || 0)
      lines.push({ time: minutes * 60 + seconds + milliseconds / 1000, text: content })
    }
  }

  return lines.sort((a, b) => a.time - b.time)
}

export function findCurrentLrcLine(lines: LrcLine[], time: number): number {
  if (!lines.length || !Number.isFinite(time) || time < lines[0].time) return -1

  let low = 0
  let high = lines.length - 1
  while (low <= high) {
    const middle = Math.floor((low + high) / 2)
    if (lines[middle].time <= time) low = middle + 1
    else high = middle - 1
  }
  return high
}
