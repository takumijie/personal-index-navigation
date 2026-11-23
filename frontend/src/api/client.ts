import type { ApiErrorBody } from '@/types'

export class ApiError extends Error {
  status: number
  code: string | number

  constructor(message: string, status = 0, code: string | number = 'NETWORK_ERROR') {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

type Envelope<T> = { data?: T; error?: ApiErrorBody }
export type RequestOptions = RequestInit & { timeoutMs?: number }

export const DEFAULT_REQUEST_TIMEOUT_MS = 15_000
export const SLOW_REQUEST_TIMEOUT_MS = 5 * 60_000

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { timeoutMs = DEFAULT_REQUEST_TIMEOUT_MS, signal: externalSignal, ...fetchOptions } = options
  const headers = new Headers(fetchOptions.headers)
  if (fetchOptions.body && !(fetchOptions.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const controller = new AbortController()
  let timedOut = false
  const abortFromOutside = () => controller.abort(externalSignal?.reason)
  if (externalSignal?.aborted) abortFromOutside()
  else externalSignal?.addEventListener('abort', abortFromOutside, { once: true })
  const timer = setTimeout(() => {
    timedOut = true
    controller.abort()
  }, timeoutMs)

  const cancellationError = () => {
    if (timedOut) return new ApiError('请求超时，请稍后重试', 0, 'TIMEOUT')
    if (externalSignal?.aborted) return new ApiError('请求已取消', 0, 'REQUEST_ABORTED')
    return null
  }

  try {
    let response: Response
    try {
      response = await fetch(`/api/v1${path}`, {
        ...fetchOptions,
        headers,
        credentials: 'include',
        signal: controller.signal,
      })
    } catch {
      throw cancellationError() || new ApiError('无法连接服务器，请稍后重试')
    }

    let payload: Envelope<T> | null = null
    try {
      payload = await response.json() as Envelope<T>
    } catch {
      const cancelled = cancellationError()
      if (cancelled) throw cancelled
      if (!response.ok) throw new ApiError(`请求失败（${response.status}）`, response.status)
    }

    if (!response.ok || payload?.error) {
      throw new ApiError(
        payload?.error?.message || `请求失败（${response.status}）`,
        response.status,
        payload?.error?.code || response.status,
      )
    }
    if (!payload || !('data' in payload)) throw new ApiError('服务器响应格式不正确', response.status)
    return payload.data as T
  } finally {
    clearTimeout(timer)
    externalSignal?.removeEventListener('abort', abortFromOutside)
  }
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : '操作失败，请稍后重试'
}
