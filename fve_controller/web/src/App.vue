<script setup lang="ts">
import {version} from '../package.json'
import {useLiveTelemetry} from './live'
import LiveOverview from './LiveOverview.vue'
import VictronReadings from './VictronReadings.vue'
import TuvReadings from './TuvReadings.vue'
import TeslaReadings from './TeslaReadings.vue'
import PowerReadings from './PowerReadings.vue'
const {victron,tuv,tesla,power}=useLiveTelemetry()
</script>
<template>
<main>
  <header><a class="brand" href="./"><img class="brand-icon" :src="'./app-icon.png'" width="36" height="36" alt=""> FVE Controller</a><span class="badge"><i></i> Sledovací režim</span></header>
  <LiveOverview :victron="victron" :tuv="tuv" :tesla="tesla" :power="power"/>
  <details class="card diagnostics live-diagnostics"><summary>Diagnostika</summary><p>FVE Controller {{version}} · pouze čtení · řízení doplňkem vypnuté.</p><p class="muted">Podrobnosti příjmu, platnosti a časů jednotlivých údajů. Zastaralé nebo neplatné hodnoty se v hlavním přehledu zobrazují jako pomlčka.</p><VictronReadings :data="victron.data" :error="victron.error"/><TuvReadings :data="tuv.data" :error="tuv.error"/><PowerReadings :data="power.data" :error="power.error"/><TeslaReadings :data="tesla.data" :error="tesla.error"/></details>
</main>
</template>
