<script setup lang="ts">
import {computed} from 'vue'
import EnergyIcon from './EnergyIcon.vue'
import {numeric,solarTotal,format,qualities,sourceStatus} from './live'
import type {Reading,Source,VictronTelemetry,TuvTelemetry,TeslaTelemetry} from './live'
const props=defineProps<{victron:Source<VictronTelemetry>;tuv:Source<TuvTelemetry>;tesla:Source<TeslaTelemetry>}>()
const strings=[['solar_roof','Střecha'],['solar_shelter','Přístřešek'],['solar_fence','Plot']]
const vReady=computed(()=>!props.victron.error&&!!props.victron.data?.enabled&&!!props.victron.data?.connected)
const tReady=computed(()=>!props.tuv.error&&!!props.tuv.data?.enabled&&!!props.tuv.data?.connected)
const carReady=computed(()=>!props.tesla.error&&!!props.tesla.data?.enabled&&!!props.tesla.data?.connected)
function v(key:string){return numeric(props.victron.data?.readings[key],vReady.value)}
function t(key:string){return numeric(props.tuv.data?.readings[key],tReady.value)}
function car(key:string){return numeric(props.tesla.data?.readings[key],carReady.value)}
const total=computed(()=>solarTotal(props.victron.data?.readings,vReady.value))
const nominal=computed(()=>numeric(props.tuv.data?.nominal_power,tReady.value))
const direction=computed(()=>{
  const watts=v('battery')
  return watts===null?'Tok není dostupný':watts>0?'Nabíjí se':watts<0?'Vybíjí se':'Bez toku'
})
function kw(value:number|null){return value===null?'—':format(value/1000,2)}
function quality(reading:Reading|undefined,ready:boolean,mqtt=false){
  if(!ready)return 'Údaj není dostupný'
  return reading?.quality==='valid'&&mqtt?'Přijatá data':qualities[reading?.quality??'missing']??'Údaj není dostupný'
}
function stamp(reading?:Reading,mqtt=false){
  const at=mqtt?reading?.at:reading?.source_at
  return at?`${mqtt?'Příjem':'Hlášení HA'} ${new Date(at).toLocaleString('cs-CZ')}`:''
}
function share(key:string){return total.value!==null&&total.value>0?Math.max(0,(v(key)??0)/total.value*100):0}
function percent(value:number|null){return value===null?0:Math.min(100,Math.max(0,value))}
const charging=computed(()=>{
  const r=props.tesla.data?.readings.charging
  if(!carReady.value||r?.quality!=='valid')return 'Stav nabíjení není dostupný'
  const labels:Record<string,string>={starting:'Spouští se',charging:'Nabíjí se',stopped:'Zastaveno',complete:'Dokončeno',disconnected:'Odpojeno',no_power:'Bez napájení'}
  return labels[r.text??'']??'Stav nabíjení není dostupný'
})
const cable=computed(()=>car('connected')===null?'—':car('connected')===1?'Připojený':'Odpojený')
const pump=computed(()=>t('pump')===null?'—':t('pump')===1?'Zapnuté':'Vypnuté')
</script>
<template>
  <div class="live-overview">
    <section class="energy-overview" aria-labelledby="solar-live-title">
      <div class="energy-heading">
        <div class="solar-total"><div class="eyebrow"><EnergyIcon name="sun"/><h2 id="solar-live-title">Solární výroba</h2></div><div class="metric" data-testid="solar-total">{{format(total,0)}} <small>W</small></div><p class="source-caption">{{sourceStatus(victron,true)}}</p></div>
        <div class="battery-glance"><span class="muted"><EnergyIcon name="battery"/> Baterie</span><strong>{{format(v('soc'))}} <small>%</small></strong><span class="muted">{{direction}} · {{kw(v('battery'))}} kW</span></div>
      </div>
      <div class="production-bar" role="img" :aria-label="total===null?'Součet výroby není dostupný':`Výroba ${format(total,0)} W; střecha ${format(v('solar_roof'),0)} W, přístřešek ${format(v('solar_shelter'),0)} W, plot ${format(v('solar_fence'),0)} W`"><span v-for="[key] in strings" :key="key" :class="key" :style="{width:share(key)+'%'}"></span></div>
      <div class="solar-readings"><div v-for="[key,label] in strings" :key="key" class="solar-string"><h3><i :class="key"></i>{{label}}</h3><strong>{{format(v(key),0)}} <small>W</small></strong><p :class="v(key)!==null?'muted':'quality-warning'">{{quality(victron.data?.readings[key],vReady,true)}}</p><p class="muted">{{stamp(victron.data?.readings[key],true)}}</p></div></div>
      <p v-if="total===null" class="quality-warning">Součet čeká na platná data všech tří stringů.</p>
    </section>
    <div class="live-status"><strong>Pouze sledování</strong><span>Zobrazené hodnoty jsou ze zařízení. Doplněk zatím neodesílá žádné příkazy.</span></div>
    <div class="loads">
      <section class="card load-card water" aria-labelledby="water-live-title">
        <div class="cardhead"><h2 id="water-live-title"><EnergyIcon name="water"/>Teplá voda</h2><span class="mode-pill neutral">{{sourceStatus(tuv)}}</span></div>
        <div class="load-metrics"><div v-for="[key,label] in [['upper','Horní teplota'],['lower','Spodní teplota']]" :key="key"><p class="eyebrow">{{label}}</p><div class="metric" :data-testid="'tuv-'+key">{{format(t(key))}} <small>°C</small></div><p :class="t(key)!==null?'source-caption':'quality-warning'">{{quality(tuv.data?.readings[key],tReady)}}</p><p class="source-caption">{{stamp(tuv.data?.readings[key])}}</p></div></div>
        <div class="device-detail"><div class="device-title"><strong>Stupeň ohřevu</strong><span class="muted">Jmenovitě {{kw(nominal)}} kW</span></div><div class="stages" aria-label="Hlášený celkový stupeň ohřevu"><span v-for="stage in [0,1,2,3]" :key="stage" :class="{active:nominal!==null&&nominal===stage*1000}">{{stage}} kW</span></div><p :class="nominal!==null?'muted':'quality-warning'">{{nominal===null?quality(tuv.data?.nominal_power,tReady):'Podle přepínačů v HA · příkon se neměří.'}}</p><dl><dt>Čerpadlo</dt><dd>{{pump}}<small v-if="t('pump')===null" class="reading-note quality-warning">{{quality(tuv.data?.readings.pump,tReady)}}</small></dd><dt>Řízení doplňkem</dt><dd>Vypnuté</dd></dl><p class="muted">Hlášený stupeň nepotvrzuje fyzické sepnutí spirál.</p></div>
      </section>
      <section class="card load-card car" aria-labelledby="car-live-title">
        <div class="cardhead"><h2 id="car-live-title"><EnergyIcon name="car"/>Tesla</h2><span class="mode-pill neutral">{{sourceStatus(tesla)}}</span></div>
        <div class="load-metrics"><div><p class="eyebrow">Výkon nabíječky</p><div class="metric" data-testid="tesla-power">{{kw(car('power'))}} <small>kW</small></div><p :class="car('power')!==null?'source-caption':'quality-warning'">{{quality(tesla.data?.readings.power,carReady)}}</p></div><div><p class="eyebrow">Stav nabití</p><div class="metric">{{format(car('soc'))}} <small>%</small></div><p :class="car('soc')!==null?'source-caption':'quality-warning'">{{quality(tesla.data?.readings.soc,carReady)}}</p></div></div>
        <div class="device-detail"><p class="device-message">{{charging}}</p><dl><dt>Nabíjecí kabel</dt><dd>{{cable}}<small v-if="car('connected')===null" class="reading-note quality-warning">{{quality(tesla.data?.readings.connected,carReady)}}</small></dd><dt>Skutečný proud</dt><dd><span data-testid="tesla-current">{{format(car('current'))}} A</span><small v-if="car('current')===null" class="reading-note quality-warning">{{quality(tesla.data?.readings.current,carReady)}}</small></dd><dt>Nastavený proud v HA</dt><dd><span data-testid="tesla-set-current">{{format(car('current_limit'))}} A</span><small v-if="car('current_limit')===null" class="reading-note quality-warning">{{quality(tesla.data?.readings.current_limit,carReady)}}</small></dd></dl><p class="muted">Nastavený proud je požadavek, nikoli skutečný odběr.</p><p class="muted">Hlášení HA může pocházet z mezipaměti Tessie. Auto neprobouzíme.</p></div>
      </section>
    </div>
    <section class="card battery-card live-battery" aria-labelledby="battery-live-title"><div class="cardhead"><h2 id="battery-live-title"><EnergyIcon name="battery"/>Domácí baterie</h2><span class="mode-pill neutral">{{sourceStatus(victron,true)}}</span></div><div class="battery-body"><div><div class="metric">{{format(v('soc'))}} <small>%</small></div><div class="bar" role="img" :aria-label="v('soc')===null?'Stav nabití není dostupný':`Stav nabití ${format(v('soc'))} %`"><i :style="{width:percent(v('soc'))+'%'}"></i></div><p :class="v('soc')!==null?'source-caption':'quality-warning'">{{quality(victron.data?.readings.soc,vReady,true)}} · {{stamp(victron.data?.readings.soc,true)}}</p></div><dl><dt>Tok baterie</dt><dd>{{format(v('battery'),0)}} W · {{direction}}</dd><dt>Nejnižší článek</dt><dd>{{format(v('min_cell'),3)}} V<small v-if="v('min_cell')===null" class="reading-note quality-warning">{{quality(victron.data?.readings.min_cell,vReady,true)}}</small></dd><template v-if="victron.data?.readings.max_cell?.quality!=='not_configured'"><dt>Nejvyšší článek</dt><dd>{{format(v('max_cell'),3)}} V<small v-if="v('max_cell')===null" class="reading-note quality-warning">{{quality(victron.data?.readings.max_cell,vReady,true)}}</small></dd></template></dl></div><p class="source-caption">+ nabíjení / − vybíjení · MQTT: limit stáří {{victron.data?.fresh_seconds??60}} s od příjmu.</p></section>
  </div>
</template>
