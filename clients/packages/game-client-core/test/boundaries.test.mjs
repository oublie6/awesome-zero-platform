import {test} from 'node:test'
import assert from 'node:assert/strict'
import ts from 'typescript'
import {readFile} from 'node:fs/promises'
test('pure entrypoints contain no renderer, platform SDK, adapter imports or runtime globals',async()=>{
 for(const path of ['../src/index.ts','../src/socket.ts','../../doudizhu-client/src/index.ts']){
  const source=ts.createSourceFile(path,await readFile(new URL(path,import.meta.url),'utf8'),ts.ScriptTarget.Latest,true)
  const forbidden=new Set(['wx','tt','window','document','navigator','WebSocket','sessionStorage','crypto','URL'])
  const visit=node=>{
   if(ts.isImportDeclaration(node))assert.match(node.moduleSpecifier.text,/^\.\/(index|network-timeout)\.js$/)
   if(ts.isIdentifier(node))assert.equal(forbidden.has(node.text),false,`${path}: ${node.text}`)
   ts.forEachChild(node,visit)
  };visit(source)
 }
})
test('published ESM entrypoints resolve without Cocos/mini-game globals',async()=>{
 for(const name of ['@awesome-zero-platform/game-client-core','@awesome-zero-platform/game-client-core/web','@awesome-zero-platform/game-client-core/minigame','@awesome-zero-platform/game-client-core/platform']) assert.ok(await import(name))
})
