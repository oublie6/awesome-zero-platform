import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'

// /game/ is a separate, actually exported Cocos4 build. Vite's SPA
// fallback must not silently render Vue's own index as the game iframe.
export default defineConfig({
  plugins:[vue(),{
    name:'game-artifact-boundary',
    configureServer(server){
      server.middlewares.use((req,res,next)=>{
        if(req.url?.startsWith('/game/') && !req.url.startsWith('/game-config.js')){
          // A real exported game must be staged under public/game.
          const pathname=decodeURIComponent(new URL(req.url,'http://localhost').pathname)
          if(!pathname.startsWith('/game/') || pathname.includes('..')){
            res.statusCode=400;res.end('Invalid game asset path');return
          }
          const fs=require('node:fs')
          const path=require('node:path')
          const root=path.resolve(process.cwd(),'public/game')
          const file=path.resolve(root,'.'+pathname.substring('/game'.length))
          if(!file.startsWith(root+path.sep) || !fs.existsSync(file) || !fs.statSync(file).isFile()){
            res.statusCode=404
            res.setHeader('Content-Type','text/plain; charset=utf-8')
            res.end('Cocos4 Web artifact is not staged; build and stage the game first.')
            return
          }
        }
        next()
      })
    }
  }],
  base:'/',
  server:{port:5174,host:'127.0.0.1',strictPort:true},
  build:{outDir:'dist',sourcemap:false}
})
