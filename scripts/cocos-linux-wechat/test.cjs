'use strict';
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const vm = require('node:vm');
const { execFileSync } = require('node:child_process');

test('installer refuses unknown revisions and an existing unowned platform', t => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'azp-installer-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const bin = path.join(root, 'bin'); fs.mkdirSync(bin);
  const git = path.join(bin, 'git');
  const invoke = () => execFileSync(process.execPath, [path.join(__dirname, 'install.cjs'), root], {
    env: { ...process.env, PATH: bin + path.delimiter + process.env.PATH }, stdio: 'pipe',
  });
  fs.writeFileSync(git, '#!/bin/sh\nprintf %s unknown-revision\n', { mode: 0o700 });
  assert.throws(invoke, /Unsupported cocos-cli revision/);
  const dest = path.join(root, 'packages/platforms/wechatgame'); fs.mkdirSync(dest, { recursive: true });
  fs.writeFileSync(path.join(dest, 'sentinel'), 'operator-owned');
  fs.writeFileSync(git, '#!/bin/sh\nprintf %s f506fa3de403b48b4833334fa48dddd27bb26364\n');
  assert.throws(invoke, /Refusing to replace/);
  assert.equal(fs.readFileSync(path.join(dest, 'sentinel'), 'utf8'), 'operator-owned');
});

test('platform rejects invalid orientation and remote resources', () => {
  const hooks = require('./hooks.cjs');
  for (const options of [
    { packages: { wechatgame: { orientation: 'invalid' } } },
    { packages: { wechatgame: { orientation: 'landscape' } }, server: 'https://example.org/' },
  ]) assert.throws(() => hooks.onBeforeCompressSettings(options, { settings: { screen: {} } }));
});

test('generated Mini Game entry boots through CommonJS SystemJS and official-adapter slots', async t => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'azp-linux-wechat-'));
  t.after(() => fs.rmSync(temp, { recursive: true, force: true }));
  const plugin = path.join(temp, 'plugin'); fs.mkdirSync(plugin);
  fs.copyFileSync(path.join(__dirname, 'hooks.cjs'), path.join(plugin, 'hooks.cjs'));
  fs.copyFileSync(path.join(__dirname, 'audit.cjs'), path.join(plugin, 'audit.cjs'));
  const hooks = require(path.join(plugin, 'hooks.cjs'));
  const adapter = path.join(temp, 'bin/adapter/minigame/wechat'); fs.mkdirSync(adapter, { recursive: true });
  for (const name of ['web-adapter.js', 'engine-adapter.js']) fs.writeFileSync(path.join(adapter, name), '// adapter fixture');
  const root = path.join(temp, 'out'); fs.mkdirSync(root);
  const options = { platform: 'wechatgame', engineInfo: { typescript: { path: temp } },
    packages: { wechatgame: { orientation: 'landscape', appid: '' } }, buildEngineParam: {}, buildScriptParam: { flags: {} } };
  fs.mkdirSync(path.join(temp, 'temp/declarations'), { recursive: true });
  fs.writeFileSync(path.join(temp, 'temp/declarations/cc.d.ts'), 'declare module "cc" {}');
  fs.writeFileSync(path.join(temp, 'temp/tsconfig.cocos.json'), JSON.stringify({ compilerOptions: {
    types: ['./declarations/cc'], strictNullChecks: false, noImplicitAny: false,
  } }));
  fs.writeFileSync(path.join(temp, 'tsconfig.json'), JSON.stringify({ compilerOptions: { strict: true } }));
  hooks.onAfterInit(options, { paths: { projectRoot: temp } }); hooks.onAfterBundleInit(options);
  const generated = JSON.parse(fs.readFileSync(path.join(temp, 'temp/tsconfig.cocos.json'))).compilerOptions;
  assert.equal(generated.noImplicitAny, true);
  assert.equal(generated.strictNullChecks, true);
  assert.equal(generated.types[0], path.join(temp, 'temp/declarations/cc'));
  assert.equal(options.buildEngineParam.platform, 'WECHAT');
  assert.equal(options.buildScriptParam.importMapFormat, 'commonjs');
  assert.deepEqual(options.buildScriptParam.targets, { chrome: '90' });
  hooks.onAfterBuild(options, { paths: { dir: root, systemJs: path.join(root, 'system.js'), importMap: path.join(root, 'map.js'), applicationJS: path.join(root, 'app.js') } });
  assert.equal(JSON.parse(fs.readFileSync(path.join(root, 'game.json'))).deviceOrientation, 'landscape');
  const events = [], engine = {};
  class Application { init(cc) { assert.equal(cc, engine); events.push('init'); } start() { events.push('start'); } }
  const system = { warmup(data) { assert.deepEqual(data.importMap, { imports: {} }); events.push('warmup'); },
    async import(name) { return name === 'cc' ? engine : { Application }; } };
  vm.runInNewContext(fs.readFileSync(path.join(root, 'game.js'), 'utf8'), {
    System: system, console, require(name) { events.push(name); return name === './map.js' ? { default: { imports: {} } } : {}; },
  });
  await new Promise(resolve => setImmediate(resolve));
  assert.deepEqual(events.slice(-3), ['./engine-adapter.js', 'init', 'start']);
  assert.throws(() => hooks.onAfterBuild(options, { paths: {
    dir: root, systemJs: path.dirname(root), importMap: path.join(root, 'map.js'), applicationJS: path.join(root, 'app.js'),
  } }), /escapes package/);
});

test('generated crypto audit distinguishes minified numeric powers and rejects BigInt lowering', t => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'azp-bigint-audit-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const file = path.join(root, 'chunk.js');
  const { audit } = require('./audit.cjs');
  fs.writeFileSync(file, 'const n=Math.pow(2,32)-1,b=BigInt(n); const c=2n**256n;');
  assert.equal(audit(root), 1);
  for (const bad of ['Math.pow(2n,256n)', 'Math["pow"](2n,256n)', 'const x=BigInt(2);Math.pow(x,BigInt(256))', 'Math.pow(c.Fn.ORDER,n)']) {
    fs.writeFileSync(file, bad); assert.throws(() => audit(root), /BigInt/);
  }
  fs.writeFileSync(file, 'const url="//# sourceMappingURL=data:abc";\n//# sourceMappingURL=data:application/json;base64,e30=');
  audit(root, { stripInlineMaps: true });
  assert.equal(fs.readFileSync(file, 'utf8'), 'const url="//# sourceMappingURL=data:abc";\n\n');
});
