package doudizhu

import (
	"math/rand"
	"time"
	"chessSever/program/game"
	"chessSever/program/game/poker"
	"sync"
	"fmt"
	"chessSever/program/game/msg"
	"strconv"
	"errors"
	"encoding/json"
)

type Doudizhu struct {
	id int                               //游戏ID
	name string                          //游戏名称

	playerNum int						 //玩家数
	deckNum int                          //几副牌

	sync.RWMutex                         //操作playNum以及player时加锁

	IsPlaying  bool                      //是否正在游戏中
	baseScore int						 //底分
	currMulti int                        //当前倍率
	CalledLoardNum  int                  //叫过地主的人数
	lordIndex int                        //地主索引
	CurrPlayerIndex int                  //当前叫地主或者出牌人的index
	FirstCallScoreIndex int              //第一个叫地主的人的index
	OutCardIndexs []int                  //出完牌的用户index

	Players []game.IPlayer               //玩家数组
	pokerCards []*poker.PokerCard        //当前游戏中的所有的牌
	lastCards  *game.LastCardsType 		 //最后的出牌结构
	playerCards [][]*poker.PokerCard     //同桌不同玩家的牌的切片
	bottomCards []*poker.PokerCard       //底牌
}

var originDoudizhu Doudizhu

func init(){
	//其他索引型属性不能在此赋值
	originDoudizhu = Doudizhu{
		name:game.GetGameName(game.GAME_TYPE_OF_DOUDOZHU),
		playerNum:3,
		deckNum:1,
		baseScore:10,
		currMulti:1,
		IsPlaying:false,
		CalledLoardNum:0,
		lordIndex:-1,
		CurrPlayerIndex:-1,
		id:-1,
	}
}

func GetDoudizhu(baseScore int) game.IGame{

	newDou := originDoudizhu

	newDou.Lock()
	newDou.baseScore = baseScore
	newDou.pokerCards = []*poker.PokerCard{}
	newDou.Players = []game.IPlayer{}
	newDou.playerCards = [][]*poker.PokerCard{[]*poker.PokerCard{},[]*poker.PokerCard{},[]*poker.PokerCard{}}
	newDou.bottomCards = []*poker.PokerCard{}
	newDou.id = game.GetRoom().AddGame(game.GAME_TYPE_OF_DOUDOZHU,&newDou)
	newDou.Unlock()

	newDou.initCards()

	game.GetRoom().AddGame(newDou.GetGameType(),&newDou)
	return &newDou
}

