'use strict';

const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');
if (process.platform !== 'linux') throw new Error('This integration is qualified only on Linux');
if (!process.argv[2]) throw new Error('Usage: node install.cjs /path/to/pinned/cocos-cli');
const cli = fs.realpathSync(process.argv[2]);
const revision = execFileSync('git', ['-C', cli, 'rev-parse', 'HEAD'], { encoding: 'utf8' }).trim();
if (revision !== 'f506fa3de403b48b4833334fa48dddd27bb26364') {
  throw new Error('Unsupported cocos-cli revision; inspect its platform contract before installing');
}
const dest = path.join(cli, 'packages/platforms/wechatgame');
const owner = path.join(dest, 'awesome-zero-platform-owner');
if (fs.existsSync(dest) && !fs.existsSync(owner)) throw new Error('Refusing to replace an existing platform');
fs.mkdirSync(dest, { recursive: true });
fs.rmSync(path.join(dest, 'fix-bigint.cjs'), { force: true });
for (const file of ['config.cjs', 'hooks.cjs', 'audit.cjs']) fs.copyFileSync(path.join(__dirname, file), path.join(dest, file));
fs.writeFileSync(path.join(dest, 'package.json'), JSON.stringify({
  name: 'awesome-zero-platform-wechatgame', version: '0.1.0', type: 'commonjs',
  contributes: { builder: { register: true, platform: 'wechatgame', config: './config.cjs', hooks: './hooks.cjs' } },
}, null, 2));
fs.writeFileSync(owner, 'User-authorized Linux native builder experiment. Remove this directory to roll back.\n');
console.log('Installed external wechatgame platform for pinned Linux native cocos-cli');
