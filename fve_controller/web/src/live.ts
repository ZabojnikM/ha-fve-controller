import {onMounted,onUnmounted,ref} from 'vue'

export type Reading={value:number|null;text?:string;unit:string;quality:string;source_at?:string|null;at?:string|null;retained?:boolean}
export type VictronTelemetry={enabled:boolean;connected:boolean;status:string;fresh_seconds:number;readings:Record<string,Reading>}
export type PumpControl={mode?:string;ready?:boolean;manual_request?:boolean|null;enabled:boolean;owner:string;status:string;reason:string;desired:boolean|null;sent:boolean|null;confirmed:boolean|null;sent_at?:string|null;error?:string;process_request:boolean;service_request:boolean;service_until?:string|null;last_service_day:string}
export type TuvControl={mode:'auto'|'manual';pump_owned:boolean;automatic_heating:boolean;heating_owner:string}
export type TuvTelemetry={enabled:boolean;connected:boolean;status:string;received_at:string|null;readings:Record<string,Reading>;nominal_power:Reading;pump_control?:PumpControl;tuv_control?:TuvControl;control_token?:string}
export type TeslaTelemetry={enabled:boolean;connected:boolean;status:string;received_at:string|null;readings:Record<string,Reading>}
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
  return {victron,tuv,tesla,refresh}
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
