import fs from 'node:fs';import {createWebAuth,SESSION_KEY} from './input/clients/game-web/src/auth.ts';
const config={apiBase:'https://fixture.invalid',socketEndpoint:'wss://fixture.invalid/ws',revealManifestRoots:{}},okay=data=>new Response(JSON.stringify({code:'OK',data}),{status:200});
const cases=[];
for(const mode of ['dispose-during-save','logout-during-save','login-during-restore-load','dispose-during-final-restore-load','shared-login-during-final-restore-load']){
 const data=new Map();let auth,peer,armed=false,gets=0,newLogin,logout,calls=[];
 const storage={getItem(k){const v=data.get(k)??null;if(armed&&((mode==='login-during-restore-load'&&++gets===2)||(['dispose-during-final-restore-load','shared-login-during-final-restore-load'].includes(mode)&&++gets===2))){armed=false;if(mode.startsWith('shared'))newLogin=peer.login('new','password');else if(mode.startsWith('login'))newLogin=auth.login('new','password');else auth.dispose()}return v},setItem(k,v){data.set(k,v);if(armed&&mode.endsWith('during-save')){armed=false;if(mode.startsWith('dispose'))auth.dispose();else logout=auth.logout()}},removeItem(k){data.delete(k)}};
 const fetcher=async(url,opts)=>{const path=new URL(url).pathname;calls.push(path);if(path==='/auth/login'){const id=JSON.parse(opts.body).identifier;return okay({accessToken:id,tokenType:'Bearer'})}if(path==='/auth/session')return okay({account:{id:opts.headers.Authorization.slice(7),displayName:'fixture'}});return okay({loggedOut:true})};
 auth=createWebAuth(config,fetcher,storage);peer=createWebAuth(config,fetcher,storage);
 let returned;
 if(mode.endsWith('during-save')){armed=true;returned=await auth.login('old','password');if(logout)await logout}
 else{await auth.login('old','password');armed=true;returned=await auth.restore();if(newLogin)await newLogin}
 cases.push({name:mode,returnedIdentity:returned?.id??null,currentIdentity:auth.account?.id??null,authenticated:auth.authenticated,stored:data.get(SESSION_KEY)??null,calls,pass:returned===null});auth.dispose();
}
const out={source:'747597181a8497547e4c3b591e73efcd84d1c5e1',scope:'Actual Vue WebAuth with controlled synchronous StoragePort reentry; not ordinary native sessionStorage behavior',cases,pass:cases.filter(x=>x.pass).length,fail:cases.filter(x=>!x.pass).length};fs.writeFileSync(new URL('./vue-reentry.json',import.meta.url),JSON.stringify(out,null,2));console.log(JSON.stringify(out));
