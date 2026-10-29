const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const {extensionFixture,event}=require('../test-fixture.cjs');
const score={overall_score:85,categories:[],missing_keywords:[],strengths:[],summary:'Saved match',from_cache:true};
const tick=()=>new Promise(resolve=>setImmediate(resolve));
function deferred(){let resolve,reject;const promise=new Promise((done,fail)=>{resolve=done;reject=fail});return{promise,resolve,reject}}
async function fixture(settings={}){
  const {JSDOM}=await import('../../fe/node_modules/jsdom/lib/api.js');
  const dom=new JSDOM(fs.readFileSync(`${__dirname}/sidepanel.html`,'utf8'),{url:'https://fixture.invalid',runScripts:'outside-only'});
  const context=dom.getInternalVMContext(),calls=[],notices=[],listeners={};
  const held={};let confirmAction=()=>true;
  context.fetch=async(url,options={})=>{
    calls.push({url,options});
    if(url.endsWith('/auth/refresh'))return Response.json({access_token:'rotated',refresh_token:'rotated-refresh'});
    if(url.endsWith('/auth/login'))return Response.json({tokens:{access_token:'account-b',refresh_token:'refresh-b'}});
    if(url.endsWith('/resume-builder'))return held.builders?.promise??Response.json({items:[]});
    if(url.includes('/resume-builder/')&&held.profile)return held.profile.promise;
    if(url.endsWith('/autofill-profile')&&held.profile){if(held.profilePaid&&options.method!=='POST')return Response.json({},{status:404});return held.profile.promise;}
    if(url.endsWith('/companies')&&held.company)return held.company.promise;
    if(url.includes('/jobs?')&&held.jobs)return held.jobs.promise;
    if(url.endsWith('/jobs/parse'))return held.parse?.promise??Response.json({title:'Parsed fixture',url:'https://jobs.example.invalid'});
    if(url.includes('/match-score?'))return held.lookup?.promise??Response.json(score);
    if(url.endsWith('/subscription')){if(held.subscription){const value=held.subscription;held.subscription=null;return value.promise}return Response.json({credits:{remaining:held.profilePaid?30:5},credit_costs:{match:5,job_parse:2,autofill_import:7}})}
    if(url.endsWith('/resumes?limit=50')&&held.resumes)return held.resumes.promise;
    if(url.endsWith('/resumes?limit=50')||url.includes('/jobs?'))return Response.json({items:[]});
    if(url.endsWith('/match-score')&&options.method==='POST'&&held.paid)return held.paid.promise;
    if(url.endsWith('/jobs')&&options.method==='POST')return Response.json({id:'new-job',title:'New job',url:'https://fixture.invalid/new-job'});
    return Response.json({...score,from_cache:false});
  };
  context.confirm=notice=>{notices.push(notice);return confirmAction()};
  const engine=extensionFixture({fetch:context.fetch,...settings});
  context.chrome={...engine.chromeFor(),tabs:{query:async()=>[{id:1,active:true,url:'https://jobs.example.invalid'}],onActivated:{addListener:fn=>{listeners.activated=fn}},onUpdated:{addListener:fn=>{listeners.updated=fn}},create:()=>{}}};
  vm.runInContext(fs.readFileSync(`${__dirname}/../shared/api.js`,'utf8'),context);
  vm.runInContext(fs.readFileSync(`${__dirname}/sidepanel.js`,'utf8'),context);
  vm.runInContext(fs.readFileSync(`${__dirname}/autofill.js`,'utf8'),context);
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
  return{engine,dom,context,calls,notices,held,start,change,click,setConfirm:fn=>{confirmAction=fn},result:()=>dom.window.document.getElementById('sp-match-result'),error:()=>dom.window.document.getElementById('sp-match-error')};
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
  const f=await fixture();try{f.held.lookup={promise:Promise.resolve(miss())};const balance=deferred();f.held.subscription=balance;await f.start();await f.change('saveAnother');balance.resolve(Response.json({credits:{remaining:5},credit_costs:{match:5,job_parse:2,autofill_import:7}}));await tick();await tick();assert.equal(f.notices.length,0);assert.equal(posts(f).length,0)}finally{f.dom.window.close()}
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

for(const prefix of ['rb:','up:'])test(`held ${prefix} profile cannot persist after logout and B login`,async()=>{
  const f=await fixture();try{
    const select=f.dom.window.document.getElementById('autofill-resume-select');select.innerHTML=`<option value="${prefix}resume">Synthetic Resume</option>`;select.value=prefix+'resume';
    f.held.profile=deferred();f.click('btn-set-autofill');await tick();
    f.click('btn-logout');await tick();
    await vm.runInContext("JobberAPI.login('synthetic','synthetic')",f.context);
    f.held.profile.resolve(Response.json({contact:{full_name:'Account A private fixture'}}));await tick();await tick();
    assert.equal(f.engine.local.autofillProfile,undefined);assert.equal(f.dom.window.document.getElementById('profile-name').textContent,'');
  }finally{f.dom.window.close()}
});
test('password is cleared and masked on successful login, logout and expiry',async()=>{
  const f=await fixture();try{
    const password=f.dom.window.document.getElementById('login-password');
    for(const action of ['login','logout','expiry']){
      password.value='synthetic-password';password.type='text';
      if(action==='login')f.dom.window.document.getElementById('login-form').dispatchEvent(new f.dom.window.Event('submit',{cancelable:true}));
      if(action==='logout')f.click('btn-logout');
      if(action==='expiry')vm.runInContext('handleSessionExpired()',f.context);
      await tick();await tick();assert.equal(password.value,'');assert.equal(password.type,'password');
    }
  }finally{f.dom.window.close()}
});
for(const loggedIn of [true,false])test(`persistent pending listener handles future commands after initial ${loggedIn?'consumption':'logged-out login'}`,async()=>{
  const settings={pending:{timestamp:Date.now(),pageData:{text:'Synthetic job fixture text',url:'https://jobs.example.invalid'}}};
  if(!loggedIn)settings.stored={};
  const f=await fixture(settings);try{
    if(!loggedIn){assert.equal(f.calls.filter(call=>call.url.endsWith('/jobs/parse')).length,0);await vm.runInContext("JobberAPI.login('synthetic','synthetic')",f.context);await tick();await tick()}
    const first=f.calls.filter(call=>call.url.endsWith('/jobs/parse')).length;
    // fixture clears the call history after init, so the initial consumed request is absent.
    assert.equal(first,loggedIn?0:1);
    await f.engine.storage.session.set({pendingParse:{timestamp:Date.now(),pageData:{text:'Second synthetic posting',url:'https://jobs.example.invalid/second'},sessionId:f.engine.local.authSessionId}});
    await tick();await tick();assert.equal(f.calls.filter(call=>call.url.endsWith('/jobs/parse')).length,first+1);
    assert.equal(f.engine.session.pendingParse,undefined);
  }finally{f.dom.window.close()}
});
for(const departure of ['logout','activeTab','navigation','pagehide','tab'])test(`parse credit lookup after ${departure} never confirms or dispatches`,async()=>{
  const f=await fixture();try{
    f.held.subscription=deferred();const held=f.held.subscription;
    const running=vm.runInContext("parseWithData({text:'Synthetic job description',url:'https://jobs.example.invalid'})",f.context);await tick();await f.change(departure);
    held.resolve(Response.json({credits:{remaining:30},credit_costs:{job_parse:2}}));await running;await tick();assert.equal(f.notices.length,0);assert.equal(f.calls.filter(call=>call.url.endsWith('/jobs/parse')).length,0);
  }finally{f.dom.window.close()}
});

test('routine worker token rotation preserves an unrelated in-flight match',async()=>{
  const f=await fixture();try{
    f.held.lookup=deferred();await f.start();f.engine.alarm.emit({name:'jobber-token-refresh'});await tick();await tick();
    assert.equal(f.engine.local.accessToken,'rotated');f.held.lookup.resolve(Response.json(score));await tick();await tick();assert.equal(f.result().classList.contains('hidden'),false);
  }finally{f.dom.window.close()}
});

for(const departure of ['activeTab','navigation','tab'])test(`cancelled parse after ${departure} restores idle and accepts a retry`,async()=>{
  const f=await fixture();try{
    const balance=deferred();f.held.subscription=balance;
    const old=vm.runInContext("parseWithData({text:'Old synthetic description',url:'https://jobs.example.invalid/old'})",f.context);await tick();
    await f.change(departure);
    const doc=f.dom.window.document;
    assert.equal(doc.getElementById('save-loading').classList.contains('hidden'),true);
    assert.equal(doc.getElementById('save-idle').classList.contains('hidden'),false);
    if(departure==='tab')doc.querySelector('[data-tab="save"]').click();
    const retry=vm.runInContext("parseWithData({text:'Retry synthetic description',url:'https://jobs.example.invalid/retry'})",f.context);await retry;
    balance.resolve(Response.json({credits:{remaining:30},credit_costs:{job_parse:2}}));await old;
    assert.equal(doc.getElementById('save-preview').classList.contains('hidden'),false);
    assert.equal(doc.getElementById('save-loading').classList.contains('hidden'),true);
    assert.equal(f.calls.filter(call=>call.url.endsWith('/jobs/parse')).length,1);
  }finally{f.dom.window.close()}
});
for(const prefix of ['rb:','up:'])for(const departure of ['activeTab','navigation','tab'])test(`cancelled ${prefix} profile after ${departure} permits the same selection retry`,async()=>{
  const f=await fixture();try{
    const doc=f.dom.window.document,select=doc.getElementById('autofill-resume-select'),button=doc.getElementById('btn-set-autofill');
    select.innerHTML=`<option value="${prefix}resume">Synthetic resume</option>`;select.value=prefix+'resume';
    const old=deferred();f.held.profile=old;f.click('btn-set-autofill');await tick();assert.equal(button.disabled,true);
    await f.change(departure);assert.equal(button.disabled,false);assert.equal(button.textContent,'Set as Autofill Profile');
    old.resolve(Response.json({contact:{full_name:'Old synthetic profile'}}));await tick();await tick();
    if(departure==='tab'){
      f.held.builders={promise:Promise.resolve(Response.json({items:[{id:'resume',title:'Synthetic builder'}]}))};
      f.held.resumes={promise:Promise.resolve(Response.json({items:[{id:'resume',title:'Synthetic upload'}]}))};
      doc.querySelector('[data-tab="autofill"]').click();await tick();await tick();select.value=prefix+'resume';
    }
    f.held.profile={promise:Promise.resolve(Response.json({contact:{full_name:'Retry synthetic profile'}}))};
    f.click('btn-set-autofill');await tick();await tick();
    assert.equal(f.engine.local.autofillProfile.contact.full_name,'Retry synthetic profile');assert.equal(button.disabled,false);assert.equal(button.textContent,'Set as Autofill Profile');
  }finally{f.dom.window.close()}
});
for(const phase of ['importedUrls','savedCount'])for(const method of ['get','set'])test(`save bookkeeping ${phase} ${method} stays ordered before account replacement`,async()=>{
  const f=await fixture();try{
    const area=phase==='importedUrls'?f.engine.storage.local:f.engine.storage.session;
    const original=area[method],gate=deferred(),arrived=deferred();let hold=true;
    area[method]=async value=>{if((method==='get'?value===phase:phase in value)&&hold){hold=false;arrived.resolve();await gate.promise}return original(value)};
    const doc=f.dom.window.document;doc.getElementById('preview-title').value='Account A synthetic job';doc.getElementById('save-form').dispatchEvent(new f.dom.window.Event('submit',{cancelable:true}));await arrived.promise;
    f.click('btn-logout');await tick();
    const replacement=vm.runInContext("JobberAPI.logout().then(()=>JobberAPI.login('synthetic','synthetic'))",f.context);
    await tick();assert.equal(f.engine.local.accessToken,'synthetic');
    gate.resolve();await replacement;await tick();await tick();
    assert.equal(f.engine.local.accessToken,'account-b');assert.equal(doc.getElementById('save-success').classList.contains('hidden'),true);assert.equal(doc.getElementById('btn-save').disabled,false);assert.equal(vm.runInContext('lastSavedJob',f.context),null);
    const countBefore=f.engine.session.savedCount||0;
    doc.getElementById('preview-title').value='Account B synthetic job';doc.getElementById('save-form').dispatchEvent(new f.dom.window.Event('submit',{cancelable:true}));await tick();await tick();
    assert.equal(doc.getElementById('save-success').classList.contains('hidden'),false);assert.equal(doc.getElementById('btn-save').disabled,false);assert.equal(f.engine.session.savedCount,countBefore+1);
  }finally{f.dom.window.close()}
});
for(const cancel of [true,false])test(`save snapshots submitted fields and ${cancel?'cancels':'continues'} after held company response`,async()=>{
  const f=await fixture();try{
    const doc=f.dom.window.document;
    const original={'preview-title':'Submitted Role','preview-company':'Submitted Company','preview-source':'Submitted Source','preview-url':'https://fixture.invalid/submitted','preview-salary':'123','preview-location':'Submitted City','preview-description':'Submitted Description'};
    for(const [id,value]of Object.entries(original))doc.getElementById(id).value=value;
    f.held.company=deferred();doc.getElementById('save-form').dispatchEvent(new f.dom.window.Event('submit',{cancelable:true}));await tick();
    if(cancel)await f.change('tab');
    for(const id of Object.keys(original))doc.getElementById(id).value='Changed after submit';
    f.held.company.resolve(Response.json({id:'submitted-company'}));await tick();await tick();
    const posts=f.calls.filter(call=>call.url.endsWith('/jobs')&&call.options.method==='POST');assert.equal(posts.length,cancel?0:1);
    if(!cancel){assert.deepEqual(JSON.parse(posts[0].options.body),{title:'Submitted Role',company_id:'submitted-company',source:'Submitted Source',url:'https://fixture.invalid/submitted',description:'Salary: 123\nLocation: Submitted City\n\nSubmitted Description'});assert.equal(doc.getElementById('save-success').classList.contains('hidden'),false)}
  }finally{f.dom.window.close()}
});
test('late account A jobs failure cannot hide account B loading or show its error',async()=>{
  const f=await fixture();try{
    const old=deferred();f.held.jobs=old;vm.runInContext('void loadJobs()',f.context);await tick();
    f.click('btn-logout');await tick();await vm.runInContext("JobberAPI.login('synthetic','synthetic')",f.context);await tick();
    const current=deferred();f.held.jobs=current;vm.runInContext('void loadJobs()',f.context);await tick();
    old.reject(new Error('Failed to fetch old A jobs'));await tick();await tick();
    assert.equal(f.dom.window.document.getElementById('jobs-loading').classList.contains('hidden'),false);assert.equal(f.dom.window.document.getElementById('jobs-error').classList.contains('hidden'),true);
    current.resolve(Response.json({items:[]}));await tick();await tick();assert.equal(f.dom.window.document.getElementById('jobs-loading').classList.contains('hidden'),true);
  }finally{f.dom.window.close()}
});
test('late delivery of A refresh expiry while loading match resumes cannot log B out',async()=>{
  const f=await fixture();try{
    const gate=deferred(),arrived=deferred(),send=f.context.chrome.runtime.sendMessage;
    f.context.chrome.runtime.sendMessage=async message=>{const result=await send(message);if(message.action==='authRefresh'&&result.error==='SESSION_EXPIRED'){arrived.resolve();await gate.promise}return result};
    const workerFetch=f.engine.context.fetch;
    f.engine.context.fetch=async(url,options)=>url.endsWith('/auth/refresh')?Response.json({},{status:401}):workerFetch(url,options);
    f.held.resumes={promise:Promise.resolve(Response.json({},{status:401}))};
    const loading=vm.runInContext('loadSpMatchResumes()',f.context);await arrived.promise;
    await vm.runInContext("JobberAPI.login('synthetic','synthetic')",f.context);await tick();
    gate.resolve();await loading;await tick();await tick();
    assert.equal(f.engine.local.accessToken,'account-b');assert.equal(vm.runInContext('JobberAPI.isLoggedIn()',f.context),true);assert.equal(f.dom.window.document.getElementById('view-main').classList.contains('hidden'),false);
  }finally{f.dom.window.close()}
});
for(const method of ['get','set'])for(const departure of ['selection','activeTab'])test(`profile cancellation during worker ${method} after ${departure} rejects obsolete commit and permits retry`,async()=>{
  const f=await fixture();try{
    const doc=f.dom.window.document,select=doc.getElementById('autofill-resume-select');select.innerHTML='<option value="rb:first">First</option><option value="rb:second">Second</option>';select.value='rb:first';
    const send=f.context.chrome.runtime.sendMessage,gate=deferred(),arrived=deferred();let storing=false,hold=true,oldRequest;
    f.context.chrome.runtime.sendMessage=message=>{if(message.action==='authStoreProfile'){storing=true;oldRequest ||= message.request}return send(message)};
    const area=f.engine.storage.local,original=area[method];
    area[method]=async value=>{if(storing&&hold&&(method==='get'?Array.isArray(value)&&value.includes('refreshToken'):!!value.autofillProfile)){hold=false;arrived.resolve();await gate.promise}return original(value)};
    f.held.profile={promise:Promise.resolve(Response.json({contact:{full_name:'Obsolete First profile'}}))};f.click('btn-set-autofill');await arrived.promise;
    if(departure==='selection'){select.value='rb:second';select.dispatchEvent(new f.dom.window.Event('change'))}else await f.change('activeTab');
    await tick();await tick();
    const contentRead=f.engine.chromeFor({id:'fixture',tab:{id:1},url:'https://fixture.invalid'}).runtime.sendMessage({action:'getAutofillProfile'});
    select.value='rb:first';f.held.profile={promise:Promise.resolve(Response.json({contact:{full_name:'Current retry profile'}}))};f.click('btn-set-autofill');
    gate.resolve();assert.equal((await contentRead).profile,null);await tick();await tick();await tick();
    assert.equal(f.engine.local.autofillProfile.contact.full_name,'Current retry profile');assert.equal(doc.getElementById('profile-name').textContent,'Current retry profile');
    await send({action:'authCancelProfile',request:oldRequest});await tick();
    assert.equal(f.engine.local.autofillProfile.contact.full_name,'Current retry profile');assert.equal((await f.engine.chromeFor().runtime.sendMessage({action:'getAutofillProfile'})).profile.contact.full_name,'Current retry profile');
  }finally{f.dom.window.close()}
});
for(const uploaded of [false,true])test(`pending ${uploaded?'paid uploaded':'builder'} profile activates after worker restart without another HTTP or AI operation`,async()=>{
  const f=await fixture();try{
    const select=f.dom.window.document.getElementById('autofill-resume-select');select.innerHTML=`<option value="${uploaded?'up:':'rb:'}resume">Synthetic resume</option>`;
    f.held.profile=deferred();f.held.profilePaid=uploaded;f.click('btn-set-autofill');await tick();await tick();
    const profileCalls=()=>f.calls.filter(call=>call.url.endsWith(uploaded?'/autofill-profile':'/resume-builder/resume'));
    assert.equal(profileCalls().length,uploaded?2:1);assert.equal(f.notices.length,uploaded?1:0);
    assert.ok(Object.keys(f.engine.session).some(key=>key.startsWith('autofillPending:')));
    f.engine.restartWorker();f.held.profile.resolve(Response.json({contact:{full_name:'Recovered pending profile'}}));await tick();await tick();await tick();
    assert.equal(f.engine.local.autofillProfile.contact.full_name,'Recovered pending profile');assert.equal(f.dom.window.document.getElementById('profile-name').textContent,'Recovered pending profile');
    assert.equal(profileCalls().length,uploaded?2:1);assert.equal(profileCalls().filter(call=>call.options.method==='POST').length,uploaded?1:0);assert.equal(f.notices.length,uploaded?1:0);
  }finally{f.dom.window.close()}
});
