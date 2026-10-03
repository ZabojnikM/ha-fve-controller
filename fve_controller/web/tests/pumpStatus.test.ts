import {test} from 'node:test'
import assert from 'node:assert/strict'
import {pumpStatus} from '../src/pumpStatus.ts'
import type {PumpControl} from '../src/live.ts'

function fixture(status='idle'):PumpControl{return {enabled:true,owner:'addon',status,reason:'Teploty nevyžadují promíchávání',desired:false,sent:false,confirmed:false,process_request:false,service_request:false,last_service_day:'',temperature_fresh_seconds:120}}
test('automatic ownership is independent of whether pump is running',()=>{
  const c=fixture();assert.equal(pumpStatus(c,true).label,'Automatika čerpadla')
  c.enabled=false;assert.equal(pumpStatus(c,true).label,'Řízení doplňkem nepřevzato')
  assert.equal(pumpStatus(c,false).label,'Stav automatiky není dostupný')
})
test('pending command is normal waiting, timeout/error/unavailable are distinct problems',()=>{
  assert.equal(pumpStatus(fixture('waiting_confirmation'),true).problem,false)
  for(const status of ['confirmation_timeout','command_error','input_invalid','unavailable','retry_wait','service_error']){
    assert.equal(pumpStatus(fixture(status),true).problem,true,status)
  }
  const c=fixture('command_error');c.error='Povel se nepodařilo doručit'
  assert.equal(pumpStatus(c,true).reason,c.error)
})

test('manual idle shows manual ownership without claiming automation',()=>{const c=fixture();c.mode='manual';assert.equal(pumpStatus(c,true).label,'Ruční ovládání čerpadla')})
