'use strict';

// Narrow external platform extension for the pinned official CLI.
exports.default = {
  displayName: 'Linux native WeChat Mini Game (experimental)',
  platformType: 'WECHAT',
  options: {
    appid: { type: 'string', default: '' },
    orientation: { type: 'string', default: 'landscape' },
  },
  hooks: './hooks.cjs',
  commonOptions: {
    polyfills: { default: { asyncFunctions: true } },
    nativeCodeBundleMode: { default: 'both' },
  },
  textureCompressConfig: { platformType: 'miniGame', support: { rgb: [], rgba: [] } },
  assetBundleConfig: {
    platformType: 'miniGame',
    supportedCompressionTypes: ['none', 'merge_dep', 'merge_all_json'],
  },
  buildTemplateConfig: { templates: [], version: '0.1.0' },
};
