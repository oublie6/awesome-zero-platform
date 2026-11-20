import { Color, EventTouch, Graphics, Label, Layers, Node, UITransform } from 'cc'
import type { PrivateHandProjection } from '@awesome-zero-platform/doudizhu-client/view'
import { faceOfCard, handFan, publicCardCodes, relativeOpponents, sortHandCards, tableActionCenterY, tableHeaderSlots, tableUnit, tableCaptionFont, tableNoticeText, tableNoticePlacement } from './TableGeometry.ts'
import { TurnClockAnchor } from './TurnClock.ts'

export interface TableDisplay {
  roomID:string
  handID:string
  version:number
  phase:string
  view:PrivateHandProjection | null
  selected:ReadonlySet<string>
  needsReconciliation:boolean
  actionInFlight:boolean
  finalSummary:string
}
export interface TableCallbacks {
  selectCard(card:string):void
  bid(score:number):void
  play():void
  pass():void
  refresh():void
  evidence():void
  retryUnknown():void
  discardUnknown():void
  back():void
}

const FELT=new Color(15,77,68,255)
const FELT_INNER=new Color(18,92,79,255)
const FELT_DARK=new Color(9,48,47,255)
const GOLD=new Color(226,190,112,255)
const PALE=new Color(248,239,208,255)
const WHITE=new Color(255,251,237,255)
const MUTED=new Color(162,203,187,255)
const RED=new Color(184,49,49,255)
const INK=new Color(31,40,47,255)
const TRANSPARENT=new Color(0,0,0,0)

function safePhase(phase:string):string{
  if(phase==='BIDDING')return '叫地主'
  if(phase==='PLAYING')return '出牌中'
  if(phase==='COMPLETED')return '已结算'
  if(phase==='ABORTED')return '已结束'
  return '等待开局'
}
function smallID(value:string):string{
  return value.length>15?value.slice(0,12)+'…':value
}
function seatNumber(value:unknown):number{
  return typeof value==='number' && Number.isInteger(value) && value>=1 && value<=3?value:0
}
function currentBidder(view:PrivateHandProjection|null):number{
  return seatNumber(view?.public.bidding?.currentBidder)
}
function highestBid(view:PrivateHandProjection|null):number{
  const value=view?.public.bidding?.highestScore
  return typeof value==='number' && Number.isInteger(value) && value>=0 && value<=3?value:0
}
function publicPlaying(view:PrivateHandProjection|null):{cards:string[];seat:number;passSeat:number}{
  const playing=view?.public.playing
  if(!playing || typeof playing!=='object')return {cards:[],seat:0,passSeat:0}
  const seat=seatNumber(playing.leadingSeat)
  const cards=publicCardCodes(playing.leadingCards)
  const history=playing.history
  const last=Array.isArray(history) && history.length>0?history[history.length-1]:null
  const passSeat=last && typeof last==='object' && last.type==='PASS'?seatNumber(last.seat):0
  return {cards,seat:cards.length>0?seat:0,passSeat}
}

/**
 * Dedicated presentation layer for the actual three-seat game.
 * It never mutates a projection, invents cards or decides game legality.
 * Every callback delegates to the existing authority-backed actions.
 */
