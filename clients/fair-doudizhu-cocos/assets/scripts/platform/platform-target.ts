/** Runtime platform detection is confined to this composition boundary. */
export type RuntimeTarget = 'web' | 'wechat' | 'douyin' | 'unknown'
export function detectRuntimeTarget(): RuntimeTarget {
  const environment = globalThis as typeof globalThis & { wx?: unknown; tt?: unknown; document?: unknown }
  if (environment.wx) return 'wechat'
  if (environment.tt) return 'douyin'
  if (environment.document) return 'web'
  return 'unknown'
}