//增加玩家
func (dou *Doudizhu) AddPlayer(currPlayer game.IPlayer) error {
	dou.Lock()
	if dou.IsPlaying{
		dou.Unlock()
		return errors.New("游戏进行中，无法加入游戏")
	}
	if len(dou.Players) > dou.playerNum{
		dou.Unlock()
		panic("player Num ")
		return errors.New("游戏数据出错")
	}

	if len(dou.Players) == dou.playerNum{
		for i,p := range dou.Players{
			if p == nil{
				dou.Players[i] = currPlayer
				fmt.Println("有新玩家加入游戏"+strconv.Itoa(dou.id))
				dou.Unlock()
				currPlayer.SetIndex(i)
				game.BindPlayerGame(currPlayer,dou)
				dou.BroadCastMsg(p,msg.MSG_TYPE_OF_JOIN_TABLE,"玩家加入游戏")
				return nil
			}else{
				if(i == len(dou.Players)){
					dou.Unlock()
					return errors.New("该游戏玩家已满")
				}
			}
		}
	}else{
		dou.Players = append(dou.Players,currPlayer)
		fmt.Println("有新玩家加入桌子"+strconv.Itoa(dou.id))
		i := len(dou.Players)-1
		dou.Unlock()
		game.BindPlayerGame(currPlayer,dou)
		currPlayer.SetIndex(i)
		dou.BroadCastMsg(currPlayer,msg.MSG_TYPE_OF_JOIN_TABLE,"玩家加入游戏")
		return nil
	}
	return nil
}
//移除玩家
func (dou *Doudizhu) RemovePlayer(player game.IPlayer) error{
	dou.Lock()
	if dou.IsPlaying{
		return errors.New("游戏进行中，无法移除玩家")
	}
	dou.Players[player.GetIndex()] = nil
	dou.Unlock()
	player.SetIndex(-1)
	game.UnbindPlayerGame(player,dou)
	dou.BroadCastMsg(player,msg.MSG_TYPE_OF_LEAVE_TABLE,"玩家离开游戏")
	fmt.Println("游戏"+strconv.Itoa(dou.id)+"移除玩家"+strconv.Itoa(player.GetPlayerUser().Id))
	return nil
}
//玩家准备
func (dou *Doudizhu) PlayerReady(p game.IPlayer){

	dou.BroadCastMsg(p,msg.MSG_TYPE_OF_READY,"玩家已准备")

	userAllReady := false
	if len(dou.Players) == dou.playerNum{

		for _,p := range dou.Players{
			if p != nil{
				if p.GetReadyStatus(){
					userAllReady = true
				}else{
					userAllReady = false
				}
			}else{
				userAllReady = false
			}
		}
	}
	//用户都准备好了，则发牌
	if userAllReady {
		dou.Lock()
		fmt.Println("桌子"+strconv.Itoa(dou.id)+"的玩家都准备好了")
		dou.IsPlaying = true
		dou.Unlock()
		dou.dealCards()
	}
}
//玩家取消准备
func (dou *Doudizhu)PlayerUnReady(p game.IPlayer){
	dou.BroadCastMsg(p,msg.MSG_TYPE_OF_UN_READY,"玩家取消准备")
}
//发牌
func (dou *Doudizhu) dealCards(){
	dou.shuffleCards()
	dou.Lock()
	dou.playerCards[0] = dou.pokerCards[:17]
	dou.playerCards[1] = dou.pokerCards[17:34]
	dou.playerCards[2] = dou.pokerCards[34:51]
	dou.bottomCards = dou.pokerCards[51:]
	dou.Unlock()

	dou.sortPlayerCards()

	for i,p := range dou.Players{
		p.SetPokerCards(dou.playerCards[i])
	}

	dou.nextCallLoard()
}
//叫地主
func (dou *Doudizhu) nextCallLoard(){

	var currPlayer game.IPlayer
	dou.Lock()
	if dou.CurrPlayerIndex < 0{
		rand.Seed(time.Now().Unix())
		dou.CurrPlayerIndex = int(rand.Int31n(int32(dou.playerNum-1)))
		dou.FirstCallScoreIndex = dou.CurrPlayerIndex
		currPlayer = dou.Players[dou.CurrPlayerIndex]
		dou.Unlock()
	}else{
		dou.Unlock()
		currPlayer = dou.getNextPlayer()
	}
	currPlayer.StartCallScore()
}

func (dou *Doudizhu) PlayerCallScore(currPlayer game.IPlayer,score int){
	if score != 0{
		dou.BroadCastMsg(currPlayer,msg.MSG_TYPE_OF_CALL_SCORE,"用户抢地主")
	}else{
		dou.BroadCastMsg(currPlayer,msg.MSG_TYPE_OF_CALL_SCORE,"用户不抢地主")
	}
	currIndex := dou.getCurrPlayerIndex(currPlayer)
	dou.Lock()
	dou.CalledLoardNum++
	//如果还有第一个人叫地主，其余人都不抢，则叫地主结束，不用等第一个人二次叫地主
	if score == 0 && dou.CalledLoardNum == dou.playerNum {
		//第一个人叫了地主
		if dou.lordIndex == dou.FirstCallScoreIndex{
			dou.Unlock()
			dou.callLoardEnd()
		}else if dou.lordIndex == -1{//无人叫地主,从新发牌
			dou.Unlock()
			dou.restart()
		}
	}else{
		//直到第一个人二次抢地主结束
		if dou.CalledLoardNum == dou.playerNum+1 {
			if score != 0 {
				dou.lordIndex = currIndex
			}
			dou.Unlock()
			dou.callLoardEnd()
		}else{
			if score != 0 {
				dou.lordIndex = currIndex
			}
			dou.Unlock()
			dou.nextCallLoard()
		}
	}
}

