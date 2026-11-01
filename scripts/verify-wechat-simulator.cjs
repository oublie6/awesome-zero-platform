#!/usr/bin/env node
// Operator-run smoke test for the pinned Linux WeChat developer tool.
// Requires a logged-in tool, an open built project and its debugger panel.
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')

const evidence = path.resolve(process.argv[2] || '.runtime/goal0045-wechat')
const {chromium} = require(process.env.COCOS_SMOKE_PLAYWRIGHT_MODULE || 'playwright')
const endpoint = process.env.WECHAT_SMOKE_CDP_URL || 'http://127.0.0.1:19227'
const expectedStatus = '游戏功能正在开发，暂未开放房间操作'

async function main() {
  fs.mkdirSync(evidence, {recursive:true, mode:0o700})
  const report = {runtime:'Linux community WeChat developer-tool simulator', result:'failed'}
  let browser
  let evaluate
  let registered = false
  try {
    browser = await chromium.connectOverCDP(endpoint, {timeout:15000})
    const pages = browser.contexts().flatMap(context => context.pages())
    let page
    for (const candidate of pages) {
      if (await candidate.locator('[title="编译"]').count()) { page = candidate; break }
    }
    assert.ok(page, 'Open the built Mini Game project first')
    const debuggerView = page.locator('webview').nth(1)
    assert.equal(await debuggerView.count(), 1, 'Open the tool debugger panel first')

    // executeJavaScript on gamePage alone selects the outer SDK context, not
    // necessarily the context in which the actual Cocos application runs.
    const contextId = await debuggerView.evaluate(element => element.executeJavaScript(`
      (async () => {
        const target = SDK.targetManager.targets()[0]
        for (const context of target.model(SDK.RuntimeModel).executionContexts()) {
          const result = await target.runtimeAgent().invoke_evaluate({
            expression:'typeof System === "object" && typeof cc === "object" && !!cc.director?.getScene()',
            contextId:context.id, returnByValue:true
          })
          if (result.result?.value === true) return context.id
        }
        return null
      })()`))
    assert.ok(contextId, 'Wait for the actual Cocos scene to load')
    evaluate = async expression => {
      const response = await debuggerView.evaluate((element, params) => element.executeJavaScript(`
        (async () => {
          const result = await SDK.targetManager.targets()[0].runtimeAgent().invoke_evaluate(${JSON.stringify(params)})
          return JSON.stringify({value:result.result?.value, error:result.exceptionDetails?.exception?.description})
        })()`), {expression, contextId, returnByValue:true, awaitPromise:true})
      const result = JSON.parse(response)
      assert.equal(result.error, undefined, result.error)
      return result.value
    }
    report.scene = JSON.parse(await evaluate(`(async () => {
      const engine = await System.import('cc')
      const content = engine.director.getScene().getChildByName('Canvas')?.getChildByName('SafeContent')
      return JSON.stringify({
        title:content?.getChildByName('GameTitle')?.getComponent(engine.Label)?.string,
        status:content?.getChildByName('GameStatus')?.getComponent(engine.Label)?.string,
        platform:wx.getSystemInfoSync().platform
      })
    })()`))
    assert.equal(report.scene.title, '公平斗地主')
    assert.equal(report.scene.status, expectedStatus)
    assert.equal(report.scene.platform, 'devtools')
    const probeName = '__goal0045SmokeInput'
    assert.equal(await evaluate(`typeof globalThis.${probeName}`), 'undefined', 'A previous probe must be cleaned up first')
    registered = true
    await evaluate(`(async () => {
      const engine = await System.import('cc')
      const probe = globalThis.${probeName} = {engine, count:0}
      probe.handler = () => probe.count++
      engine.input.on(engine.Input.EventType.TOUCH_START, probe.handler)
      return true
    })()`)
    report.touch = JSON.parse(await debuggerView.evaluate(element => element.executeJavaScript(`
      (async () => {
        const agent = SDK.targetManager.targets()[0].inputAgent()
        const start = await agent.invoke_dispatchTouchEvent({type:'touchStart', touchPoints:[{x:100,y:100,id:1}]})
        const end = await agent.invoke_dispatchTouchEvent({type:'touchEnd', touchPoints:[]})
        return JSON.stringify({startError:start.getError?.(), endError:end.getError?.()})
      })()`)))
    assert.equal(report.touch.startError, undefined)
    assert.equal(report.touch.endError, undefined)
    // An awaited task boundary lets the renderer deliver the native touch.
    report.touch.received = await evaluate(`new Promise(resolve => setTimeout(() => resolve(globalThis.${probeName}.count), 250))`)
    assert.equal(report.touch.received, 1, 'Cocos must receive the injected touch')
    await page.screenshot({path:path.join(evidence, 'wechat-scene.png')})
    fs.chmodSync(path.join(evidence, 'wechat-scene.png'), 0o600)
    report.result = 'passed'
  } catch (error) {
    report.error = String(error.message || error)
    throw error
  } finally {
    try {
      if (registered) {
        report.listenerRemoved = await evaluate(`(() => {
          const probe = globalThis.__goal0045SmokeInput
          if (probe?.handler) probe.engine.input.off(probe.engine.Input.EventType.TOUCH_START, probe.handler)
          delete globalThis.__goal0045SmokeInput
          return typeof globalThis.__goal0045SmokeInput === 'undefined'
        })()`)
        assert.equal(report.listenerRemoved, true)
      }
    } catch (error) {
      report.result = 'failed'
      report.cleanupError = String(error.message || error)
      throw error
    } finally {
      report.completedAt = new Date().toISOString()
      fs.writeFileSync(path.join(evidence, 'wechat-runtime.json'), JSON.stringify(report,null,2)+'\n', {mode:0o600})
      if (browser) await browser.close() // Disconnect CDP; do not quit the tool.
    }
  }
}
main().catch(error => { console.error(error.message); process.exitCode = 1 })
