'use strict';
const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const os=require('node:os');
const {configure,publicConfig}=require('./configure.cjs');
test('Web endpoints require HTTPS/WSS without credentials or fragments',()=>{
 const ok=publicConfig({apiBase:'https://api.example.test',socketEndpoint:'wss://api.example.test/ws'});
 assert.equal(ok.apiBase,'https://api.example.test');
 assert.equal(ok.socketEndpoint,'wss://api.example.test/ws');
 assert.equal(publicConfig({}),null);
 for(const bad of [
  {apiBase:'http://api.example.test',socketEndpoint:'wss://api.example.test/ws'},
  {apiBase:'https://api.example.test',socketEndpoint:'ws://api.example.test/ws'},
  {apiBase:'https://name:secret@api.example.test',socketEndpoint:'wss://api.example.test/ws'},
  {apiBase:'https://api.example.test/?token=a',socketEndpoint:'wss://api.example.test/ws'},
  {apiBase:'https://api.example.test'}
 ])assert.throws(()=>publicConfig(bad));
});
test('Cocos Web index injects public config before game scripts, idempotently',t=>{
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'azp-web-config-'));
 t.after(()=>fs.rmSync(dir,{recursive:true,force:true}));
 const index=path.join(dir,'index.html');
 fs.writeFileSync(index,'<!doctype html><html><head><title>Game</title></head><body><script src="index.js"></script></body></html>');
 configure(dir,{apiBase:'https://api.example.test',socketEndpoint:'wss://api.example.test/ws'});
 configure(dir,{apiBase:'https://api.example.test',socketEndpoint:'wss://api.example.test/ws'});
 const html=fs.readFileSync(index,'utf8');
 assert.equal(html.split('game-config.js').length-1,1);
 assert.ok(html.indexOf('game-config.js')<html.indexOf('index.js'));
 const publicJS=fs.readFileSync(path.join(dir,'game-config.js'),'utf8');
 assert.match(publicJS,/https:\/\/api.example.test/);
 assert.doesNotMatch(publicJS,/AppSecret|accessToken|password/);
});
test('missing endpoints produce deliberate fail-closed static asset',t=>{
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'azp-web-empty-'));
 t.after(()=>fs.rmSync(dir,{recursive:true,force:true}));
 fs.writeFileSync(path.join(dir,'index.html'),'<head></head><body></body>');
 assert.equal(configure(dir,{}),null);
 assert.match(fs.readFileSync(path.join(dir,'game-config.js'),'utf8'),/= null;/);
});
test('Web build invokes real pinned Cocos CLI web-desktop export',()=>{
 const source=fs.readFileSync(path.join(__dirname,'build.cjs'),'utf8');
 assert.match(source,/--platform','web-desktop'/);
 assert.match(source,/verify-toolchain.cjs/);
 assert.match(source,/audit.cjs/);
 assert.doesNotMatch(source,/mock|fakeWebOutput/);
});

test('browser smoke exercises rendered Cocos scene and Web login inputs',()=>{
 const src=fs.readFileSync(path.join(__dirname,'smoke.cjs'),'utf8');
 assert.match(src,/System\.import\('cc'\)/);
 assert.match(src,/page\.mouse\.click/);
 assert.match(src,/password must use secret EditBox mode/);
 assert.match(src,/CancelWebLogin/);
});
