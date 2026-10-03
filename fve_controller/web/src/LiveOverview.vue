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
const inverter=computed(()=>v('inverter'))
function load(value:number|null,max:number){return value===null?0:Math.min(100,Math.max(0,Math.abs(value)/max*100))}
function share(key:string){return total.value!==null?Math.max(0,(v(key)??0)/Math.max(8000,total.value)*100):0}
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
    <section class="power-overview" aria-label="Výroba, baterie a měniče">
      <div class="power-tile solar-total">
        <div class="eyebrow"><EnergyIcon name="sun"/><h2>Solární výroba</h2></div>
        <div class="metric" data-testid="solar-total">{{format(total,0)}} <small>W</small></div>
        <div class="production-bar power-bar" role="meter" aria-label="Solární výroba" :aria-valuenow="total===null?undefined:Math.min(8000,total)" aria-valuemin="0" aria-valuemax="8000" :aria-valuetext="total===null?'Nedostupné':`${format(total,0)} W`"><span v-for="[key] in strings" :key="key" :class="key" :style="{width:share(key)+'%'}"></span></div>
        <div class="bar-scale"><span>0</span><span>8 kW</span></div>
        <div class="solar-readings"><div v-for="[key,label] in strings" :key="key" class="solar-string"><h3><i :class="key"></i>{{label}}</h3><strong>{{format(v(key),0)}} <small>W</small></strong></div></div>
      </div>
      <div class="power-tile battery-tile" :class="v('battery')===null?'unknown':(v('battery')??0)<0?'discharging':'charging'">
        <div class="eyebrow"><EnergyIcon name="battery"/><h2>Baterie</h2></div>
        <div class="battery-soc metric">{{format(v('soc'))}} <small>%</small></div>
        <div class="soc-bar" role="meter" aria-label="Stav nabití baterie" :aria-valuenow="v('soc')??undefined" aria-valuemin="0" aria-valuemax="100" :aria-valuetext="v('soc')===null?'Nedostupné':`${format(v('soc'))} %`"><i :style="{width:percent(v('soc'))+'%'}"></i></div>
        <div class="battery-flow"><strong>{{direction}}</strong><span data-testid="battery-power">{{kw(v('battery'))}} <small>kW</small></span></div>
        <div class="power-bar" role="meter" aria-label="Velikost toku baterie" :aria-valuenow="v('battery')===null?undefined:Math.min(7000,Math.abs(v('battery')!))" aria-valuemin="0" aria-valuemax="7000" :aria-valuetext="`${direction} ${kw(v('battery'))} kW`"><i :style="{width:load(v('battery'),7000)+'%'}"></i></div>
        <div class="bar-scale"><span>0</span><span>7 kW</span></div>
      </div>
      <div class="power-tile inverter-tile">
        <div class="eyebrow"><h2>Výkon měničů</h2></div>
        <div class="metric" data-testid="inverter-power">{{kw(inverter)}} <small>kW</small></div>
        <div class="power-bar" role="meter" aria-label="Výkon měničů" :aria-valuenow="inverter===null?undefined:Math.min(7000,inverter)" aria-valuemin="0" aria-valuemax="7000" :aria-valuetext="inverter===null?'Nedostupné':`${kw(inverter)} kW`"><i :style="{width:load(inverter,7000)+'%'}"></i></div>
        <div class="bar-scale"><span>0</span><span>7 kW</span></div>
      </div>
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
  </div>
</template>
