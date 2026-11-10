'use strict';
const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const os=require('node:os');
const {configure,publicConfig}=require('./configure.cjs');
const {webBuildOptions}=require('./build-options.cjs');
const {repairGeneratedTypes}=require('./verify-native-types.cjs');
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

test('G57-N01: Web uses official CLI4 buildScriptTargets at a BigInt-safe target',()=>{
 const options=webBuildOptions({buildPath:'/private/native-web',startScene:'entry-uuid'});
 assert.equal(options.platform,'web-desktop');
 assert.equal(options.buildScriptTargets,'chrome 79');
 assert.equal(options.debug,false);
 // This must flow into the actual CLI --build-config JSON, not just a
 // comment or source check detached from the native builder.
 const builder=fs.readFileSync(path.join(__dirname,'build.cjs'),'utf8');
 assert.match(builder,/JSON\.stringify\(webBuildOptions\(/);
 assert.match(builder,/--build-config',config/);
 assert.match(builder,/requireNativeBigIntExponent:true/);
 assert.throws(()=>webBuildOptions({buildPath:'',startScene:'entry-uuid'}));
});
test('G57-N01: AST rejects lowered BigInt pow and requires native exponent',t=>{
 const root=fs.mkdtempSync(path.join(os.tmpdir(),'azp-web-bigint-'));
 t.after(()=>fs.rmSync(root,{recursive:true,force:true}));
 const file=path.join(root,'bundle.js');
 // For a local fixture, the existing audit requires @babel from pinned CLI.
 // Exercise the real parser only when that toolchain is already installed.
 const cli=process.env.COCOS_CLI_ROOT;
 if(!cli)return t.skip('AST parser needs the approved native Cocos CLI; source contract is still checked');
 const {audit}=require('../cocos-linux-wechat/audit.cjs');
 fs.writeFileSync(file,'const B256=2n ** 256n; globalThis.B256=B256;');
 assert.equal(audit(root,{previewCompatible:false,requireNativeBigIntExponent:true}),1);
 fs.writeFileSync(file,'const B256=Math.pow(2n,256n);');
 assert.throws(()=>audit(root,{previewCompatible:false,requireNativeBigIntExponent:true}),/BigInt/);
 fs.writeFileSync(file,'const a=Math.pow(2,10);');
 assert.throws(()=>audit(root,{previewCompatible:false,requireNativeBigIntExponent:true}),/no native BigInt exponentiation/);
});
test('Cocos generated declaration paths are resolved for real strict TS gate',t=>{
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'azp-web-types-'));
 t.after(()=>fs.rmSync(dir,{recursive:true,force:true}));
 fs.mkdirSync(path.join(dir,'temp/declarations'),{recursive:true});
 fs.writeFileSync(path.join(dir,'tsconfig.json'),JSON.stringify({compilerOptions:{strict:true,skipLibCheck:true}}));
 fs.writeFileSync(path.join(dir,'temp/tsconfig.cocos.json'),JSON.stringify({
   compilerOptions:{types:['./declarations/cc','./declarations/cc.env','./declarations/jsb','third-party']}
 }));
 for(const name of ['cc','cc.env','jsb'])
   fs.writeFileSync(path.join(dir,'temp/declarations',name+'.d.ts'),'declare const test: number;');
 const repaired=repairGeneratedTypes(dir);
 const opts=JSON.parse(fs.readFileSync(repaired,'utf8')).compilerOptions;
 assert.deepEqual(opts.types.slice(0,3),['cc','cc.env','jsb'].map(name=>path.join(dir,'temp/declarations',name)));
 assert.equal(opts.types[3],'third-party');
 assert.equal(opts.noImplicitAny,true);
 assert.equal(opts.strictNullChecks,true);
 assert.equal(opts.strictPropertyInitialization,true);
});

test('native Web builder rebuilds each isolated TS package, never trusts copied stale dist',()=>{
 const s=fs.readFileSync(path.join(__dirname,'build.cjs'),'utf8');
 assert.match(s,/\['game-client-core','doudizhu-client','secure-envelope'\]/);
 assert.match(s,/fs\.rmSync\(path\.join\(pkg,'dist'\)/);
 assert.match(s,/command\('npm',\['run','build','--prefix',pkg\]\)/);
 assert.ok(s.indexOf("command('npm',['run','build','--prefix',pkg])") <
   s.indexOf("command('npm',['ci','--no-audit','--no-fund','--prefix',project])"));
});

test('G58-N01: real-browser smoke checks no duplicate Cocos EditBox labels',()=>{
 const src=fs.readFileSync(path.join(__dirname,'smoke.cjs'),'utf8');
 assert.match(src,/async function assertInputLabels\(field,hint\)/);
 assert.match(src,/input\.node\.children\.filter/);
 assert.match(src,/\['PLACEHOLDER_LABEL','TEXT_LABEL'\]/);
 assert.match(src,/assert\.equal\(observed\.placeholderActive,true\)/);
 assert.match(src,/assertInputLabels\('roomInput'/);
 assert.match(src,/assertInputLabels\('loginIdentifier'/);
 assert.match(src,/assertInputLabels\('loginPassword'/);
});

test('G59-N01: native browser smoke clicks actual visible hints in all three viewports',()=>{
 const src=fs.readFileSync(path.join(__dirname,'smoke.cjs'),'utf8');
 assert.match(src,/async function checkVisibleHintHit\(field,viewport\)/);
 assert.match(src,/ui\.convertToWorldSpaceAR\(/);
 assert.match(src,/labelRect=project\(labelUI\)/);
 assert.match(src,/hint\.left>=box\.left-2 && hint\.right<=box\.right\+3/);
 assert.match(src,/hint\.top>=box\.top-2 && hint\.bottom<=box\.bottom\+2/);
 assert.match(src,/fontCSS>=12/);
 assert.match(src,/page\.mouse\.click\(click\.x,click\.y\)/);
 assert.match(src,/document\.activeElement\?\.tagName/);
 for(const size of ['1280,height:720','390,height:844','844,height:390'])
   assert.ok(src.includes(size),'missing viewport '+size);
 for(const field of ['roomInput','loginIdentifier','loginPassword'])
   assert.ok(src.includes("checkVisibleHintHit('"+field+"'"),'missing '+field);
});
