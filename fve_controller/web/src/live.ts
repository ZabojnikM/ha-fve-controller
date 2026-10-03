import {onMounted,onUnmounted,ref} from 'vue'

export type Reading={value:number|null;text?:string;unit:string;quality:string;source_at?:string|null;at?:string|null;retained?:boolean}
export type VictronTelemetry={enabled:boolean;connected:boolean;status:string;fresh_seconds:number;readings:Record<string,Reading>}
export type TuvTelemetry={enabled:boolean;connected:boolean;status:string;received_at:string|null;temperature_fresh_seconds:number;readings:Record<string,Reading>;nominal_power:Reading}
export type TeslaTelemetry={enabled:boolean;connected:boolean;status:string;received_at:string|null;fresh_seconds:number;readings:Record<string,Reading>}
export type Source<T>={data?:T;error:boolean}

// One refresh per source, shared by the overview and diagnostics. No simulated fallback.
export function useLiveTelemetry(){
  const victron=ref<Source<VictronTelemetry>>({error:false})
  const tuv=ref<Source<TuvTelemetry>>({error:false})
  const tesla=ref<Source<TeslaTelemetry>>({error:false})
  let pending=false,stopped=false
  async function fetchSource<T>(path:string):Promise<Source<T>>{
    try{
      const response=await fetch(path,{signal:AbortSignal.timeout(5000),cache:'no-store'})
      if(!response.ok)throw Error()
      return {data:await response.json(),error:false}
    }catch{return {error:true}}
  }
  async function refresh(){
    if(pending)return
    pending=true
    try{
      const results=await Promise.all([fetchSource<VictronTelemetry>('api/victron'),fetchSource<TuvTelemetry>('api/tuv'),fetchSource<TeslaTelemetry>('api/tesla')])
      if(!stopped){[victron.value,tuv.value,tesla.value]=results}
    }finally{pending=false}
  }
  let timer:ReturnType<typeof setInterval>
  onMounted(()=>{refresh();timer=setInterval(refresh,2000)})
  onUnmounted(()=>{stopped=true;clearInterval(timer)})
  return {victron,tuv,tesla}
}

export function numeric(reading:Reading|undefined,available:boolean):number|null{
  return available&&reading?.quality==='valid'&&typeof reading.value==='number'&&Number.isFinite(reading.value)?reading.value:null
}
export function solarTotal(readings:Record<string,Reading>|undefined,available:boolean):number|null{
  const values=['solar_roof','solar_shelter','solar_fence'].map(key=>numeric(readings?.[key],available))
  return values.every(v=>v!==null)?values.reduce<number>((sum,v)=>sum+(v??0),0):null
}
export function format(value:number|null,decimals=1){
  return value===null?'—':value.toLocaleString('cs-CZ',{maximumFractionDigits:decimals})
}
export const qualities:Record<string,string>={valid:'Hlášení HA',missing:'Údaj chybí',not_configured:'Údaj není připojený',invalid:'Neplatný údaj',stale:'Zastaralé hlášení',offline:'Spojení přerušeno',retained:'Uložená zpráva · stáří neověřeno',conflict:'Stupeň neurčený · rozporné přepínače'}
export function sourceStatus(source:Source<{enabled:boolean;connected:boolean;status:string}>,mqtt=false){
  if(source.error)return 'Backend nedostupný'
  if(!source.data)return 'Načítám data'
  const statuses:Record<string,string>={disabled:'Čtení vypnuté',connecting:'Připojuje se',subscribing:'Přihlašuje odběr',listening:mqtt?'MQTT · připojeno':'HA · připojeno',offline:'Spojení přerušeno',unauthorized:'HA odmítl přístup',no_token:'Chybí přístup k HA',error:'Data nelze načíst',stale:'Spojení zastaralo',subscription_error:'Chyba odběru MQTT'}
  return statuses[source.data.status]??'Stav není dostupný'
}
