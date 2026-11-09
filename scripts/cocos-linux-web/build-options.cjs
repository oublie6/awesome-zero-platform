'use strict';
// Pin the SCRIPT transform independently of Cocos engine/platform build.
// Cocos CLI f506fa3... reads buildScriptTargets into
// options.buildScriptParam.targets (worker/builder/index.ts), which drives
// ModLo/Babel and preserves BigInt exponentiation for Chrome >= 67.
// Chrome79 also lowers ?? and optional chaining before WeChat preview.
function webBuildOptions({buildPath,startScene}={}) {
 if(typeof buildPath!=='string'||!buildPath||typeof startScene!=='string'||!startScene)
   throw new Error('real Cocos buildPath and startScene required');
 return {
   platform:'web-desktop',debug:false,outputName:'fair-doudizhu-web',
   buildPath,startScene,
   // G57-N01: without an explicit script target, CLI4 lowers
   // 2n ** 256n into Math.pow(2n,256n), crashing at module init.
   // Use the CLI's public string buildScriptTargets build option, not
   // post-build replacement of generated cryptographic JS.
   buildScriptTargets:'chrome 79'
 };
}
module.exports={webBuildOptions};
