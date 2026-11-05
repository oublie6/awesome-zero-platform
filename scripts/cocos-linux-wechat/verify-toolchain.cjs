'use strict';
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
const {execFileSync} = require('node:child_process');
const root = path.resolve(__dirname, '../..');
const pins = JSON.parse(fs.readFileSync(path.join(__dirname,'toolchain.json'),'utf8'));
const project = JSON.parse(fs.readFileSync(path.join(root,pins.project,'package.json'),'utf8'));
assert.equal(project.creator?.version,pins.engineVersion,'Client engine declaration must match native engine pin');
assert.equal(pins.platform,'wechatgame');
assert.equal(pins.cliRevision,'f506fa3de403b48b4833334fa48dddd27bb26364');
assert.equal(pins.engineRevision,'d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7');
assert.ok(fs.existsSync(path.join(__dirname,'engine-profile.json')));
assert.ok(fs.existsSync(path.join(__dirname,'install.cjs')));
if (process.env.COCOS_CLI_ROOT) {
  const cli=fs.realpathSync(process.env.COCOS_CLI_ROOT);
  const revision=execFileSync('git',['-C',cli,'rev-parse','HEAD'],{encoding:'utf8'}).trim();
  assert.equal(revision,pins.cliRevision,'Unsupported native CLI commit');
  assert.ok(fs.existsSync(path.join(cli,'dist/cli.js')),'Compiled CLI is required for native export');
}
if (process.env.COCOS_ENGINE_ROOT) {
  const engine=fs.realpathSync(process.env.COCOS_ENGINE_ROOT);
  const revision=execFileSync('git',['-C',engine,'rev-parse','HEAD'],{encoding:'utf8'}).trim();
  assert.equal(revision,pins.engineRevision,'Unsupported native engine revision');
}
console.log('Linux-native Cocos client/CLI/engine versions are pinned consistently');
