package adapter

import (
	"errors"
	"fmt"
	"grid-crypto-real/api"
	"grid-crypto-real/config"
)

//注文の構造体です。注文時に使用します
type Order struct {
	Price  float64 `json:"price"`
	Limit  float64 `json:"limit"`
	Amount float64 `json:"amount"`
	UseJpy float64 `json:"useJpy"`
}

//もろもろ情報
var latestAccountInfo *api.AccountInfo
var latestActiveOrderInfo *api.ActiveOrder
var latestTradeHistory *api.TradeHistory
var latestBoard *api.Board

//口座情報/未約定注文/取引履歴/板情報を取得し最新化します。成功した場合はtrueを返します。
func UpdateAllInfo() (bool, error) {
	latestAccountInfo = &api.AccountInfo{}
	latestActiveOrderInfo = &api.ActiveOrder{}
	latestTradeHistory = &api.TradeHistory{}
	latestBoard = &api.Board{}

	ai, errAI := api.GetAccountInfo()
	if errAI != nil {
		return false, errAI
	}

	if ai.Success != 1 {
		return false, errors.New(ai.Error)
	}
	latestAccountInfo = ai

	ao, errAO := api.GetActiveOrder()
	if errAO != nil {
		return false, errAO
	}
	if ao.Success != 1 {
		return false, errors.New(ao.Error)
	}
	latestActiveOrderInfo = ao

	th, errTH := api.GetTradeHistory()
	if errTH != nil {
		return false, errTH
	}
	if th.Success != 1 {
		return false, errors.New(th.Error)
	}
	latestTradeHistory = th

	b, errB := api.GetBoard()
	if errB != nil {
		return false, errB
	}
	latestBoard = b

	return true, nil
}

//保有資産をログに出力します
func PrintDeposit() {
	fmt.Println("----保有資産-----")
	fmt.Printf("btc:%f(1BTC=%f)\n", latestAccountInfo.Return.Deposit.Btc)
	fmt.Printf("jpy:%f\n", latestAccountInfo.Return.Deposit.Jpy)
	fmt.Printf("総資産:%f\n", (latestBoard.Asks[0][0]*latestAccountInfo.Return.Deposit.Btc)+latestAccountInfo.Return.Deposit.Jpy)
	fmt.Printf("pos:%d\n", GetPositionNum())
	fmt.Println("-------------\n")
}

//取引履歴をログに出力します
func PrintTradeInfo() {
	fmt.Println("----取引履歴-----")
	api.PrettyPrint(latestTradeHistory)
	fmt.Println("-----------------")
}

//未約定注文をログに出力します
func PrintOrderInfo() {
	fmt.Println("----未約定注文-----")
	api.PrettyPrint(latestActiveOrderInfo)
	fmt.Println("-------------------")
}

//取引履歴と設定を元に適切な注文を作成します。
//ポジションが無い場合は成行買い、
//ポジションが存在する場合は最後に約定した価格から、下のグリッド価格を算出し注文structを作成します
func GetOrderFromLastTradePriceAndConfig() *Order {
	positionNum := GetPositionNum()
	remainJpy := GetRemainJpy()
	remainPosition := config.MaxPositionCount - positionNum
	useJpy := remainJpy / float64(remainPosition)
	price := 0.0

	if GetLastPrice() == 0 || GetPositionNum() == 0 {
		return GetMarketPriceOrder(useJpy, 50000000)
	}

	if isLastTradeLong() {
		price = GetLastPrice() * (1 - config.BuyRange)
	} else {
		price = GetLastPrice() / (1 + config.TakeProfitRange)
	}

	//ポジション数が1の時は現時点価格からレンジ下げた価格を購入価格とする
	if GetPositionNum() == 1 {
		price = latestBoard.Bids[0][0] * (1 - config.BuyRange)
	}
	limit := price * (1 + config.TakeProfitRange)

	//TODO:コメントに指値額をいれてそれをGetLastPrice()としてあつかう
	retOrder := &Order{
		price,
		limit,
		useJpy / price,
		useJpy,
	}
	return retOrder
}

