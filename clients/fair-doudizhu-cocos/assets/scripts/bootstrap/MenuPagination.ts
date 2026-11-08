/** Pure safe-area menu paging. No platform SDK or Cocos dependencies. */
export type MenuScreen = 'home'|'rooms'|'rules'|'table'
export interface MenuWindow {
  columns:number
  rows:number
  rowPitch:number
  buttonHeight:number
  firstY:number
  page:number
  pages:number
  start:number
  end:number
}
export function menuWindow(count:number,width:number,height:number,portrait:boolean,
  screen:MenuScreen,requestedPage:number):MenuWindow{
  if(!Number.isSafeInteger(count) || count<0)throw new Error('invalid action count')
  const columns=screen==='home'||screen==='rules'?1:screen==='table'?(portrait?4:6):(portrait?2:3)
  const safeHeight=Math.max(1,height)
  const rowPitch=Math.max(38,Math.min(56,safeHeight*0.106))
  const buttonHeight=Math.max(34,Math.min(52,rowPitch-4))
  const firstY=-safeHeight*0.103
  const bottomCenter=-safeHeight/2+11+buttonHeight/2
  const rows=Math.max(1,Math.floor((firstY-bottomCenter)/rowPitch)+1)
  // Reserve a dedicated navigation row when there are multiple pages.
  // On very short screens use one cycling navigation slot.
  const allFit=count<=columns*rows
  const capacity=allFit?Math.max(1,columns*rows):
    rows===1?Math.max(1,columns-1):Math.max(1,(rows-1)*columns)
  const pages=Math.max(1,Math.ceil(count/capacity))
  const page=Number.isSafeInteger(requestedPage)?Math.max(0,Math.min(pages-1,requestedPage)):0
  return {columns,rows,rowPitch,buttonHeight,firstY,page,pages,
    start:page*capacity,end:Math.min(count,(page+1)*capacity)}
}