func (dou *Doudizhu) restart(){
		dou.Lock()
		dou.CalledLoardNum = 0
		dou.lordIndex  = -1
		dou.CurrPlayerIndex = -1
		dou.Unlock()
		dou.dealCards()
}
func (dou *Doudizhu) callLoardEnd(){
	dou.Lock()
	dou.CurrPlayerIndex = dou.lordIndex
	dou.CalledLoardNum = 0
	dou.Unlock()
	fmt.Println("叫地主结束"+strconv.Itoa(dou.lordIndex)+"成为地主")

	for _,card := range dou.bottomCards{
		dou.playerCards[dou.lordIndex] = append(dou.playerCards[dou.lordIndex],card)
	}

	poker.CommonSort(dou.playerCards[dou.lordIndex])
	dou.Players[dou.lordIndex].SetPokerCards(dou.playerCards[dou.lordIndex])

	dou.BroadCastMsg(dou.Players[dou.lordIndex],msg.MSG_TYPE_OF_SEND_BOTTOM_CARDS,"发放底牌")
	fmt.Println("底牌发送完毕，开始游戏")
	dou.play(dou.Players[dou.lordIndex])
}

func (dou *Doudizhu) play(currPlayer game.IPlayer){
	if currPlayer == nil{
		currPlayer = dou.Players[dou.lordIndex]
	}
	currPlayer.StartPlay()
}

func (dou *Doudizhu) PlayerPlayCards(p game.IPlayer,cardIndexs []int){
	//符合出牌规则才允许出牌
	if dou.getCurrPlayerIndex(p) != dou.CurrPlayerIndex{
		p.PlayCardError("还没到您的出牌次序")
		return
	}

	cards := []*poker.PokerCard{}
	for _,card := range p.GetPlayerCards(cardIndexs){
		//判断是否是之前出过的牌
		cards = append(cards,card)
	}

	lastCards,err := dou.matchRoles(dou.getCurrPlayerIndex(p),cards)
	if err == nil{
		//第一个出牌，或者上一次出牌没人管，或者出牌大于上家，此时满足出牌要求
		if  dou.lastCards == nil || lastCards.PlayerIndex == dou.lastCards.PlayerIndex ||
			(lastCards.CardsType == dou.lastCards.CardsType &&
				lastCards.CardMinAndMax["min"] > dou.lastCards.CardMinAndMax["min"] &&
				lastCards.CardMinAndMax["max"] > dou.lastCards.CardMinAndMax["min"]){

			if lastCards.PlayerCardIndexs == nil{
				lastCards.PlayerCardIndexs = []int{}
			}

			for _,index := range cardIndexs{
				lastCards.PlayerCardIndexs = append(lastCards.PlayerCardIndexs,index)
			}

			isBomb := false
			dou.Lock()
			dou.lastCards = lastCards
			if dou.lastCards.CardsType == game.POKERS_TYPE_COMMON_BOMB ||
				dou.lastCards.CardsType == game.POKERS_TYPE_JOKER_BOMB{
				isBomb = true
				dou.currMulti *= 2
			}
			dou.Unlock()
			if(isBomb){
				dou.BroadCastMsg(p,msg.MSG_TYPE_OF_SCORE_CHANGE,"分数加倍")
			}
			//出牌成功，给前端提示
			p.PlayCardSuccess(cardIndexs)

			dou.BroadCastMsg(p,msg.MSG_TYPE_OF_PLAY_CARD,"玩家出牌")
			//玩家的牌全部出完了
			if p.IsOutOfCards() {
				if dou.OutCardIndexs == nil{
					dou.OutCardIndexs = []int{}
				}

				currIndex := dou.getCurrPlayerIndex(p)
				dou.OutCardIndexs = append(dou.OutCardIndexs,currIndex)

				if currIndex == dou.lordIndex{
					dou.gameOver()
					return
				}else{
					if len(dou.OutCardIndexs) == 2{
						dou.gameOver()
						return
					}
				}
			}
			//下一个玩家出牌
			dou.play(dou.getNextPlayer())
			//todo 此处判断下家的牌是否都小于当前玩家，给出要不起的提示
		}else{
			p.PlayCardError("出牌必须大于上一家")
		}

	}else{
		p.PlayCardError(err.Error())
	}
}

