<script setup lang="ts">
import {ref} from 'vue'
import {version} from '../package.json'
import {useLiveTelemetry} from './live'
import LiveOverview from './LiveOverview.vue'
import VictronReadings from './VictronReadings.vue'
import TuvReadings from './TuvReadings.vue'
import TeslaReadings from './TeslaReadings.vue'
import SimulationPanel from './SimulationPanel.vue'
const {victron,tuv,tesla}=useLiveTelemetry()
const simulationOpen=ref(false)
</script>
<template>
<main>
  <header><a class="brand" href="./"><img class="brand-icon" :src="'./app-icon.png'" width="36" height="36" alt=""> FVE Controller</a><span class="badge"><i></i> Sledovací režim</span></header>
  <div class="intro"><div><h1>Energie doma</h1><p class="muted">Slunce, baterie a vaše spotřebiče v jednom přehledu.</p></div><span class="mode-pill neutral">Živá data</span></div>
  <LiveOverview :victron="victron" :tuv="tuv" :tesla="tesla"/>
  <details class="card diagnostics live-diagnostics"><summary>Zdroje dat a diagnostika</summary><p class="muted">Podrobnosti příjmu, platnosti a časů jednotlivých údajů.</p><VictronReadings :data="victron.data" :error="victron.error"/><TuvReadings :data="tuv.data" :error="tuv.error"/><TeslaReadings :data="tesla.data" :error="tesla.error"/></details>
  <details class="simulation-details" @toggle="simulationOpen=($event.target as HTMLDetailsElement).open"><summary>Simulátor a modelová doporučení</summary><SimulationPanel v-if="simulationOpen"/></details>
  <footer>FVE Controller {{version}}<span>Pouze čtení · řízení vypnuté</span></footer>
</main>
</template>
