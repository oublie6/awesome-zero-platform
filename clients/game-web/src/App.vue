<script setup lang="ts">
import {computed,onMounted,onUnmounted,ref} from 'vue'
import {createWebAuth,runtimeConfig,type UserAccount} from './auth.ts'

type Page='home'|'login'|'register'|'lobby'|'game'
type PublicRuntime=typeof globalThis & {__FAIR_DOUDIZHU_CONFIG__?:unknown}
const config=runtimeConfig(window.location.origin,(globalThis as PublicRuntime).__FAIR_DOUDIZHU_CONFIG__)
const client=config?createWebAuth(config,fetch.bind(globalThis),sessionStorage):null
const page=ref<Page>('home')
const checking=ref(true)
const busy=ref(false)
const account=ref<UserAccount|null>(null)
const identifier=ref('')
const password=ref('')
const registerName=ref('')
const registerEmail=ref('')
const registerDisplay=ref('')
const registerPassword=ref('')
const registerConfirm=ref('')
const notice=ref('')
const error=ref('')
type BootPhase='waiting'|'document'|'engine'|'scene'|'session'|'ready'|'failed'
const gameMounted=ref(false)
const gameStatus=ref('等待加载游戏资源')
const gameFrame=ref<HTMLIFrameElement|null>(null)
const gameInstance=ref(0)
const gamePhase=ref<BootPhase>('waiting')
const gameLoadSlow=ref(false)
const gameLoadError=ref('')
const loggedIn=computed(()=>!!account.value)
const isReady=computed(()=>gamePhase.value==='ready')
const confirmedStages=computed(()=>({
  waiting:0,document:1,engine:2,scene:3,session:3,ready:4,failed:0
})[gamePhase.value])
const stageWidth=computed(()=>confirmedStages.value*25+'%')
const protocol='azp-doudizhu-frame-v1'
let slowTimer:ReturnType<typeof setTimeout>|null=null
let loadDeadline:ReturnType<typeof setTimeout>|null=null
function clearLoadWatchdog(){
  if(slowTimer!==null)clearTimeout(slowTimer)
  if(loadDeadline!==null)clearTimeout(loadDeadline)
  slowTimer=null;loadDeadline=null
}
function armLoadWatchdog(){
  clearLoadWatchdog()
  gameLoadSlow.value=false
  if(isReady.value)return
  slowTimer=setTimeout(()=>{if(!isReady.value)gameLoadSlow.value=true},15000)
  loadDeadline=setTimeout(()=>{
    if(isReady.value)return
    gamePhase.value='failed'
    gameLoadError.value='启动阶段长时间没有响应，请检查网络或重新加载游戏。'
    gameStatus.value=gameLoadError.value
  },60000)
}
function failGameLoad(message:string){
  clearLoadWatchdog()
  gamePhase.value='failed'
  gameLoadError.value=message
  gameStatus.value=message
}
function reloadGame(){
  if(!account.value || !client?.authenticated)return
  clearLoadWatchdog()
  gamePhase.value='waiting'
  gameStatus.value='等待游戏页面响应'
  gameLoadError.value=''
  gameLoadSlow.value=false
  gameMounted.value=true
  gameInstance.value++
  armLoadWatchdog()
}

