'use strict';
// Run against an already-open Linux developer-tool project. No login/upload.
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
const { chromium } = require(process.env.COCOS_SMOKE_PLAYWRIGHT_MODULE || '../../clients/admin-web/node_modules/playwright');
const evidence = path.resolve(process.argv[2] || '.runtime/linux-wechat');
const report = { result: 'failed', runtime: 'Linux community WeChat simulator', checks: [] };

(async () => {
  fs.mkdirSync(evidence, { recursive: true, mode: 0o700 });
  let browser;
  try {
    browser = await chromium.connectOverCDP(process.env.WECHAT_SMOKE_CDP_URL || 'http://127.0.0.1:19227', { timeout: 15000 });
    let page;
    for (const candidate of browser.contexts().flatMap(context => context.pages())) {
      if (await candidate.locator('[title="编译"]').count()) { page = candidate; break; }
    }
    assert.ok(page, 'Open the exported game and debugger first');
    const view = page.locator('webview').nth(1);
    let contextId;
    for (let attempt = 0; attempt < 30; attempt++) {
      contextId = await view.evaluate(element => element.executeJavaScript(`(async () => {
        if (typeof SDK === 'undefined') return null;
        for (const target of SDK.targetManager.targets()) {
          for (const context of target.model(SDK.RuntimeModel).executionContexts()) {
            const response = await target.runtimeAgent().invoke_evaluate({
              expression: 'typeof cc === "object" && !!cc.director?.getScene()',
              contextId: context.id, returnByValue: true
            });
            if (response.result?.value) return context.id;
          }
        }
        return null;
      })()`));
      if (contextId) break;
      await new Promise(resolve => setTimeout(resolve, 2000));
    }
    assert.ok(contextId, 'Actual Cocos scene must be running');
    async function evaluate(expression) {
      const value = await view.evaluate((element, args) => element.executeJavaScript(`(async () => {
        const response = await SDK.targetManager.targets()[0].runtimeAgent().invoke_evaluate(${JSON.stringify(args)});
        return JSON.stringify({ value: response.result?.value, error: response.exceptionDetails?.exception?.description });
      })()`), { expression, contextId, awaitPromise: true, returnByValue: true });
      const decoded = JSON.parse(value);
      assert.equal(decoded.error, undefined, decoded.error);
      return decoded.value;
    }
    async function state() {
      return JSON.parse(await evaluate(`(() => {
        const canvas = cc.director.getScene().getChildByName('Canvas');
        const content = canvas.getChildByName('SafeContent');
        return JSON.stringify({ title: content.getChildByName('GameTitle').getComponent(cc.Label).string,
          screen: canvas.getComponent('GameBootstrap').screen, sdk: wx.getSystemInfoSync().platform });
      })()`));
    }
    async function touch(name, expected) {
      // Native SDK input coordinates are absolute IDE-page coordinates. The
      // framebuffer and displayed simulator can have different pixel ratios.
      const point = JSON.parse(await evaluate(`(() => {
        const canvas = cc.director.getScene().getChildByName('Canvas');
        const node = canvas.getComponent('GameBootstrap').buttons.find(node => node.name === ${JSON.stringify(name)});
        if (!node) throw Error('Missing button');
        const camera = canvas.getChildByName('Camera').getComponent('cc.Camera');
        const p = camera.worldToScreen(node.worldPosition);
        return JSON.stringify({ x: p.x / cc.game.canvas.width, y: 1 - p.y / cc.game.canvas.height });
      })()`));
      const box = await page.locator('webview').first().boundingBox(); assert.ok(box);
      point.x = box.x + point.x * box.width; point.y = box.y + point.y * box.height;
      await view.evaluate((element, point) => element.executeJavaScript(`(async () => {
        const input = SDK.targetManager.targets()[0].inputAgent();
        const start = await input.invoke_dispatchTouchEvent({type:'touchStart',touchPoints:[{x:${point.x},y:${point.y},id:1}]});
        if (start.getError?.()) throw Error(start.getError());
        await new Promise(resolve => setTimeout(resolve,120));
        const end = await input.invoke_dispatchTouchEvent({type:'touchEnd',touchPoints:[]});
        if (end.getError?.()) throw Error(end.getError());
      })()`), point);
      for (let retry = 0; retry < 30; retry++) {
        const current = await state();
        if (expected(current)) { report.checks.push(name); return; }
        await new Promise(resolve => setTimeout(resolve, 200));
      }
      throw Error('Native button failed: ' + name);
    }
    report.initial = await state();
    assert.equal(report.initial.title, '公平斗地主');
    assert.equal(report.initial.sdk, 'devtools');
    report.nativeBigInt = await evaluate('(2n ** 256n).toString(16) === "1" + "0".repeat(64)');
    assert.equal(report.nativeBigInt, true);
    report.secureEntropyBytes = await evaluate(`new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(Error('Secure entropy timeout')), 5000);
      const manager = wx.getUserCryptoManager();
      manager.getRandomValues({ length: 32, success(value) {
        clearTimeout(timer); const bytes = new Uint8Array(value.randomValues);
        const count = bytes.byteLength; bytes.fill(0); resolve(count);
      }, fail(error) { clearTimeout(timer); reject(error); } });
    })`);
    assert.equal(report.secureEntropyBytes, 32);
    await touch('RulesEntry', state => state.title === '玩法介绍');
    await touch('BackHome', state => state.title === '公平斗地主');
    await touch('RoomEntry', state => state.screen === 'rooms');
    report.final = await state(); report.result = 'passed';
    await page.screenshot({ path: path.join(evidence, 'linux-wechat-scene.png') });
    fs.chmodSync(path.join(evidence, 'linux-wechat-scene.png'), 0o600);
  } catch (error) {
    report.error = error.message; process.exitCode = 1;
  } finally {
    fs.writeFileSync(path.join(evidence, 'native-smoke.json'), JSON.stringify(report, null, 2), { mode: 0o600 });
    await browser?.close();
    console.log(JSON.stringify(report));
  }
})();
