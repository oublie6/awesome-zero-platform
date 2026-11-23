'use strict';
const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const os=require('node:os');
const vm=require('node:vm');
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
 assert.match(html,/window\.__AZP_CONFIG_READY__ = new Promise/);
 assert.match(html,/node\.async = true/);
 assert.match(html,/node\.onerror = function\(\) \{ finish\(false\)/);
 assert.doesNotMatch(html,/<script\s+src="\.\/game-config\.js"/);
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

test('G60-N01: credential input mode is set before native scene activation',()=>{
 const src=fs.readFileSync(path.join(__dirname,
   '../../clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts'),'utf8');
 const factory=src.split('private createInput(')[1].split('private createRoomInput(')[0];
 assert.ok(factory.indexOf('node.addComponent(EditBox)')<
   factory.indexOf('this.content!.addChild(node)'));
 assert.ok(factory.indexOf('input.inputMode=singleLine?')<
   factory.indexOf('this.content!.addChild(node)'));
 assert.ok(factory.indexOf('input.inputFlag=EditBox.InputFlag.PASSWORD')<
   factory.indexOf('this.content!.addChild(node)'));
 const login=src.split('private createLoginInput(')[1].split('private addMenu(')[0];
 assert.match(login,/secret,true/);
 const smoke=fs.readFileSync(path.join(__dirname,'smoke.cjs'),'utf8');
 assert.match(smoke,/native\.type,field==='loginPassword'\?'password':'text'/);
 assert.match(smoke,/native\.tag,'INPUT'/);
 assert.match(smoke,/requestAnimationFrame\(\(\)=>requestAnimationFrame\(resolve\)\)/);
});

test('Goal0065 independent Cocos HTML renders splash before engine scripts and reports actual events',()=>{
 const template=fs.readFileSync(path.join(__dirname,
  '../../clients/fair-doudizhu-cocos/build-templates/web-desktop/index.ejs'),'utf8');
 const splash=template.indexOf('id="azp-boot"');
 const engine=template.indexOf('<%- include(cocosTemplate, {}) %>');
 assert.ok(splash>0 && engine>splash,'splash must exist before the Cocos engine bootstrap');
 assert.match(template,/window\.__AZP_BOOT__ = \{ stage, done, fail \}/);
 assert.match(template,/window\.__AZP_BOOT__\?\.stage\('engine'\)/);
 assert.match(template,/message\('BOOT_STAGE', name\)/);
 assert.match(template,/message\('BOOT_FAILED'\)/);
 assert.match(template,/message\('BOOT_RETRY_REQUESTED'\)/);
 const bootstrap=fs.readFileSync(path.join(__dirname,
  '../../clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts'),'utf8');
 assert.match(bootstrap,/this\.reportNativeBoot\('stage','scene'\)/);
 assert.match(bootstrap,/this\.reportNativeBoot\(active\?'done':'fail'\)/);
 assert.ok(bootstrap.indexOf("if(active)this.renderScreen('rooms')")<
   bootstrap.indexOf("this.notifyWebHost(active?'SESSION_ACTIVE':'SESSION_FAILED')"));
})


test('Goal0065 R1: socket must actually be connected before GAME_READY completion',()=>{
 const source=fs.readFileSync(path.join(__dirname,
   '../../clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts'),'utf8');
 assert.match(source,/const active=!!this\.online\?\.session\.authenticated\s*&&\s*!!this\.online\?\.shell\.commandTransportReady/);
 assert.ok(source.indexOf("if(active)this.renderScreen('rooms')") <
   source.indexOf("this.notifyWebHost(active?'SESSION_ACTIVE':'SESSION_FAILED')"));
 assert.match(source,/__AZP_CONFIG_READY__\?:Promise<boolean>/);
 assert.match(source,/if\(ok\)this\.notifyWebHost\('GAME_READY'\)/);
});
test('Goal0065 R1: Cocos watchdog really pauses while hidden and resumes',()=>{
 const template=fs.readFileSync(path.join(__dirname,
   '../../clients/fair-doudizhu-cocos/build-templates/web-desktop/index.ejs'),'utf8');
 const script=template.split('<body>')[1].match(/<script>([\s\S]*?)<\/script>/)?.[1];
 assert.ok(script,'native loader bootstrap must exist in body before Cocos');
 const handlers=new Map(), sent=[], clocks=new Map();
 let count=0, frameVisible=true, hidden=false;
 const classes=new Set();
 const root={
   classList:{add:x=>classes.add(x),remove:x=>classes.delete(x),contains:x=>classes.has(x)},
   querySelector:()=>({setAttribute:()=>{}}),remove:()=>{}
 };
 const elements={
   'azp-boot':root,'azp-boot-fill':{style:{}},
   'azp-boot-status':{textContent:''},'azp-boot-step':{textContent:''},
   'azp-boot-retry':{addEventListener:()=>{}}
 };
 const doc={
   get hidden(){return hidden},
   getElementById:x=>elements[x],
   addEventListener:(kind,fn)=>handlers.set('document:'+kind,fn)
 };
 const parent={postMessage:m=>sent.push(m)};
 const win={
   parent,location:{origin:'https://test.example'},
   frameElement:{getClientRects:()=>frameVisible?[{width:500}]:[]},
   addEventListener:(kind,fn)=>handlers.set('window:'+kind,fn)
 };
 vm.runInNewContext(script,{
   window:win,document:doc,console,
   HTMLScriptElement:class{},HTMLLinkElement:class{},
   setTimeout:(fn,ms)=>{clocks.set(++count,{fn,ms});return count},
   clearTimeout:id=>clocks.delete(id)
 });
 assert.equal(clocks.size,2);
 frameVisible=false;
 handlers.get('window:message')({
   source:parent,origin:win.location.origin,
   data:{protocol:'azp-doudizhu-frame-v1',type:'BOOT_PAUSE'}
 });
 assert.equal(clocks.size,0,'hidden loader must stop both native timers');
 win.__AZP_BOOT__.fail();
 assert.equal(classes.has('failed'),false,'real error is deferred while hidden');
 frameVisible=true;
 handlers.get('window:message')({
   source:parent,origin:win.location.origin,
   data:{protocol:'azp-doudizhu-frame-v1',type:'BOOT_RESUME'}
 });
 assert.equal(classes.has('failed'),true,'real deferred error must surface when visible');
 assert.equal(sent.filter(x=>x.type==='BOOT_FAILED').length,1);
 // Paused watchdogs are also suspended by the browser when tab is hidden.
 assert.equal(clocks.size,0);
});
test('Goal0065 R1: visible suspended loader re-arms and stale origin cannot resume',()=>{
 const html=fs.readFileSync(path.join(__dirname,
   '../../clients/fair-doudizhu-cocos/build-templates/web-desktop/index.ejs'),'utf8');
 assert.match(html,/event\.source !== window\.parent \|\| event\.origin !== window\.location\.origin/);
 assert.match(html,/event\.data\.type === 'BOOT_PAUSE'/);
 assert.match(html,/event\.data\.type === 'BOOT_RESUME'/);
 assert.match(html,/suspended = !visible\(\)/);
 assert.match(html,/if \(suspended \|\| !visible\(\)\)/);
 assert.match(html,/deferredFailure = true/);
 assert.match(html,/document\.addEventListener\('visibilitychange'/);
});
test('Goal0065 R1: public config is async without exposing startup to a sync head stall',async t=>{
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'azp-config-async-'));
 t.after(()=>fs.rmSync(dir,{recursive:true,force:true}));
 fs.writeFileSync(path.join(dir,'index.html'),'<html><head></head><body><script src="engine.js"></script></body></html>');
 configure(dir,{apiBase:'https://api.example.test',socketEndpoint:'wss://api.example.test/ws'});
 const html=fs.readFileSync(path.join(dir,'index.html'),'utf8');
 const loader=html.match(/<script>([\s\S]*?)<\/script>/)?.[1];
 assert.ok(loader);
 let pending;
 const doc={head:{appendChild:node=>{pending=node}},createElement:()=>({})};
 const context={window:{},document:doc,setTimeout:()=>1,clearTimeout:()=>{}};
 vm.runInNewContext(loader,context);
 assert.equal(pending.src,'./game-config.js');
 assert.equal(pending.async,true);
 pending.onload();
 assert.equal(await context.window.__AZP_CONFIG_READY__,true);
 assert.ok(html.indexOf('window.__AZP_CONFIG_READY__') < html.indexOf('engine.js'));
});
