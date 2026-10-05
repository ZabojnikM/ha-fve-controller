<script setup lang="ts">
import {version} from '../package.json'
import {useLiveTelemetry} from './live'
import LiveOverview from './LiveOverview.vue'
import VictronReadings from './VictronReadings.vue'
import TuvReadings from './TuvReadings.vue'
import TeslaReadings from './TeslaReadings.vue'
const {victron,tuv,tesla,refresh}=useLiveTelemetry()
</script>
<template>
<main>
  <header><a class="brand" href="./"><img class="brand-icon" :src="'./app-icon.png'" width="36" height="36" alt=""> FVE Controller</a><span class="badge"><i></i> {{tuv.error?'Režim není dostupný':tuv.data?.tuv_control?.pump_owned?(tuv.data?.tuv_control?.mode==='manual'?'TUV · Ruční ovládání':'TUV · Automatika'):'Sledovací režim'}}</span></header>
  <LiveOverview :victron="victron" :tuv="tuv" :tesla="tesla" @changed="refresh"/>
  <details class="card diagnostics live-diagnostics"><summary>Diagnostika</summary><p>FVE Controller {{version}} · {{tuv.data?.tuv_control?.pump_owned?(tuv.data?.tuv_control?.mode==='manual'?'ruční ovládání předaných výstupů TUV':'automatické promíchávání TUV'):'pouze čtení · řízení doplňkem vypnuté'}}.</p><p class="muted">Podrobnosti příjmu, platnosti a časů jednotlivých údajů. Neplatné nebo nedostupné hodnoty se zobrazují jako pomlčka; u Victron MQTT se kontroluje také stáří příjmu.</p><VictronReadings :data="victron.data" :error="victron.error"/><TuvReadings :data="tuv.data" :error="tuv.error"/><TeslaReadings :data="tesla.data" :error="tesla.error"/></details>
</main>
</template>
