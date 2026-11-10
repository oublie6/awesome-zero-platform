/** Pure safe-area menu paging. No platform SDK or Cocos dependencies. */
export type MenuScreen = 'home'|'rooms'|'rules'|'table'|'login'
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
  const columns=screen==='home'||screen==='rules'||screen==='login'?1:
    screen==='table'?(portrait?Math.max(2,Math.min(4,Math.floor(width/90))):Math.max(3,Math.min(6,Math.floor(width/120)))):
    portrait?(width<600?1:2):(width<640?2:3)
  const safeHeight=Math.max(1,height)
  const rowPitch=Math.max(38,Math.min(56,safeHeight*0.106))
  const buttonHeight=Math.max(34,Math.min(52,rowPitch-4))
  // Login has two EditBoxes above its action buttons. The default menu
  // origin intersects the password input on short landscape screens.
  const firstY=-safeHeight*(screen==='login'?0.15:0.103)
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
