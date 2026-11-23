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
 assert.match(src,/v-if="gameMounted" v-show="page==='game'"/)
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
 assert.match(source,/if\(page\.value==='game'\)failGameLoad\(/)
 assert.match(source,/if\(page\.value==='game'\)reloadGame\(\)/)
 assert.match(source,/if\(gamePhase\.value==='failed'\)return/)
 assert.match(source,/if\(gamePhase\.value!=='session' \|\| !account\.value \|\| !client\?\.authenticated\)return/)
 assert.match(source,/event\.source!==gameFrame\.value\?\.contentWindow/)
 assert.match(source,/event\.origin!==window\.location\.origin/)
})
