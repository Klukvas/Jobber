const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const crypto=require('node:crypto');
const {extensionFixture}=require('../test-fixture.cjs');
const source=fs.readFileSync(`${__dirname}/api.js`,'utf8');
const tick=()=>new Promise(resolve=>setImmediate(resolve));
function deferred(){let resolve;const promise=new Promise(done=>resolve=done);return{resolve,promise}}
function panel(engine,fetch,confirm=()=>true){return vm.runInNewContext(`${source}\nJobberAPI`,{chrome:engine.chromeFor(),fetch,crypto,confirm})}
const loginResponse=()=>Response.json({tokens:{access_token:'account-b',refresh_token:'refresh-b'}});
test('all panels, quick save and alarm share one refresh and adopt rotated credentials',async()=>{
  const held=deferred();let refreshes=0;
  const fetch=async(url,options={})=>{
    if(url.endsWith('/auth/refresh')){refreshes++;return held.promise}
    if(options.headers?.Authorization==='Bearer synthetic')return Response.json({},{status:401});
    return Response.json({id:'saved-job',title:'Job',items:[]});
  };
  const engine=extensionFixture({fetch});const a=panel(engine,fetch),b=panel(engine,fetch);
  const requests=[a.listJobs(),b.listJobs(),engine.chromeFor().runtime.sendMessage({action:'quickSave',title:'Job'})];
  await tick();engine.alarm.emit({name:'jobber-token-refresh'});await tick();assert.equal(refreshes,1);
  held.resolve(Response.json({access_token:'rotated',refresh_token:'rotated-refresh'}));
  await Promise.all(requests);await tick();assert.equal(refreshes,1);assert.equal(engine.local.accessToken,'rotated');assert.equal(engine.local.refreshToken,'rotated-refresh');
});
for(const status of [200,401,500])test(`stale refresh ${status} cannot overwrite or remove replacement account`,async()=>{
  const held=deferred();const fetch=async url=>url.endsWith('/auth/refresh')?held.promise:loginResponse();
  const engine=extensionFixture({fetch});const a=panel(engine,fetch),b=panel(engine,fetch);await a.init();await b.init();
  const refresh=a.refreshAccessToken().then(()=>null,error=>error.message);await tick();
  await b.logout();await b.login('synthetic@example.invalid','synthetic');
  held.resolve(Response.json({access_token:'stale-a',refresh_token:'stale-a'},{status}));
  assert.equal(await refresh,'AI_CANCELLED');assert.equal(engine.local.accessToken,'account-b');assert.equal(engine.local.refreshToken,'refresh-b');
});
for(const kind of ['network','500','429'])test(`transient ${kind} preserves credentials and backs off refresh retries`,async()=>{
  let count=0;const fetch=async()=>{count++;if(kind==='network')throw new Error('offline');return Response.json({},{status:Number(kind)})};
  const engine=extensionFixture({fetch});const api=panel(engine,fetch);await api.init();
  await assert.rejects(api.refreshAccessToken(),/AUTH_RETRY_LATER/);await assert.rejects(api.refreshAccessToken(),/AUTH_RETRY_LATER/);
  assert.equal(count,1);assert.equal(engine.local.accessToken,'synthetic');assert.equal(engine.local.refreshToken,'synthetic');
});
test('true invalid refresh clears credentials and private profile',async()=>{
  const fetch=async()=>Response.json({},{status:401});const engine=extensionFixture({fetch});engine.local.autofillProfile={contact:{full_name:'Synthetic'}};
  const api=panel(engine,fetch);await api.init();await assert.rejects(api.refreshAccessToken(),/SESSION_EXPIRED/);
  assert.equal(engine.local.accessToken,undefined);assert.equal(engine.local.autofillProfile,undefined);
});
test('absent session never refreshes or dispatches API requests',async()=>{
  let count=0;const fetch=async()=>{count++;return Response.json({})};const engine=extensionFixture({fetch,stored:{}});const api=panel(engine,fetch);
  assert.equal(await api.init(),false);await assert.rejects(api.listJobs(),/SESSION_EXPIRED/);await assert.rejects(api.refreshAccessToken(),/AI_CANCELLED/);assert.equal(count,0);
});
test('content script cannot invoke any privileged auth route',async()=>{
  let calls=0;const engine=extensionFixture({fetch:async()=>{calls++;return loginResponse()}});
  const runtime=engine.chromeFor({id:'fixture',tab:{id:1},url:'https://hostile.invalid'}).runtime;
  for(const action of ['authSnapshot','authLogin','authLogout','authRefresh','authStoreProfile','authTakePending','authBeginProfile','authCancelProfile','authTrackImportedUrl','authIncrementSavedCount'])assert.equal(await runtime.sendMessage({action,profile:{}}),undefined);
  assert.equal(calls,0);assert.equal(engine.local.accessToken,'synthetic');assert.equal(engine.local.autofillProfile,undefined);
});
test('only exact trusted API origins are used for auth',async()=>{
  for(const base of ['https://jobber-app.com.evil.invalid','https://jobber-app.com@evil.invalid','https://jobber-app.com/extra']){
    let called;const fetch=async url=>{called=url;return loginResponse()};const engine=extensionFixture({fetch,stored:{apiBase:base}});const api=panel(engine,fetch);await api.login('synthetic','synthetic');assert.equal(called,'https://jobber-app.com/api/v1/auth/login');
  }
});
for(const operation of ['parseJob','getUploadedAutofillProfile','getMatchScore'])test(`${operation} never confirms or posts A content after B login`,async()=>{
  const balance=deferred();let notices=0;const posts=[];
  const fetch=async(url,options={})=>{
    if(url.endsWith('/auth/login'))return loginResponse();
    if(url.endsWith('/subscription'))return balance.promise;
    if(options.method==='POST'){posts.push({url,options});return Response.json({})}
    return Response.json({error_code:'MATCH_SCORE_NOT_READY'},{status:404});
  };
  const engine=extensionFixture({fetch});const a=panel(engine,fetch,()=>{notices++;return true}),b=panel(engine,fetch);await b.init();
  const pending=a[operation]('synthetic A content','synthetic').then(()=>null,error=>error.message);await tick();
  await b.logout();await b.login('synthetic','synthetic');balance.resolve(Response.json({credits:{remaining:30},credit_costs:{job_parse:2,autofill_import:7,match:5}}));
  assert.equal(await pending,'AI_CANCELLED');assert.equal(notices,0);assert.equal(posts.length,0);
});
test('late login response cannot restore a session invalidated by another panel',async()=>{
  const held=deferred();const fetch=async()=>held.promise;const engine=extensionFixture({fetch});const a=panel(engine,fetch),b=panel(engine,fetch);await b.init();
  const pending=a.login('synthetic','synthetic').then(()=>null,error=>error.message);await tick();await b.logout();held.resolve(loginResponse());assert.equal(await pending,'AI_CANCELLED');assert.equal(engine.local.accessToken,undefined);
});
for(const operation of ['parseJob','getUploadedAutofillProfile','getMatchScore'])test(`${operation} checks cancellation at the final async predispatch boundary`,async()=>{
  const gate=deferred(),arrived=deferred(),controller=new AbortController();let confirmed=false,snapshots=0,posts=0;
  const fetch=async(url,options={})=>{
    if(url.endsWith('/subscription'))return Response.json({credits:{remaining:30},credit_costs:{job_parse:2,autofill_import:7,match:5}});
    if(options.method==='POST'){posts++;return Response.json({})}
    return Response.json({error_code:'MATCH_SCORE_NOT_READY'},{status:404});
  };
  const engine=extensionFixture({fetch}),chrome=engine.chromeFor();const send=chrome.runtime.sendMessage;
  chrome.runtime.sendMessage=async message=>{
    const result=await send(message);
    if(message.action==='authSnapshot'&&confirmed&&++snapshots===3){arrived.resolve();await gate.promise}
    return result;
  };
  const api=vm.runInNewContext(`${source}\nJobberAPI`,{chrome,fetch,crypto,confirm:()=>{confirmed=true;return true}});
  const options={consentSignal:controller.signal};
  const pending=(operation==='getUploadedAutofillProfile'?api[operation]('synthetic',options):api[operation]('synthetic','synthetic',options)).then(()=>null,error=>error.message);
  await arrived.promise;controller.abort();gate.resolve();assert.equal(await pending,'AI_CANCELLED');assert.equal(posts,0);
});
test('simultaneous panels atomically consume a pending command once and reject another account marker',async()=>{
  const engine=extensionFixture({fetch:async()=>Response.json({}),pending:{timestamp:Date.now(),sessionId:'session-a',pageData:{text:'synthetic'}}});
  const a=engine.chromeFor().runtime,b=engine.chromeFor().runtime;
  const results=await Promise.all([a.sendMessage({action:'authTakePending',sessionId:'session-a'}),b.sendMessage({action:'authTakePending',sessionId:'session-a'})]);
  assert.equal(results.filter(result=>result.data).length,1);
  await engine.storage.session.set({pendingParse:{timestamp:Date.now(),sessionId:'another-account'}});
  assert.equal((await a.sendMessage({action:'authTakePending',sessionId:'session-a'})).data,null);
});
test('actual pending producer and consumer interleaving preserves the newly published command exactly once',async()=>{
  const engine=extensionFixture({fetch:async()=>Response.json({}),pending:{timestamp:Date.now(),sessionId:'session-a',pageData:{text:'old command'}}});
  engine.workerChrome.scripting.executeScript=async()=>[{result:{text:'new command',url:'https://fixture.invalid/new'}}];
  const publishing=deferred(),publishGate=deferred(),removeGate=deferred();
  const originalSet=engine.storage.session.set,originalRemove=engine.storage.session.remove;
  engine.storage.session.set=async value=>{if(value.pendingParse?.pageData?.text==='new command'){publishing.resolve();await publishGate.promise}return originalSet(value)};
  engine.storage.session.remove=async keys=>{if(keys==='pendingParse')await removeGate.promise;return originalRemove(keys)};
  const producer=vm.runInContext('openSidePanelWithPageData({id:1,windowId:1})',engine.context);
  await publishing.promise;
  const consumer=engine.chromeFor().runtime.sendMessage({action:'authTakePending',sessionId:'session-a'});await tick();
  publishGate.resolve();await producer;removeGate.resolve();
  const first=await consumer;
  const next=await engine.chromeFor().runtime.sendMessage({action:'authTakePending',sessionId:'session-a'});
  assert.equal([first.data,next.data].filter(value=>value?.pageData?.text==='new command').length,1);
  assert.equal(engine.session.pendingParse,undefined);
});
for(const phase of ['importedUrls','savedCount'])for(const method of ['get','set'])test(`worker quick-save ${phase} ${method} transaction cannot resume under B`,async()=>{
  const job={id:'private-a',title:'Private synthetic A job',url:'https://fixture.invalid/private-a'};
  const fetch=async url=>url.endsWith('/auth/login')?loginResponse():Response.json(job);
  const engine=extensionFixture({fetch}),api=panel(engine,fetch);await api.init();
  const area=phase==='importedUrls'?engine.storage.local:engine.storage.session;
  const original=area[method],gate=deferred(),arrived=deferred(),events=[];let hold=true;
  area[method]=async value=>{if((method==='get'?value===phase:phase in value)&&hold){hold=false;arrived.resolve();await gate.promise}if(method==='set'&&phase in value)events.push({kind:'write',session:engine.local.authSessionId});return original(value)};
  engine.workerChrome.action.setBadgeText=async()=>events.push({kind:'badge',session:engine.local.authSessionId});
  const quickSave=new Promise(resolve=>engine.workerChrome.runtime.onMessage.listeners[0]({action:'quickSave',title:job.title,url:job.url},{id:'fixture',tab:{id:1},url:'https://fixture.invalid'},result=>{events.push({kind:'response',session:engine.local.authSessionId,result});resolve(result)}));
  await arrived.promise;
  const replacement=api.logout().then(()=>api.login('synthetic','synthetic'));await tick();assert.equal(engine.local.accessToken,'synthetic');
  gate.resolve();const result=await quickSave;await replacement;
  assert.equal(result.ok,true);assert.equal(engine.local.accessToken,'account-b');assert.ok(events.length>=2);assert.ok(events.every(event=>event.session==='session-a'));
});
for(const operation of ['createCompany','createJob'])test(`${operation} checks cancellation immediately before dispatch`,async()=>{
  const arrived=deferred(),gate=deferred(),controller=new AbortController();let posts=0;
  const fetch=async()=>{posts++;return Response.json({})};const engine=extensionFixture({fetch}),chrome=engine.chromeFor();
  const send=chrome.runtime.sendMessage;chrome.runtime.sendMessage=async message=>{const result=await send(message);if(message.action==='authSnapshot'){arrived.resolve();await gate.promise}return result};
  const api=vm.runInNewContext(`${source}\nJobberAPI`,{chrome,fetch,crypto});
  const session={accessToken:'synthetic',authSessionId:'session-a',apiBase:'https://jobber-app.com'};
  const pending=api[operation](operation==='createCompany'?'Synthetic':{title:'Synthetic'}, {session,consentSignal:controller.signal}).then(()=>null,error=>error.message);
  await arrived.promise;controller.abort();gate.resolve();assert.equal(await pending,'AI_CANCELLED');assert.equal(posts,0);
});
test('committed profile survives worker recreation and another panel stale cancellation cannot revoke it',async()=>{
  const fetch=async()=>Response.json({});const engine=extensionFixture({fetch}),a=panel(engine,fetch),b=panel(engine,fetch);await a.init();await b.init();
  const first=a.beginAutofillRequest();await first.ready;await a.storeAutofillProfile({contact:{full_name:'First panel'}},first);
  const second=b.beginAutofillRequest();await second.ready;await b.storeAutofillProfile({contact:{full_name:'Second panel committed'}},second);
  await engine.chromeFor({id:'fixture',documentId:'other-document',url:'chrome-extension://fixture/sidepanel/sidepanel.html'}).runtime.sendMessage({action:'authCancelProfile',request:second.request});
  first.cancel();await tick();await tick();
  assert.equal((await engine.chromeFor().runtime.sendMessage({action:'getAutofillProfile'})).profile.contact.full_name,'Second panel committed');
  const restarted=extensionFixture({fetch,stored:{...engine.local}});
  const response=await restarted.chromeFor({id:'fixture',tab:{id:1},url:'https://fixture.invalid'}).runtime.sendMessage({action:'getAutofillProfile'});
  assert.deepEqual(Object.keys(response),['profile']);assert.equal(response.profile.contact.full_name,'Second panel committed');
});
test('pending registration canceled before worker restart cannot be committed afterward',async()=>{
  const fetch=async()=>Response.json({}),engine=extensionFixture({fetch}),api=panel(engine,fetch);await api.init();const pending=api.beginAutofillRequest();await pending.ready;
  pending.cancel();await tick();await tick();engine.restartWorker();
  await assert.rejects(api.storeAutofillProfile({contact:{full_name:'Canceled'}},pending),/AI_CANCELLED/);assert.equal(engine.local.autofillProfile,undefined);
});
test('superseded pending revision stays rejected after worker restart while its replacement commits',async()=>{
  const fetch=async()=>Response.json({}),engine=extensionFixture({fetch}),api=panel(engine,fetch);await api.init();const first=api.beginAutofillRequest();await first.ready;const second=api.beginAutofillRequest();await second.ready;engine.restartWorker();
  await assert.rejects(api.storeAutofillProfile({contact:{full_name:'Superseded'}},first),/AI_CANCELLED/);await api.storeAutofillProfile({contact:{full_name:'Replacement'}},second);
  assert.equal((await engine.chromeFor().runtime.sendMessage({action:'getAutofillProfile'})).profile.contact.full_name,'Replacement');
});
test('recovered pending registration rejects another document, session or revision and preserves the original owner',async()=>{
  const fetch=async()=>Response.json({}),engine=extensionFixture({fetch}),api=panel(engine,fetch);await api.init();const pending=api.beginAutofillRequest();await pending.ready;engine.restartWorker();
  for(const change of ['document','session','revision']){
    const request={...pending.request};if(change==='session')request.sessionId='another-session';if(change==='revision')request.revision++;
    const sender=change==='document'?{id:'fixture',documentId:'another-document',url:'chrome-extension://fixture/sidepanel/sidepanel.html'}:undefined;
    const result=await engine.chromeFor(sender).runtime.sendMessage({action:'authStoreProfile',request,profile:{contact:{full_name:'Hijack'}}});assert.equal(result.error,'AI_CANCELLED');
  }
  const foreign=engine.chromeFor({id:'fixture',documentId:'another-document',url:'chrome-extension://fixture/sidepanel/sidepanel.html'}).runtime;
  assert.equal((await foreign.sendMessage({action:'authBeginProfile',request:{...pending.request,revision:pending.request.revision+1}})).error,'AI_CANCELLED');
  await api.storeAutofillProfile({contact:{full_name:'Original owner'}},pending);assert.equal(engine.local.autofillProfile.contact.full_name,'Original owner');
});
