<script setup lang="ts">
import {computed,ref} from 'vue'
import type {Source,TuvTelemetry} from './live'
const props=defineProps<{source:Source<TuvTelemetry>}>()
const emit=defineEmits<{changed:[]}>()
const busy=ref(false),error=ref(''),accepted=ref('')
const central=computed(()=>props.source.data?.tuv_control)
const pump=computed(()=>props.source.data?.pump_control)
const owned=computed(()=>!props.source.error&&!!props.source.data?.control_token&&central.value?.pump_owned===true)
const manual=computed(()=>central.value?.mode==='manual')
async function command(path:string,body:object){
 busy.value=true;error.value='';accepted.value=''
 try{
  const r=await fetch(path,{method:'POST',headers:{'Content-Type':'application/json','X-FVE-TUV':props.source.data?.control_token??''},body:JSON.stringify(body),signal:AbortSignal.timeout(5000)})
  if(!r.ok)throw new Error((await r.text()).trim()||'Povel se nepodařilo přijmout')
  accepted.value='Požadavek přijat · čeká na aktualizaci stavu';emit('changed')
 }catch(e){error.value=e instanceof Error?e.message:'Povel se nepodařilo přijmout'}finally{busy.value=false}
}
function state(v:boolean|null|undefined){return v==null?'—':v?'Zap':'Vyp'}
</script>
<template>
 <div class="tuv-control">
  <p class="eyebrow">Režim TUV</p>
  <div class="control-buttons" aria-label="Režim TUV">
   <button :aria-pressed="central?.mode==='auto'" :disabled="!owned||busy" @click="command('api/tuv/mode',{mode:'auto'})">Automatika TUV</button>
   <button :aria-pressed="manual" :disabled="!owned||busy" @click="command('api/tuv/mode',{mode:'manual'})">Ruční ovládání</button>
  </div>
  <p class="muted">Ohřev řídí Node-RED. Režim nyní ovládá pouze čerpadlo.</p>
  <div v-if="owned&&manual" class="control-buttons" aria-label="Ruční čerpadlo">
   <button :disabled="busy||!pump?.ready" @click="command('api/tuv/pump',{on:true})">Čerpadlo Zap</button>
   <button :disabled="busy" @click="command('api/tuv/pump',{on:false})">Čerpadlo Vyp</button>
  </div>
  <p v-if="owned" class="pump-feedback">Požadavek: <strong>{{state(pump?.desired)}}</strong> · Odesláno: <strong>{{state(pump?.sent)}}</strong> · HA: <strong>{{source.error?'—':state(pump?.confirmed)}}</strong></p>
  <p v-if="error||accepted" role="status" :class="{problem:!!error}">{{error||accepted}}</p>
 </div>
</template>
