const fs = require('node:fs');
const vm = require('node:vm');
const crypto = require('node:crypto');
const event = () => { const listeners=[]; return { addListener: fn=>listeners.push(fn), emit: (...args)=>listeners.map(fn=>fn(...args)), listeners }; };
function extensionFixture({fetch,stored={accessToken:'synthetic',refreshToken:'synthetic',authSessionId:'session-a'},pending}={}) {
  const local={...stored},session={pendingParse:pending};
  const changed=event(),messages=event(),alarm=event();
  const storageArea=(data,area)=>({
    get: async keys=>Object.fromEntries((typeof keys==='string'?[keys]:keys||Object.keys(data)).map(key=>[key,data[key]])),
    set: async values=>{const changes={};for(const [key,value]of Object.entries(values)){const oldValue=data[key];data[key]=value;if(oldValue!==value)changes[key]={oldValue,newValue:value}}if(Object.keys(changes).length)changed.emit(changes,area)},
    remove: async keys=>{const changes={};for(const key of typeof keys==='string'?[keys]:keys){if(key in data){changes[key]={oldValue:data[key]};delete data[key]}}if(Object.keys(changes).length)changed.emit(changes,area)},
  });
  const storage={local:storageArea(local,'local'),session:storageArea(session,'session'),onChanged:changed};
  const workerChrome={storage,runtime:{id:'fixture',getURL:p=>`chrome-extension://fixture/${p}`,onMessage:messages,onInstalled:event(),onStartup:event()},sidePanel:{setPanelBehavior:async()=>{},open:async()=>{}},contextMenus:{onClicked:event(),removeAll:async()=>{},create:()=>{}},commands:{onCommand:event()},alarms:{onAlarm:alarm,get:async()=>({}),create:()=>{}},tabs:{query:async()=>[],sendMessage:async()=>{}},scripting:{executeScript:async()=>[],insertCSS:async()=>{}},action:{setBadgeText:async()=>{},setBadgeBackgroundColor:async()=>{}}};
  let context;
  function restartWorker(){
    for(const surface of [messages,alarm,workerChrome.runtime.onInstalled,workerChrome.runtime.onStartup,workerChrome.contextMenus.onClicked,workerChrome.commands.onCommand])surface.listeners.length=0;
    context=vm.createContext({chrome:workerChrome,fetch,crypto,URL,console});
    context.importScripts=rel=>vm.runInContext(fs.readFileSync(`${__dirname}/${rel}`,'utf8'),context);
    vm.runInContext(fs.readFileSync(`${__dirname}/background.js`,'utf8'),context);
    return context;
  }
  restartWorker();
  function chromeFor(sender={id:'fixture',documentId:'fixture-panel-document',url:'chrome-extension://fixture/sidepanel/sidepanel.html'}) {
    return {storage,runtime:{id:'fixture',sendMessage:message=>new Promise((resolve,reject)=>{
      try {const handled=messages.listeners.some(listener=>listener(message,sender,resolve));if(!handled)resolve(undefined)}catch(error){reject(error)}
    })}};
  }
  return {local,session,storage,chromeFor,get context(){return context},restartWorker,workerChrome,alarm};
}
module.exports={extensionFixture,event};
