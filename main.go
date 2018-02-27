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
		adapter.CancelLowestOrderIfOrderFull()
		adapter.PrintOrderInfo()
		adapter.PrintDeposit()

		if config.Debug == 1 {
			continue
		}

		orders := adapter.GetOrderFromLastTradePriceAndConfig()
		for _, order := range orders {
			if adapter.HasRangeBuyOrder(order.Price) {
				fmt.Println("すでに同様の注文/もしくは高い注文があるためスキップします", order.Price)
			} else {
				adapter.BuyFromOrder(order)
			}
		}

	}
}
