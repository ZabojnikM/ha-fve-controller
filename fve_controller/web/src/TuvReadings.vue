<script setup lang="ts">
import {computed} from 'vue'
import {numeric,format} from './live'
import {tuvCharge} from './tuvCharge'
import {tuvStatus} from './tuvStatus'
import {pumpStatus} from './pumpStatus'
import type {TuvTelemetry,Reading} from './live'
const props=defineProps<{data?:TuvTelemetry;error:boolean}>()
const control=computed(()=>pumpStatus(props.data?.pump_control,!props.error&&!!props.data))
const charge=computed(()=>{
  const ready=!props.error&&!!props.data?.enabled&&!!props.data?.connected
  return tuvStatus(props.data,ready).sensorFault?null:tuvCharge(numeric(props.data?.readings.upper,ready),numeric(props.data?.readings.lower,ready))
})
const statuses:Record<string,string>={disabled:'Čtení vypnuté',connecting:'Připojuje se',listening:'Čte z Home Assistantu',no_token:'Chybí přístup k HA · restartujte doplněk',offline:'HA nedostupný',unauthorized:'HA odmítl přístup',error:'Data HA nelze načíst',stale:'Spojení zastaralo'}
const qualities:Record<string,string>={valid:'Stav z HA',missing:'Entita chybí',invalid:'Neplatný stav nebo jednotka',stale:'Zastaralé hlášení',offline:'Spojení přerušeno'}
function value(r?:Reading){return !props.error&&r?.quality==='valid'&&r.value!==null?r.value.toLocaleString('cs-CZ',{maximumFractionDigits:1}):'—'}
function quality(r?:Reading){return props.error?'Backend nedostupný':qualities[r?.quality??'missing']}
function pump(){const r=props.data?.readings.pump;return !props.error&&r?.quality==='valid'?(r.value===1?'Zapnuté':'Vypnuté'):'—'}
function binary(key:string,on:string,off:string){
  const r=props.data?.readings[key]
  return !props.error&&props.data?.connected&&r?.quality==='valid'?(r.value===1?on:off):'—'
}
function uptime(){
  const seconds=numeric(props.data?.readings.uptime,!props.error&&!!props.data?.connected)
  if(seconds===null)return '—'
  const minutes=Math.floor(seconds/60)
  return `${Math.floor(minutes/1440)} d ${Math.floor(minutes/60)%24} h ${minutes%60} min`
}
function system(){const r=props.data?.readings.system;return !props.error&&props.data?.connected&&r?.quality==='valid'?r.text:'—'}
function onoff(v:boolean|null|undefined){return v===true?'Zapnuto':v===false?'Vypnuto':'—'}
</script>
<template>
  <section class="card victron-card" aria-labelledby="tuv-live-title">
    <div class="cardhead"><h2 id="tuv-live-title">Teplá voda · skutečná data</h2><span class="mode-pill neutral">{{error?'Backend nedostupný':statuses[data?.status??'connecting']}}</span></div>
    <p v-if="error" class="quality-warning" role="alert">Aktuální data nelze načíst.</p>
    <p v-else-if="data&&!data.enabled" class="muted">Čtení TUV zapněte v konfiguraci doplňku.</p>
    <div v-else-if="data" class="victron-readings">
      <div v-for="[key,label] in [['upper','Horní teplota'],['lower','Spodní teplota']]" :key="key" class="victron-reading">
        <h3>{{label}}</h3><strong>{{value(data.readings[key])}} <small>°C</small></strong>
        <p :class="data.readings[key]?.quality==='valid'&&!error?'muted':'quality-warning'">{{quality(data.readings[key])}}</p>
        <p v-if="data.readings[key]?.source_at" class="muted">Hlášení HA {{new Date(data.readings[key].source_at!).toLocaleString('cs-CZ')}}</p>
      </div>
      <div class="victron-reading"><h3>Jmenovitý výkon stupně</h3><strong>{{value(data.nominal_power)}} <small>W</small></strong><p :class="data.nominal_power.quality==='valid'&&!error?'muted':'quality-warning'">{{quality(data.nominal_power)}}</p><p class="muted">Podle selectu · příkon se neměří</p></div>
      <div class="victron-reading"><h3>Čerpadlo</h3><strong>{{pump()}}</strong><p :class="data.readings.pump?.quality==='valid'&&!error?'muted':'quality-warning'">{{quality(data.readings.pump)}}</p></div>
      <div class="victron-reading tuv-system-reading"><h3>Stav firmware</h3><strong>{{system()}}</strong><p :class="data.readings.system?.quality==='valid'&&!error?'muted':'quality-warning'">{{quality(data.readings.system)}}</p><p class="muted">Aktivní = ohřev povolen, nikoli zapnutý. Text ukazuje jen první blokaci.</p></div>
      <div class="victron-reading"><h3>Přetížení X16</h3><strong>{{binary('overload','Aktivní','Neaktivní')}}</strong><p :class="data.readings.overload?.quality==='valid'&&!error?'muted':'quality-warning'">{{quality(data.readings.overload)}}</p></div>
      <div class="victron-reading"><h3>Napájení čidel · Y15</h3><strong>{{binary('sensor_reset','Odpojené','Zapnuté')}}</strong><p :class="data.readings.sensor_reset?.quality==='valid'&&!error?'muted':'quality-warning'">{{quality(data.readings.sensor_reset)}}</p><p class="muted">Y15 ON odpojuje napájení. Nový firmware obnovy je nutné ověřit na zařízení.</p></div>
      <div class="victron-reading"><h3>Doba běhu Kicony</h3><strong>{{uptime()}}</strong><p :class="data.readings.uptime?.quality==='valid'&&!error?'muted':'quality-warning'">{{quality(data.readings.uptime)}}</p><p class="muted">Pokles může znamenat restart; příčinu neurčuje.</p></div>
    </div>
    <div class="pump-diagnostics" v-if="data?.pump_control" aria-label="Automatika čerpadla">
      <h3>Řízení čerpadla</h3><p :class="control.problem?'quality-warning':'muted'"><strong>{{control.label}}</strong> · {{control.reason}}</p>
      <dl><dt>Správce výstupu</dt><dd>{{data.pump_control.enabled?'Doplněk':'Řízení doplňkem nepřevzato'}}</dd><dt>Požadovaný stav</dt><dd>{{error?'—':onoff(data.pump_control.desired)}}</dd><dt>Poslední přijatý povel</dt><dd>{{error?'—':onoff(data.pump_control.sent)}}</dd><dt>Stav hlášený HA</dt><dd>{{error?'—':onoff(data.pump_control.confirmed)}}</dd><dt>Teplotní promíchávání</dt><dd>{{error?'—':data.pump_control.process_request?'Požadováno':'Nepožadováno'}}</dd><dt>Servisní protočení</dt><dd>{{error?'—':data.pump_control.service_request?'Požadováno':'Nepožadováno'}}</dd><dt>Poslední rezervovaný servisní den</dt><dd>{{error?'—':data.pump_control.last_service_day||'Dosud žádný'}}</dd></dl>
      <p class="muted">Zapnout horní &gt; 58 °C a spodní &lt; 57 °C; vypnout horní &lt; 57 °C nebo spodní ≥ 57 °C. Denně v 18:30 Europe/Prague protočení 30 s od hlášeného zapnutí, sloučené s promícháváním. Limit stáří teplot pro řízení {{data.pump_control.temperature_fresh_seconds}} s. Při nedostupnosti nelze potvrdit vypnutí. Servisní den je rezervace, nikoli doklad dokončeného běhu.</p>
    </div>
    <p class="muted victron-note">Čtení TUV · obnova každých 5 s. Select hlásí nastavení výstupů firmwarem; nepotvrzuje fyzické sepnutí spirál. X16 a Y15 se čtou nezávisle na textu systému. Stav firmware má limit hlášení 60 s, doba běhu 180 s. Teploty mají limit stáří {{data?.temperature_fresh_seconds??900}} s od hlášení do HA. Živá data zatím nevstupují do simulačních doporučení.</p>
    <p class="muted victron-note"><strong>Odhad nabití TUV: {{format(charge)}} %.</strong> Nádrž 300 l, výška 170 cm, čidla 40/130 cm. Počítá se 170 vrstev po 1 cm; mimo čidla je teplota konstantní, mezi nimi lineární. Nabití každé vrstvy je omezené na 0–100 % v rozsahu 45–57 °C. Při horní teplotě ≤ 45 °C je výsledek 0 %. Výsledek je průměr vrstev, zaokrouhlený na 0,1 %. Při neplatné či zastaralé teplotě, obnově nebo hlášené poruše čidel je nedostupný. Horní teplota pochází z jiného zařízení; TUV2 se nečte. Jde o orientační zásobu tepla, nikoli procento litrů vody na sprchování; výpočet neovládá ohřev.</p>
  </section>
</template>
