export class Task {
  events = new Map(); closes = 0; sent = []
  add(name, cb) { const set = this.events.get(name) ?? new Set(); set.add(cb); this.events.set(name, set) }
  off(name, cb) { this.events.get(name)?.delete(cb) }
  emit(name, value) { for (const cb of [...(this.events.get(name) ?? [])]) cb(value) }
  onOpen(cb) { this.add('open', cb) } offOpen(cb) { this.off('open', cb) }
  onClose(cb) { this.add('close', cb) } offClose(cb) { this.off('close', cb) }
  onError(cb) { this.add('error', cb) } offError(cb) { this.off('error', cb) }
  onMessage(cb) { this.add('message', cb) } offMessage(cb) { this.off('message', cb) }
  close() { this.closes++; this.emit('close') }
  send({data, success}) { this.sent.push(data); success?.() }
  get listeners() { return [...this.events.values()].reduce((n, s) => n+s.size, 0) }
}
