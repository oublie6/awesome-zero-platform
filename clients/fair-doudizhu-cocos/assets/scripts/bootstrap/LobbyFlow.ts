/** Pure, server-authoritative lobby presentation. No Cocos or platform imports. */
export type RoomPhase = 'WAITING_PLAYERS'|'READY'|'HAND_ACTIVE'|'CLOSED'
export type LobbyStage = 'signed-out'|'entry'|'syncing'|'waiting'|'contributing'|'playing'|'settled'|'reconcile'
export type LobbyCommand =
  'create'|'join'|'discover'|'sync-room'|'ready'|'start'|'contribute'|
  'sync-fairness'|'table'|'evidence'|'verify-own'|'retry-room'|'retry-fairness'|'more'
export type RoomInputPurpose = 'none'|'join'|'contribute'|'verify'
export interface AuthorizedLobbyRoom {
  roomId:string
  member:boolean
  status:RoomPhase
  occupied:number
  seats?:ReadonlyArray<{position:number;accountId:string;ready:boolean}>
  activeHandId?:string
  lastHandId?:string
}
export interface LobbyContext {
  authenticated:boolean
  roomId:string
  room:AuthorizedLobbyRoom|null
  roomUnknown:boolean
  fairnessUnknown:boolean
  fairnessMode:string|null
  fairness:{handId:string;phase:string;revealed:boolean}|null
  handPhase:string|null
  recoverableRooms:number
  /** Server-validated session identity for display-only owner/ready gates. */
  viewerAccountID?:string|null
}
export interface LobbyPlan {
  stage:LobbyStage
  title:string
  hint:string
  input:RoomInputPurpose
  primary:readonly LobbyCommand[]
  secondary:readonly LobbyCommand[]
  seats:readonly {position:number;occupied:boolean;ready:boolean}[]
}

export function planLobby(ctx:LobbyContext):LobbyPlan {
  const base=(stage:LobbyStage,title:string,hint:string,input:RoomInputPurpose,
    primary:LobbyCommand[],secondary:LobbyCommand[]=[],seats:LobbyPlan['seats']=[]):LobbyPlan=>
    ({stage,title,hint,input,primary,secondary,seats})
  if(!ctx.authenticated)
    return base('signed-out','房间大厅','登录后即可和好友一起玩','none',[])
  // Unknown durable commands must be reconciled by exact identity. A room
  // query is NOT permission to create a replacement command or clear UNKNOWN.
  if(ctx.roomUnknown)
    return base('reconcile','房间操作待确认','先核对原操作，不要重复创建或加入','none',['retry-room'])
  if(ctx.fairnessUnknown)
    return base('reconcile','随机贡献待确认','先核对原提交，不要再次生成贡献命令','none',['retry-fairness'])
  if(!ctx.roomId)
    return base('entry','游戏大厅','和好友开一桌，或输入房间号加入','join',
      ['create','join'],['discover','more'])
  const room=ctx.room
  if(!room || room.roomId!==ctx.roomId || !room.member || room.status==='CLOSED')
    return base('syncing','正在确认房间','需要先通过服务器确认成员与房间状态','none',['sync-room'])
  const roomSeats=(room.seats||[]).slice(0,3).sort((a,b)=>a.position-b.position)
  const viewer=ctx.viewerAccountID
    ? roomSeats.find(seat=>seat.accountId===ctx.viewerAccountID):undefined
  if(ctx.viewerAccountID && !viewer)
    return base('syncing','正在确认房间','账号与房间成员信息不一致，请重新同步','none',['sync-room'])
  const seats=roomSeats.map(seat=>({
    position:seat.position,occupied:!!seat.accountId,ready:!!seat.accountId&&seat.ready
  }))
  const roomTitle='好友房间 · '+room.roomId
  if(room.activeHandId){
    const own=ctx.fairness?.handId===room.activeHandId?ctx.fairness:null
    if(ctx.handPhase==='COMPLETED'||ctx.handPhase==='ABORTED')
      return base('settled',roomTitle,'本局已结束，可以查看归档证明',
        ctx.fairnessMode==='client-contributed-server-custodied-v1'?'verify':'none',
        ['evidence'],ctx.fairnessMode==='client-contributed-server-custodied-v1'?['verify-own','more']:['more'],seats)
    if(ctx.handPhase==='BIDDING'||ctx.handPhase==='PLAYING')
      return base('playing',roomTitle,'牌局进行中，点击继续牌局','none',['table'],['sync-room','more'],seats)
    if(ctx.fairnessMode==='client-contributed-server-custodied-v1' && own){
      if(!own.revealed && own.phase!=='BIDDING' && own.phase!=='PLAYING')
        return base('contributing',roomTitle,'请输入本人随机验证字符串，提交后由服务器加密托管',
          'contribute',['contribute'],['sync-fairness','more'],seats)
      return base('waiting',roomTitle,'已提交贡献，等待其他玩家和公开信标','none',
        ['sync-fairness'],['more'],seats)
    }
    return base('syncing',roomTitle,'正在确认本局公平阶段','none',
      ['sync-fairness'],['more'],seats)
  }
  const allReady=room.status==='READY'
  const someoneNotReady=seats.some(seat=>seat.occupied&&!seat.ready)
  // Room creation permanently binds the owner to seat 1; Room.StartHand
  // authorizes only that owner. The backend remains the final authority.
  const canStart=allReady && (!viewer || viewer.position===1)
  const canReady=room.status==='WAITING_PLAYERS' &&
    (viewer?!viewer.ready:someoneNotReady)
  const primary:LobbyCommand[]=canStart?['start']:canReady?['ready']:['sync-room']
  const secondary:LobbyCommand[]=[]
  if(room.lastHandId){
    secondary.push('evidence')
    if(ctx.fairnessMode==='client-contributed-server-custodied-v1')
      secondary.push('verify-own')
  }
  secondary.push('sync-room','more')
  return base(room.lastHandId?'settled':'waiting',roomTitle,
    allReady?(viewer&&viewer.position!==1?'等待房主开始本局':'全员准备就绪，可以开始本局'):
      room.occupied<3?'等待好友加入，已到场 '+room.occupied+'/3':
        '等待成员准备，已到场 '+room.occupied+'/3',
    room.lastHandId&&ctx.fairnessMode==='client-contributed-server-custodied-v1'?'verify':'none',
    primary,secondary,seats)
}
