const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const http = require('node:http')
const {chromium} = require(process.env.COCOS_SMOKE_PLAYWRIGHT_MODULE || 'playwright')

async function main() {
  const root = path.resolve(process.argv[2])
  const evidence = path.resolve(process.argv[3])
  fs.mkdirSync(evidence, {recursive: true})
  const errors = [], warnings = [], failures = [], checks = []
  const report = {result: 'failed', errors, warnings, failures, checks}
  const types = {'.html':'text/html', '.js':'text/javascript', '.json':'application/json', '.css':'text/css', '.wasm':'application/wasm', '.png':'image/png'}
  const server = http.createServer((request, response) => {
    const pathname = decodeURIComponent(new URL(request.url, 'http://localhost').pathname)
    if (pathname === '/favicon.ico') { response.writeHead(204); response.end(); return }
    const file = path.resolve(root, '.' + (pathname === '/' ? '/index.html' : pathname))
    if (!file.startsWith(root + path.sep)) { response.writeHead(403); response.end(); return }
    if (!fs.existsSync(file) || !fs.statSync(file).isFile()) { response.writeHead(404); response.end(); return }
    response.writeHead(200, {'Content-Type':types[path.extname(file)] || 'application/octet-stream', 'Cache-Control':'no-store'})
    fs.createReadStream(file).pipe(response)
  })
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve))
  let browser, page
  try {
    browser = await chromium.launch({headless:true, args:['--use-angle=swiftshader', '--enable-unsafe-swiftshader']})
    page = await browser.newPage({viewport:{width:1280, height:720}})
    page.on('pageerror', error => errors.push(error.message))
    page.on('console', message => { if (message.type() === 'error') errors.push(message.text()); else if (message.type() === 'warning') warnings.push(message.text()) })
    page.on('requestfailed', request => failures.push({url:request.url(), error:request.failure()?.errorText}))
    page.on('response', response => { if (response.status() >= 400) failures.push({url:response.url(), status:response.status()}) })
    await page.goto(`http://127.0.0.1:${server.address().port}/`, {waitUntil:'load'})
    await page.waitForFunction(() => !!globalThis.System?.import, undefined, {timeout:90000})
    await page.evaluate(async () => { globalThis.__cocosSmokeEngine = await System.import('cc') })
    // Poll synchronously: a Promise is truthy even when its eventual result is false.
    await page.waitForFunction(() => {
      const cc = globalThis.__cocosSmokeEngine
      const canvas = cc.director.getScene()?.getChildByName('Canvas')
      return !!canvas?.getChildByName('SafeContent')?.getChildByName('GameStatus')
    }, undefined, {timeout:90000})
    for (const [width, height] of [[1280,720],[844,390],[390,844]]) {
      await page.setViewportSize({width, height})
      await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))))
      const state = await page.evaluate(async () => {
        const cc = globalThis.__cocosSmokeEngine
        const canvas = cc.director.getScene().getChildByName('Canvas')
        const content = canvas.getChildByName('SafeContent')
        const title = content.getChildByName('GameTitle')
        const status = content.getChildByName('GameStatus')
        const rect = node => {const r=node.getComponent(cc.UITransform).getBoundingBoxToWorld(); return {x:r.x, y:r.y, width:r.width, height:r.height}}
        const domCanvas = document.getElementById('GameCanvas').getBoundingClientRect()
        return {canvasRect:{x:domCanvas.x,y:domCanvas.y,width:domCanvas.width,height:domCanvas.height},
          title:title.getComponent(cc.Label).string, status:status.getComponent(cc.Label).string,
          active:title.activeInHierarchy && status.activeInHierarchy,
          content:rect(content), titleBounds:rect(title), statusBounds:rect(status),
          bootstrap:!!canvas.getComponent('GameBootstrap')}
      })
      assert.ok(state.canvasRect.x >= -1 && state.canvasRect.y >= -1, 'canvas must start inside the viewport')
      assert.ok(state.canvasRect.x+state.canvasRect.width <= width+1 && state.canvasRect.y+state.canvasRect.height <= height+1, 'canvas must fit the actual browser viewport')
      assert.equal(state.title,'公平斗地主')
      assert.match(state.status,/请选择/)
      assert.ok(state.active && state.bootstrap, 'actual scene/bootstrap must be active')
      for (const rect of [state.titleBounds,state.statusBounds]) {
        assert.ok(rect.width > 0 && rect.height > 0)
        assert.ok(rect.x >= state.content.x-1 && rect.y >= state.content.y-1)
        assert.ok(rect.x+rect.width <= state.content.x+state.content.width+1)
        assert.ok(rect.y+rect.height <= state.content.y+state.content.height+1)
      }
      checks.push({width,height,...state})
      await page.screenshot({path:path.join(evidence,`entry-${width}x${height}.png`)})
    }
    const disposed = await page.evaluate(async () => {
      const cc = globalThis.__cocosSmokeEngine
      const bootstrap = cc.director.getScene().getChildByName('Canvas').getComponent('GameBootstrap')
      const shell = bootstrap.shell
      bootstrap.destroy()
      await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))
      return {shellCleared:bootstrap.shell === null, status:shell.hand.status, snapshot:shell.hand.snapshot,
        contentCleared:cc.director.getScene().getChildByName('Canvas').getChildByName('SafeContent') === null,
        resizeListener:cc.view.hasEventListener('canvas-resize',bootstrap.layout,bootstrap),
        designListener:cc.view.hasEventListener('design-resolution-changed',bootstrap.layout,bootstrap)}
    })
    assert.deepEqual(disposed,{shellCleared:true,status:'disconnected',snapshot:null,contentCleared:true,resizeListener:false,designListener:false})
    report.disposal = disposed
    assert.deepEqual(errors,[])
    assert.deepEqual(failures,[])
    report.result='passed'
  } catch (error) {
    report.error=error.message
    if (page) await page.screenshot({path:path.join(evidence,'runtime-failure.png')}).catch(()=>{})
    throw error
  } finally {
    report.completedAt=new Date().toISOString()
    fs.writeFileSync(path.join(evidence,'browser-runtime.json'),JSON.stringify(report,null,2))
    await browser?.close()
    await new Promise(resolve=>server.close(resolve))
  }
}
main().catch(error=>{console.error(error.message);process.exitCode=1})
