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
const gameStatus=ref('等待加载游戏资源')
const gameFrame=ref<HTMLIFrameElement|null>(null)
const gameInstance=ref(0)
const loggedIn=computed(()=>!!account.value)
const isReady=computed(()=>gameStatus.value==='游戏就绪')
const protocol='azp-doudizhu-frame-v1'

function communicate(type:'SESSION_AVAILABLE'|'SESSION_REVOKED'|'OPEN_ROOMS'){
  const frame=gameFrame.value?.contentWindow
  if(!frame)return
  // Room IDs are non-secret user intentions, never client-authoritative state.
  frame.postMessage({protocol,type},window.location.origin)
}
function onGameMessage(event:MessageEvent){
  if(page.value!=='game' || event.source!==gameFrame.value?.contentWindow ||
     event.origin!==window.location.origin)return
  const message=event.data
  if(!message || typeof message!=='object' || message.protocol!==protocol ||
     typeof message.type!=='string')return
  if(message.type==='GAME_READY'){
    if(account.value && client?.authenticated){
      gameStatus.value='验证服务器会话中'
      communicate('SESSION_AVAILABLE')
    }else {
      gameStatus.value='需要先登录平台账号'
      page.value='login'
    }
  }else if(message.type==='SESSION_ACTIVE'){
    gameStatus.value='游戏就绪'
    communicate('OPEN_ROOMS')
  }else if(message.type==='SESSION_FAILED'){
    gameStatus.value='游戏连接失败，请检查服务端 HTTPS/WSS 配置或重新进入'
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
  // Drop the game frame immediately: a previous account must not continue
  // submitting commands while logout's server revocation is in flight.
  page.value='home'
  account.value=null
  password.value=''
  try{await client?.logout()}
  catch{error.value='已清理浏览器会话，但服务端注销状态需要确认'}
  finally{busy.value=false}
}
function openGame(){
  if(!account.value)return
  error.value=''
  // The authenticated Cocos lobby is the only user-facing room entry.
  // No room mutation is synthesized by Vue; server-authoritative recovery
  // and exact UNKNOWN reconciliation stay within the game coordinator.
  gameStatus.value='等待 Cocos4 Web 产物启动'
  gameInstance.value++
  page.value='game'
}

onMounted(()=>{
  window.addEventListener('message',onGameMessage)
  void initialize()
})
onUnmounted(()=>window.removeEventListener('message',onGameMessage))
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
        <section v-else-if="page==='game'" class="game-layout">
          <div class="game-toolbar">
            <div><div class="eyebrow">LIVE GAME</div><h2>好友牌桌</h2><p>{{gameStatus}}</p></div>
            <div class="game-toolbar-actions">
              <button type="button" class="ghost-button compact" @click="gameInstance++;gameStatus='重新加载游戏中'">重新加载</button>
              <button type="button" class="ghost-button compact" @click="page='lobby'">返回网站</button>
            </div>
          </div>
          <div class="game-frame-shell">
            <iframe :key="gameInstance" ref="gameFrame" src="/game/index.html"
              title="公平斗地主 Cocos4 互动牌桌" referrerpolicy="no-referrer" allow="fullscreen" />
            <div class="frame-footer"><span><span class="status-dot" :class="{ready:isReady}"></span> {{gameStatus}}</span><span>游戏进度由服务器保存</span></div>
          </div>
        </section>
      </template>
    </main>

    <footer class="site-footer"><span>AZP / FAIR DOUDIZHU</span><span>共享游戏协议 · Vue 3 + Cocos Creator + Go</span></footer>
  </div>
</template>
