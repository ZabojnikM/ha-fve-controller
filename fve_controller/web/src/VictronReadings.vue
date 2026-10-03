<script setup lang="ts">
import {computed} from 'vue'
import type {VictronTelemetry} from './live'
const props=defineProps<{data?:VictronTelemetry;error:boolean}>()
const labels=[['soc','Stav nabití'],['battery','Tok baterie'],['min_cell','Nejnižší článek'],['max_cell','Nejvyšší článek'],['solar_roof','Střecha'],['solar_shelter','Přístřešek'],['solar_fence','Plot'],['inverter','Výkon měničů']]
const visibleLabels=computed(()=>labels.filter(([key])=>key!=='max_cell'||props.data?.readings[key]?.quality!=='not_configured'))
const qualities:Record<string,string>={valid:'Přijatá data',invalid:'Neplatné měření',missing:'Čeká na první zprávu',not_configured:'Topic není nastaven',retained:'Uložená zpráva · stáří neověřeno',stale:'Zastaralá data',offline:'Spojení přerušeno'}
const statuses:Record<string,string>={disabled:'Nepřipojeno',connecting:'Připojuje se',subscribing:'Přihlašuje odběr',listening:'Čte z MQTT',offline:'Spojení přerušeno',subscription_error:'Odběr se nepodařil · zkontrolujte oprávnění a restartujte doplněk'}
function value(key:string){const r=props.data?.readings[key];return !props.error && r?.quality==='valid' && r.value!==null?r.value.toLocaleString('cs-CZ',{maximumFractionDigits:3}):'—'}
</script>
<template>
  <section class="card victron-card" aria-labelledby="victron-title">
    <div class="cardhead"><h2 id="victron-title">Victron · skutečná data</h2><span class="mode-pill neutral">{{error?'Backend nedostupný':statuses[data?.status ?? 'connecting']}}</span></div>
    <p v-if="error" class="quality-warning" role="alert">Aktuální data nelze načíst.</p>
    <p v-else-if="!data?.enabled" class="muted">Pro připojení zapněte MQTT v konfiguraci doplňku a nastavte topics. </p>
    <template v-else>
      <div class="victron-readings">
        <div v-for="[key,label] in visibleLabels" :key="key" class="victron-reading">
          <h3>{{label}}</h3><strong>{{value(key)}} <small>{{data.readings[key]?.unit}}</small></strong>
          <p :class="!error && data.readings[key]?.quality==='valid'?'muted':'quality-warning'">{{error?'Backend nedostupný':qualities[data.readings[key]?.quality ?? 'missing']}}</p>
          <p v-if="data.readings[key]?.at" class="muted">Příjem {{new Date(data.readings[key].at!).toLocaleTimeString('cs-CZ')}}</p>
        </div>
      </div>
      <p class="muted victron-note">Pouze čtení · limit stáří od příjmu {{data.fresh_seconds}} s. + nabíjení / − vybíjení. Výkon měničů: AC odběr na výstupu L1; bargraf do 7 kW. Tato data zatím nevstupují do simulačních doporučení.</p>
    </template>
  </section>
</template>
