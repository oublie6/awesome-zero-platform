#!/usr/bin/env node
'use strict'
// Cocos Creator 3.8.8's Babel exponent transform emits Math.pow(BigInt,...)
// for @noble cryptographic code. JS Math.pow rejects BigInt, whereas native
// ** preserves the required exact integer semantics. Rewrite only the
// audit-identified third-party BigInt forms in *generated* Creator chunks.
// Never touch source, dependencies, normal Math.pow or arbitrary expressions.
const fs=require('node:fs')
const path=require('node:path')

const known = [
  ['Math.pow(2n, 256n)','(2n ** 256n)'],
  ['Math.pow(c.Fn.ORDER, _4n)','(c.Fn.ORDER ** _4n)'],
  ['Math.pow(_2n, BigInt(254))','(_2n ** BigInt(254))'],
  ['Math.pow(_2n, BigInt(447))','(_2n ** BigInt(447))'],
  ['Math.pow(_2n, BigInt(251))','(_2n ** BigInt(251))'],
  ['Math.pow(_2n, BigInt(445))','(_2n ** BigInt(445))'],
]
function transform(source) {
  let output=source,rewrites=0
  for(const [from,to] of known){
    if(output.includes(from)) {
      output=output.split(from).join(to)
      rewrites++
    }
  }
  // Explicitly reject unknown BigInt exponent lowering instead of silently
  // shipping a cryptographic module that crashes at import time.
  const suspect=/Math\.pow\([^\r\n;]{0,115}(?:\b\d+n\b|\b_\d+n\b|BigInt\(|\.Fn\.ORDER)/g
  const leftovers=output.match(suspect)||[]
  const unknown=leftovers.filter(value=>!/Math\.pow\(2, 8 \* Nn\)/.test(value))
  if(unknown.length) throw new Error('unknown Cocos BigInt exponent transform: '+unknown.slice(0,4).join('; '))
  return {output,rewrites}
}
function apply(root) {
  const files=[]
  function walk(dir) {
    for(const entry of fs.readdirSync(dir,{withFileTypes:true})){
      const item=path.join(dir,entry.name)
      if(entry.isDirectory())walk(item)
      else if(entry.isFile() && item.endsWith('.js'))files.push(item)
    }
  }
  walk(root)
  let rewritten=0
  for(const file of files){
    const original=fs.readFileSync(file,'utf8')
    const {output,rewrites}=transform(original)
    if(rewrites>0){
      fs.writeFileSync(file,output,'utf8')
      rewritten+=rewrites
    }
  }
  process.stdout.write('Cocos verified BigInt exponent fixes: '+rewritten+' in '+files.length+' JS files\n')
}
if(require.main===module){
  try{
    const root=process.argv[2]
    if(!root||!fs.statSync(root).isDirectory())throw new Error('pass an existing Creator build output directory')
    apply(path.resolve(root))
  }catch(error){console.error(error.message);process.exitCode=1}
}
module.exports={transform,apply}
