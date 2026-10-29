const {test}=require('node:test');
const assert=require('node:assert/strict');
const path=require('node:path');
const fs=require('node:fs');
const {chromium}=require('../../fe/node_modules/playwright');
const {extensionFixture}=require('../test-fixture.cjs');
const root=path.resolve(__dirname,'..');
test('real browser sidepanel loads all controllers, consumes repeated commands and clears login/profile secrets',async()=>{
  let parses=0;
  const fixtureFetch=async(url,options={})=>{
    if(url.endsWith('/auth/login'))return Response.json({tokens:{access_token:'synthetic',refresh_token:'synthetic-refresh'}});
    if(url.endsWith('/subscription'))return Response.json({credits:{remaining:30,granted:30},credit_costs:{job_parse:2,match:5,autofill_import:7}});
    if(url.endsWith('/jobs/parse')){parses++;return Response.json({title:`Synthetic role ${parses}`,url:'https://fixture.invalid/job'})}
    if(url.endsWith('/resume-builder'))return Response.json({items:[{id:'synthetic',title:'Synthetic Builder'}]});
    if(url.endsWith('/resume-builder/synthetic'))return Response.json({contact:{full_name:'Synthetic Browser Applicant'}});
    return Response.json({items:[]});
  };
  const engine=extensionFixture({stored:{},fetch:fixtureFetch});
  const context=await chromium.launchPersistentContext('',{executablePath:'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',headless:true});
  const page=await context.newPage();const errors=[];page.on('pageerror',error=>errors.push(error.message));page.on('dialog',dialog=>dialog.accept());
  try{
    const runtime=engine.chromeFor().runtime;
    await page.exposeFunction('fixtureMessage',message=>runtime.sendMessage(message));
    await page.exposeFunction('fixtureStorage',(area,method,args)=>engine.storage[area][method](args));
    await page.addInitScript(()=>{
      const storage=area=>Object.fromEntries(['get','set','remove'].map(method=>[method,args=>window.fixtureStorage(area,method,args)]));
      window.fixtureChanges=[];
      window.chrome={storage:{local:storage('local'),session:storage('session'),onChanged:{addListener:fn=>fixtureChanges.push(fn)}},runtime:{sendMessage:message=>window.fixtureMessage(message)},permissions:{request:async()=>true},tabs:{query:async()=>[{id:1,active:true,url:'https://fixture.invalid/job'}],onActivated:{addListener:()=>{}},onUpdated:{addListener:()=>{}},create:()=>{}}};
    });
    engine.storage.onChanged.addListener((changes,area)=>{void page.evaluate(({changes,area})=>window.fixtureChanges?.forEach(fn=>fn(changes,area)),{changes,area}).catch(()=>{})});
    await page.route('**/*',async route=>{
      if(new URL(route.request().url()).origin==='https://jobber-app.com'){
        const headers={'access-control-allow-origin':'*','access-control-allow-headers':'*','access-control-allow-methods':'GET,POST,OPTIONS'};
        if(route.request().method()==='OPTIONS')return route.fulfill({status:204,headers,body:''});
        const response=await fixtureFetch(route.request().url(),{method:route.request().method(),headers:route.request().headers(),body:route.request().postData()});
        return route.fulfill({status:response.status,headers,contentType:'application/json',body:await response.text()});
      }
      const pathname=new URL(route.request().url()).pathname;
      const relative=pathname==='/'?'sidepanel/sidepanel.html':pathname.slice(1);
      if(!['sidepanel/sidepanel.html','sidepanel/sidepanel.js','sidepanel/autofill.js','sidepanel/sidepanel.css','shared/api.js'].includes(relative))return route.abort();
      return route.fulfill({path:path.join(root,relative),contentType:relative.endsWith('.js')?'application/javascript':relative.endsWith('.css')?'text/css':'text/html'});
    });
    await page.goto('https://panel.fixture.invalid/sidepanel/sidepanel.html');
    await page.locator('#login-email').fill('synthetic@example.invalid');await page.locator('#login-password').fill('synthetic-password');await page.locator('#btn-toggle-password').click();
    assert.equal(await page.locator('#login-password').getAttribute('type'),'text');await page.locator('#btn-login').click();await page.locator('#view-main').waitFor({state:'visible'});
    assert.equal(await page.locator('#login-password').inputValue(),'');assert.equal(await page.locator('#login-password').getAttribute('type'),'password');
    for(let index=1;index<=2;index++){
      await engine.storage.session.set({pendingParse:{timestamp:Date.now(),sessionId:engine.local.authSessionId,pageData:{text:'Synthetic job description for local browser fixture',url:'https://fixture.invalid/job'}}});
      await page.waitForFunction(value=>document.querySelector('#preview-title').value===`Synthetic role ${value}`,index);
      assert.equal(parses,index);
    }
    await page.locator('[data-tab="autofill"]').click();await page.locator('#autofill-resume-select').selectOption('rb:synthetic');await page.locator('#btn-set-autofill').click();
    await page.waitForFunction(()=>document.querySelector('#profile-name').textContent==='Synthetic Browser Applicant');assert.equal(engine.local.autofillProfile.contact.full_name,'Synthetic Browser Applicant');
    await page.locator('#btn-logout').click();await page.locator('#view-login').waitFor({state:'visible'});await page.waitForFunction(()=>document.querySelector('#profile-name').textContent==='');
    assert.equal(engine.local.autofillProfile,undefined);assert.equal(await page.locator('#login-password').inputValue(),'');assert.deepEqual(errors,[]);
  }finally{await context.close()}
});
