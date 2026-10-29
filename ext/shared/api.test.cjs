const { test } = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const crypto = require('node:crypto');
const source = fs.readFileSync(`${__dirname}/api.js`, 'utf8');
function fixture({accept=true,remaining=30,cached=false,language='en',refresh=false,matchResponse}={}) {
  const calls=[],notices=[];
  const context={crypto,navigator:{language},confirm:notice=>{notices.push(notice);return accept;},chrome:{storage:{local:{get:async()=>({accessToken:'synthetic',refreshToken:'synthetic'}),set:async()=>{},remove:async()=>{}}}},fetch:async(url,options={})=>{
    calls.push({url,options});
    if(url.endsWith('/subscription'))return Response.json({credits:{remaining},credit_costs:{job_parse:2,match:5,autofill_import:7}});
    if(url.endsWith('/auth/refresh'))return Response.json({access_token:'synthetic-new',refresh_token:'synthetic-new'});
    if(url.endsWith('/autofill-profile')&&!options.method)return cached?Response.json({contact:{email:'test@example.invalid'}}):Response.json({},{status:404});
    if(url.includes('/match-score?'))return matchResponse?.()??(cached?Response.json({overall_score:85,categories:[],missing_keywords:[],strengths:[],summary:'Saved result',from_cache:true}):Response.json({error_code:'MATCH_SCORE_NOT_READY'},{status:404}));
    if(refresh&&calls.filter(c=>c.options.method==='POST'&&!c.url.includes('/auth/')).length===1)return Response.json({},{status:401});
    return Response.json({ok:true});
  }};
  return {api:vm.runInNewContext(`${source}\nJobberAPI`,context),calls,notices};
}
test('all AI starts quote cost and carry an operation key',async()=>{
  for(const [action,cost] of [['parseJob',2],['getMatchScore',5],['getUploadedAutofillProfile',7]]) {
    const f=fixture();await f.api[action]('synthetic','synthetic');
    assert.equal(f.notices.length,1);assert.match(f.notices[0],new RegExp(`Cost: ${cost} AI credits`));
    assert.match(f.notices[0],/Errors after AI starts still use credits/);
    assert.match(f.calls.find(c=>c.options.method==='POST').options.headers['Idempotency-Key'],/^[0-9a-f-]{36}$/);
  }
});
test('cancel and insufficient credits dispatch no AI',async()=>{
  for(const settings of [{accept:false},{remaining:0}]){const f=fixture(settings);await assert.rejects(f.api.parseJob('text','url'));assert.equal(f.calls.filter(c=>c.options.method==='POST').length,0);}
});
test('cached autofill stays read-only without consent',async()=>{const f=fixture({cached:true});await f.api.getUploadedAutofillProfile('id');assert.equal(f.notices.length,0);assert.equal(f.calls.length,1);assert.equal(f.calls[0].options.method,undefined);});
test('401 replay retains its key without a second confirmation',async()=>{const f=fixture({refresh:true});await f.api.init();await f.api.parseJob('text','url');const posts=f.calls.filter(c=>c.options.method==='POST'&&!c.url.includes('/auth/'));assert.equal(posts.length,2);assert.equal(f.notices.length,1);assert.equal(posts[0].options.headers['Idempotency-Key'],posts[1].options.headers['Idempotency-Key']);});
test('credit warnings support Russian and Ukrainian',async()=>{for(const [language,word]of [['ru','Стоимость'],['uk','Вартість']]){const f=fixture({language});await f.api.parseJob('text','url');assert.match(f.notices[0],new RegExp(word));}});

for(const remaining of [0,1,2,3,4]) {
  test(`match cache at balance ${remaining} needs no credits or consent`,async()=>{
    const f=fixture({remaining,cached:true}); const result=await f.api.getMatchScore('job','resume');
    assert.equal(result.overall_score,85);assert.equal(result.from_cache,true);assert.equal(f.calls.length,1);assert.equal(f.notices.length,0);assert.match(f.calls[0].url,/match-score\?job_id=job&resume_id=resume/);
  });
  test(`match miss at balance ${remaining} dispatches no paid request`,async()=>{
    const f=fixture({remaining});await assert.rejects(f.api.getMatchScore('job','resume'),/Not enough AI credits/);assert.equal(f.calls.filter(c=>c.options.method==='POST').length,0);
  });
}
test('match miss cancellation and confirmation retain the paid path',async()=>{
  for(const accept of [false,true]){const f=fixture({accept,remaining:5});if(accept) await f.api.getMatchScore('job','resume');else await assert.rejects(f.api.getMatchScore('job','resume'),/AI_CANCELLED/);assert.equal(f.notices.length,1);assert.equal(f.calls.filter(c=>c.options.method==='POST').length,accept?1:0);}
});
test('only exact match cache miss permits the paid path',async()=>{
  for(const [status,body] of [[403,{error_code:'FORBIDDEN'}],[404,{error_code:'JOB_NOT_FOUND'}],[404,{}],[500,{}],[200,{}],[200,{overall_score:85,from_cache:false}]]) {
    const f=fixture({matchResponse:()=>Response.json(body,{status})});await assert.rejects(f.api.getMatchScore('job','resume'));assert.equal(f.notices.length,0);assert.equal(f.calls.filter(c=>c.options.method==='POST').length,0);
  }
});
