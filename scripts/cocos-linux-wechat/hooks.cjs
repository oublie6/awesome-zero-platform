'use strict';

const fs = require('node:fs');
const path = require('node:path');
exports.throwError = true;

function local(root, file) {
  const rel = path.relative(root, file).split(path.sep).join('/');
  if (!rel || rel === '..' || rel.startsWith('../') || path.isAbsolute(rel)) throw new Error('Output path escapes package');
  return './' + rel;
}

exports.onAfterInit = function (options, result) {
  // CLI4 currently emits relative declaration names that tsc resolves relative
  // to typeRoots rather than this file. Use the actual generated declarations;
  // retain the source project's compiler settings, including its strict gate.
  const project = result.paths.projectRoot;
  const generated = path.join(project, 'temp/tsconfig.cocos.json');
  const config = JSON.parse(fs.readFileSync(generated, 'utf8'));
  const source = JSON.parse(fs.readFileSync(path.join(project, 'tsconfig.json'), 'utf8'));
  const compiler = config.compilerOptions;
  compiler.types = (compiler.types || []).map(type => {
    const name = path.basename(type);
    if (!['cc', 'cc.custom-macro', 'jsb', 'cc.env'].includes(name)) return type;
    const declaration = path.join(project, 'temp/declarations', name + '.d.ts');
    if (!fs.statSync(declaration).isFile()) throw new Error('Missing real engine declaration');
    return declaration.slice(0, -5);
  });
  Object.assign(compiler, source.compilerOptions);
  if (compiler.strict === true) Object.assign(compiler, {
    strictNullChecks: true, noImplicitAny: true, strictPropertyInitialization: true,
  });
  fs.writeFileSync(generated, JSON.stringify(config, null, 2) + '\n');
  options.buildEngineParam.platform = 'WECHAT';
  options.buildEngineParam.platformType = 'WECHAT';
  options.buildEngineParam.assetURLFormat = 'runtime-resolved';
  options.buildEngineParam.split = false;
};

exports.onAfterBundleInit = function (options) {
  options.buildScriptParam.system = { preset: 'commonjs-like' };
  options.buildScriptParam.importMapFormat = 'commonjs';
  options.buildScriptParam.flags.WEBGPU = false;
  // This client already requires native BigInt for HPKE/BLS. Preserve native
  // exponentiation instead of lowering it to Number-only Math.pow. Device
  // compatibility beyond the tested SDK runtime remains a separate gate.
  options.buildScriptParam.targets = { chrome: '90' };
};

exports.onBeforeCompressSettings = function (options, result) {
  const orientation = options.packages.wechatgame.orientation;
  if (!['portrait', 'landscape'].includes(orientation)) throw new Error('Invalid orientation');
  if (options.server) throw new Error('Remote resources are not qualified by this initial builder');
  result.settings.screen.orientation = orientation;
};

exports.onAfterBuild = function (options, result) {
  const root = result.paths.dir;
  const adapter = path.join(options.engineInfo.typescript.path, 'bin/adapter/minigame/wechat');
  for (const file of ['web-adapter.js', 'engine-adapter.js']) {
    fs.copyFileSync(path.join(adapter, file), path.join(root, file));
  }
  const relative = file => local(root, file);
  const preloads = [result.paths.polyfillsJs, result.paths.systemJs].filter(Boolean)
    .map(file => `require(${JSON.stringify(relative(file))});`).join('\n');
  const entry = `'use strict';
globalThis.__wxRequire = require;
require('./web-adapter.js');
${preloads}
const mapFile = ${JSON.stringify(relative(result.paths.importMap))};
const importedMap = require(mapFile);
System.warmup({
  importMap: importedMap.default || importedMap,
  importMapUrl: mapFile,
  defaultHandler: url => require('.' + url),
});
System.import(${JSON.stringify(relative(result.paths.applicationJS))})
  .then(async ({ Application }) => {
    const cc = await System.import('cc');
    require('./engine-adapter.js');
    const app = new Application();
    await app.init(cc);
    await app.start();
  }).catch(error => { console.error(error); });
`;
  fs.writeFileSync(path.join(root, 'game.js'), entry);
  fs.writeFileSync(path.join(root, 'game.json'), JSON.stringify({
    deviceOrientation: options.packages.wechatgame.orientation,
    networkTimeout: { request: 15000, connectSocket: 15000, downloadFile: 15000 },
  }, null, 2));
  fs.writeFileSync(path.join(root, 'project.config.json'), JSON.stringify({
    appid: options.packages.wechatgame.appid,
    projectname: 'fair-doudizhu-linux', compileType: 'game', miniprogramRoot: './',
    setting: { urlCheck: true, enhance: false, es6: false, minified: false },
  }, null, 2));
  fs.writeFileSync(path.join(root, 'linux-builder.json'), JSON.stringify({
    builder: 'awesome-zero-platform-linux-wechat-0.1.0',
    platform: options.platform, enginePlatform: 'WECHAT',
    engineVersion: options.engineInfo.version || null,
  }, null, 2));
  const checked = require('./audit.cjs').audit(root, { stripInlineMaps: !options.debug });
  console.log('Parsed ' + checked + ' generated JS files; BigInt exponent audit passed');
};
