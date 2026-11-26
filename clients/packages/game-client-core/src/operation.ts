/** Portable, local operation lifetime. No SDK/renderer or AbortController
 * dependency. Cancellation is not a rollback of an already dispatched write. */
export class OperationCancelled extends Error {
  constructor(){super('operation cancelled');this.name='OperationCancelled'}
}
export class OperationScope {
  private state:'active'|'cancelled'|'finished'='active'
  private dispatched=false
  private readonly listeners=new Set<()=>void>()
  constructor(private readonly release:()=>void=()=>{}){}
  get active():boolean{return this.state==='active'}
  get committed():boolean{return this.dispatched}
  markDispatched():void{
    if(!this.active)throw new OperationCancelled()
    this.dispatched=true
  }
  cancel():boolean{
    if(!this.active)return true
    if(this.dispatched)return false
    this.end('cancelled')
    return true
  }
  /** Disposal can abandon observation of a dispatched write, not undo it. */
  abandon():void{if(this.active)this.end('cancelled')}
  finish():void{if(this.active)this.end('finished')}
  private end(state:'cancelled'|'finished'):void{
    this.state=state
    const callbacks:Array<()=>void>=[]
    this.listeners.forEach(fn=>callbacks.push(fn))
    this.listeners.clear()
    this.release()
    callbacks.forEach(fn=>fn())
  }
  /** Portable cancellation registration for nested SDK/transport adapters. */
  onCancel(handler:()=>void):()=>void{
    if(!this.active){handler();return ()=>{}}
    this.listeners.add(handler)
    return ()=>{this.listeners.delete(handler)}
  }
  /** Start synchronously; detach timers/observers on all completion paths.
   * The underlying SDK may still return; its result is consumed but ignored. */
  wait<T>(start:()=>Promise<T>,timeoutMs=15_000):Promise<T>{
    if(!this.active)return Promise.reject(new OperationCancelled())
    if(!Number.isSafeInteger(timeoutMs)||timeoutMs<1||timeoutMs>2_147_483_647)
      return Promise.reject(new RangeError('invalid operation timeout'))
    return new Promise<T>((resolve,reject)=>{
      let settled=false
      const cleanup=()=>{clearTimeout(timer);this.listeners.delete(cancel)}
      const fail=(error:unknown)=>{if(settled)return;settled=true;cleanup();reject(error)}
      const cancel=()=>fail(new OperationCancelled())
      const timer=setTimeout(()=>fail(new Error('operation timed out')),timeoutMs)
      this.listeners.add(cancel)
      try{
        Promise.resolve(start()).then(value=>{
          if(settled)return
          settled=true;cleanup();resolve(value)
        },fail)
      }catch(error){fail(error)}
    })
  }
}
/** A single owned slot; an old finally can never release a new operation. */
export class OperationSlot {
  private owner:OperationScope|null=null
  begin():OperationScope|null{
    if(this.owner)return null
    const scope=new OperationScope(()=>{if(this.owner===scope)this.owner=null})
    this.owner=scope
    return scope
  }
  cancel():boolean{return this.owner?.cancel()??true}
  abandon():void{this.owner?.abandon()}
}
