package api

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/go-resty/resty"
)

const (
	PrivateAPIEndpoint  = "https://api.zaif.jp/tapi"
	BtcBoardAPIEndpoint = "https://api.zaif.jp/api/1/depth/btc_jpy"
	APIKey              = "d95945e8-d8f0-40dd-90a9-e8aa2e9f15c6"
	APISecret           = ""
	AccountInfoMethod   = "get_info2"
	TradeHistoryMethod  = "trade_history"
	TradeMethod         = "trade"
	ActiveOrderMethod   = "active_orders"
	CancelOrderMethod   = "cancel_order"

	CommentPrefix = "fromBot"
)

//板情報
type Board struct {
	Asks [][]float64 `json:"asks"`
	Bids [][]float64 `json:"bids"`
}

type AccountInfo struct {
	Success int    `json:"success"`
	Error   string `json:"error"`
	Return  struct {
		Funds struct {
			Jpy float64 `json:"jpy"`
			Btc float64 `json:"btc"`
		} `json:"funds"`
		Deposit struct {
			Jpy float64 `json:"jpy"`
			Btc float64 `json:"btc"`
		} `json:"deposit"`
		Rights struct {
			Info         float64 `json:"info"`
			Trade        float64 `json:"trade"`
			Withdraw     float64 `json:"withdraw"`
			PersonalInfo float64 `json:"personal_info"`
			IDInfo       float64 `json:"id_info"`
		} `json:"rights"`
		OpenOrders int `json:"open_orders"`
		ServerTime int `json:"server_time"`
	} `json:"return"`
}

type TradeHistory struct {
	Success int    `json:"success"`
	Error   string `json:"error"`
	Return  []struct {
		ID           int         `json:"id"`
		CurrencyPair string      `json:"currency_pair"`
		Action       string      `json:"action"`
		Amount       float64     `json:"amount"`
		Price        float64     `json:"price"`
		Fee          float64     `json:"fee"`
		FeeAmount    float64     `json:"fee_amount"`
		YourAction   string      `json:"your_action"`
		Bonus        interface{} `json:"bonus"`
		Timestamp    string      `json:"timestamp"`
		Comment      string      `json:"comment"`
	} `json:"return"`
}

type ActiveOrder struct {
	Success int    `json:"success"`
	Error   string `json:"error"`
	Return  []struct {
		ID           int     `json:"id"`
		CurrencyPair string  `json:"currency_pair"`
		Action       string  `json:"action"`
		Amount       float64 `json:"amount"`
		Price        float64 `json:"price"`
		Timestamp    string  `json:"timestamp"`
		Comment      string  `json:"comment"`
	} `json:"return"`
}

type TradeResponse struct {
	Success int    `json:"success"`
	Error   string `json:"error"`
	Return  struct {
		Received float64 `json:"received"`
		Remains  float64 `json:"remains"`
		OrderID  int     `json:"order_id"`
		Funds    struct {
			Jpy float64 `json:"jpy"`
			Btc float64 `json:"btc"`
		} `json:"funds"`
	} `json:"return"`
}

type CancelOrderResponse struct {
	Success int    `json:"success"`
	Error   string `json:"error"`
	Return  struct {
		OrderID int `json:"order_id"`
		Funds   struct {
			Jpy float64 `json:"jpy"`
			Btc float64 `json:"btc"`
		} `json:"funds"`
	} `json:"return"`
}

//nonceはunixtime+incrementalCounterで算出されます
var incrementalCounter float64

//取引履歴を取得します
func GetTradeHistory() (*TradeHistory, error) {
	tradeHistory, err := fetchPrivateAPI(tradeHistroyParamString(), &TradeHistory{}, sanitizeTradeHistoryJsonString)
	if err != nil {
		return nil, err
	}

	return tradeHistory.(*TradeHistory), nil
}

func GetActiveOrder() (*ActiveOrder, error) {
	activeOrder, err := fetchPrivateAPI(activeOrderParamString(), &ActiveOrder{}, sanitizeActiveOrderJsonString)
	if err != nil {
		return nil, err
	}
	return activeOrder.(*ActiveOrder), nil
}

//アカウント情報を取得します
func GetAccountInfo() (*AccountInfo, error) {
	accountInfo, err := fetchPrivateAPI(accountInfoRequestParamString(), &AccountInfo{}, nil)
	if err != nil {
		return nil, err
	}
	return accountInfo.(*AccountInfo), nil
}

func SellBtc(amount float64) (*TradeResponse, error) {
	tradeResponse, err := fetchPrivateAPI(sellBtcParamString(amount), &TradeResponse{}, nil)
	if err != nil {
		return nil, err
	}
	return tradeResponse.(*TradeResponse), nil

}

//板情報を取得します
func GetBoard() (*Board, error) {
	return fetchBoardAPI()
}

func GetLongPosition(price float64, limit float64, amount float64) (*TradeResponse, error) {
	tradeResponse, err := fetchPrivateAPI(LongParamString(price, limit, amount, CommentPrefix), &TradeResponse{}, nil)
	if err != nil {
		return nil, err
	}
	return tradeResponse.(*TradeResponse), nil
}

func CancelOrder(orderID int) (*CancelOrderResponse, error) {
	cancelResponse, err := fetchPrivateAPI(cancelOrderParamString(orderID), &CancelOrderResponse{}, nil)
	if err != nil {
		fmt.Println("注文キャンセル時にエラーが発生しました", err)
		return nil, err
	}
	return cancelResponse.(*CancelOrderResponse), nil

}

