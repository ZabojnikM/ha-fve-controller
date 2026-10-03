<script setup lang="ts">
import {computed} from 'vue'
import {numeric,format} from './live'
import {tuvCharge} from './tuvCharge'
import type {TuvTelemetry,Reading} from './live'
const props=defineProps<{data?:TuvTelemetry;error:boolean}>()
const charge=computed(()=>{
  const ready=!props.error&&!!props.data?.enabled&&!!props.data?.connected
  return tuvCharge(numeric(props.data?.readings.upper,ready),numeric(props.data?.readings.lower,ready))
})
const statuses:Record<string,string>={disabled:'Čtení vypnuté',connecting:'Připojuje se',listening:'Čte z Home Assistantu',no_token:'Chybí přístup k HA · restartujte doplněk',offline:'HA nedostupný',unauthorized:'HA odmítl přístup',error:'Data HA nelze načíst',stale:'Spojení zastaralo'}
const qualities:Record<string,string>={valid:'Stav z HA',missing:'Entita chybí',invalid:'Neplatný stav nebo jednotka',stale:'Zastaralá teplota',offline:'Spojení přerušeno',conflict:'Stupeň neurčený · musí být zapnutý právě jeden přepínač'}
function value(r?:Reading){return !props.error&&r?.quality==='valid'&&r.value!==null?r.value.toLocaleString('cs-CZ',{maximumFractionDigits:1}):'—'}
function quality(r?:Reading){return props.error?'Backend nedostupný':qualities[r?.quality??'missing']}
function pump(){const r=props.data?.readings.pump;return !props.error&&r?.quality==='valid'?(r.value===1?'Zapnuté':'Vypnuté'):'—'}
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
    <p class="muted victron-note"><strong>Odhad nabití TUV: {{format(charge)}} %.</strong> Nádrž 300 l, výška 170 cm, čidla 40/130 cm. Počítá se 170 vrstev po 1 cm; mimo čidla je teplota konstantní, mezi nimi lineární. Nabití každé vrstvy je omezené na 0–100 % v rozsahu 45–57 °C. Při horní teplotě ≤ 45 °C je výsledek 0 %. Výsledek je průměr vrstev, zaokrouhlený na 0,1 %. Při neplatné nebo zastaralé teplotě je nedostupný. Jde o orientační zásobu tepla, nikoli procento litrů vody na sprchování; výpočet neovládá ohřev.</p>
  </section>
</template>
