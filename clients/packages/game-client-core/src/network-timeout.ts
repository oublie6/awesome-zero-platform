/** Portable network deadline; avoid host timer overflow and unbounded pending work. */
export function networkTimeout(value = 15_000): number {
  if (!Number.isSafeInteger(value) || value < 1 || value > 2_147_483_647) throw new RangeError('invalid network timeout')
  return value
}
