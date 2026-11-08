'use strict';
// Non-destructive native Linux exporter. Only a pinned, already-built official
// Cocos CLI/engine is accepted. No Wine, other-OS guest or browser-build relabel.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {spawnSync} = require('node:child_process');
if (process.platform !== 'linux') throw Error('Linux-only Cocos export');
const root=path.resolve(__dirname,'../..');
const cli=process.env.COCOS_CLI_ROOT;
if (!cli) throw Error('COCOS_CLI_ROOT must reference a pinned compiled official CLI');
const pins=JSON.parse(fs.readFileSync(path.join(__dirname,'toolchain.json')));
function command(name,args,env) {
  const result=spawnSync(name,args,{cwd:root,encoding:'utf8',timeout:600000,stdio:'inherit',env:env||process.env});
  if(result.error)throw result.error;
  if(result.status!==0)throw Error(name+' exited '+result.status);
}
command(process.execPath,[path.join(__dirname,'verify-toolchain.cjs')]);
command(process.execPath,[path.join(__dirname,'install.cjs'),cli]);
const work=fs.mkdtempSync(path.join(os.tmpdir(),'azp-native-cocos-'));
const output=path.resolve(process.argv[2]||path.join(root,'.runtime/linux-native-cocos'));
const preserve=process.env.COCOS_KEEP_BUILD_INPUTS==='1';
try {
  const source=path.join(root,'clients');
  const copy=path.join(work,'clients');
  fs.cpSync(source,copy,{recursive:true,filter:filename=>{
    const relative=path.relative(source,filename).split(path.sep);
    return !relative.some(x=>['node_modules','.git','temp','build','.runtime'].includes(x));
  }});
  const project=path.join(copy,'fair-doudizhu-cocos');
  const settings=path.join(project,'settings');
  fs.mkdirSync(settings,{recursive:true});
  fs.copyFileSync(path.join(__dirname,'engine-profile.json'),path.join(settings,'cocos.config.json'));
  // Local file: packages are copied without node_modules. npm ci for the
  // Cocos project does not restore the transitive runtime dependencies in
  // those source directories; the actual CLI follows the file-linked paths.
  // Provision each isolated package explicitly before installing the project.
  for(const packageName of ['game-client-core','doudizhu-client']){
    command('npm',['ci','--no-audit','--no-fund','--prefix',path.join(copy,'packages',packageName)]);
  }
  command('npm',['install','--package-lock=false','--ignore-scripts','--no-audit','--no-fund',
    '--prefix',path.join(copy,'packages','secure-envelope')]);
  command('npm',['ci','--no-audit','--no-fund','--prefix',project]);
  const scene=JSON.parse(fs.readFileSync(path.join(project,'assets/scenes/Entry.scene.meta'),'utf8'));
  assert.match(scene.uuid,/^[a-f0-9-]{36}$/i);
  const optsPath=path.join(work,'wechat-options.json');
  fs.writeFileSync(optsPath,JSON.stringify({
    platform:pins.platform,debug:false,outputName:'fair-doudizhu-wechat',
    buildPath:output,startScene:scene.uuid,
    packages:{wechatgame:{appid:process.env.WECHAT_APPID||'',orientation:'landscape'}}
  },null,2));
  fs.mkdirSync(output,{recursive:true});
  const env={...process.env,NODE_OPTIONS:'--max-old-space-size=2048',
    PATH:path.join(cli,'node_modules/.bin')+path.delimiter+process.env.PATH};
  command('xvfb-run',['-a',process.execPath,path.join(cli,'dist/cli.js'),
    '--no-interactive','build','--project',project,'--platform',pins.platform,'--build-config',optsPath],env);
  const packages=[];
  function visit(directory,level=0){
    if(level>5)return;
    for(const item of fs.readdirSync(directory,{withFileTypes:true})){
      const file=path.join(directory,item.name);
      if(item.isDirectory())visit(file,level+1);
      else if(item.name==='game.js' && fs.existsSync(path.join(directory,'linux-builder.json')))
        packages.push(directory);
    }
  }
  visit(output);
  if(packages.length!==1)throw Error('Expected exactly one actual native WECHAT package; found '+packages.length);
  const pkg=packages[0];
  const result=JSON.parse(fs.readFileSync(path.join(pkg,'linux-builder.json'),'utf8'));
  assert.equal(result.platform,pins.platform);
  assert.equal(result.enginePlatform,'WECHAT');
  for(const file of ['game.js','game.json','project.config.json','engine-adapter.js','web-adapter.js'])
    assert.ok(fs.statSync(path.join(pkg,file)).size>0,'Missing native entry: '+file);
  const preview=JSON.parse(fs.readFileSync(path.join(pkg,'project.config.json'),'utf8'));
  assert.equal(preview.setting.urlCheck,true,'Never disable HTTPS/domain qualification');
  assert.equal(preview.setting.es6,false,'WeChat ES6 transpilation destroys native BigInt');
  assert.equal(preview.setting.minified,true,'Preview package must enable size-conscious minification');
  const checked=require('./audit.cjs').audit(pkg);
  console.log('Pinned Linux-native WECHAT export verified: '+pkg+' ('+checked+' audited JS files)');
  console.log('After real WeChat preview compilation, audit the processed JS separately.');
}finally{
  if(preserve){console.log('Isolated build inputs retained: '+work)}
  else fs.rmSync(work,{recursive:true,force:true});
}
