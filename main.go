package main

import (
	"fmt"
	"grid-crypto-real/adapter"
	"grid-crypto-real/config"
	"time"
)

func main() {
	songiriMode := false
	for {
		time.Sleep(1 * time.Second) // 3秒休む
		_, err := adapter.UpdateAllInfo()
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println("==================================================")
		adapter.PrintOrderInfo()
		adapter.PrintDeposit()
		songiriMode = adapter.ShouldSongiri()

		if config.Debug == 1 {
			continue
		}

		//損切りモード
		if songiriMode {
			ret, errCancel := adapter.CancelAllOrder()
			if ret == true && errCancel == nil {
				if adapter.SellAllBtc() {
					songiriMode = false
				}
			}
			continue
		}
		order := adapter.GetOrderFromLastTradePriceAndConfig()
		if adapter.IsSameOrHigherOrderExist(order) && adapter.GetLongOrderCount() == 1 {
			fmt.Println("すでに同様の注文/もしくは高い注文があるためスキップします")
			continue
		}
		res, err := adapter.CancelAllLongOrder()
		if err != nil && !res {
			fmt.Printf("キャンセル時にエラーが発生しました.%v\n", err)
			continue
		}
		adapter.BuyFromOrder(order)
	}
}