export class GameTableView {
  private readonly root:Node
  private data:TableDisplay|null=null
  private width=760
  private height=440
  private unit=1
  private noticeText=''
  private countdownLabel:Label|null=null
  private tickHandle:ReturnType<typeof setInterval>|null=null
  setNotice(message:string):void{
    this.noticeText=tableNoticeText(message)
    if(this.data)this.layout(this.width,this.height)
  }
  constructor(parent:Node,private readonly callbacks:TableCallbacks,
    private readonly turnClock:TurnClockAnchor){
    this.root=new Node('ActualThreeSeatTable')
    this.root.layer=Layers.Enum.UI_2D
    this.root.addComponent(UITransform)
    parent.addChild(this.root)
  }
  destroy():void {
    if(this.tickHandle!==null){clearInterval(this.tickHandle);this.tickHandle=null}
    if(this.root.isValid)this.root.destroy()
    this.data=null
  }
  render(data:TableDisplay,width:number,height:number):void {
    if(this.data?.handID!==data.handID)this.noticeText=''
    // The anchor is owned by GameBootstrap, not this disposable table view.
    // Returning to rooms and reopening must not restart a stale snapshot clock.
    this.turnClock.observe(data.handID,data.version,data.view?.public??{})
    this.data=data
    this.layout(width,height)
    if(this.tickHandle===null)this.tickHandle=setInterval(()=>this.updateCountdown(),250)
  }
  layout(width:number,height:number):void {
    if(!this.root.isValid || !this.data)return
    this.width=width
    this.height=height
    this.countdownLabel=null
    for(const child of [...this.root.children]){
      child.removeFromParent()
      child.destroy()
    }
    const data=this.data
    const w=Math.max(240,width), h=Math.max(280,height)
    const u=tableUnit(w,h)
    this.unit=u
    this.root.getComponent(UITransform)!.setContentSize(w,h)

    // Original drawn felt and gold edging. No copied commercial assets.
    this.panel('TableFelt',0,0,w,h,FELT_DARK,TRANSPARENT,0)
    this.panel('FeltInset',0,0,w-8*u,h-8*u,FELT,GOLD,Math.max(12,26*u))
    this.panel('FeltCentre',0,h*.06,w*.65,Math.min(h*.51,310*u),FELT_INNER,
      new Color(52,120,101,255),Math.min(75*u,h*.21))
    this.label('TableWatermark','公 平 竞 技',0,h*.065,26*u,new Color(75,136,116,255),w*.45,32*u)

    const view=data.view
    const pub=view?.public
    const me=view?.position??0
    const landlord=seatNumber(pub?.landlordSeat)
    const turn= data.phase==='BIDDING'?currentBidder(view):seatNumber(pub?.playingSeat)
    const myTurn=me!==0 && turn===me &&
      !data.needsReconciliation && !data.actionInFlight
    const final=data.phase==='COMPLETED'||data.phase==='ABORTED'

    // Separate top chrome: room/hand/version are informational, not game actions.
    const top=h/2-28*u
    const header=tableHeaderSlots(w,u)
    this.label('TableBrand','斗地主  ·  '+safePhase(data.phase),
      header.leftX,top,22*u,GOLD,header.leftWidth,38*u)
    this.label('RoomVersion','房间 '+smallID(data.roomID||'—')+'  ·  v'+data.version,
      header.leftX,top-29*u,14*u,MUTED,header.leftWidth,24*u)
    const buttonW=header.rightWidth
    this.button('BackToRoom','返回',header.rightX,top,buttonW,35*u,true,this.callbacks.back,true)
    this.button('UpdateHand','同步',header.rightX,top-43*u,buttonW,34*u,
      !!data.handID,this.callbacks.refresh,true)

    // Countdown is presentation-only and driven by the server's timeline.
    this.countdownLabel=this.label('AuthoritativeTurnClock','等待计时同步',
      0,top,24*u,GOLD,header.clockWidth,37*u)
    this.updateCountdown()
    
    // Top central landlord kitty: reveal only exactly the cards that Go marks public.
    const kitty=Array.isArray(pub?.landlordCards)?pub!.landlordCards:[]
    const safeKitty=kitty.length===3 && kitty.every(c=>typeof c==='string')?kitty:[]
    const kittyY=h/2-98*u
    const smallCardW=29*u,smallCardH=43*u
    for(let i=0;i<3;i++){
      const x=(i-1)*35*u
      if(safeKitty.length===3) this.faceCard('LandlordPublic'+i,safeKitty[i],
        x,kittyY,smallCardW,smallCardH,u*.70,false)
      else this.cardBack('LandlordBack'+i,x,kittyY,smallCardW,smallCardH,u*.70)
    }
    this.label('KittyLabel','地主底牌',0,kittyY-32*u,13*u,MUTED,130*u,21*u)

    // Both opponents rotate relative to my true server-assigned seat.
    if(me>=1&&me<=3&&pub?.seats?.length===3){
      const relative=relativeOpponents(me)
      const seatY=Math.min(h*.17,h/2-165*u)
      for(const [side,position] of [['left',relative.left],['right',relative.right]] as const){
        const player=pub.seats.find(x=>x.position===position)
        if(!player)continue
        const x=(side==='left'?-1:1)*(w/2-Math.min(86*u,w*.245))
        const current=turn===position
        const role=landlord===position?'地主':'农民'
        const panelWidth=Math.min(w*.38,135*u)
        this.panel('Seat'+position,x,seatY,panelWidth,77*u,
          current?new Color(39,107,85,255):FELT_DARK,current?GOLD:new Color(50,112,92,255),16*u)
        this.label('Name'+position,(side==='left'?'上家':'下家')+' · '+role,
          x,seatY+19*u,16*u,current?GOLD:PALE,panelWidth-8*u,25*u)
        this.label('Remaining'+position,'剩余 '+player.remainingCards+' 张',
          x,seatY-12*u,17*u,WHITE,panelWidth-8*u,26*u)
        for(let i=0;i<3;i++)
          this.cardBack('OppBack'+position+'-'+i,
            x+(i-1)*14*u,seatY-65*u,28*u,38*u,u*.65)
      }
    } else if(!final){
      this.label('NoHandSeats','等待服务端发牌后展示三家座位',0,h*.2,19*u,MUTED,w*.76,28*u)
    }

    // Central trick accepts Go's Base64 []uint8 public wire representation.
    const playing=publicPlaying(view??null)
    const trickY=Math.max(5*u,h*.035)
    const trickCards=playing.cards
    const trickCardWidth=Math.min(51*u,Math.max(26,w*.12))
    const trickWidth=Math.min(w*.57,560*u)
    const trickStep=trickCards.length<=1?0:
      Math.min(trickCardWidth*.67,(trickWidth-trickCardWidth)/(trickCards.length-1))
    const trickSpan=trickCardWidth+Math.max(0,trickCards.length-1)*trickStep
    if(trickCards.length>0){
      this.label('LeadingSeat','座位 '+playing.seat+' 出牌',0,trickY+67*u,16*u,GOLD,w*.55,24*u)
      for(let i=0;i<trickCards.length;i++){
        this.faceCard('PublicTrick'+i,trickCards[i],
          -trickSpan/2+trickCardWidth/2+i*trickStep,trickY,
          trickCardWidth,trickCardWidth*1.4,Math.min(u,0.9),false)
      }
    }else{
      const result=final?(data.finalSummary||'本局已结束'):
        data.phase==='BIDDING'?'正在叫地主':
        turn?('等待座位 '+turn+' 出牌'):'等待服务端牌局状态'
      this.label('EmptyTrick',result,0,trickY+10*u,20*u,PALE,w*.58,58*u)
    }
    // Temporary command feedback takes precedence over historical hints;
    // both otherwise occupy the narrow strip between the trick and actions.
    if(playing.passSeat && !final && !this.noticeText)
      this.label('LastPass','座位 '+playing.passSeat+' 不出',0,trickY-55*u,15*u,MUTED,w*.5,22*u)

    const cards=final?[]:Array.isArray(view?.cards)?sortHandCards(view!.cards):[]
    const selected=data.selected
    const fan=handFan(cards.length,w,h,selected,cards)
    const controlsY=tableActionCenterY(fan,w,h,u)
    // Operations are always in their own row; cards never share pagination.
    if(data.needsReconciliation){
      // These are the only actionable controls until the original durable
      // request is resolved. Replacing (rather than stacking on) the main row
      // leaves a dedicated, non-overlapping message row for ACK loss.
      const bw=Math.min(150*u,w*.39)
      this.button('RetryUnknown','核对上次操作',-bw*.54,controlsY,bw,36*u,true,this.callbacks.retryUnknown)
      this.button('DiscardUnknown','按新快照放弃',bw*.54,controlsY,bw,36*u,true,this.callbacks.discardUnknown)
    }else if(data.phase==='BIDDING'){
      const highest=highestBid(view??null)
      const scores=[0,1,2,3]
      const bw=Math.min(110*u,(w-30*u)/4-6*u)
      scores.forEach((score,i)=>this.button('Bid'+score,
        score===0?'不叫':score+' 分',
        (i-1.5)*(bw+6*u),controlsY,bw,38*u,
        myTurn&&(score===0||score>highest),()=>this.callbacks.bid(score)))
    }else if(data.phase==='PLAYING'){
      const bw=Math.min(150*u,w*.30)
      this.button('PassTurn','不出',-bw*.58,controlsY,bw,42*u,
        myTurn && playing.cards.length>0,this.callbacks.pass)
      this.button('PlaySelected','出牌'+(selected.size?' ('+selected.size+')':''),
        bw*.58,controlsY,bw,42*u,myTurn&&selected.size>0,this.callbacks.play,true)
    }
    // Main hand: stable sorted rank order, full hand visible and raised on tap.
    // Draw backs / opponents separately: only these private cards are exposed.
    cards.forEach((code,i)=>{
      const pos=fan.places[i]
      const card=this.faceCard('HandCard_'+code,code,pos.x,pos.y,
        fan.cardWidth,fan.cardHeight,u,selected.has(code))
      card.angle=pos.angle
      card.on(Node.EventType.TOUCH_END,(_event:EventTouch)=>this.callbacks.selectCard(code))
    })
    if(this.noticeText){
      const feedback=tableNoticePlacement(controlsY,w,h,u)
      this.label('TableNotice',this.noticeText,0,feedback.y,
        feedback.font,new Color(255,216,143,255),w*.74,feedback.height)
    }
    if(me>=1 && me<=3){
      const mineRole=landlord===me?'地主':'农民'
      const ownText='我('+me+') · '+mineRole+' · '+cards.length+'张'
      this.label('OwnSeatInfo',ownText,header.leftX,top-56*u,
        15*u,MUTED,header.leftWidth,23*u)
    }
    if((final || data.phase==='COMPLETED') && !data.needsReconciliation){
      const bw=Math.min(136*u,w*.27)
      this.button('VerifyFinalEvidence','核验结算',0,controlsY,2*bw,38*u,
        !!data.handID,this.callbacks.evidence)
    }
    if(!view && data.handID && !final && !this.noticeText){
      this.label('ServerSyncNeeded','请同步服务端私有牌局快照',0,-h*.23,15*u,MUTED,w*.66,25*u)
    }
  }

