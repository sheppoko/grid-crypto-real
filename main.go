package main

import (
	"fmt"
	"grid-crypto-real/adapter"
	"grid-crypto-real/config"
	"time"
)

func main() {
	for {
		time.Sleep(2 * time.Second) // 休む
		_, err := adapter.UpdateAllInfo()
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println("==================================================")
		adapter.PrintOrderInfo()
		adapter.PrintDeposit()

		if config.Debug == 1 {
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
