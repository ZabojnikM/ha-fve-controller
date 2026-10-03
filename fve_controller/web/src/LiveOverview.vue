<script setup lang="ts">
import {computed} from 'vue'
import EnergyIcon from './EnergyIcon.vue'
import {numeric,solarTotal,format} from './live'
import type {Source,VictronTelemetry,TuvTelemetry,TeslaTelemetry} from './live'
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
        <div class="solar-total"><div class="eyebrow"><EnergyIcon name="sun"/><h2 id="solar-live-title">Solární výroba</h2></div><div class="metric" data-testid="solar-total">{{format(total,0)}} <small>W</small></div></div>
        <div class="battery-glance"><span class="muted"><EnergyIcon name="battery"/> Baterie</span><strong>{{format(v('soc'))}} <small>%</small></strong><span class="muted">{{direction}} · {{kw(v('battery'))}} kW</span></div>
      </div>
      <div class="production-bar" role="img" :aria-label="total===null?'Součet výroby není dostupný':`Výroba ${format(total,0)} W; střecha ${format(v('solar_roof'),0)} W, přístřešek ${format(v('solar_shelter'),0)} W, plot ${format(v('solar_fence'),0)} W`"><span v-for="[key] in strings" :key="key" :class="key" :style="{width:share(key)+'%'}"></span></div>
      <div class="solar-readings"><div v-for="[key,label] in strings" :key="key" class="solar-string"><h3><i :class="key"></i>{{label}}</h3><strong>{{format(v(key),0)}} <small>W</small></strong></div></div>

    </section>

    <div class="loads">
      <section class="card load-card water" aria-labelledby="water-live-title">
        <div class="cardhead"><h2 id="water-live-title"><EnergyIcon name="water"/>Teplá voda</h2></div>
        <div class="load-metrics"><div v-for="[key,label] in [['upper','Horní teplota'],['lower','Spodní teplota']]" :key="key"><p class="eyebrow">{{label}}</p><div class="metric" :data-testid="'tuv-'+key">{{format(t(key))}} <small>°C</small></div></div></div>
        <div class="device-detail"><div class="device-title"><strong>Stupeň ohřevu</strong></div><div class="heating-status" :class="nominal===null?'unknown':nominal===0?'off':'on'"><i aria-hidden="true"></i><strong>{{nominal===null?'Stupeň není známý':nominal===0?'Ohřev vypnutý':`Zvolený ohřev ${format(nominal/1000,0)} kW`}}</strong></div><div class="stages heating-stages" aria-label="Hlášený celkový stupeň ohřevu"><span v-for="stage in [0,1,2,3]" :key="stage" :class="{active:nominal!==null&&nominal===stage*1000,off:stage===0,unknown:nominal===null}"><strong>{{stage}} kW</strong><small>{{nominal===null?'Neurčeno':nominal===stage*1000?(stage===0?'✓ Vypnuto':'✓ Aktivní'):'Neaktivní'}}</small></span></div><dl><dt>Čerpadlo</dt><dd><span class="state-tag" :class="t('pump')===null?'unknown':t('pump')===1?'on':'off'">{{pump}}</span></dd></dl></div>
      </section>
      <section class="card load-card car" aria-labelledby="car-live-title">
        <div class="cardhead"><h2 id="car-live-title"><EnergyIcon name="car"/>Tesla</h2></div>
        <div class="load-metrics"><div><p class="eyebrow">Výkon nabíječky</p><div class="metric" data-testid="tesla-power">{{kw(car('power'))}} <small>kW</small></div></div><div><p class="eyebrow">Stav nabití</p><div class="metric">{{format(car('soc'))}} <small>%</small></div></div></div>
        <div class="device-detail"><p class="device-message">{{charging}}</p><dl><dt>Nabíjecí kabel</dt><dd>{{cable}}</dd><dt>Skutečný proud</dt><dd><span data-testid="tesla-current">{{format(car('current'))}} A</span></dd><dt>Nastavený proud</dt><dd><span data-testid="tesla-set-current">{{format(car('current_limit'))}} A</span></dd></dl></div>
      </section>
    </div>
    <section class="card battery-card live-battery" aria-labelledby="battery-live-title"><div class="cardhead"><h2 id="battery-live-title"><EnergyIcon name="battery"/>Domácí baterie</h2></div><div class="battery-body"><div><div class="metric">{{format(v('soc'))}} <small>%</small></div><div class="bar" role="img" :aria-label="v('soc')===null?'Stav nabití není dostupný':`Stav nabití ${format(v('soc'))} %`"><i :style="{width:percent(v('soc'))+'%'}"></i></div></div><dl><dt>Tok baterie</dt><dd>{{format(v('battery'),0)}} W · {{direction}}</dd><dt>Nejnižší článek</dt><dd>{{format(v('min_cell'),3)}} V</dd><template v-if="victron.data?.readings.max_cell?.quality!=='not_configured'"><dt>Nejvyšší článek</dt><dd>{{format(v('max_cell'),3)}} V</dd></template></dl></div></section>
  </div>
</template>
