'use strict';
// Linux-native Cocos4 Web export. Does not generate or simulate a browser build.
const assert=require('node:assert/strict');
const fs=require('node:fs');
const os=require('node:os');
const path=require('node:path');
const {spawnSync}=require('node:child_process');
const {webBuildOptions}=require('./build-options.cjs');
const {verifyNativeTypes}=require('./verify-native-types.cjs');
if(process.platform!=='linux')throw Error('Linux-only Cocos web export');
const root=path.resolve(__dirname,'../..');
const cli=process.env.COCOS_CLI_ROOT;
if(!cli)throw Error('COCOS_CLI_ROOT must reference the pinned compiled official CLI');
function command(name,args,env){
  const result=spawnSync(name,args,{cwd:root,encoding:'utf8',timeout:600000,stdio:'inherit',env:env||process.env});
  if(result.error)throw result.error;
  if(result.status!==0)throw Error(name+' exited '+result.status);
}
command(process.execPath,[path.resolve(root,'scripts/cocos-linux-wechat/verify-toolchain.cjs')]);
const work=fs.mkdtempSync(path.join(os.tmpdir(),'azp-native-cocos-web-'));
const output=path.resolve(process.argv[2]||path.join(root,'.runtime/linux-native-web'));
try{
  const source=path.join(root,'clients'),copy=path.join(work,'clients');
  fs.cpSync(source,copy,{recursive:true,filter:filename=>{
    const relative=path.relative(source,filename).split(path.sep);
    return !relative.some(x=>['node_modules','.git','temp','build','.runtime'].includes(x));
  }});
  const project=path.join(copy,'fair-doudizhu-cocos');
  const settings=path.join(project,'settings');
  fs.mkdirSync(settings,{recursive:true});
  fs.copyFileSync(path.resolve(root,'scripts/cocos-linux-wechat/engine-profile.json'),
    path.join(settings,'cocos.config.json'));
  for(const packageName of ['game-client-core','doudizhu-client'])
    command('npm',['ci','--no-audit','--no-fund','--prefix',path.join(copy,'packages',packageName)]);
  command('npm',['install','--package-lock=false','--ignore-scripts','--no-audit','--no-fund',
    '--prefix',path.join(copy,'packages','secure-envelope')]);
  command('npm',['ci','--no-audit','--no-fund','--prefix',project]);
  const scene=JSON.parse(fs.readFileSync(path.join(project,'assets/scenes/Entry.scene.meta'),'utf8'));
  assert.match(scene.uuid,/^[a-f0-9-]{36}$/i);
  const config=path.join(work,'web-build-options.json');
  fs.writeFileSync(config,JSON.stringify(webBuildOptions({
    buildPath:output,startScene:scene.uuid
  }),null,2));
  fs.mkdirSync(output,{recursive:true});
  const env={...process.env,NODE_OPTIONS:'--max-old-space-size=2048',
    PATH:path.join(cli,'node_modules/.bin')+path.delimiter+process.env.PATH};
  command('xvfb-run',['-a',process.execPath,path.join(cli,'dist/cli.js'),
    '--no-interactive','build','--project',project,'--platform','web-desktop','--build-config',config],env);
  // CLI4 reports success even when its internal tsc logs TS2688; check
  // the actual generated Cocos4 declaration set with the project strict gate.
  verifyNativeTypes(project);
  const pages=[];
  function scan(dir,depth=0){
    if(depth>5)return;
    for(const entry of fs.readdirSync(dir,{withFileTypes:true})){
      const file=path.join(dir,entry.name);
      if(entry.isDirectory())scan(file,depth+1);
      else if(entry.name==='index.html')pages.push(dir);
    }
  }
  scan(output);
  if(pages.length!==1)throw Error('Expected one actual Cocos web index; found '+pages.length);
  const location=pages[0];
  const jsFiles=[];
  function collect(dir){
    for(const entry of fs.readdirSync(dir,{withFileTypes:true})){
      const file=path.join(dir,entry.name);
      if(entry.isDirectory())collect(file);
      else if(entry.name.endsWith('.js'))jsFiles.push(file);
    }
  }
  collect(location);
  if(!jsFiles.length)throw Error('no emitted web JS');
  const checked=require('../cocos-linux-wechat/audit.cjs').audit(location,{
    previewCompatible:false,requireNativeBigIntExponent:true
  });
  console.log('AST verified '+checked+' real Web JS files, including native BigInt exponentiation');
  const roots=process.env.WEB_REVEAL_MANIFEST_ROOTS?
    JSON.parse(process.env.WEB_REVEAL_MANIFEST_ROOTS):{};
  require('./configure.cjs').configure(location,{
    apiBase:process.env.WEB_API_BASE,
    socketEndpoint:process.env.WEB_SOCKET_ENDPOINT,
    revealManifestRoots:roots
  });
  console.log('Pinned Linux-native Cocos Web export verified: '+location+' ('+jsFiles.length+' JS files)');
}finally{
  if(process.env.COCOS_KEEP_BUILD_INPUTS==='1')console.log('Isolated web inputs retained: '+work);
  else fs.rmSync(work,{recursive:true,force:true});
}
