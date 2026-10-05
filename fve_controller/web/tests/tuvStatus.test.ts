import {test} from 'node:test'
import assert from 'node:assert/strict'
import {tuvStatus} from '../src/tuvStatus.ts'
import type {TuvTelemetry} from '../src/live.ts'

function fixture(state='Aktivní'):TuvTelemetry{
  return {enabled:true,connected:true,status:'listening',received_at:null,
    nominal_power:{value:0,unit:'W',quality:'valid'},readings:{
      system:{value:null,text:state,unit:'',quality:'valid'},
      overload:{value:0,unit:'',quality:'valid'},sensor_reset:{value:0,unit:'',quality:'valid'},
    }}
}
test('enabled system is distinct from selected power; every block has a readable label',()=>{
  assert.equal(tuvStatus(fixture(),true).label,'Ohřev povolen')
  for(const state of ['Obnova čidel TUV','Zablokováno (Teplota)','Zablokováno (Porucha čidel po 3 resetech)','Zablokováno (Porucha čidla)','Zablokováno (Watchdog)','Zablokováno (Přetížení)']){
    const status=tuvStatus(fixture(state),true)
    assert.equal(status.tone,'blocked')
    assert.notEqual(status.label,'Stav systému není dostupný')
  }
})
test('independent inputs reveal delayed or concurrent blocks',()=>{
  const data=fixture()
  data.readings.overload.value=1
  assert.equal(tuvStatus(data,true).label,'Blokováno přetížením')
  data.readings.sensor_reset.value=1
  assert.equal(tuvStatus(data,true).label,'Obnova čidel')
  assert.equal(tuvStatus(data,true).sensorFault,true)
  data.readings.system.text='Zablokováno (Teplota)'
  assert.equal(tuvStatus(data,true).label,'Blokováno přehřátím')
  assert.match(tuvStatus(data,true).details,/X16/)
  assert.match(tuvStatus(data,true).details,/Napájení čidel/)
})
test('missing, stale and disconnected data cannot display a permitted system',()=>{
  assert.equal(tuvStatus(undefined,false).tone,'unknown')
  const data=fixture()
  data.readings.system.quality='stale'
  assert.equal(tuvStatus(data,true).tone,'unknown')
  data.readings.system.quality='valid'
  data.readings.sensor_reset.value=1
  assert.equal(tuvStatus(data,false).sensorFault,false)
  assert.equal(tuvStatus(data,false).tone,'unknown')
  data.readings.sensor_reset.quality='offline'
  assert.equal(tuvStatus(data,true).sensorFault,false)
})
test('sensor faults hide the heat estimate; temperature/watchdog blocks do not invalidate measured temperatures',()=>{
  assert.equal(tuvStatus(fixture('Zablokováno (Porucha čidla)'),true).sensorFault,true)
  assert.equal(tuvStatus(fixture('Zablokováno (Porucha čidel po 3 resetech)'),true).sensorFault,true)
  assert.equal(tuvStatus(fixture('Zablokováno (Teplota)'),true).sensorFault,false)
  assert.equal(tuvStatus(fixture('Zablokováno (Watchdog)'),true).sensorFault,false)
})
