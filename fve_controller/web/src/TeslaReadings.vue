<script setup lang="ts">
import type {TeslaTelemetry,Reading} from './live'
const props=defineProps<{data?:TeslaTelemetry;error:boolean}>()
const labels=[['soc','Stav nabití'],['connected','Nabíjecí kabel'],['charging','Stav nabíjení'],['power','Výkon nabíječky'],['current','Skutečný proud'],['current_limit','Nastavený proud']]
const statuses:Record<string,string>={disabled:'Čtení vypnuté',connecting:'Připojuje se',listening:'Čte z Home Assistantu',no_token:'Chybí přístup k HA · restartujte doplněk',offline:'HA nedostupný',unauthorized:'HA odmítl přístup',error:'Data HA nelze načíst',stale:'Spojení zastaralo'}
const qualities:Record<string,string>={valid:'Hlášení HA',missing:'Entita chybí',not_configured:'Entita není nastavená',invalid:'Neplatný stav nebo jednotka',stale:'Zastaralé hlášení',offline:'Spojení přerušeno'}
const charging:Record<string,string>={charging:'Nabíjí se',complete:'Dokončeno',disconnected:'Odpojeno',stopped:'Zastaveno',starting:'Spouští se',no_power:'Bez napájení'}
function value(key:string){const r=props.data?.readings[key];if(props.error||r?.quality!=='valid')return '—';if(key==='charging')return charging[r.text??'']??'—';if(key==='connected')return r.value===1?'Připojený':'Odpojený';return r.value===null?'—':r.value.toLocaleString('cs-CZ',{maximumFractionDigits:1})}
</script>
<template>
  <section class="card victron-card" aria-labelledby="tesla-live-title">
    <div class="cardhead"><h2 id="tesla-live-title">Tesla · skutečná data</h2><span class="mode-pill neutral">{{error?'Backend nedostupný':statuses[data?.status??'connecting']}}</span></div>
    <p v-if="error" class="quality-warning" role="alert">Aktuální data nelze načíst.</p>
    <p v-else-if="data&&!data.enabled" class="muted">Čtení Tesly zapněte v konfiguraci doplňku společně s přístupem k HA.</p>
    <div v-else-if="data" class="victron-readings tesla-readings">
      <div v-for="[key,label] in labels" :key="key" class="victron-reading">
        <h3>{{label}}</h3><strong>{{value(key)}} <small>{{data.readings[key]?.unit}}</small></strong>
        <p :class="data.readings[key]?.quality==='valid'&&!error?'muted':'quality-warning'">{{qualities[data.readings[key]?.quality??'missing']}}</p>
        <p v-if="key==='current_limit'" class="muted">Nastavení v HA · není skutečný proud</p>
        <p v-if="data.readings[key]?.source_at" class="muted">Hlášení HA {{new Date(data.readings[key].source_at!).toLocaleString('cs-CZ')}}</p>
      </div>
    </div>
    <p class="muted victron-note">Pouze čtení · auto neprobouzíme. Data z HA mohou pocházet z mezipaměti Tessie; limit stáří hlášení je {{data?.fresh_seconds??900}} s. Živá data zatím nevstupují do simulačních doporučení.</p>
  </section>
</template>
