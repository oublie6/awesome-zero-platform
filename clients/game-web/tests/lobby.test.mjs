import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFile} from 'node:fs/promises'

test('Vue is an account entry, not a second room lobby',async()=>{
 const source=await readFile(new URL('../src/App.vue',import.meta.url),'utf8')
 const lobby=source.split('<section v-else-if="page===\'lobby\'"')[1]
   .split('<section v-if="gameMounted"')[0]
 assert.equal((lobby.match(/@click="openGame\(\)"/g)||[]).length,1)
 assert.match(lobby,/进入游戏大厅/)
 assert.doesNotMatch(lobby,/openGame\('(CREATE_ROOM|JOIN_ROOM)'\)/)
 assert.doesNotMatch(lobby,/恢复房间|输入朋友提供的房间号|创建好友房间/)
 assert.match(source,/event\.source!==gameFrame\.value\?\.contentWindow/)
 assert.match(source,/event\.origin!==window\.location\.origin/)
 assert.match(source,/communicate\('SESSION_AVAILABLE'\)/)
 assert.match(source,/communicate\('OPEN_ROOMS'\)/)
 assert.match(source,/communicate\('SESSION_REVOKED'\)/)
 assert.doesNotMatch(source,/postMessage\(\{protocol,type,.*roomId/)
})

test('Goal0065 same-session site navigation retains iframe while logout destroys it',async()=>{
 const src=await readFile(new URL('../src/App.vue',import.meta.url),'utf8')
 const open=src.split('function openGame(){')[1].split('onMounted(')[0]
 assert.match(open,/if\(!gameMounted\.value \|\| gamePhase\.value==='failed'\)reloadGame\(\)/)
 assert.doesNotMatch(open,/gameInstance\.value\+\+/)
 const logout=src.split('async function logout(){')[1].split('function openGame(){')[0]
 assert.ok(logout.indexOf('gameMounted.value=false')<logout.indexOf('account.value=null'))
 assert.match(src,/v-if="gameMounted"[\s\S]*?:class="\['game-layout'/)
 assert.match(src,/:aria-hidden="page!=='game'" :inert="page!=='game'"/)
 assert.doesNotMatch(src,/v-show="page==='game'"/)
 assert.match(src,/:key="gameInstance"/)
 assert.match(src,/event\.source!==gameFrame\.value\?\.contentWindow/)
 assert.match(src,/event\.origin!==window\.location\.origin/)
})
test('Goal0065 startup reports actual stages and offers bounded failure/retry',async()=>{
 const source=await readFile(new URL('../src/App.vue',import.meta.url),'utf8')
 assert.match(source,/message\.type==='BOOT_STAGE'/)
 assert.match(source,/message\.type==='BOOT_FAILED'/)
 assert.match(source,/message\.type==='BOOT_RETRY_REQUESTED'/)
 assert.match(source,/message\.type==='SESSION_ACTIVE'/)
 assert.match(source,/gamePhase\.value='ready'/)
 assert.match(source,/aria-valuemax="4"/)
 assert.match(source,/gameLoadError/)
 assert.match(source,/loadDeadline=setTimeout/)
 assert.doesNotMatch(source,/Math\.random|setInterval\(/)
})

test('Goal0065 R1: hidden iframe messages cannot override fail-closed boot lifecycle',async()=>{
 const source=await readFile(new URL('../src/App.vue',import.meta.url),'utf8')
 assert.match(source,/communicate\('BOOT_PAUSE'\)/)
 assert.match(source,/communicate\('BOOT_RESUME'\)/)
 assert.match(source,/\{flush:'post'\}/)
 assert.match(source,/if\(page\.value==='game' && !document\.hidden\)\s*failGameLoad\(/)
 assert.match(source,/if\(page\.value==='game'\)reloadGame\(\)/)
 assert.match(source,/if\(gamePhase\.value==='failed'\)return/)
 assert.match(source,/if\(gamePhase\.value!=='session' \|\| !account\.value \|\| !client\?\.authenticated\)return/)
 assert.match(source,/event\.source!==gameFrame\.value\?\.contentWindow/)
 assert.match(source,/event\.origin!==window\.location\.origin/)
})

test('Goal0065 R1: background tabs also suspend Vue watchdog',async()=>{
 const source=await readFile(new URL('../src/App.vue',import.meta.url),'utf8')
 assert.match(source,/if\(isReady\.value \|\| document\.hidden \|\| page\.value!=='game'\)return/)
 assert.match(source,/document\.addEventListener\('visibilitychange',onVisibilityChange\)/)
 assert.match(source,/document\.removeEventListener\('visibilitychange',onVisibilityChange\)/)
 assert.match(source,/if\(document\.hidden\)\{\s*clearLoadWatchdog\(\)/)
})

test('Goal0065 Round2: native CSS/session failure stays deferred in retained background iframe',async()=>{
 const source=await readFile(new URL('../src/App.vue',import.meta.url),'utf8')
 const boot=source.split("}else if(message.type==='BOOT_FAILED'")[1].split("}else if(message.type==='BOOT_RETRY_REQUESTED'")[0]
 const session=source.split("}else if(message.type==='SESSION_FAILED'")[1].split('\n}')[0]
 assert.match(boot,/page\.value==='game' && !document\.hidden/)
 assert.match(session,/page\.value==='game' && !document\.hidden/)
 assert.match(source,/if\(gamePhase\.value!=='session' \|\| !account\.value \|\| !client\?\.authenticated\)return/)
})

test('Goal0065 R3: parked iframe keeps its geometry but cannot receive interaction',async()=>{
 const source=await readFile(new URL('../src/App.vue',import.meta.url),'utf8')
 const css=await readFile(new URL('../src/styles.css',import.meta.url),'utf8')
 assert.match(source,/:class="\['game-layout',\{'game-layout-parked':page!=='game'\}\]"/)
 assert.match(source,/:aria-hidden="page!=='game'" :inert="page!=='game'"/)
 assert.doesNotMatch(source,/v-show="page==='game'"/)
 assert.match(css,/\.page-body\{position:relative\}/)
 const parked=css.split('.game-layout.game-layout-parked{')[1].split('}')[0]
 assert.match(parked,/position:absolute/)
 assert.match(parked,/left:-200vw/)
 assert.match(parked,/width:100%/)
 assert.match(parked,/pointer-events:none/)
 assert.doesNotMatch(parked,/display\s*:\s*none|visibility\s*:\s*hidden|width\s*:\s*0/)
 assert.match(source,/if\(gamePhase\.value==='session' && !document\.hidden\)/)
})

test('G65 R9: external session replacement drops old host presentation without revoking replacement',async()=>{
 const source=await readFile(new URL('../src/App.vue',import.meta.url),'utf8')
 const reconcile=source.split('function reconcileLocalIdentity(){')[1].split('function onStorageChange')[0]
 assert.match(reconcile,/client\?\.authenticated/)
 assert.match(reconcile,/communicate\('SESSION_REVOKED'\)/)
 assert.match(reconcile,/gameMounted\.value=false/)
 assert.match(reconcile,/account\.value=null/)
 assert.doesNotMatch(reconcile,/\.logout\(|removeItem\(/)
 assert.match(source,/window\.addEventListener\('storage',onStorageChange\)/)
 assert.match(source,/window\.removeEventListener\('storage',onStorageChange\)/)
})

test('G65 R10: unmount releases pending Web identity work without clearing a replacement',async()=>{
 const source=await readFile(new URL('../src/App.vue',import.meta.url),'utf8')
 assert.match(source,/onUnmounted\(\(\)=>\{[\s\S]*?\(\)=>client\?\.dispose\(\)/)
})

test('G65 R14: Vue logout distinguishes confirmed server revocation from uncertain browser cleanup',async()=>{
 const source=await readFile(new URL('../src/App.vue',import.meta.url),'utf8')
 const logout=source.split('async function logout(){')[1].split('function openGame(){')[0]
 assert.match(logout,/result\?\.storageCleanupFailed/)
 assert.match(logout,/服务端已注销，但浏览器本地凭据清理未能确认/)
 assert.match(logout,/服务端注销状态未确认，浏览器可能仍保留凭据/)
 assert.doesNotMatch(logout,/已清理浏览器会话/)
 assert.ok(logout.indexOf('account.value=null')<logout.indexOf('await client?.logout()'))
})
