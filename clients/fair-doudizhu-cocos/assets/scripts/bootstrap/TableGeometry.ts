/** Presentation-only helpers. The Go hand state remains authoritative. */
export const CARD_STRENGTH: Readonly<Record<string, number>> = Object.freeze({
  '3':3,'4':4,'5':5,'6':6,'7':7,'8':8,'9':9,'10':10,
  'J':11,'Q':12,'K':13,'A':14,'2':15,'XJ':16,'YJ':17,
})
const SUIT_ORDER: Readonly<Record<string,number>> = Object.freeze({S:4,H:3,C:2,D:1})
const SUIT_GLYPH: Readonly<Record<string,string>> = Object.freeze({S:'♠',H:'♥',C:'♣',D:'♦'})

export function faceOfCard(code:string):{rank:string;suit:string;red:boolean;joker:boolean}{
  if(code==='YJ')return {rank:'大王',suit:'★',red:true,joker:true}
  if(code==='XJ')return {rank:'小王',suit:'★',red:false,joker:true}
  const suit=code.slice(0,1)
  const rank=code.slice(1)
  if(!(suit in SUIT_ORDER) || !(rank in CARD_STRENGTH) || rank==='XJ'||rank==='YJ')
    throw new Error('invalid card code')
  return {rank,suit:SUIT_GLYPH[suit],red:suit==='H'||suit==='D',joker:false}
}

/** Big joker > small joker > 2 > A ... 3; suit is a display-only tie break. */
export function sortHandCards(cards:readonly string[]):string[]{
  return [...cards].sort((a,b)=>{
    const rank=(code:string)=>CARD_STRENGTH[code==='XJ'||code==='YJ'?code:code.slice(1)]??-1
    const strength=rank(b)-rank(a)
    if(strength!==0)return strength
    return (SUIT_ORDER[b[0]]??0)-(SUIT_ORDER[a[0]]??0)
  })
}

/** Go public playing.Snapshot encodes carddeck.Card as canonical numeric IDs.
 * This conversion may only be used for public leadingCards/history, never to
 * infer unrevealed opponent hands. */
export function publicCardCode(id:unknown):string|null{
  if(typeof id!=='number'||!Number.isInteger(id)||id<0||id>53)return null
  if(id===52)return 'XJ'
  if(id===53)return 'YJ'
  const rank=['3','4','5','6','7','8','9','10','J','Q','K','A','2'][Math.floor(id/4)]
  const suit=['C','D','H','S'][id%4]
  return suit+rank
}
export function publicCardCodes(raw:unknown):string[]{
  if(!Array.isArray(raw)||raw.length>20)return []
  const result:string[]=[]
  for(const id of raw){
    const card=publicCardCode(id)
    if(card===null)return []
    result.push(card)
  }
  return sortHandCards(result)
}
export function relativeOpponents(mySeat:number):{left:number;right:number}{
  if(!Number.isInteger(mySeat)||mySeat<1||mySeat>3)throw new Error('invalid own seat')
  return {left:(mySeat+1)%3+1,right:mySeat%3+1}
}
export interface HandCardPlace{
  x:number
  y:number
  angle:number
}
export interface HandFan{
  cardWidth:number
  cardHeight:number
  step:number
  places:readonly HandCardPlace[]
}
/** All cards stay on-screen for 17-20-card hands; no pagination. */
export function handFan(count:number,width:number,height:number,selected:ReadonlySet<string>,
  cards:readonly string[]):HandFan{
  if(!Number.isInteger(count)||count<0||count>20||count!==cards.length)
    throw new Error('invalid hand size')
  const w=Math.max(240,width),h=Math.max(280,height)
  const unit=Math.max(0.72,Math.min(1.35,Math.min(w/760,h/440)))
  const cardWidth=Math.min(78*unit,w*0.21)
  const cardHeight=cardWidth*1.47
  const available=Math.max(cardWidth,w-28*unit)
  const step=count>1?Math.min(cardWidth*0.64,(available-cardWidth)/(count-1)):0
  const span=cardWidth+Math.max(0,count-1)*step
  const baseY=-h/2+10*unit+cardHeight/2
  const places=cards.map((card,i)=>{
    const norm=count<=1?0:(2*i-(count-1))/(count-1)
    return {
      x:-span/2+cardWidth/2+i*step,
      y:baseY+7*unit*(1-norm*norm)+(selected.has(card)?15*unit:0),
      angle:-norm*6,
    }
  })
  return {cardWidth,cardHeight,step,places}
}

/** The three top chrome slots must not overlap at supported viewport widths. */
export function tableHeaderSlots(width:number,scale:number):{
  leftX:number;leftWidth:number;clockWidth:number;rightX:number;rightWidth:number
}{
  const w=Math.max(240,width)
  const u=Math.max(.72,Math.min(1.35,scale))
  const leftWidth=Math.min(w*.30,220*u)
  const rightWidth=Math.max(58,75*u)
  return {
    leftX:-w/2+14*u+leftWidth/2,leftWidth,
    clockWidth:Math.min(w*.24,205*u),
    rightX:w/2-rightWidth/2-14*u,rightWidth,
  }
}

/** Lifted selected cards stay below the operating controls' touch targets. */
export function tableActionCenterY(fan:HandFan,height:number,scale:number):number{
  const u=Math.max(.72,Math.min(1.35,scale))
  return fan.places.length?fan.places[0].y+fan.cardHeight/2+50*u:-height/2+90*u
}
