package game

import (
	"chessSever/program/game/poker"
	"github.com/sirupsen/logrus"
)

const (
	GAME_TYPE_OF_DOUDOZHU = iota
	GAME_TYPE_OF_SHENGJI
	GAME_TYPE_OF_BAOHUANG
	GAME_TYPE_OF_ZHAJINHUA
)

var gameIDNameDic map[int]string

func init(){
	gameIDNameDic = make(map[int]string)
	gameIDNameDic[GAME_TYPE_OF_DOUDOZHU] = "斗地主"
	gameIDNameDic[GAME_TYPE_OF_SHENGJI] = "升级"
	gameIDNameDic[GAME_TYPE_OF_BAOHUANG] = "保皇"
	gameIDNameDic[GAME_TYPE_OF_ZHAJINHUA] = "斗地主"
}

func GetGameName(gameID int) string{
	name,ok := gameIDNameDic[gameID]
	if ok{
		return name
	}else{
		logrus.Error("未定义游戏名称")
		return "未定义游戏名称"
	}
}

//游戏使用接口类型，便于实现多态
type IGame interface {
	GetGameID() int                                                          //获取游戏id
	GetGameName() string                                                     //获取游戏名称
	GetGameType() int                                                        //获取游戏类型
	GetLastCard() *LastCardsType                                             //获取游戏最后出的牌

	AddPlayer(p IPlayer) error                                               //游戏添加玩家
	RemovePlayer(p IPlayer) error                                            //游戏移除玩家
	SayToOthers(p IPlayer,msg []byte)										 //跟其他玩家说话
	SayToAnother(p IPlayer,otherIndex int,msg []byte)						 //跟一个玩家说话
	PlayerReady(p IPlayer)													 //玩家准备
	PlayerUnReady(p IPlayer)											     //玩家取消准备
	PlayerCallScore(p IPlayer,score int)								     //玩家叫地主
	PlayerPlayCards(p IPlayer,cardsIndex []int)								 //玩家出牌
	PlayerPassCard(p IPlayer)												 //玩家过牌
	HintCards(p IPlayer) []int  									         //提示玩家可出的牌
	BroadCastMsg(p IPlayer,msgType int,msg string)

}

type LastCardsType struct{
	PlayerCardIndexs []int		 	//扑克牌在出牌玩家所有牌中的index
	PlayerIndex int              	//出牌的玩家index
	CardsType int				 	//牌的类型
	Cards []*poker.PokerCard	    //出的牌
	CardMinAndMax map[string]int    //出的牌的map,min为主体牌的最小值，max为有主体牌的最大值，比如三带一，三为主体牌，带一为辅助牌
}

func NewLastCards(playerIndex int,cardsType int,cards []*poker.PokerCard,minValue int,maxValue int) *LastCardsType {
	lastCards :=  &LastCardsType{
		PlayerIndex:playerIndex,
		CardsType:cardsType,
		Cards:cards,
		CardMinAndMax:make(map[string]int),
	}
	lastCards.CardMinAndMax["min"] = minValue
	lastCards.CardMinAndMax["max"] = maxValue
	return lastCards
}