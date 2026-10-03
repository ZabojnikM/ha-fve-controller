import type {TuvTelemetry} from './live'

const labels:Record<string,string>={
  'Aktivní':'Ohřev povolen',
  'Obnova čidel TUV':'Obnova čidel',
  'Zablokováno (Teplota)':'Blokováno přehřátím',
  'Zablokováno (Porucha čidel po 3 resetech)':'Porucha čidel po 3 pokusech o obnovu',
  'Zablokováno (Porucha čidla)':'Porucha čidel',
  'Zablokováno (Watchdog)':'Blokováno watchdogem · chyběl heartbeat',
  'Zablokováno (Přetížení)':'Blokováno přetížením',
}

export function tuvStatus(data:TuvTelemetry|undefined,available:boolean){
  const r=data?.readings.system
  const raw=available&&r?.quality==='valid'?r.text:undefined
  const on=(key:string)=>available&&data?.readings[key]?.quality==='valid'&&data.readings[key].value===1
  const overload=on('overload')
  const reset=on('sensor_reset')
  const sensorFault=reset||!!raw&&['Obnova čidel TUV','Zablokováno (Porucha čidla)','Zablokováno (Porucha čidel po 3 resetech)'].includes(raw)
  // Inputs can reveal a block before the firmware's next textual report.
  const label=raw==='Aktivní'&&reset?'Obnova čidel':raw==='Aktivní'&&overload?'Blokováno přetížením':raw&&labels[raw]?labels[raw]:'Stav systému není dostupný'
  const tone=raw==='Aktivní'&&!overload&&!reset?'ready':(raw&&labels[raw]||overload||reset)?'blocked':'unknown'
  const details=[reset?'Napájení čidel je odpojené':null,overload?'Přetížení X16 je aktivní':null].filter(Boolean).join(' · ')
  return {label,tone,details,sensorFault}
}