//与えられた引数を元にfetchPrivateApiし、interfaceにマーシャルします
func fetchPrivateAPI(queryString string, result interface{}, sanitizeFunction func(string) string) (interface{}, error) {
	resty.SetTimeout(time.Duration(30 * time.Second))
	//jsonレスポンスの補正が不要な場合
	if sanitizeFunction == nil {
		resp, err := resty.R().SetHeader("key", APIKey).
			SetHeader("Content-type", "application/x-www-form-urlencoded").
			SetBody(queryString).
			SetResult(result).
			SetHeader("sign", signature(queryString)).
			Post(PrivateAPIEndpoint)
		if err != nil {
			return nil, err
		}
		return resp.Result(), nil

	}
	//jsonレスポンスが必要な場合
	resp, err := resty.R().SetHeader("key", APIKey).
		SetHeader("Content-type", "application/x-www-form-urlencoded").
		SetBody(queryString).
		SetHeader("sign", signature(queryString)).
		Post(PrivateAPIEndpoint)
	responseBody := sanitizeFunction(resp.String())
	jsonBytes := ([]byte)(responseBody)
	err = json.Unmarshal(jsonBytes, result)
	if err != nil {
		log.Printf("JSON補正に失敗しました", err)
		return nil, err
	}
	return result, nil
}

func fetchBoardAPI() (*Board, error) {
	resp, err := resty.R().SetResult(&Board{}).Get(BtcBoardAPIEndpoint)
	if err != nil {
		return nil, err
	}
	board := resp.Result().(*Board)
	if board == nil || board.Bids == nil || len(board.Bids) == 0 || len(board.Bids[0]) == 0 {
		return nil, errors.New("不正な買い板を取得しました")
	}
	return resp.Result().(*Board), nil
}

//糞構造であるTradeHistoryAPIのjsonレスポンスをまともな形に修正します
func sanitizeTradeHistoryJsonString(body string) string {

	rep := regexp.MustCompile(`"return": {}`)
	body = rep.ReplaceAllString(body, `"return": []`)
	rep = regexp.MustCompile(`"return": {(.*)}}}`)
	body = rep.ReplaceAllString(body, `"return": [$1}]}`)
	rep = regexp.MustCompile(`"(\d*)":(.){`)
	body = rep.ReplaceAllString(body, `{"id": $1, `)
	return body
}

//糞構造であるActiveOrderAPIのjsonレスポンスをまともな形に修正します
func sanitizeActiveOrderJsonString(body string) string {
	return sanitizeTradeHistoryJsonString(body)
}

//accountInfo呼び出しに必要なリクエストパラメータ文字列を取得します
func accountInfoRequestParamString() string {
	base := commonPrivateRequestParamString()
	retString := base + "&method=" + AccountInfoMethod
	return retString
}

func tradeHistroyParamString() string {
	base := commonPrivateRequestParamString()
	retString := base + "&count=3&order=DESC&currency_pair=btc_jpy&method=" + TradeHistoryMethod
	return retString
}

func activeOrderParamString() string {
	base := commonPrivateRequestParamString()
	retString := base + "&count=3&currency_pair=btc_jpy&method=" + ActiveOrderMethod
	return retString
}

func LongParamString(price float64, limit float64, amount float64, comment string) string {
	amount = Round(amount, 1.0, 4)
	base := commonPrivateRequestParamString()
	priceString := strconv.Itoa(Round5(price))
	limitString := strconv.Itoa(Round5(limit))
	amoutString := strconv.FormatFloat(amount, 'f', 4, 64)
	retString := base + "&currency_pair=btc_jpy&action=bid&price=" + priceString + "&limit=" + limitString + "&amount=" + amoutString + "&comment=" + comment + "&method=" + TradeMethod
	return retString
}

func cancelOrderParamString(orderID int) string {
	base := commonPrivateRequestParamString()
	retString := base + "&order_id=" + strconv.Itoa(orderID) + "&method=" + CancelOrderMethod
	return retString
}

func sellBtcParamString(amount float64) string {
	base := commonPrivateRequestParamString()
	amoutString := strconv.FormatFloat(amount, 'f', 4, 64)
	retString := base + "&currency_pair=btc_jpy&action=ask&price=5&amount=" + amoutString + "&comment=sell_with_market_price&method=" + TradeMethod
	return retString
}

//プライベートAPI呼び出しに必要な共通のリクエストパラ���ータを取得します
func commonPrivateRequestParamString() string {
	incrementalCounter += 0.001
	nonce := strconv.FormatFloat(float64(time.Now().Unix())+incrementalCounter, 'f', 3, 64)
	retString := "nonce=" + nonce
	return retString
}

//queryStringの署名文字列を返却しま���
func signature(queryString string) string {
	hash := hmac.New(sha512.New, []byte(APISecret))
	hash.Write([]byte(queryString))
	signature := hex.EncodeToString(hash.Sum(nil))
	return signature
}

//PrettyPrint オブジェクトなどを可視性高くprintします
func PrettyPrint(v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	println(string(b))
}

//5単位に丸めます
func Round5(num float64) int {
	ceilPrice := int(math.Trunc(num))
	ceilPrice = (ceilPrice/5)*5 + 5
	return ceilPrice
}

func Round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}
