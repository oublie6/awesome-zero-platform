import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFile} from 'node:fs/promises'

test('Vue is an account entry, not a second room lobby',async()=>{
 const source=await readFile(new URL('../src/App.vue',import.meta.url),'utf8')
 const lobby=source.split('<section v-else-if="page===\'lobby\'"')[1]
   .split('<section v-else-if="page===\'game\'"')[0]
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
