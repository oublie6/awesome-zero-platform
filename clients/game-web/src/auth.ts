/** Browser game auth uses only existing Go identity APIs.
 * No duplicate account database, local game authority or password storage. */
export const SESSION_KEY='fair-doudizhu-session'
export interface PublicConfig {
  apiBase:string
  socketEndpoint:string
  revealManifestRoots:Record<string,string>
}
export interface UserAccount {id:string;displayName:string}
interface Envelope {code?:unknown;data?:unknown}
type Fetcher=typeof fetch
type StoragePort=Pick<Storage,'getItem'|'setItem'|'removeItem'>
const loginEpochs=new WeakMap<StoragePort,number>()
type PendingLogin={epoch:number;cancel:()=>void}
const pendingLogins=new WeakMap<StoragePort,PendingLogin>()
// The synchronous write is provisional until its caller rechecks ownership.
// Separate from pendingLogin: cancellation may release that marker in setItem.
type StorageWrite={token:string;generation:number;epoch:number}
const storageWrites=new WeakMap<StoragePort,StorageWrite>()

function record(value:unknown):value is Record<string,unknown>{
  return !!value && typeof value==='object' && !Array.isArray(value)
}
function bearer(raw:unknown):string{
  if(!record(raw)||raw.tokenType!=='Bearer' || typeof raw.accessToken!=='string' ||
      !raw.accessToken.trim() || /[\r\n]/.test(raw.accessToken) ||
      raw.accessToken.length>8192)throw Error('服务端未返回有效访问令牌')
  return raw.accessToken
}
function account(raw:unknown):UserAccount{
  if(!record(raw)||!record(raw.account)||typeof raw.account.id!=='string'||
    !raw.account.id || typeof raw.account.displayName!=='string')
    throw Error('服务器会话身份无效')
  return {id:raw.account.id,displayName:raw.account.displayName}
}
export function runtimeConfig(origin:string,raw:unknown):PublicConfig|null{
  if(!record(raw))return null
  if(typeof raw.apiBase!=='string'||typeof raw.socketEndpoint!=='string')return null
  try{
    const api=new URL(raw.apiBase),socket=new URL(raw.socketEndpoint)
    const current=new URL(origin)
    // Same-origin deployment is required: Vue and trusted Cocos frame share
    // the same tab's sessionStorage, and no cross-origin token transport exists.
    if(current.protocol!=='https:' || api.protocol!=='https:' || api.origin!==current.origin ||
       socket.protocol!=='wss:' || socket.host!==current.host ||
       api.username || api.password || api.search || api.hash ||
       socket.username || socket.password || socket.search || socket.hash)return null
    return {apiBase:raw.apiBase.replace(/\/$/,''),
      socketEndpoint:raw.socketEndpoint,
      revealManifestRoots:record(raw.revealManifestRoots)?
        raw.revealManifestRoots as Record<string,string>:{}}
  }catch{return null}
}
function validatedData(raw:unknown):unknown{
  if(!record(raw)||raw.code!=='OK'||!record(raw.data))
    throw Error('服务器拒绝了身份验证')
  return raw.data
}
export function createWebAuth(config:PublicConfig,fetcher:Fetcher,storage:StoragePort,timeoutMs=12000){
  if(!Number.isSafeInteger(timeoutMs)||timeoutMs<1||timeoutMs>60000)throw Error('invalid identity request timeout')
  let generation=0
  let active:UserAccount|null=null
  let token:string|null=null
  let ownedToken:string|null=null
  let activeEpoch=-1
  let pendingLogin:PendingLogin|null=null
  let pendingRequest:AbortController|null=null
  let writing:StorageWrite|null=null
  const epoch=()=>loginEpochs.get(storage)??0
  const owns=(mine:number,expectedEpoch:number)=>generation===mine && epoch()===expectedEpoch
  // Cleanup is diagnostic, not a business outcome. Never log raw adapter
  // errors (they may contain credentials), and isolate diagnostics too.
  function cleanup(step:()=>void,message:string):boolean{
    try{step();return true}
    catch{
      try{console.warn(message)}catch{/* diagnostic cannot replace business result */}
      return false
    }
  }
  // A read may synchronously start another identity operation. Only act on
  // its result if the same caller still owns the decision after it returns.
  // True means no cleanup error, not proof that an unrelated token was removed.
  function clearStored(expected:string,current:()=>boolean):boolean{
    return cleanup(()=>{
      if(!current())return
      const saved=storage.getItem(SESSION_KEY)
      if(current() && saved===expected)storage.removeItem(SESSION_KEY)
    },'web session storage cleanup failed')
  }
  function detachAttempt(){
    const login=pendingLogin,request=pendingRequest
    pendingLogin=null;pendingRequest=null
    return ()=>{
      cleanup(()=>login?.cancel(),'web session cancellation failed')
      cleanup(()=>request?.abort(),'web request cancellation failed')
    }
  }
  function currentIdentity():UserAccount|null{
    const mine=generation,expectedEpoch=activeEpoch,expected=token,identity=active
    if(!expected || !identity || !owns(mine,expectedEpoch))return null
    let saved:string|null
    try{saved=storage.getItem(SESSION_KEY)}catch{return null}
    return owns(mine,expectedEpoch) && token===expected && active===identity &&
      activeEpoch===expectedEpoch && saved===expected?identity:null
  }
  async function request(path:string,method:string,body?:unknown,accessToken?:string,owner?:AbortSignal):Promise<unknown>{
    const controller=new AbortController()
    const cancel=()=>controller.abort()
    if(owner?.aborted)throw Error('登录操作已取消')
    owner?.addEventListener('abort',cancel,{once:true})
    const timeout=setTimeout(cancel,timeoutMs)
    try{
      const response=await fetcher(config.apiBase+path,{
        method,signal:controller.signal,credentials:'omit',cache:'no-store',redirect:'error',
        headers:{...body===undefined?{}:{'Content-Type':'application/json'},
          ...accessToken?{Authorization:'Bearer '+accessToken}:{}},
        body:body===undefined?undefined:JSON.stringify(body)
      })
      if(!response.ok){
        if(response.status===401)throw Error('账号或密码错误，或者会话已过期')
        throw Error('服务端请求失败（HTTP '+response.status+'）')
      }
      // Keep the same deadline through body consumption, not only headers.
      return await response.json()
    }finally{
      cleanup(()=>clearTimeout(timeout),'web request timer cleanup failed')
      cleanup(()=>owner?.removeEventListener('abort',cancel),'web request listener cleanup failed')
    }
  }
  async function validate(accessToken:string,owner?:AbortSignal):Promise<UserAccount>{
    const data=validatedData(await request('/auth/session','GET',undefined,accessToken,owner))
    return account(data)
  }
  async function restore():Promise<UserAccount|null>{
    const mine=++generation
    const cancelPrevious=detachAttempt()
    const loginEpoch=epoch()
    const current=()=>owns(mine,loginEpoch) && !pendingLogins.has(storage) && !storageWrites.has(storage)
    ownedToken=null
    active=null;token=null
    cancelPrevious()
    if(!current())return null
    const stored=storage.getItem(SESSION_KEY)
    if(!current() || !stored)return null
    ownedToken=stored
    const requestOwner=new AbortController()
    pendingRequest=requestOwner
    try{
      const identity=await validate(stored,requestOwner.signal)
      if(!current())return null
      const saved=storage.getItem(SESSION_KEY)
      if(!current())return null
      if(saved!==stored){ownedToken=null;return null}
      token=stored;active=identity;activeEpoch=loginEpoch
      return identity
    }catch{
      if(owns(mine,loginEpoch)){
        // Clear local ownership before external cleanup, never afterwards.
        token=null;active=null;ownedToken=null
        clearStored(stored,current)
      }
      return null
    }finally{if(pendingRequest===requestOwner)pendingRequest=null}
  }
  async function register(username:string,email:string,displayName:string,password:string):Promise<UserAccount>{
    if(!/^[A-Za-z0-9][A-Za-z0-9._-]{2,31}$/.test(username) ||
       !email.trim() || email.length>320 || !displayName.trim() ||
       new TextEncoder().encode(password).length<8 || new TextEncoder().encode(password).length>128)
      throw Error('请填写有效用户名、邮箱、昵称和至少 8 字节密码')
    // Registration never persists credentials or grants a session.
    const data=validatedData(await request('/auth/register','POST',{username,email:email.trim(),displayName:displayName.trim(),password}))
    return account(data)
  }
  async function login(identifier:string,password:string):Promise<UserAccount|null>{
    const username=identifier.trim()
    if(!username||username.length>256||!password||password.length>128)
      throw Error('请输入有效账号和密码')
    const mine=++generation
    const cancelPrevious=detachAttempt()
    const sharedPrevious=pendingLogins.get(storage)
    const loginEpoch=(loginEpochs.get(storage)??0)+1
    loginEpochs.set(storage,loginEpoch)
    const requestOwner=new AbortController()
    const release=()=>{
      if(pendingLogin===operation)pendingLogin=null
      if(pendingLogins.get(storage)===operation)pendingLogins.delete(storage)
      if(pendingRequest===requestOwner)pendingRequest=null
    }
    const operation:PendingLogin={epoch:loginEpoch,cancel:()=>{release();requestOwner.abort()}}
    pendingLogin=operation
    pendingLogins.set(storage,operation)
    pendingRequest=requestOwner
    const current=()=>generation===mine && loginEpochs.get(storage)===loginEpoch && !requestOwner.signal.aborted
    try{
      cancelPrevious()
      cleanup(()=>sharedPrevious?.cancel(),'web session cancellation failed')
      if(!current())return null
      const reply=await request('/auth/login','POST',{identifier:username,password},undefined,requestOwner.signal)
      if(!current())return null
      const received=bearer(validatedData(reply))
      const identity=await validate(received,requestOwner.signal)
      if(!current())return null
      const write:StorageWrite={token:received,generation:mine,epoch:loginEpoch}
      writing=write
      storageWrites.set(storage,write)
      let committed=false
      try{
        storage.setItem(SESSION_KEY,received)
        if(!current())return null
        const saved=storage.getItem(SESSION_KEY)
        if(!current() || saved!==received)return null
        ownedToken=received;token=received;active=identity;activeEpoch=loginEpoch
        committed=true
        return identity
      }finally{
        // A canceled/failed write must never delete a replacement credential.
        // Keep the lease until this synchronous cleanup has fully returned.
        if(!committed)clearStored(received,()=>storageWrites.get(storage)===write && epoch()===write.epoch)
        if(storageWrites.get(storage)===write)storageWrites.delete(storage)
        if(writing===write)writing=null
      }
    }catch(error){if(!current())return null;throw error}
    finally{release()}
  }
  async function logout():Promise<{storageCleanupFailed:boolean}>{
    // A login can be inside setItem: capture its verified, provisional token
    // before invalidating the attempt, even if the write has not returned.
    const old=writing?.generation===generation?writing.token:token||ownedToken
    const mine=++generation,loginEpoch=epoch()
    const cancelPrevious=detachAttempt()
    token=null;active=null;ownedToken=null
    const storageCleanupFailed=old!==null && !clearStored(old,()=>owns(mine,loginEpoch))
    cancelPrevious()
    // Storage or cancellation failure cannot skip revocation of the captured
    // credential. Preserve real server errors, never revoke a replacement.
    if(old){
      const result=validatedData(await request('/auth/logout','POST',{},old))
      if(!record(result) || result.loggedOut!==true)throw Error('服务端注销状态未确认')
    }
    return {storageCleanupFailed}
  }
  function dispose(){
    ++generation
    const cancelPrevious=detachAttempt()
    token=null;active=null;ownedToken=null
    cancelPrevious()
  }
  return {register,login,restore,logout,dispose,get account(){return currentIdentity()},get authenticated(){return currentIdentity()!==null}}
}
