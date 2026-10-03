<script setup lang="ts">
import {format,numeric} from './live'
import type {PowerTelemetry} from './live'
defineProps<{data?:PowerTelemetry;error:boolean}>()
const qualities:Record<string,string>={valid:'Hlášení HA',missing:'Entita chybí',invalid:'Neplatná hodnota nebo jednotka',stale:'Zastaralé hlášení',offline:'Spojení přerušeno'}
</script>
<template>
  <section class="card victron-card">
    <div class="cardhead"><h2>Výkon měničů · skutečná data</h2><span class="mode-pill neutral">{{error?'Backend nedostupný':data?.connected?'Čte z Home Assistantu':'HA nedostupný'}}</span></div>
    <strong>{{format(numeric(data?.readings.inverter,!error&&!!data?.enabled&&!!data?.connected),0)}} W</strong>
    <p class="muted">sensor.vystupni_vykon · {{qualities[data?.readings.inverter?.quality??'missing']}}</p>
    <p v-if="data?.readings.inverter?.source_at" class="muted">Hlášení HA {{new Date(data.readings.inverter.source_at).toLocaleString('cs-CZ')}}</p>
    <p class="muted victron-note">Pouze čtení · obnova každých 5 s. Limit stáří hlášení {{data?.fresh_seconds??60}} s; stáří spojení 20 s. Jednotky W/kW se převádějí na W. Bargrafy: výroba 8 kW, baterie a měniče 7 kW; překročení zaplní bargraf, číselná hodnota zůstává úplná. Články baterie jsou v diagnostice Victronu.</p>
  </section>
</template>
