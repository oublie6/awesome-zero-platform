export interface TokenState { accessToken: string; refreshToken: string; accessExpiresAt: string; refreshExpiresAt: string }
export interface TokenVault { read(): TokenState | null; write(tokens: TokenState): void; clear(): void }
const key = 'awesome-zero-admin.tokens'
export const sessionTokenVault: TokenVault = {
  read() { const raw = sessionStorage.getItem(key); if (!raw) return null; try { return JSON.parse(raw) as TokenState } catch { sessionStorage.removeItem(key); return null } },
  write(tokens) { sessionStorage.setItem(key, JSON.stringify(tokens)) },
  clear() { sessionStorage.removeItem(key) }
}