func (dou *Doudizhu) gameOver(){
	//todo结算分数
	if len(dou.OutCardIndexs) == 1 {
		dou.BroadCastMsg(nil,msg.MSG_TYPE_OF_GAME_OVER,"游戏结束,地主胜利")
	}else{
		if dou.OutCardIndexs[1] == dou.lordIndex{
			dou.BroadCastMsg(nil,msg.MSG_TYPE_OF_GAME_OVER,"游戏结束,地主胜利")
		}else{
			dou.BroadCastMsg(nil,msg.MSG_TYPE_OF_GAME_OVER,"游戏结束,农民胜利")
		}
	}
}

func (dou *Doudizhu) PlayerPassCard(currPlayer game.IPlayer){
	//之前出牌是当前玩家则不能过牌，第一个出牌玩家也不能过牌
	if dou.lastCards != nil && dou.getCurrPlayerIndex(currPlayer) != dou.lastCards.PlayerIndex{
		currPlayer.PlayCardSuccess([]int{})
		dou.BroadCastMsg(currPlayer,msg.MSG_TYPE_OF_PASS,"用户过牌")
		dou.play(dou.getNextPlayer())
	}else{
		currPlayer.PlayCardError("第一个出牌的玩家不能过牌")
	}
}

func (dou *Doudizhu) getNextPlayer() game.IPlayer{
	dou.Lock()
	defer dou.Unlock()
	if(dou.CurrPlayerIndex >= dou.playerNum-1){
		dou.CurrPlayerIndex = 0
	}else{
		dou.CurrPlayerIndex++
	}

	return dou.Players[dou.CurrPlayerIndex]
}

func (dou *Doudizhu) getCurrPlayerIndex(currPlayer game.IPlayer) int {
	return currPlayer.GetIndex()
}

