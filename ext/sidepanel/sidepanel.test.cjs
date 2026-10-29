const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const score={overall_score:85,categories:[],missing_keywords:[],strengths:[],summary:'Saved match',from_cache:true};
const tick=()=>new Promise(resolve=>setImmediate(resolve));
function deferred(){let resolve;const promise=new Promise(done=>{resolve=done});return{promise,resolve}}
async function fixture(){
  const {JSDOM}=await import('../../fe/node_modules/jsdom/lib/api.js');
  const dom=new JSDOM(fs.readFileSync(`${__dirname}/sidepanel.html`,'utf8'),{url:'https://fixture.invalid',runScripts:'outside-only'});
  const context=dom.getInternalVMContext(),calls=[],notices=[],listeners={};
  const held={};let confirmAction=()=>true;
  context.fetch=async(url,options={})=>{
    calls.push({url,options});
    if(url.includes('/match-score?'))return held.lookup?.promise??Response.json(score);
    if(url.endsWith('/subscription')){if(held.subscription){const value=held.subscription;held.subscription=null;return value.promise}return Response.json({credits:{remaining:5},credit_costs:{match:5}})}
    if(url.endsWith('/resumes?limit=50')&&held.resumes)return held.resumes.promise;
    if(url.endsWith('/resumes?limit=50')||url.includes('/jobs?'))return Response.json({items:[]});
    if(url.endsWith('/match-score')&&options.method==='POST'&&held.paid)return held.paid.promise;
    if(url.endsWith('/jobs')&&options.method==='POST')return Response.json({id:'new-job',title:'New job'});
    return Response.json({...score,from_cache:false});
  };
  context.confirm=notice=>{notices.push(notice);return confirmAction()};
  context.chrome={storage:{local:{get:async()=>({accessToken:'synthetic'}),set:async()=>{},remove:async()=>{}},session:{get:async()=>({}),set:async()=>{},remove:async()=>{}},onChanged:{addListener:fn=>{listeners.storage=fn}}},tabs:{query:async()=>[{id:1,active:true,url:'https://jobs.example.invalid'}],onActivated:{addListener:fn=>{listeners.activated=fn}},onUpdated:{addListener:fn=>{listeners.updated=fn}},create:()=>{}},runtime:{sendMessage:async()=>({})}};
  vm.runInContext(fs.readFileSync(`${__dirname}/../shared/api.js`,'utf8'),context);
  vm.runInContext(fs.readFileSync(`${__dirname}/sidepanel.js`,'utf8'),context);
  await tick();await tick();
  vm.runInContext("lastSavedJob={id:'old-job',title:'Old job'};showSaveState('success')",context);
  const select=dom.window.document.getElementById('sp-match-resume');select.innerHTML='<option value="old-resume">Old resume</option><option value="new-resume">New resume</option>';select.value='old-resume';calls.length=0;
  const click=id=>dom.window.document.getElementById(id).click();
  const start=async()=>{click('sp-btn-check-match');await tick()};
  const change=async kind=>{
    if(kind==='saveAnother')click('btn-save-another');
    if(kind==='tab')dom.window.document.querySelector('[data-tab="jobs"]').click();
    if(kind==='resume'){select.value='new-resume';select.dispatchEvent(new dom.window.Event('change'))}
    if(kind==='logout')click('btn-logout');
    if(kind==='activeTab')listeners.activated();
    if(kind==='navigation')listeners.updated(1,{url:'https://other.example.invalid'},{active:true});
    if(kind==='pagehide')dom.window.dispatchEvent(new dom.window.Event('pagehide'));
    if(kind==='newJob'){dom.window.document.getElementById('preview-title').value='New job';dom.window.document.getElementById('save-form').dispatchEvent(new dom.window.Event('submit',{cancelable:true}))}
    await tick();
  };
  return{dom,context,calls,notices,held,start,change,click,setConfirm:fn=>{confirmAction=fn},result:()=>dom.window.document.getElementById('sp-match-result'),error:()=>dom.window.document.getElementById('sp-match-error')};
}
const miss=()=>Response.json({error_code:'MATCH_SCORE_NOT_READY'},{status:404});
const posts=f=>f.calls.filter(c=>c.url.endsWith('/match-score')&&c.options.method==='POST');
for(const departure of ['saveAnother','tab','resume','logout','activeTab','navigation','pagehide','newJob']){
  test(`delayed cache miss after ${departure} never confirms or posts`,async()=>{
    const f=await fixture();try{f.held.lookup=deferred();await f.start();await f.change(departure);f.held.lookup.resolve(miss());await tick();await tick();
    assert.equal(f.notices.length,0);assert.equal(posts(f).length,0);assert.equal(f.result().classList.contains('hidden'),true);assert.equal(f.error().classList.contains('hidden'),true)}finally{f.dom.window.close()}
  });
}
test('departed context never paints a delayed cached hit or error',async()=>{
  for(const response of [Response.json(score),Response.json({error_code:'UNAUTHORIZED'},{status:403})]){const f=await fixture();try{f.held.lookup=deferred();await f.start();await f.change('saveAnother');f.held.lookup.resolve(response);await tick();await tick();assert.equal(f.result().classList.contains('hidden'),true);assert.equal(f.error().classList.contains('hidden'),true)}finally{f.dom.window.close()}}
});
test('departure during credit lookup prevents the later confirmation',async()=>{
  const f=await fixture();try{f.held.lookup={promise:Promise.resolve(miss())};const balance=deferred();f.held.subscription=balance;await f.start();await f.change('saveAnother');balance.resolve(Response.json({credits:{remaining:5},credit_costs:{match:5}}));await tick();await tick();assert.equal(f.notices.length,0);assert.equal(posts(f).length,0)}finally{f.dom.window.close()}
});
test('departure at the confirmation boundary prevents paid dispatch',async()=>{
  const f=await fixture();try{f.held.lookup={promise:Promise.resolve(miss())};f.setConfirm(()=>{f.click('btn-save-another');return true});await f.start();await tick();assert.equal(f.notices.length,1);assert.equal(posts(f).length,0)}finally{f.dom.window.close()}
});
test('active context retains free hits, cancellation and confirmed paid results',async()=>{
  for(const state of ['hit','cancel','accept']){const f=await fixture();try{if(state!=='hit')f.held.lookup={promise:Promise.resolve(miss())};f.setConfirm(()=>state==='accept');await f.start();await tick();assert.equal(f.notices.length,state==='hit'?0:1);assert.equal(posts(f).length,state==='accept'?1:0);assert.equal(f.result().classList.contains('hidden'),state==='cancel')}finally{f.dom.window.close()}}
});

