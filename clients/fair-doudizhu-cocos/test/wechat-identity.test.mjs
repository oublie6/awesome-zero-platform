import {test} from 'node:test'
import assert from 'node:assert/strict'
import {WeChatBindingRequiredError,createWeChatIdentityExchange,validateWeChatAccess}
  from '../dist/game/WeChatIdentityExchange.js'

function fakePlatform(reply){
  const calls=[]
  return {
    target:'wechat',
    http:{request:async(path,options)=>{calls.push({path,options});return reply}},
    calls,
  }
}
test('an unbound verified WeChat code cannot become an authenticated AccountID session',async()=>{
  const proof='a'.repeat(43)
  const platform=fakePlatform({code:'OK',data:{status:'binding_required',pendingToken:proof,expiresInSeconds:300}})
  const exchange=createWeChatIdentityExchange(platform)
  await assert.rejects(exchange.exchange('wechat','valid-one-time-code'),error=>{
    assert.ok(error instanceof WeChatBindingRequiredError)
    assert.equal(error.pendingToken,proof)
    assert.ok(!error.message.includes(proof))
    return true
  })
  assert.equal(platform.calls.length,1)
  assert.equal(platform.calls[0].path,'/auth/wechat/code')
  assert.equal(platform.calls[0].options.method,'POST')
})
test('an already bound verified WeChat identity yields only server-issued token',async()=>{
  const platform=fakePlatform({code:'OK',data:{tokenType:'Bearer',accessToken:'signed-access-token'}})
  assert.deepEqual(await createWeChatIdentityExchange(platform).exchange('wechat','valid-code'),
    {accessToken:'signed-access-token'})
})
test('untrusted WeChat return values and malformed codes fail closed',async()=>{
  const platform=fakePlatform({code:'OK',data:{status:'binding_required',pendingToken:'short'}})
  const exchange=createWeChatIdentityExchange(platform)
  await assert.rejects(exchange.exchange('wechat','valid-code'),/invalid authenticated token/)
  await assert.rejects(exchange.exchange('web','valid-code'),/verified WeChat code required/)
  await assert.rejects(exchange.exchange('wechat','bad\ncode'),/verified WeChat code required/)
  assert.equal(platform.calls.length,1)
})
test('session verification is always server-backed',async()=>{
  const valid=fakePlatform({code:'OK',data:{account:{id:'uuid-01',displayName:'玩家'}}})
  assert.equal(await validateWeChatAccess(valid,'valid-access'),true)
  const invalid=fakePlatform({code:'OK',data:{}})
  assert.equal(await validateWeChatAccess(invalid,'valid-access'),false)
  assert.equal(await validateWeChatAccess(valid,'bad\ntoken'),false)
})