func (dou *Doudizhu) BroadCastMsg(player game.IPlayer,msgType int,hints string){

	newMsg := msg.NewBraodCastMsg()
	newMsg.SubMsgType = msgType

	dou.RLock()
	defer dou.RUnlock()

	if player != nil{
		newMsg.PlayerId = player.GetPlayerUser().Id
		for i,p := range dou.Players{
			if p != nil{
				newMsg.PlayerIndexIdDic["id"+strconv.Itoa(p.GetPlayerUser().Id)] = i
			}
		}
	}

	switch msgType{
	case msg.MSG_TYPE_OF_READY:
		newMsg.Msg = strconv.Itoa(player.GetPlayerUser().Id)+"已准备"
	case msg.MSG_TYPE_OF_UN_READY:
		newMsg.Msg = strconv.Itoa(player.GetPlayerUser().Id)+"取消准备"
	case msg.MSG_TYPE_OF_JOIN_TABLE:
		newMsg.Msg = strconv.Itoa(player.GetPlayerUser().Id)+"加入游戏"
	case msg.MSG_TYPE_OF_LEAVE_TABLE:
		newMsg.Msg = strconv.Itoa(player.GetPlayerUser().Id)+"离开游戏"
	case msg.MSG_TYPE_OF_PLAY_CARD:
		newMsg.Msg = strconv.Itoa(player.GetPlayerUser().Id)+"出牌"
		for _,card := range dou.lastCards.Cards{
			newMsg.Cards = append(newMsg.Cards,card)
		}
	case msg.MSG_TYPE_OF_PASS:
		newMsg.Msg = strconv.Itoa(player.GetPlayerUser().Id)+"过牌"
	case msg.MSG_TYPE_OF_CALL_SCORE:
		newMsg.Msg = strconv.Itoa(player.GetPlayerUser().Id)+"叫地主"
		newMsg.Score = dou.currMulti
	case msg.MSG_TYPE_OF_SCORE_CHANGE:
		newMsg.Msg = "基础变动"
		newMsg.Score = dou.currMulti
	case msg.MSG_TYPE_OF_SEND_BOTTOM_CARDS:
		newMsg.Msg = "发放底牌"
		newMsg.Cards = dou.bottomCards
		newMsg.PlayerId = player.GetPlayerUser().Id
	case msg.MSG_TYPE_OF_GAME_OVER:
		newMsg.Msg = "游戏结束，结算积分"
		newMsg.Score = dou.currMulti
		for _,index := range dou.OutCardIndexs{
			if index == dou.lordIndex{
				newMsg.SettleInfoDic["id"+strconv.Itoa(dou.Players[index].GetPlayerUser().Id)] = "+"+strconv.Itoa(dou.currMulti*dou.baseScore*2)
			}else{
				newMsg.SettleInfoDic["id"+strconv.Itoa(dou.Players[index].GetPlayerUser().Id)] = "+"+strconv.Itoa(dou.currMulti*dou.baseScore)
			}
		}

		for i,player := range dou.Players{
			_,ok := newMsg.SettleInfoDic["id"+strconv.Itoa(player.GetPlayerUser().Id)]
			if !ok{
				if i == dou.lordIndex{
					newMsg.SettleInfoDic["id"+strconv.Itoa(dou.Players[i].GetPlayerUser().Id)] = "-"+strconv.Itoa(dou.currMulti*dou.baseScore*2)
				}else{
					newMsg.SettleInfoDic["id"+strconv.Itoa(dou.Players[i].GetPlayerUser().Id)] = "-"+strconv.Itoa(dou.currMulti*dou.baseScore)
				}
			}
		}
	}

	msgJson,err := json.Marshal(newMsg)
	if err != nil {
		panic(err.Error())
	}

	for _,player := range dou.Players{
		if player != nil{
			player.SendMsg(msgJson)
		}
	}
}
func (dou *Doudizhu)GetGameName() string{
	return dou.name
}

func (dou *Doudizhu)GetGameID() int{
	return dou.id
}

func (dou *Doudizhu)GetGameType() int{
	return game.GAME_TYPE_OF_DOUDOZHU
}

//初始化游戏中的牌
func (dou *Doudizhu) initCards(){
	dou.Lock()
	defer dou.Unlock()

	for i:=0;i<dou.deckNum;i++{
		deck := poker.CreateDeck()
		for i,_ := range deck.Cards{
			dou.pokerCards = append(dou.pokerCards,&deck.Cards[i])
		}
	}
}

//洗牌
func (dou *Doudizhu) shuffleCards(){
	dou.Lock()
	defer dou.Unlock()

	rand.Seed(time.Now().Unix())
	for i := len(dou.pokerCards) - 1; i > 0; i-- {
		num := rand.Intn(i + 1)
		dou.pokerCards[i], dou.pokerCards[num] = dou.pokerCards[num], dou.pokerCards[i]
	}
}

func (dou *Doudizhu)HintCards(p game.IPlayer) []int{
	//todo
	return []int{}
}

//检查出牌是否符合规则
func (dou *Doudizhu) matchRoles(currPlayerIndex int,pokers []*poker.PokerCard) (*game.LastCardsType,error){
	return CheckRules(currPlayerIndex,pokers)
}

//对玩家手中扑克牌，按照从小到大排序
func (dou *Doudizhu)sortPlayerCards(){
	dou.Lock()
	defer dou.Unlock()
	for _,cards := range dou.playerCards{
		poker.CommonSort(cards)
	}
}

func (dou *Doudizhu)SayToOthers(p game.IPlayer,msg []byte){
	//todo
}

func (dou *Doudizhu)SayToAnother(p game.IPlayer,otherIndex int,msg []byte){
	//todo
}