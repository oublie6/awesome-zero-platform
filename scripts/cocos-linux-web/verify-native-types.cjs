'use strict';
// Do not accept CLI4's success code when its generated declaration-based
// TypeScript check failed. This is the same resolution correction used by
// the pinned native WeChat builder, but does not mutate shared CLI installs.
const fs=require('node:fs');
const path=require('node:path');
const {spawnSync}=require('node:child_process');
const engineTypes=new Set(['cc','cc.custom-macro','cc.env','jsb']);
function repairGeneratedTypes(project){
 const source=JSON.parse(fs.readFileSync(path.join(project,'tsconfig.json'),'utf8'));
 const file=path.join(project,'temp/tsconfig.cocos.json');
 const generated=JSON.parse(fs.readFileSync(file,'utf8'));
 if(!generated.compilerOptions || !Array.isArray(generated.compilerOptions.types))
   throw new Error('Cocos did not generate expected real declaration types');
 const compiler=generated.compilerOptions;
 compiler.types=compiler.types.map(type=>{
   const name=path.basename(type);
   if(!engineTypes.has(name))return type;
   const declaration=path.join(project,'temp/declarations',name+'.d.ts');
   if(!fs.statSync(declaration).isFile())
     throw new Error('Missing Cocos4 generated declaration: '+name);
   return declaration.slice(0,-5);
 });
 Object.assign(compiler,source.compilerOptions);
 if(compiler.strict===true)
   Object.assign(compiler,{strictNullChecks:true,noImplicitAny:true,strictPropertyInitialization:true});
 fs.writeFileSync(file,JSON.stringify(generated,null,2)+'\n');
 return file;
}
function verifyNativeTypes(project){
 repairGeneratedTypes(project);
 const tsc=path.join(project,'node_modules/typescript/bin/tsc');
 if(!fs.statSync(tsc).isFile())throw new Error('Missing pinned project TypeScript');
 const result=spawnSync(process.execPath,[tsc,'-p',path.join(project,'tsconfig.json'),'--noEmit'],{
   cwd:project,encoding:'utf8',stdio:'inherit',timeout:180000
 });
 if(result.error)throw result.error;
 if(result.status!==0)throw new Error('Real generated Cocos4 TypeScript gate failed: '+result.status);
 console.log('Real generated Cocos4 TypeScript declaration gate passed');
}
module.exports={repairGeneratedTypes,verifyNativeTypes};
