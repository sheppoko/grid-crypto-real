package util

import (
	"fmt"
	"log"
	"os"
)

func WriteFile(filePath string, body string) {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		//エラー処理
		log.Fatal(err)
	}
	defer file.Close()
	fmt.Fprintln(file, body) //書き込み
}