  private updateCountdown():void{
    if(!this.countdownLabel?.isValid || !this.data)return
    const seconds=this.turnClock.remainingSeconds()
    const phase=this.data.phase
    if(phase!=='BIDDING' && phase!=='PLAYING'){
      this.countdownLabel.string='—'
      return
    }
    this.countdownLabel.string=seconds===null?'等待计时同步':seconds===0?'00s':String(seconds).padStart(2,'0')+'s'
    this.countdownLabel.color=seconds!==null && seconds<=5?new Color(255,117,90,255):GOLD
  }
  private node(name:string,x:number,y:number,w:number,h:number):Node{
    const node=new Node(name)
    node.layer=Layers.Enum.UI_2D
    node.addComponent(UITransform).setContentSize(Math.max(1,w),Math.max(1,h))
    this.root.addChild(node)
    node.setPosition(x,y,0)
    return node
  }
  private panel(name:string,x:number,y:number,w:number,h:number,
    fill:Color,border:Color,radius:number):Node{
    const node=this.node(name,x,y,w,h)
    const graphics=node.addComponent(Graphics)
    graphics.fillColor=fill
    graphics.roundRect(-w/2,-h/2,w,h,radius)
    graphics.fill()
    if(border.a>0){
      graphics.strokeColor=border
      graphics.lineWidth=1.6
      graphics.roundRect(-w/2+1,-h/2+1,w-2,h-2,Math.max(0,radius-1))
      graphics.stroke()
    }
    return node
  }
  private label(name:string,text:string,x:number,y:number,font:number,color:Color,w:number,h:number):Label{
    const node=this.node(name,x,y,w,h)
    const label=node.addComponent(Label)
    label.string=text
    label.fontSize=font
    label.lineHeight=font+5
    label.horizontalAlign=Label.HorizontalAlign.CENTER
    label.verticalAlign=Label.VerticalAlign.CENTER
    label.color=color
    label.overflow=Label.Overflow.SHRINK
    label.enableWrapText=false
    return label
  }
  private button(name:string,text:string,x:number,y:number,w:number,h:number,
    enabled:boolean,click:()=>void,primary=false):void{
    const fill=enabled?(primary?new Color(172,92,39,255):new Color(26,90,80,255)):
      new Color(53,85,79,255)
    const border=enabled?(primary?new Color(245,193,110,255):new Color(100,157,131,255)):
      new Color(71,109,96,255)
    const node=this.panel(name,x,y,w,h,fill,border,Math.min(13,h*.36))
    // The logical Cocos canvas can be 1280px wide but only 390 CSS px
    // wide on a portrait phone. A fixed 18 logical-px cap gave 5.48px
    // unreadable labels. Scale font with the same unit as the button, and
    // bound it by available width for long captions to prevent SHRINK.
    const captionSize=tableCaptionFont(text,w,h,this.unit)
    const label=this.label(name+'Caption',text,x,y,captionSize,
      enabled?WHITE:MUTED,w-6*this.unit,h-4*this.unit)
    label.node.setParent(node)
    label.node.setPosition(0,0,0)
    if(enabled){
      node.on(Node.EventType.TOUCH_END,(_event:EventTouch)=>{
        if(this.root.isValid)click()
      })
    }
  }
  private faceCard(name:string,code:string,x:number,y:number,w:number,h:number,
    scale:number,selected:boolean):Node{
    const node=this.panel(name,x,y,w,h,WHITE,selected?GOLD:new Color(216,221,213,255),6*scale)
    const face=faceOfCard(code)
    const color=face.red?RED:INK
    const left=x-w/2
    const top=y+h/2
    const rankFont=Math.min(25*scale,w*.39)
    const rank=this.label(name+'Rank',face.rank,
      left+w*.23,top-h*.19,rankFont,color,w*.42,h*.28)
    rank.node.setParent(node)
    rank.node.setPosition(-w*.27,h*.31,0)
    const suit=this.label(name+'Suit',face.suit,
      left+w*.22,top-h*.42,Math.min(23*scale,w*.37),color,w*.42,h*.25)
    suit.node.setParent(node)
    suit.node.setPosition(-w*.28,h*.08,0)
    if(w>48*scale) {
      const mark=this.label(name+'Face',face.suit,
        x+w*.17,y-h*.15,Math.min(39*scale,w*.51),new Color(color.r,color.g,color.b,90),
        w*.55,h*.54)
      mark.node.setParent(node)
      mark.node.setPosition(w*.17,-h*.15,0)
    }
    return node
  }
  private cardBack(name:string,x:number,y:number,w:number,h:number,scale:number):Node{
    const panel=this.panel(name,x,y,w,h,new Color(171,58,57,255),new Color(249,218,163,255),4*scale)
    const inner=this.panel(name+'Inner',x,y,w-7*scale,h-7*scale,
      new Color(138,48,51,255),GOLD,2*scale)
    this.label(name+'BackMark','✦',x,y,Math.min(21*scale,w*.52),GOLD,w*.7,h*.5)
    return panel
  }
}
