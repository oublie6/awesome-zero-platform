const fs=require('node:fs'),path=require('node:path'),{spawnSync}=require('node:child_process'),E=__dirname,R='/root/awesome-zero-platform',input=E+'/input',project=input+'/clients/fair-doudizhu-cocos',out=E+'/native2';
function run(args){const r=spawnSync(args[0],args.slice(1),{cwd:R,env:process.env,stdio:'inherit',timeout:720000});if(r.error)throw r.error;if(r.status!==0)throw Error('Build stage failed with exit '+r.status)}
run([process.execPath,R+'/scripts/cocos-linux-wechat/verify-toolchain.cjs']);
for(const pkg of ['game-client-core'])run(['npm','run','build','--prefix',input+'/clients/packages/'+pkg]);
const scene=JSON.parse(fs.readFileSync(project+'/assets/scenes/Entry.scene.meta'));fs.writeFileSync(input+'/web-build-options.json',JSON.stringify(require(R+'/scripts/cocos-linux-web/build-options.cjs').webBuildOptions({buildPath:out,startScene:scene.uuid})));
run(['xvfb-run','-a',process.execPath,process.env.COCOS_CLI_ROOT+'/dist/cli.js','--no-interactive','build','--project',project,'--platform','web-desktop','--build-config',input+'/web-build-options.json']);
require(R+'/scripts/cocos-linux-web/verify-native-types.cjs').verifyNativeTypes(project);
const n=require(R+'/scripts/cocos-linux-wechat/audit.cjs').audit(out,{previewCompatible:false,requireNativeBigIntExponent:true});console.log('Official fixed-candidate export and strict declarations/AST PASS; JS='+n);
