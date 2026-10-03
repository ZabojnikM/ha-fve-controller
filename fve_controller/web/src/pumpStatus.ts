import type {PumpControl} from './live'

export function pumpStatus(control:PumpControl|undefined,available:boolean){
  if(!available)return {label:'Stav automatiky není dostupný',reason:'Čeká na spojení s doplňkem',problem:true}
  if(!control?.enabled)return {label:'Řízení doplňkem nepřevzato',reason:'Čerpadlo se pouze sleduje',problem:false}
  const labels:Record<string,string>={
    waiting:'Čeká na živá data',idle:'Automatika čerpadla',sending:'Odesílá povel čerpadlu',
    waiting_confirmation:'Čeká na potvrzení čerpadla',unavailable:'Stav čerpadla není dostupný',
    input_invalid:'Čerpadlo zastaveno podle hlášení HA · neplatná data',
    command_error:'Povel čerpadla se nepodařilo odeslat',confirmation_timeout:'Čerpadlo nepotvrdilo povel',
    retry_wait:'Čeká na opakování povelu',service_error:'Servisní protočení se nepodařilo připravit',
  }
  const label=control.mode==='manual'&&control.status==='idle'?'Ruční ovládání čerpadla':labels[control.status]??'Stav automatiky není dostupný'
  const problem=!['idle','sending','waiting_confirmation'].includes(control.status)
  return {label,reason:control.error||control.reason,problem}
}
