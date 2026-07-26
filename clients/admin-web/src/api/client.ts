import { sessionTokenVault, type TokenState } from './tokenVault'

interface Envelope<T> { code: string; message: string; requestId: string; data: T }
interface TokenResponse extends TokenState { tokenType: string; account: { id: string; displayName: string } }
export class ApiError extends Error { constructor(message: string, public code: string, public status: number, public requestId = '') { super(message) } }
let refreshPromise: Promise<TokenState> | null = null

async function parse<T>(response: Response): Promise<T> {
  let body: Envelope<T> | undefined
  try { body = await response.json() as Envelope<T> } catch { throw new ApiError('服务返回了不可解析的响应', 'INVALID_RESPONSE', response.status) }
  if (!response.ok || body.code !== 'OK') throw new ApiError(body.message || '请求失败', body.code, response.status, body.requestId)
  return body.data
}

async function refresh(): Promise<TokenState> {
  const current = sessionTokenVault.read()
  if (!current?.refreshToken) throw new ApiError('登录已失效', 'UNAUTHORIZED', 401)
  if (!refreshPromise) {
    refreshPromise = fetch('/api/auth/refresh', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ refreshToken: current.refreshToken }) })
      .then(parse<TokenResponse>).then(result => { const next: TokenState = result; sessionTokenVault.write(next); window.dispatchEvent(new CustomEvent('admin-auth-token')); return next })
      .catch(error => { sessionTokenVault.clear(); window.dispatchEvent(new CustomEvent('admin-auth-expired')); throw error })
      .finally(() => { refreshPromise = null })
  }
  return refreshPromise
}

export async function request<T>(path: string, init: RequestInit = {}, retry = true): Promise<T> {
  const headers = new Headers(init.headers)
  if (!headers.has('Content-Type') && init.body) headers.set('Content-Type', 'application/json')
  const token = sessionTokenVault.read()?.accessToken
  if (token) headers.set('Authorization', `Bearer ${token}`)
  let response = await fetch(`/api${path}`, { ...init, headers })
  if (response.status === 401 && retry && !path.startsWith('/auth/')) {
    await refresh()
    const next = sessionTokenVault.read()?.accessToken
    if (next) headers.set('Authorization', `Bearer ${next}`)
    response = await fetch(`/api${path}`, { ...init, headers })
  }
  return parse<T>(response)
}

export async function login(identifier: string, password: string): Promise<TokenResponse> {
  const result = await request<TokenResponse>('/auth/login', { method: 'POST', body: JSON.stringify({ identifier, password }) }, false)
  sessionTokenVault.write(result)
  window.dispatchEvent(new CustomEvent('admin-auth-token'))
  return result
}
export async function logout(): Promise<void> { try { await request('/auth/logout', { method: 'POST' }, false) } finally { sessionTokenVault.clear() } }
export function hasToken(): boolean { return Boolean(sessionTokenVault.read()?.accessToken) }