//板を参照し使用するJPYから適切な注文を作成します
func GetMarketPriceOrder(useJpy float64, limit float64) *Order {
	canAmount := 0.0
	lastJpy := useJpy
	retPrice := 0.0
	for _, row := range latestBoard.Asks {
		tmpLastJpy := lastJpy
		lastJpy -= row[0] * row[1]
		if lastJpy < 0 {
			canAmount += tmpLastJpy / row[0]
			retPrice = row[0]
			break
		} else {
			canAmount += row[1]
		}
	}
	retOrder := &Order{
		retPrice,
		limit,
		canAmount,
		useJpy,
	}
	return retOrder
}

//使用可能な残りJPYを返却します
func GetRemainJpy() float64 {
	return latestAccountInfo.Return.Funds.Jpy
}

//通っていない買い注文があるかどうかを返却します
func GetLongOrderCount() int {
	ret := 0
	for _, order := range latestActiveOrderInfo.Return {
		if order.Action == "bid" {
			ret++
		}
	}
	return ret
}

//最後の取引の約定価格を返却します,action:ask(売り) bid(買い)
func GetLastPrice() float64 {

	if len(latestTradeHistory.Return) == 0 {
		return 0
	}

	lastTrade := latestTradeHistory.Return[0]
	return lastTrade.Price
}

//最後の取引履歴が買いかどうかを返却します
func isLastTradeLong() bool {
	if len(latestTradeHistory.Return) == 0 {
		return false
	}

	lastTrade := latestTradeHistory.Return[0]
	if lastTrade.YourAction == "bid" {
		return true
	}
	return false

}

//現在保有しているポジション数を返却します（=通っていない売り注文の数です）
func GetPositionNum() int {
	count := 0
	for _, order := range latestActiveOrderInfo.Return {
		if order.Action == "ask" {
			count++
		}
	}
	return count
}

//注文structから実際に注文を行います
//最大ポジション数に達した場合は実行されません
func BuyFromOrder(order *Order) {

	if GetPositionNum() >= config.MaxPositionCount {
		fmt.Println("最大ポジション数に達しました。")
		return
	}

	res, err := api.GetLongPosition(order.Price, order.Limit, order.Amount)
	if err != nil {
		fmt.Print(err)
		return
	}
	if res.Success != 1 {
		fmt.Print(res.Error)
		return
	}
	fmt.Printf("注文に成功しました。\n")
	api.PrettyPrint(order)
}

//現時点と同じか、高い価格の注文があるかどうかを返却します
func IsSameOrHigherOrderExist(order *Order) bool {

	amount := float64(api.Round(order.Amount, 1.0, 4))
	price := float64(api.Round5(order.Price))

	for _, serverOrder := range latestActiveOrderInfo.Return {
		if serverOrder.Amount == amount && serverOrder.Price == price && serverOrder.Action == "bid" {
			return true
		}
		if serverOrder.Price > order.Price && serverOrder.Action == "bid" {
			return true
		}
	}

	return false
}

//全てのLong注文をキャンセルします
func CancelAllLongOrder() (bool, error) {
	for _, order := range latestActiveOrderInfo.Return {
		if order.Action == "bid" {
			cancelResponse, errCancel := api.CancelOrder(order.ID)
			if errCancel != nil {
				return false, errCancel
			}
			if cancelResponse.Success != 1 {
				return false, errors.New(cancelResponse.Error)
			}
		}
	}
	fmt.Println("全ての買い注文をキャンセルしました")
	return true, nil
}

//全ての注文をキャンセルします
func CancelAllOrder() (bool, error) {
	for _, order := range latestActiveOrderInfo.Return {
		cancelResponse, errCancel := api.CancelOrder(order.ID)
		if errCancel != nil {
			return false, errCancel
		}
		if cancelResponse.Success != 1 {
			return false, errors.New(cancelResponse.Error)
		}
		fmt.Println("注文を1本キャンセルしました")
	}
	fmt.Println("全ての注文をキャンセルしました")
	return true, nil

}

//全てのBTCを成行で売却します
func SellAllBtc() bool {
	res, err := api.SellBtc(latestAccountInfo.Return.Deposit.Btc)
	if err != nil {
		fmt.Println("BTC売却に失敗しました")
		fmt.Println(err)
		return false
	}
	if res.Success != 1 {
		fmt.Println("BTC売却に失敗しました")
		fmt.Println(res.Error)
		return false
	}
	fmt.Println("BTCを売却しました")
	return true
}

func ShouldSongiri() bool {
	if GetPositionNum() >= config.MaxPositionCount {
		if latestBoard.Asks[0][0] < GetLastPrice()*(1-config.BuyRange) {
			return true
		}
	}
	return false
}
