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
export function createWebAuth(config:PublicConfig,fetcher:Fetcher,storage:StoragePort){
  let generation=0
  let active:UserAccount|null=null
  let token:string|null=null
  async function request(path:string,method:string,body?:unknown,accessToken?:string):Promise<unknown>{
    const controller=new AbortController()
    const timeout=setTimeout(()=>controller.abort(),12000)
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
      return response.json()
    }finally{clearTimeout(timeout)}
  }
  async function validate(accessToken:string):Promise<UserAccount>{
    const data=validatedData(await request('/auth/session','GET',undefined,accessToken))
    return account(data)
  }
  async function restore():Promise<UserAccount|null>{
    const mine=++generation
    const stored=storage.getItem(SESSION_KEY)
    if(!stored){active=null;token=null;return null}
    try{
      const identity=await validate(stored)
      if(generation!==mine)return active
      token=stored;active=identity
      return identity
    }catch{
      if(generation===mine){
        if(storage.getItem(SESSION_KEY)===stored)storage.removeItem(SESSION_KEY)
        token=null;active=null
      }
      return null
    }
  }
  async function login(identifier:string,password:string):Promise<UserAccount|null>{
    const mine=++generation
    const username=identifier.trim()
    if(!username||username.length>256||!password||password.length>128)
      throw Error('请输入有效账号和密码')
    const reply=validatedData(await request('/auth/login','POST',{identifier:username,password}))
    const received=bearer(reply)
    // Verify server account before persisting; stale login responses after
    // logout/account switch cannot reintroduce an old access token.
    const identity=await validate(received)
    if(generation!==mine)return active
    storage.setItem(SESSION_KEY,received)
    token=received;active=identity
    return identity
  }
  async function logout():Promise<void>{
    ++generation
    const old=token || storage.getItem(SESSION_KEY)
    token=null;active=null
    storage.removeItem(SESSION_KEY)
    if(old)await request('/auth/logout','POST',{},old)
  }
  return {login,restore,logout,get account(){return active},get authenticated(){return !!token && !!active}}
}
