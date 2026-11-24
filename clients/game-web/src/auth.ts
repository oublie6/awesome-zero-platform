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
  async function request(path:string,method:string,body?:unknown,accessToken?:string):Promise<unknown>{
    const controller=new AbortController()
    const timeout=setTimeout(()=>controller.abort(),timeoutMs)
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
    }finally{clearTimeout(timeout)}
  }
  async function validate(accessToken:string):Promise<UserAccount>{
    const data=validatedData(await request('/auth/session','GET',undefined,accessToken))
    return account(data)
  }
  async function restore():Promise<UserAccount|null>{
    const mine=++generation
    const stored=storage.getItem(SESSION_KEY)
    const loginEpoch=loginEpochs.get(storage)??0
    ownedToken=stored
    active=null;token=null
    if(!stored)return null
    try{
      const identity=await validate(stored)
      if(generation!==mine)return null
      if((loginEpochs.get(storage)??0)!==loginEpoch || storage.getItem(SESSION_KEY)!==stored){
        ownedToken=null;return null
      }
      token=stored;active=identity
      return identity
    }catch{
      if(generation===mine){
        if((loginEpochs.get(storage)??0)===loginEpoch && storage.getItem(SESSION_KEY)===stored)
          storage.removeItem(SESSION_KEY)
        token=null;active=null;ownedToken=null
      }
      return null
    }
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
    const mine=++generation
    const username=identifier.trim()
    if(!username||username.length>256||!password||password.length>128)
      throw Error('请输入有效账号和密码')
    const loginEpoch=(loginEpochs.get(storage)??0)+1
    loginEpochs.set(storage,loginEpoch)
    const reply=validatedData(await request('/auth/login','POST',{identifier:username,password}))
    const received=bearer(reply)
    // Verify server account before persisting; stale login responses after
    // logout/account switch cannot reintroduce an old access token.
    const identity=await validate(received)
    if(generation!==mine || loginEpochs.get(storage)!==loginEpoch)return null
    storage.setItem(SESSION_KEY,received)
    ownedToken=received;token=received;active=identity
    return identity
  }
  async function logout():Promise<void>{
    ++generation
    const old=token || ownedToken
    token=null;active=null;ownedToken=null
    if(old && storage.getItem(SESSION_KEY)===old)storage.removeItem(SESSION_KEY)
    if(old)await request('/auth/logout','POST',{},old)
  }
  return {register,login,restore,logout,get account(){return active},get authenticated(){return !!token && !!active}}
}