function communicate(type:'SESSION_AVAILABLE'|'SESSION_REVOKED'|'OPEN_ROOMS'){
  const frame=gameFrame.value?.contentWindow
  if(!frame)return
  // Room IDs are non-secret user intentions, never client-authoritative state.
  frame.postMessage({protocol,type},window.location.origin)
}
function onGameMessage(event:MessageEvent){
  if(!gameMounted.value || event.source!==gameFrame.value?.contentWindow ||
     event.origin!==window.location.origin)return
  const message=event.data
  if(!message || typeof message!=='object' || message.protocol!==protocol ||
     typeof message.type!=='string')return
  if(message.type==='BOOT_STAGE' && !isReady.value && gamePhase.value!=='failed'){
    const stage=message.stage
    if(stage!=='document' && stage!=='engine' && stage!=='scene')return
    const ranks:Record<string,number>={waiting:0,document:1,engine:2,scene:3,session:4}
    if((ranks[stage]??0)>(ranks[gamePhase.value]??0)){
      gamePhase.value=stage
      const stageLabels={document:'正在读取游戏页面',
        engine:'游戏引擎初始化完成',scene:'游戏场景已准备'}
      gameStatus.value=stageLabels[stage as keyof typeof stageLabels]
      armLoadWatchdog()
    }
  }else if(message.type==='BOOT_FAILED' && !isReady.value){
    failGameLoad('游戏资源或引擎启动失败，请重新加载。')
  }else if(message.type==='GAME_READY'){
    if(account.value && client?.authenticated){
      if(!isReady.value){
        gamePhase.value='session'
        gameStatus.value='正在验证服务器会话并恢复房间'
        armLoadWatchdog()
      }
      communicate('SESSION_AVAILABLE')
    }else {
      failGameLoad('平台会话已过期，请重新登录。')
      gameMounted.value=false
      page.value='login'
    }
  }else if(message.type==='SESSION_ACTIVE'){
    clearLoadWatchdog()
    gamePhase.value='ready'
    gameLoadError.value=''
    gameStatus.value='游戏就绪'
    communicate('OPEN_ROOMS')
  }else if(message.type==='SESSION_FAILED'){
    failGameLoad('游戏连接失败，请检查 HTTPS/WSS 服务或重新进入。')
  }
}
async function initialize(){
  checking.value=true
  try{
    account.value=await client?.restore()??null
    page.value=account.value?'lobby':'home'
  }catch{
    account.value=null;page.value='home'
  }finally{checking.value=false}
}
async function submitLogin(){
  if(!client){error.value='网站尚未配置同源 HTTPS/WSS 游戏服务';return}
  if(busy.value)return
  busy.value=true
  error.value=''
  const user=identifier.value.trim()
  const secret=password.value
  password.value=''
  try{
    const verified=await client.login(user,secret)
    if(!verified)return
    account.value=verified
    page.value='lobby'
  }catch(e){error.value=e instanceof Error?e.message:'登录失败'}
  finally{busy.value=false}
}
async function submitRegister(){
  if(!client){error.value='网站尚未配置同源 HTTPS/WSS 服务';return}
  if(busy.value)return
  error.value=''
  if(registerPassword.value!==registerConfirm.value){error.value='两次密码不一致';return}
  busy.value=true
  const secret=registerPassword.value
  registerPassword.value=''
  registerConfirm.value=''
  try{
    await client.register(registerName.value.trim(),registerEmail.value.trim(),registerDisplay.value.trim(),secret)
    notice.value='账号已创建。当前测试版尚未验证邮箱，请使用用户名和密码登录。'
    identifier.value=registerName.value.trim()
    page.value='login'
  }catch(e){error.value=e instanceof Error?e.message:'注册失败'}
  finally{busy.value=false}
}
async function logout(){
  if(busy.value)return
  busy.value=true
  error.value=''
  communicate('SESSION_REVOKED')
  // Never retain a previous account's iframe or in-memory game state.
  clearLoadWatchdog()
  gameMounted.value=false
  gamePhase.value='waiting'
  page.value='home'
  account.value=null
  password.value=''
  try{await client?.logout()}
  catch{error.value='已清理浏览器会话，但服务端注销状态需要确认'}
  finally{busy.value=false}
}
function openGame(){
  if(!account.value || !client?.authenticated)return
  error.value=''
  // Retain one authenticated iframe across site navigation; no new requests.
  if(!gameMounted.value || gamePhase.value==='failed')reloadGame()
  page.value='game'
}

onMounted(()=>{
  window.addEventListener('message',onGameMessage)
  void initialize()
})
onUnmounted(()=>{
  window.removeEventListener('message',onGameMessage)
  clearLoadWatchdog()
})
</script>

