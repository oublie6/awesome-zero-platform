'use strict';
// Safe public runtime configuration for an exported Cocos web build.
// This is never an identity/token storage channel.
const fs=require('node:fs');
const path=require('node:path');
function endpoint(value,protocol,name) {
  if(typeof value!=='string' || !value || value.trim()!==value)throw Error(name+' is required');
  let url;
  try{url=new URL(value)}catch{throw Error(name+' must be a valid URL')}
  if(url.protocol!==protocol || !url.hostname || url.username || url.password || url.hash || url.search)
    throw Error(name+' requires '+protocol+' and no credentials, query or fragment');
  return value.replace(/\/$/,'');
}
function publicConfig({apiBase,socketEndpoint,revealManifestRoots={}}={}){
  if((!apiBase && !socketEndpoint))return null;
  if(!apiBase || !socketEndpoint)throw Error('both WEB_API_BASE and WEB_SOCKET_ENDPOINT are required');
  if(!revealManifestRoots || typeof revealManifestRoots!=='object' || Array.isArray(revealManifestRoots))
    throw Error('invalid public manifest roots');
  for(const [key,value] of Object.entries(revealManifestRoots)){
    if(!/^[A-Za-z0-9_-]{1,128}$/.test(key) || typeof value!=='string' ||
       !/^[A-Za-z0-9_-]+$/.test(value))throw Error('invalid public manifest root');
  }
  return {apiBase:endpoint(apiBase,'https:','WEB_API_BASE'),
    socketEndpoint:endpoint(socketEndpoint,'wss:','WEB_SOCKET_ENDPOINT'),
    revealManifestRoots};
}
function configure(root,settings={}){
  const folder=path.resolve(root);
  const file=path.join(folder,'index.html');
  if(!fs.statSync(file).isFile())throw Error('not a built web index.html');
  let html=fs.readFileSync(file,'utf8');
  const tag='<script src="./game-config.js"></script>';
  if(!html.includes(tag)){
    if(!/<\/head>/i.test(html))throw Error('web index missing head');
    html=html.replace(/<\/head>/i,tag+'\n</head>');
  }
  const config=publicConfig(settings);
  // Empty deployment config fails closed in GameBootstrap with a visible
  // diagnostic; static assets can be hosted without baked credentials.
  const js='globalThis.__FAIR_DOUDIZHU_CONFIG__ = '+JSON.stringify(config)+';\n';
  fs.writeFileSync(path.join(folder,'game-config.js'),js,{mode:0o644});
  fs.writeFileSync(file,html);
  return config;
}
if(require.main===module){
  const root=process.argv[2];
  if(!root)throw Error('Usage: node configure.cjs <export-root>');
  const roots=process.env.WEB_REVEAL_MANIFEST_ROOTS?
    JSON.parse(process.env.WEB_REVEAL_MANIFEST_ROOTS):{};
  const result=configure(root,{apiBase:process.env.WEB_API_BASE,
    socketEndpoint:process.env.WEB_SOCKET_ENDPOINT,revealManifestRoots:roots});
  console.log(result?'Public Web game endpoints configured':'Web game endpoints unset; login fails closed until deployment config is provided');
}
module.exports={configure,publicConfig};
