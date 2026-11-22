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

export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  if (options.body && !(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  let response: Response
  try {
    response = await fetch(`/api/v1${path}`, {
      ...options,
      headers,
      credentials: 'include',
    })
  } catch {
    throw new ApiError('无法连接服务器，请稍后重试')
  }

  let payload: Envelope<T> | null = null
  try {
    payload = await response.json() as Envelope<T>
  } catch {
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
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : '操作失败，请稍后重试'
}
