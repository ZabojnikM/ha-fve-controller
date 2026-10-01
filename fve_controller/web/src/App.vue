<script setup lang="ts">
import {ref,onMounted,onUnmounted} from 'vue'
import {version} from '../package.json'
type Sample={value:number;unit:string;at:string;valid:boolean}
type State={at:string;scenario:string;input:Record<string,Sample>;quality:Record<string,string>;decision:{mode:string;reason:string;target:number;tuv:number;car:number;balance:string;last_balance:string;next_balance:string}}
const state=ref<State>(),error=ref(''),busy=ref(false)
const history=ref<{battery:number;at:number}[]>([])
const scenarios=[['sunny','Slunečný den'],['zero','Omezené MPPT'],['balance','Balancování'],['hot','Horká TUV'],['critical','Kritické SOC'],['stale','Zastaralá data'],['invalid','Neplatný vstup'],['overload','Přetížení'],['manual','Ruční TUV']]
async function refresh(){try{const r=await fetch('api/state');if(!r.ok)throw Error();state.value=await r.json();const h=await fetch('api/history');if(!h.ok)throw Error();history.value=(await h.json()).reverse();error.value=''}catch{error.value='Spojení s backendem přerušeno. Zobrazená data mohou být zastaralá.'}}
async function scenario(event:Event){busy.value=true;try{const r=await fetch('api/scenario',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:(event.target as HTMLSelectElement).value})});if(!r.ok)throw Error();await refresh()}catch{error.value='Scénář se nepodařilo změnit.'}finally{busy.value=false}}
function value(k:string){const s=state.value;if(!s || s.quality[k]!=='valid')return '—';return s.input[k].value.toLocaleString('cs-CZ')}
function date(s:string){return s.startsWith('0001')?'Dosud nepotvrzeno':new Date(s).toLocaleString('cs-CZ')}
let timer:ReturnType<typeof setInterval>;onMounted(()=>{refresh();timer=setInterval(refresh,2000)});onUnmounted(()=>clearInterval(timer))
</script>
<template>
<main>
 <header><a class="brand" href="./"><span class="sun">☀</span> FVE <span class="muted">/ Controller</span></a><span class="badge">● SLEDOVACÍ REŽIM</span></header>
 <div class="intro"><div><p class="eyebrow">ENERGIE POD DOHLEDEM</p><h1>Domácí energetika</h1><p class="muted">Jeden přehled pro baterii, teplou vodu a auto.</p></div><label>Simulační scénář<select :value="state?.scenario" :disabled="busy" @change="scenario"><option v-for="s in scenarios" :value="s[0]">{{s[1]}}</option></select></label></div>
 <div class="notice">SIMULACE <span>Všechna data jsou modelová. Doporučení se neodesílají do zařízení.</span></div>
 <p v-if="error" role="alert" class="error">{{error}}</p>
 <template v-if="state">
 <section class="summary"><div><p class="eyebrow">DOPORUČENÝ REŽIM</p><h2>{{state.decision.mode}}</h2><p>{{state.decision.reason}}</p></div><div class="target"><span>Cílový tok baterie</span><strong>{{state.decision.target.toLocaleString('cs-CZ')}} <small>W</small></strong><span>+ nabíjení / − vybíjení</span></div></section>
 <div class="grid">
 <section class="card"><div class="cardhead"><h2>Baterie</h2><span class="symbol">▤</span></div><div class="metric">{{value('soc')}} <small>%</small></div><div class="bar"><i :style="{width:(state.quality.soc==='valid'?state.input.soc.value:0)+'%'}"></i></div><dl><dt>Simulovaný tok</dt><dd>{{value('battery')}} W</dd><dt>Nejnižší článek</dt><dd>{{value('min_cell')}} V</dd><dt>Nejvyšší článek</dt><dd>{{value('max_cell')}} V</dd></dl></section>
 <section class="card"><div class="cardhead"><h2>Teplá voda</h2><span class="symbol">≈</span></div><div class="metric">{{value('temperature')}} <small>°C</small></div><p class="muted">{{state.scenario==='manual'?'Ruční požadavek 2 kW':'Automatický příděl'}} · 3 × 1 kW</p><dl><dt>Simulovaná skutečnost</dt><dd>{{value('tuv')}} W</dd><dt>Doporučení</dt><dd>{{state.decision.tuv}} W</dd><dt>Odesláno zařízení</dt><dd>Nic</dd></dl></section>
 <section class="card"><div class="cardhead"><h2>Tesla</h2><span class="symbol">ϟ</span></div><div class="metric">{{(state.decision.car/230).toLocaleString('cs-CZ')}} <small>A</small></div><p class="muted">Doporučený proud · minimum 5 A</p><dl><dt>Simulovaná skutečnost</dt><dd>{{value('car')}} W</dd><dt>Doporučení</dt><dd>{{state.decision.car}} W</dd><dt>Odesláno zařízení</dt><dd>Nic</dd></dl></section>
 </div>
 <div class="lower"><section class="card"><p class="eyebrow">BATERIE V ROVNOVÁZE</p><h2>Balancování</h2><p class="balance">{{state.decision.balance}}</p><dl><dt>Poslední dokončení (simulace)</dt><dd>{{date(state.decision.last_balance)}}</dd><dt>Další termín</dt><dd>{{state.decision.last_balance.startsWith('0001')?'Nyní • chybí předchozí záznam':date(state.decision.next_balance)}}</dd></dl><p class="muted">Interval 14 dní · SOC ≥ 95 % · min. článek ≥ 3,439 V po 5 minut</p></section>
 <section class="card"><p class="eyebrow">POSLEDNÍ VZORKY</p><h2>Tok baterie <small class="muted"> / W</small></h2><svg viewBox="0 0 500 100" role="img" aria-label="Historie simulovaného toku baterie"><path d="M0 85H500" stroke="#d9e3df"/><polyline :points="history.map((h,i)=>`${i*500/Math.max(1,history.length-1)},${85-Math.min(4000,Math.max(-500,h.battery))*0.018}`).join(' ')" fill="none" stroke="#287f61" stroke-width="3"/></svg><p class="muted">SQLite · uchování 7 dní · zobrazeno posledních 120 vzorků</p></section></div>
 <details class="card"><summary>Diagnostika a plánování</summary><p>observe_only = true · control_enabled = false · vlastník fyzických výstupů: žádný</p><p>NT/VT a predikční plánování nejsou připojené. Limity tohoto modelu nejsou limity vaší instalace.</p><p v-for="(q,k) in state.quality">{{k}}: {{q}}</p></details>
 <footer>FVE Controller {{version}} <span>Poslední vzorek {{new Date(state.at).toLocaleTimeString('cs-CZ')}}</span></footer>
 </template><p v-else>Načítám simulaci…</p>
</main>
</template>