test('a late old request cannot reset a newer match context',async()=>{
  const f=await fixture();try{const old=deferred();f.held.lookup=old;await f.start();await f.change('resume');const current=deferred();f.held.lookup=current;await f.start();old.resolve(Response.json({},{status:500}));await tick();await tick();assert.equal(f.dom.window.document.getElementById('sp-match-loading').classList.contains('hidden'),false);assert.equal(f.error().classList.contains('hidden'),true);current.resolve(Response.json(score));await tick();await tick();assert.equal(f.result().classList.contains('hidden'),false)}finally{f.dom.window.close()}
});
test('departing after paid dispatch leaves its transport alive but suppresses its result',async()=>{
  const f=await fixture();try{f.held.lookup={promise:Promise.resolve(miss())};f.held.paid=deferred();await f.start();await tick();assert.equal(posts(f).length,1);assert.equal(posts(f)[0].options.signal,undefined);await f.change('saveAnother');f.held.paid.resolve(Response.json({...score,from_cache:false}));await tick();await tick();assert.equal(f.result().classList.contains('hidden'),true);assert.equal(f.error().classList.contains('hidden'),true)}finally{f.dom.window.close()}
});

for(const outcome of ['miss','hit','error']){
  test(`delayed resume options from the actual save handler cancel an old-resume ${outcome}`,async()=>{
    const f=await fixture();try{
      const resumes=deferred();f.held.resumes=resumes;
      await f.change('newJob');
      assert.equal(vm.runInContext('lastSavedJob.id',f.context),'new-job');
      const select=f.dom.window.document.getElementById('sp-match-resume');assert.equal(select.value,'old-resume');
      const lookup=deferred();f.held.lookup=lookup;await f.start();
      assert.match(f.calls.find(c=>c.url.includes('/match-score?')).url,/job_id=new-job&resume_id=old-resume/);
      resumes.resolve(Response.json({items:[{id:'new-resume',title:'New resume',is_active:true}]}));await tick();await tick();
      assert.equal(select.value,'new-resume');
      lookup.resolve(outcome==='miss'?miss():outcome==='hit'?Response.json(score):Response.json({error_code:'INTERNAL_ERROR'},{status:500}));await tick();await tick();
      assert.equal(f.notices.length,0);assert.equal(posts(f).length,0);
      assert.equal(f.result().classList.contains('hidden'),true);assert.equal(f.error().classList.contains('hidden'),true);
      assert.equal(f.dom.window.document.getElementById('sp-match-loading').classList.contains('hidden'),true);
    }finally{f.dom.window.close()}
  });
}
test('resume options refresh clears an already completed result for the old selection',async()=>{
  const f=await fixture();try{
    const resumes=deferred();f.held.resumes=resumes;await f.change('newJob');await f.start();await tick();
    assert.equal(f.result().classList.contains('hidden'),false);
    resumes.resolve(Response.json({items:[{id:'new-resume',title:'New resume',is_active:true}]}));await tick();await tick();
    assert.equal(f.dom.window.document.getElementById('sp-match-resume').value,'new-resume');
    assert.equal(f.result().classList.contains('hidden'),true);assert.equal(f.error().classList.contains('hidden'),true);
    assert.equal(f.notices.length,0);assert.equal(posts(f).length,0);
  }finally{f.dom.window.close()}
});
