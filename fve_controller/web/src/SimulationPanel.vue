<script setup lang="ts">
import {ref,onMounted,onUnmounted} from 'vue'
import EnergyIcon from './EnergyIcon.vue'
type Sample={value:number;unit:string;at:string;valid:boolean}
type State={at:string;scenario:string;input:Record<string,Sample>;quality:Record<string,string>;decision:{mode:string;reason:string;target:number;tuv:number;car:number;balance:string;last_balance:string;next_balance:string}}
const state=ref<State>(),error=ref(''),busy=ref(false)
const history=ref<{battery:number;at:number}[]>([])
const scenarios=[['sunny','Slunečný den'],['zero','Omezené MPPT'],['night','Noc · bez výroby'],['balance','Balancování'],['hot','Horká TUV'],['critical','Kritické SOC'],['stale','Zastaralá data'],['invalid','Neplatný vstup'],['overload','Přetížení'],['manual','Ruční TUV']]
const solarStrings=[['solar_roof','Střecha'],['solar_shelter','Přístřešek'],['solar_fence','Plot']]
function solarQuality(k:string){if(error.value)return 'Spojení přerušeno';const q=state.value?.quality[k];return q==='valid'?'Aktuální vzorek':q==='stale'?'Zastaralé měření':q==='invalid'?'Neplatné měření':'Měření není dostupné'}
function solarValue(k:string){return error.value?'—':value(k)}
async function refresh(){try{const r=await fetch('api/state');if(!r.ok)throw Error();state.value=await r.json();const h=await fetch('api/history');if(!h.ok)throw Error();history.value=(await h.json()).reverse();error.value=''}catch{error.value='Spojení s backendem přerušeno. Zobrazená data mohou být zastaralá.'}}
async function scenario(event:Event){busy.value=true;try{const r=await fetch('api/scenario',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:(event.target as HTMLSelectElement).value})});if(!r.ok)throw Error();await refresh()}catch{error.value='Scénář se nepodařilo změnit.'}finally{busy.value=false}}
function value(k:string){const s=state.value;if(error.value || !s || s.quality[k]!=='valid')return '—';return s.input[k].value.toLocaleString('cs-CZ')}
function power(k:string){const s=state.value;if(error.value || !s || s.quality[k]!=='valid')return '—';return kw(s.input[k].value)}
function kw(watts:number){return (watts/1000).toLocaleString('cs-CZ',{minimumFractionDigits:1,maximumFractionDigits:2})}
function share(k:string){const s=state.value;if(error.value || !s || s.quality.solar_total!=='valid' || s.quality[k]!=='valid' || s.input.solar_total.value<=0)return 0;return s.input[k].value/s.input.solar_total.value*100}
function batteryDirection(){const s=state.value;if(error.value || !s || s.quality.battery!=='valid')return solarQuality('battery');return s.input.battery.value>0?'Nabíjí se':s.input.battery.value<0?'Vybíjí se':'Bez toku'}
function date(s:string){return s.startsWith('0001')?'Dosud nepotvrzeno':new Date(s).toLocaleString('cs-CZ')}
let timer:ReturnType<typeof setInterval>;onMounted(()=>{refresh();timer=setInterval(refresh,2000)});onUnmounted(()=>clearInterval(timer))
</script>
<template>
<div class="simulation-panel">
  <label>Simulační scénář<select :value="state?.scenario" :disabled="busy || !state" @change="scenario"><option v-for="s in scenarios" :key="s[0]" :value="s[0]">{{s[1]}}</option></select></label>
  <div class="notice"><strong>Simulace</strong><span>Následující data jsou modelová. Doporučení se neodesílají do zařízení.</span></div>
  <p v-if="error" role="alert" class="error">{{error}}</p>
  <template v-if="state">
    <section class="energy-overview" aria-labelledby="solar-title">
      <div class="energy-heading">
        <div class="solar-total"><div class="eyebrow"><EnergyIcon name="sun"/><h2 id="solar-title">Solární výroba</h2></div><div class="metric">{{solarValue('solar_total')}} <small>W</small></div></div>
        <div class="battery-glance"><span class="muted"><EnergyIcon name="battery"/> Baterie</span><strong>{{value('soc')}} <small>%</small></strong><span class="muted">{{batteryDirection()}} · {{power('battery')}} kW</span></div>
      </div>
      <div class="production-bar" role="img" :aria-label="`Podíl stringů na výrobě: Střecha ${solarValue('solar_roof')} W, Přístřešek ${solarValue('solar_shelter')} W, Plot ${solarValue('solar_fence')} W`"><span v-for="[key] in solarStrings" :key="key" :class="key" :style="{width:share(key)+'%'}"></span></div>
      <div class="solar-readings">
        <div v-for="[key,label] in solarStrings" :key="key" class="solar-string"><h3><i :class="key"></i>{{label}}</h3><strong>{{solarValue(key)}} <small>W</small></strong><p :class="!error && state.quality[key]==='valid'?'muted':'quality-warning'">{{solarQuality(key)}}</p></div>
      </div>
      <p class="quality-warning" v-if="error || state.quality.solar_total!=='valid'">{{solarQuality('solar_total')}} · součet není dostupný</p>
    </section>
    <section class="summary" :class="{blocked:state.decision.mode==='BLOKACE' || error}">
      <div><p class="eyebrow">Doporučený režim</p><h2>{{error?'Čeká na spojení':state.decision.mode}}</h2><p>{{error?'Aktuální doporučení není dostupné.':state.decision.reason}}</p></div>
      <div class="target"><span>Cílový tok baterie</span><strong>{{error?'—':state.decision.target.toLocaleString('cs-CZ')}} <small>W</small></strong><span>+ nabíjení / − vybíjení</span></div>
    </section>
    <div class="loads">
      <section class="card load-card water">
        <div class="cardhead"><h2><EnergyIcon name="water"/>Teplá voda</h2><span class="mode-pill">{{state.scenario==='manual'?'Ruční požadavek':'Auto'}}</span></div>
        <div class="load-metrics"><div><p class="eyebrow">Simulovaný výkon</p><div class="metric">{{power('tuv')}} <small>kW</small></div></div><div><p class="eyebrow">Teplota</p><div class="metric">{{value('temperature')}} <small>°C</small></div></div></div>
        <div class="device-detail"><div class="device-title"><strong>Bojler</strong><span class="muted">3 spirály × 1 kW</span></div><div class="stages" aria-label="Doporučené stupně ohřevu"><span v-for="stage in 3" :key="stage" :class="{active:!error && state.decision.tuv>=stage*1000}">{{stage}} kW</span></div><p class="muted">Stupně ukazují doporučení simulátoru.</p><dl><dt>Doporučený výkon</dt><dd>{{error?'—':kw(state.decision.tuv)}} kW</dd><dt>Odesláno zařízení</dt><dd>Nic · pouze sledování</dd></dl></div>
      </section>
      <section class="card load-card car">
        <div class="cardhead"><h2><EnergyIcon name="car"/>Tesla</h2><span class="mode-pill neutral">Simulace</span></div>
        <div class="load-metrics"><div><p class="eyebrow">Simulovaný výkon</p><div class="metric">{{power('car')}} <small>kW</small></div></div><div><p class="eyebrow">Doporučený proud</p><div class="metric">{{error?'—':(state.decision.car/230).toLocaleString('cs-CZ',{maximumFractionDigits:1})}} <small>A</small></div></div></div>
        <div class="device-detail"><div class="device-title"><strong>Nabíjení z přebytků</strong><span class="muted">Minimum 5 A</span></div><p class="device-message">{{error?'Čeká na spojení':state.decision.car>0?'Simulátor doporučuje nabíjení.':'Simulátor nyní nabíjení nedoporučuje.'}}</p><p class="muted">Společný výkonový rozpočet · přednost TUV</p><dl><dt>Doporučený výkon</dt><dd>{{error?'—':kw(state.decision.car)}} kW</dd><dt>Odesláno zařízení</dt><dd>Nic · pouze sledování</dd></dl></div>
      </section>
    </div>
    <div class="lower">
      <section class="card battery-card"><div class="cardhead"><h2><EnergyIcon name="battery"/>Domácí baterie</h2><span class="muted">{{batteryDirection()}}</span></div><div class="metric">{{value('soc')}} <small>%</small></div><div class="bar" role="img" :aria-label="`Stav nabití baterie ${value('soc')} %`"><i :style="{width:(!error && state.quality.soc==='valid'?Math.min(100,Math.max(0,state.input.soc.value)):0)+'%'}"></i></div><dl><dt>Simulovaný tok</dt><dd>{{value('battery')}} W</dd><dt>Nejnižší článek</dt><dd>{{value('min_cell')}} V</dd><dt>Nejvyšší článek</dt><dd>{{value('max_cell')}} V</dd></dl>
      </section>
      <section class="card"><div class="cardhead"><h2>Balancování</h2><span class="muted">Každých 14 dní</span></div><p class="balance">{{error?'Aktuální stav není dostupný':state.decision.balance}}</p><dl><dt>Poslední dokončení (simulace)</dt><dd>{{date(state.decision.last_balance)}}</dd><dt>Další termín</dt><dd>{{state.decision.last_balance.startsWith('0001')?'Nyní · chybí předchozí záznam':date(state.decision.next_balance)}}</dd></dl><p class="muted balance-rule">SOC ≥ 95 % · min. článek ≥ 3,439 V po 5 minut</p></section>
    </div>
    <section class="card history-card"><div class="cardhead"><h2>Tok baterie</h2><span class="muted">Posledních 120 vzorků · W</span></div><svg viewBox="0 0 500 100" role="img" aria-label="Historie simulovaného toku baterie, kladný tok znamená nabíjení"><path d="M0 85H500" stroke="currentColor" class="chart-axis"/><polyline :points="history.map((h,i)=>`${i*500/Math.max(1,history.length-1)},${85-Math.min(4000,Math.max(-500,h.battery))*0.018}`).join(' ')" fill="none" stroke="currentColor" stroke-width="3"/></svg><p class="muted">Simulovaná historie · uchování 7 dní · + nabíjení / − vybíjení</p></section>
    <details class="card diagnostics"><summary>Diagnostika a plánování</summary><p>observe_only = true · control_enabled = false · vlastník fyzických výstupů: žádný</p><p>NT/VT a predikční plánování nejsou připojené. Limity tohoto modelu nejsou limity vaší instalace.</p><dl><template v-for="(q,k) in state.quality" :key="k"><dt>{{k}}</dt><dd>{{q}}</dd></template></dl></details>
  </template>
  <p v-else role="status">Načítám simulaci…</p>
</div>
</template>
