// Agreed estimate: constant cross-section, 170 one-centimetre layers.
// Sample each layer at its centre; interpolate only between the two sensors.
export function tuvCharge(upper:number|null,lower:number|null):number|null {
  if(upper===null||lower===null||!Number.isFinite(upper)||!Number.isFinite(lower)||upper<0||upper>100||lower<0||lower>100)return null
  if(upper<=45)return 0
  let sum=0
  for(let i=0;i<170;i++){
    const height=i+0.5
    const temperature=height<=40?lower:height>=130?upper:lower+(upper-lower)*(height-40)/90
    sum+=Math.max(0,Math.min(1,(temperature-45)/12))
  }
  return Math.round(sum/170*1000)/10
}
