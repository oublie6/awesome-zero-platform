'use strict';
// Run only against real Cocos4 web-desktop output; no fake engine or login.
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const http=require('node:http');
const {chromium}=require(process.env.COCOS_SMOKE_PLAYWRIGHT_MODULE||
  '../../clients/admin-web/node_modules/playwright');
async function main(){
 const root=path.resolve(process.argv[2]||'');
 const evidence=path.resolve(process.argv[3]||'.runtime/web-smoke');
 if(!fs.statSync(path.join(root,'index.html')).isFile())throw Error('Requires actual built Cocos web index.html');
 fs.mkdirSync(evidence,{recursive:true,mode:0o700});
 const failures=[],errors=[],checks=[];
 const types={'.html':'text/html','.js':'text/javascript','.json':'application/json',
   '.css':'text/css','.wasm':'application/wasm','.png':'image/png','.ttf':'font/ttf'};
 const server=http.createServer((req,res)=>{
   const pathname=decodeURIComponent(new URL(req.url,'http://localhost').pathname);
   const file=path.resolve(root,'.'+(pathname==='/'?'/index.html':pathname));
   if(file!==root && !file.startsWith(root+path.sep)){res.writeHead(403);res.end();return}
   if(!fs.existsSync(file)||!fs.statSync(file).isFile()){
     failures.push({path:pathname,status:404});res.writeHead(404);res.end();return
   }
   res.writeHead(200,{'Content-Type':types[path.extname(file)]||'application/octet-stream',
      'Cache-Control':'no-store'});
   fs.createReadStream(file).pipe(res);
 });
 await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
 let browser,page;
 const report={result:'failed',checks,errors,failures};
 try{
   browser=await chromium.launch({headless:true,args:['--use-angle=swiftshader',
      '--enable-unsafe-swiftshader']});
   page=await browser.newPage({viewport:{width:1280,height:720}});
   page.on('pageerror',err=>errors.push(err.message));
   page.on('console',msg=>{if(msg.type()==='error')errors.push(msg.text())});
   page.on('requestfailed',req=>failures.push({url:req.url(),error:req.failure()?.errorText}));
   await page.goto('http://127.0.0.1:'+server.address().port+'/',{waitUntil:'load'});
   await page.waitForFunction(()=>!!globalThis.System?.import,undefined,{timeout:90000});
   await page.evaluate(async()=>{globalThis.__smokeCC=await System.import('cc')});
   const state=()=>page.evaluate(()=>{
     const cc=globalThis.__smokeCC;
     const canvas=cc.director.getScene()?.getChildByName('Canvas');
     const bootstrap=canvas?.getComponent('GameBootstrap');
     return bootstrap?{screen:bootstrap.screen,
       buttons:bootstrap.buttons.map(node=>node.name),
       title:bootstrap.title?.string,
       identifier:bootstrap.loginIdentifier?.string??null,
       passwordMode:bootstrap.loginPassword?.inputFlag??null,
       passwordFlag:cc.EditBox.InputFlag.PASSWORD}:null;
   });
   await page.waitForFunction(()=>{
     const cc=globalThis.__smokeCC;
     return !!cc.director.getScene()?.getChildByName('Canvas')?.getComponent('GameBootstrap')?.buttons.length;
   },undefined,{timeout:90000});
   async function tap(name,expected){
     const point=await page.evaluate(name=>{
       const cc=globalThis.__smokeCC;
       const canvas=cc.director.getScene().getChildByName('Canvas');
       const bootstrap=canvas.getComponent('GameBootstrap');
       const node=bootstrap.buttons.find(button=>button.name===name);
       if(!node)throw Error('Missing visible game button: '+name);
       const camera=canvas.getChildByName('Camera').getComponent(cc.Camera);
       const screen=camera.worldToScreen(node.worldPosition);
       const element=document.getElementById('GameCanvas');
       const box=element.getBoundingClientRect();
       return {x:box.x+screen.x/element.width*box.width,
         y:box.y+(1-screen.y/element.height)*box.height};
     },name);
     await page.mouse.click(point.x,point.y);
     await page.waitForFunction(expected,undefined,{timeout:15000});
     checks.push(name);
   }
   assert.equal((await state()).title,'公平斗地主');
   await tap('RoomEntry',()=>globalThis.__smokeCC.director.getScene()
     .getChildByName('Canvas').getComponent('GameBootstrap').screen==='rooms');
   await tap('LoginEntry',()=>globalThis.__smokeCC.director.getScene()
     .getChildByName('Canvas').getComponent('GameBootstrap').screen==='login');
   const form=await state();
   assert.equal(form.passwordMode,form.passwordFlag,'password must use secret EditBox mode');
   assert.ok(form.buttons.includes('SubmitWebLogin'));
   await page.screenshot({path:path.join(evidence,'web-login.png')});
   await tap('CancelWebLogin',()=>globalThis.__smokeCC.director.getScene()
     .getChildByName('Canvas').getComponent('GameBootstrap').screen==='rooms');
   assert.deepEqual(errors,[]);
   assert.deepEqual(failures,[]);
   report.result='passed';
 }catch(error){report.error=error.message;throw error}
 finally{
   fs.writeFileSync(path.join(evidence,'web-smoke.json'),JSON.stringify(report,null,2),{mode:0o600});
   await browser?.close();
   await new Promise(resolve=>server.close(resolve));
 }
}
main().catch(error=>{console.error(error);process.exitCode=1});
