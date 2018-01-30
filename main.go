package main

import (
	"fmt"
	"grid-crypto-real/adapter"
	"time"
)

func main() {

	songiriMode := false
	for {
		fmt.Println("==================================================")
		_, err := adapter.UpdateAllInfo()
		if err != nil {
			fmt.Print(err)
		} else {
			adapter.PrintOrderInfo()
			adapter.PrintDeposit()
			songiriMode = adapter.ShouldSongiri()
			res, err := adapter.CancelAllLongOrder()
			if err == nil && res == true {
				//損切りモード
				if songiriMode {
					ret, errCancel := adapter.CancelAllOrder()
					if ret == true && errCancel == nil {
						if adapter.SellAllBtc() {
							songiriMode = false
						}
					}
				} else {
					adapter.BuyFromOrder(adapter.GetOrderFromLastTradePriceAndConfig())
				}

			} else {
				fmt.Printf("キャンセル時にエラーが発生しました.%v\n", err)
			}
		}
		time.Sleep(3 * time.Second) // 3秒休む
	}
	/*
		fmt.Println("↓注文キャンセル")
		for _, order := range activeOrder.Return {
			cancelResponse, errCancel := api.CancelOrder(order.ID)
			if errCancel != nil {
				fmt.Print("キャンセルに失敗", errCancel)
			} else {
				api.PrettyPrint(cancelResponse)
			}
		}

		fmt.Println("↓全BTC成行売り")
		sellResult, errs := api.SellBtc(accountInfo.Return.Deposit.Btc)
		if errs != nil {
			fmt.Println("成行売りに失敗")
			fmt.Println(errs)
		}
		api.PrettyPrint(sellResult)
	*/
}
