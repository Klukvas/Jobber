const {test}=require('node:test');
const assert=require('node:assert/strict');
const path=require('node:path');
const fs=require('node:fs');
const {chromium}=require('../../fe/node_modules/playwright');
const {extensionFixture}=require('../test-fixture.cjs');
function deferred(){let resolve;const promise=new Promise(done=>resolve=done);return{promise,resolve}}
const profile={contact:{full_name:'Synthetic Applicant',email:'synthetic@example.invalid'},educations:[{institution:'Synthetic University'}],experiences:[{company:'Synthetic Employer'}]};
async function browserFixture(name,html){
  const context=await chromium.launchPersistentContext('',{executablePath:'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',headless:true});
  const page=await context.newPage();await page.route('**/*',route=>route.fulfill({status:200,contentType:'text/html',body:html}));
  await page.goto('https://www.linkedin.com/jobs/fixture');
  const engine=extensionFixture({fetch:async(url,options={})=>Response.json({id:'synthetic-job',title:'Synthetic job',url:'https://fixture.invalid/saved'}),stored:{accessToken:'synthetic',refreshToken:'synthetic',authSessionId:'session-a',autofillSessionId:'session-a',autofillProfile:profile,autofillProfileRequest:{owner:'browser-profile',revision:1,sessionId:'session-a',apiBase:'https://cursopus.com'}}});
  const runtime=engine.chromeFor({id:'fixture',tab:{id:1},url:'https://www.linkedin.com/jobs/fixture'}).runtime;
  const state={hold:null};
  await page.exposeFunction('fixtureRuntimeMessage',async message=>{const result=await runtime.sendMessage(message);if(message.action==='getAutofillProfile'&&state.hold){const held=state.hold;state.hold=null;held.arrived.resolve();await held.gate.promise}return result});
  await page.exposeFunction('fixtureStorageGet',keys=>engine.storage.local.get(keys));
  await page.evaluate(()=>{
    window.fixture={reads:0,saves:[],listeners:[],changedKeys:[]};
    window.chrome={runtime:{id:'fixture',onMessage:{addListener:fn=>{}},sendMessage:async message=>{if(message.action==='getAutofillProfile')fixture.reads++;if(message.action==='quickSave')fixture.saves.push(message);return window.fixtureRuntimeMessage(message)}},storage:{local:{get:keys=>window.fixtureStorageGet(keys)},onChanged:{addListener:fn=>fixture.listeners.push(fn)}}};
  });
  engine.storage.onChanged.addListener((changes,area)=>{void page.evaluate(({changes,area})=>{fixture.changedKeys.push(...Object.keys(changes));fixture.listeners.forEach(fn=>fn(changes,area))},{changes,area}).catch(()=>{})});
  return{context,page,engine,state};
}
test('autofill requires trusted click, matches semantic labels, and rechecks signed-out storage',async()=>{
  const f=await browserFixture('autofill',`<form>${['Name','First Name','Last Name','School Name','University Name','Current Company Name','Company Name','User Name','Email'].map((label,index)=>`<label>${label}<input id="field-${index}" placeholder="${label}"></label>`).join('')}</form>`);
  try{
    await f.page.addScriptTag({path:path.join(__dirname,'autofill.js')});await f.page.locator('#jobber-autofill-btn').waitFor();
    await f.page.evaluate(()=>document.querySelector('#jobber-autofill-btn').click());
    assert.equal(await f.page.evaluate(()=>fixture.reads),0);assert.equal(await f.page.locator('#field-0').inputValue(),'');
    await f.page.locator('#jobber-autofill-btn').click();
    await f.page.waitForFunction(()=>document.querySelector('#field-0').value==='Synthetic Applicant');
    assert.deepEqual(await f.page.locator('input').evaluateAll(fields=>fields.map(field=>field.value)),['Synthetic Applicant','Synthetic','Applicant','Synthetic University','Synthetic University','Synthetic Employer','','','synthetic@example.invalid']);
    await f.page.evaluate(()=>document.querySelectorAll('input').forEach(field=>field.value=''));
    const held={arrived:deferred(),gate:deferred()};f.state.hold=held;
    await f.page.locator('#jobber-autofill-btn').click();await held.arrived.promise;
    await f.engine.storage.local.set({'autofillRevoked:browser-profile':1});
    await f.page.waitForFunction(()=>fixture.changedKeys.includes('autofillRevoked:browser-profile'));
    held.gate.resolve();await f.page.waitForTimeout(25);assert.equal(await f.page.locator('#field-0').inputValue(),'');
    await f.engine.storage.local.remove('accessToken');await f.page.waitForFunction(()=>fixture.changedKeys.includes('accessToken'));
    await f.page.locator('#jobber-autofill-btn').click();assert.equal(await f.page.locator('#field-0').inputValue(),'');
  }finally{await f.context.close()}
});
test('hydrated and reused SPA cards receive one trusted save control for their current job',async()=>{
  const f=await browserFixture('cards','<div class="job-card-container"><a class="job-card-list__title" href="/jobs/view/one"></a><span class="job-card-container__primary-description">Synthetic Company</span></div>');
  try{
    await f.page.addScriptTag({path:path.join(__dirname,'jobboards.js')});
    assert.equal(await f.page.locator('.jobber-save-btn').count(),0);
    await f.page.locator('a').evaluate(link=>link.textContent='Hydrated Role');await f.page.locator('.jobber-save-btn').waitFor();
    assert.equal(await f.page.locator('.jobber-save-btn').count(),1);
    await f.page.evaluate(()=>document.querySelector('.jobber-save-btn').click());assert.equal(await f.page.evaluate(()=>fixture.saves.length),0);
    await f.page.locator('.jobber-save-btn').click();await f.page.waitForFunction(()=>fixture.saves.length===1);
    assert.equal(await f.page.evaluate(()=>fixture.saves[0].title),'Hydrated Role');
    await f.page.locator('a').evaluate(link=>{link.href='/jobs/view/two';link.textContent='Reused Role'});
    await f.page.waitForFunction(()=>document.querySelector('.jobber-save-btn').textContent.includes('Save')&&!document.querySelector('.jobber-save-btn').textContent.includes('Saved'));
    assert.equal(await f.page.locator('.jobber-save-btn').count(),1);await f.page.locator('.jobber-save-btn').click();
    await f.page.waitForFunction(()=>fixture.saves.length===2);assert.equal(await f.page.evaluate(()=>fixture.saves[1].title),'Reused Role');assert.match(await f.page.evaluate(()=>fixture.saves[1].url),/\/jobs\/view\/two$/);
  }finally{await f.context.close()}
});
