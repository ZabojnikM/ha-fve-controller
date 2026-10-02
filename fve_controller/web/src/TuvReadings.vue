<script setup lang="ts">
import {ref,onMounted,onUnmounted} from 'vue'
type Reading={value:number|null;unit:string;quality:string;source_at:string|null}
type Telemetry={enabled:boolean;connected:boolean;status:string;received_at:string|null;temperature_fresh_seconds:number;readings:Record<string,Reading>;nominal_power:Reading}
const data=ref<Telemetry>(),error=ref(false)
const statuses:Record<string,string>={disabled:'Čtení vypnuté',connecting:'Připojuje se',listening:'Čte z Home Assistantu',no_token:'Chybí přístup k HA · restartujte doplněk',offline:'HA nedostupný',unauthorized:'HA odmítl přístup',error:'Data HA nelze načíst',stale:'Spojení zastaralo'}
const qualities:Record<string,string>={valid:'Stav z HA',missing:'Entita chybí',invalid:'Neplatný stav nebo jednotka',stale:'Zastaralá teplota',offline:'Spojení přerušeno',conflict:'Stupeň neurčený · musí být zapnutý právě jeden přepínač'}
function value(r?:Reading){return !error.value&&r?.quality==='valid'&&r.value!==null?r.value.toLocaleString('cs-CZ',{maximumFractionDigits:1}):'—'}
function quality(r?:Reading){return error.value?'Backend nedostupný':qualities[r?.quality??'missing']}
function pump(){const r=data.value?.readings.pump;return !error.value&&r?.quality==='valid'?(r.value===1?'Zapnuté':'Vypnuté'):'—'}
let pending=false
async function refresh(){if(pending)return;pending=true;try{const response=await fetch('api/tuv',{signal:AbortSignal.timeout(5000)});if(!response.ok)throw Error();data.value=await response.json();error.value=false}catch{error.value=true}finally{pending=false}}
let timer:ReturnType<typeof setInterval>
onMounted(()=>{refresh();timer=setInterval(refresh,2000)})
onUnmounted(()=>clearInterval(timer))
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
      <div class="victron-reading"><h3>Jmenovitý výkon stupně</h3><strong>{{value(data.nominal_power)}} <small>W</small></strong><p :class="data.nominal_power.quality==='valid'&&!error?'muted':'quality-warning'">{{quality(data.nominal_power)}}</p><p class="muted">Podle přepínačů · příkon se neměří</p></div>
      <div class="victron-reading"><h3>Čerpadlo</h3><strong>{{pump()}}</strong><p :class="data.readings.pump?.quality==='valid'&&!error?'muted':'quality-warning'">{{quality(data.readings.pump)}}</p></div>
    </div>
    <p class="muted victron-note">Pouze čtení · obnova každých 5 s. Stavy přepínačů jsou hlášení HA; nepotvrzují fyzické sepnutí spirál. Teploty mají limit stáří {{data?.temperature_fresh_seconds??900}} s od hlášení do HA. Živá data zatím nevstupují do simulačních doporučení.</p>
  </section>
</template>