<template>
  <div class="app-shell">
    <header class="topbar">
      <button class="brand" type="button" @click="page=account?'lobby':'home'" aria-label="公平斗地主首页">
        <span class="brand-symbol">♠</span>
        <span><strong>公平斗地主</strong><small>AWESOME ZERO PLATFORM</small></span>
      </button>
      <nav class="top-nav" aria-label="网站导航">
        <button type="button" :class="{selected:page==='home'||page==='lobby'}" @click="page=account?'lobby':'home'">首页</button>
        <button type="button" :disabled="!account" :class="{selected:page==='game'}" @click="openGame()">进入游戏</button>
        <button v-if="!loggedIn" type="button" class="nav-action" @click="page='login'">账号登录</button>
        <button v-else type="button" class="nav-action" :disabled="busy" @click="logout">退出登录</button>
      </nav>
    </header>

    <main class="page-body">
      <div v-if="checking" class="center-card"><div class="loading-suit">♣</div><p>正在验证平台账号会话…</p></div>
      <template v-else>
        <div v-if="error" class="error-strip" role="alert">{{ error }}</div>
        <div v-if="notice" class="form-hint" role="status">{{notice}}</div>
        <section v-if="page==='home'" class="hero">
          <div class="hero-copy">
            <div class="eyebrow"><span class="status-dot"></span> 三人好友房 · 服务端公平洗牌</div>
            <h1>一桌好牌，<br><em>公平见真章。</em></h1>
            <p>从好友开局到最后一张牌，关键规则与牌局恢复都由 Go 服务端负责。浏览器和微信小游戏共享同一套玩法与协议。</p>
            <div class="hero-actions">
              <button class="primary-button" type="button" @click="page='login'">使用平台账号登录 <span>→</span></button>
              <button class="ghost-button" type="button" @click="page='lobby'">了解游戏 <span>↗</span></button>
            </div>
            <div class="feature-line"><span>✦ 公平随机贡献</span><span>✦ 三人在线对局</span><span>✦ 服务端恢复</span></div>
          </div>
          <div class="hero-cards" aria-hidden="true">
            <div class="playing-card card-left"><span>♣</span><b>A</b></div>
            <div class="playing-card card-center"><span>♠</span><b>K</b></div>
            <div class="playing-card card-right"><span class="red">♥</span><b class="red">Q</b></div>
            <div class="card-shadow"></div>
          </div>
        </section>

        <section v-else-if="page==='login'" class="login-layout">
          <div class="login-intro"><div class="eyebrow">SECURE SIGN IN</div><h2>欢迎回来</h2><p>使用 Awesome Zero Platform 的现有账号进入游戏。无需创建新的斗地主身份。</p><p class="small-note">你的密码不会被游戏存储；牌局由服务器保存与恢复。</p></div>
          <form class="login-card" @submit.prevent="submitLogin">
            <div class="card-eyebrow">ACCOUNT ACCESS</div>
            <h3>登录平台账号</h3>
            <label for="username">用户名 / 邮箱 / 手机号</label>
            <input id="username" v-model="identifier" autocomplete="username" maxlength="256" placeholder="输入现有平台账号" required />
            <label for="password">密码</label>
            <input id="password" v-model="password" type="password" autocomplete="current-password" maxlength="128" placeholder="输入账号密码" required />
            <button class="primary-button full" type="submit" :disabled="busy || !config">{{ busy?'验证中…':'登录并继续 →' }}</button>
            <p v-if="!config" class="form-hint warning">当前部署缺少同源 HTTPS/WSS 配置，登录暂不可用。</p>
            <p class="form-hint">没有账号？ <button class="inline-link" type="button" @click="page='register'">创建平台账号</button></p>
          </form>
        </section>

        <section v-else-if="page==='register'" class="login-layout">
          <div class="login-intro"><div class="eyebrow">PLATFORM ACCOUNT</div><h2>创建统一账号</h2><p>网页、微信和未来的游戏客户端共享同一个 AccountID，牌局数据只归属于该账户。</p><p class="small-note">测试版邮箱必填，但暂不发送验证码，也不能用于密码找回。</p></div>
          <form class="login-card" @submit.prevent="submitRegister">
            <div class="card-eyebrow">CREATE ACCOUNT</div><h3>注册平台账号</h3>
            <label for="register-name">用户名</label><input id="register-name" v-model="registerName" autocomplete="username" minlength="3" maxlength="32" required placeholder="3–32 位字母、数字、._-" />
            <label for="register-email">邮箱（暂不验证）</label><input id="register-email" v-model="registerEmail" type="email" autocomplete="email" maxlength="320" required placeholder="name@example.com" />
            <label for="register-display">昵称</label><input id="register-display" v-model="registerDisplay" maxlength="120" required placeholder="显示名称" />
            <label for="register-password">密码</label><input id="register-password" v-model="registerPassword" type="password" autocomplete="new-password" minlength="8" maxlength="128" required />
            <label for="register-confirm">确认密码</label><input id="register-confirm" v-model="registerConfirm" type="password" autocomplete="new-password" minlength="8" maxlength="128" required />
            <button class="primary-button full" type="submit" :disabled="busy || !config">{{busy?'正在创建…':'创建账号'}}</button>
            <p class="form-hint">本阶段仅供受控测试使用。<button class="inline-link" type="button" @click="page='login'">已有账号？返回登录</button></p>
          </form>
        </section>
        <section v-else-if="page==='lobby'" class="lobby lobby-simplified">
          <div class="lobby-title">
            <div class="eyebrow">FAIR DOUDIZHU</div>
            <h2>{{ account?'欢迎回来，'+account.displayName:'三人好友房' }}</h2>
            <p>准备好和朋友玩一局了吗？房间创建、加入、继续游戏，都在同一个游戏大厅里完成。</p>
          </div>
          <div class="lobby-entry">
            <div class="lobby-entry-symbol" aria-hidden="true">♠</div>
            <h3>好友斗地主</h3>
            <p>创建房间邀请好友，或输入房间号加入。正在进行的牌局可以从服务器安全恢复。</p>
            <button type="button" class="primary-button lobby-entry-action" :disabled="!account" @click="openGame()">
              进入游戏大厅 <span>→</span>
            </button>
          </div>
          <p class="lobby-footnote">所有牌局由服务端同步 · 结算后可查看公平验证</p>
        </section>
        <section v-if="gameMounted" v-show="page==='game'" class="game-layout">
          <div class="game-toolbar">
            <div><div class="eyebrow">LIVE GAME</div><h2>好友牌桌</h2><p>{{gameStatus}}</p></div>
            <div class="game-toolbar-actions">
              <button type="button" class="ghost-button compact" @click="reloadGame">重新加载</button>
              <button type="button" class="ghost-button compact" @click="page='lobby'">返回网站</button>
            </div>
          </div>
          <div class="game-frame-shell">
            <iframe :key="gameInstance" ref="gameFrame" src="/game/index.html"
              title="公平斗地主 Cocos4 互动牌桌" referrerpolicy="no-referrer" allow="fullscreen" />
            <div v-if="!isReady" class="game-loading-cover" role="status" aria-live="polite">
              <div class="game-loading-content">
                <div class="game-loading-suit" aria-hidden="true">♠</div>
                <h3>公平斗地主</h3>
                <p class="game-loading-status">{{gameStatus}}</p>
                <div class="game-loading-track" role="progressbar"
                  :aria-valuenow="confirmedStages" aria-valuemin="0" aria-valuemax="4"
                  aria-label="游戏启动阶段">
                  <div class="game-loading-fill" :style="{width:stageWidth}"></div>
                </div>
                <p class="game-loading-step">已确认 {{confirmedStages}} / 4 个启动阶段 · 非下载百分比</p>
                <p v-if="gameLoadSlow && gamePhase!=='failed'" class="game-loading-note">加载时间较长，可以继续等待或重新尝试。</p>
                <p v-if="gameLoadError" class="game-loading-error" role="alert">{{gameLoadError}}</p>
                <div v-if="gameLoadSlow || gamePhase==='failed'" class="game-loading-actions">
                  <button type="button" class="primary-button" @click="reloadGame">重新加载游戏</button>
                  <button type="button" class="ghost-button compact" @click="page='lobby'">返回网站</button>
                </div>
                <p class="game-loading-footnote">牌局进度由服务器保存 · 返回网站不会重新加载引擎</p>
              </div>
            </div>
            <div class="frame-footer"><span><span class="status-dot" :class="{ready:isReady}"></span> {{gameStatus}}</span><span>游戏进度由服务器保存</span></div>
          </div>
        </section>
      </template>
    </main>

    <footer class="site-footer"><span>AZP / FAIR DOUDIZHU</span><span>共享游戏协议 · Vue 3 + Cocos Creator + Go</span></footer>
  </div>
</template>
