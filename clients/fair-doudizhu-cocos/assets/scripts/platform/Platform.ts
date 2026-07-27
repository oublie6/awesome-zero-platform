export interface GamePlatform {
  readonly name: string
  now(): number
  onPause(handler: () => void): void
  onResume(handler: () => void): void
}
