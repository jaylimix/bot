package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

// var api_key = "14b417a306cd837d3c3ec9cee6f6c4ca2468b0b06a6028c3978ba8a6287ac5c2"

var api_key = "Iv49dUKHcJ8rqypuu4SW9Xa0nLYgv75b2QtdvQtcIeP7EnhTkmRanZxtA8yQMMi7"

// var api_secret = "a6d2fabd26dbe982d0b104e41e115352dc24dfda6726725f153c05aaa6440ca3"

var api_secret = "u5ASQxwwYC4b1TJqUvLGZsqwXSXdqdIsj7uKf8X8nkXZ13xAe8gPVzc1Bq4mGF0L"

// var base_url = "https://testnet.binancefuture.com"

var base_url = "https://fapi.binance.com"

var stop_loss_percentage = 0.10

var usd_per_trade = 50.00

var close_position_hours_passed = int64(60 * 60 * 24)

var limit = "72"

type Symbols struct {
	Symbol            string
	PricePrecision    int
	QuantityPrecision int
}

type Exchange struct {
	Symbols []Symbols
}

type Positions struct {
	Symbol      string
	PositionAmt string
	UpdateTime  int64
}

type Account struct {
	Positions []Positions
}

type NewOrder struct {
	Symbol string
}

type StopOrder struct {
	Symbol    string
	StopPrice string
}

type Ticker struct {
	Price string
}

var klines [][]string

var account Account

var exchange Exchange

var ticker Ticker

var ticker_price float64

var long bool

var short bool

var symbol string

var price_precision string

var quantity_precision string

var quantity_after_per_trade_divide_by_price float64

var quantity string

var minimum_quantity_per_order float64

var new_order NewOrder

var stop_order StopOrder

var stopPrice string

var the_longest_candle float64

var current_candle_length float64

var turn_off_opening_new_positions = false

func main() {
	lambda.Start(handleRequest)
}

func handleRequest() {

	run_http("/fapi/v1/exchangeInfo", "exchange")

	run_http("/fapi/v2/account", "account")

	for _, v := range exchange.Symbols {

		long = false
		short = false
		new_order.Symbol = ""
		stop_order.Symbol = ""
		symbol = v.Symbol

		switch symbol {
		case "BTCUSDT_211231":
			continue
		case "ETHUSDT_211231":
			continue
		case "SOLUSDT":
			continue
		case "ADAUSDT":
			continue
		case "FTTUSDT":
			continue
		case "XRPUSDT":
			continue
		case "DOGEUSDT":
			continue
		case "BNBUSDT":
			continue
		case "ETHUSDT":
			continue
		case "BTCUSDT":
			continue
		}

		// if symbol != "CELOUSDT" {
		// 	continue
		// }

		if check_symbol_already_has_open_position_and_consider_closing_position(symbol) {
			continue
		}

		if turn_off_opening_new_positions {
			continue
		}

		if run_http("/fapi/v1/ticker/price?symbol="+symbol, "ticker") {
			continue
		}

		ticker_price, _ = strconv.ParseFloat(ticker.Price, 32)

		quantity_after_per_trade_divide_by_price = usd_per_trade / ticker_price

		set_minimum_quantity_per_order(v.QuantityPrecision)

		if quantity_after_per_trade_divide_by_price < minimum_quantity_per_order {
			continue
		}

		if run_http("/fapi/v1/klines?limit="+limit+"&interval=1h&symbol="+symbol, "klines") {
			continue
		}

		if !current_candle_is_the_longest_and_highest_lowest_and_set_long_or_short() {
			continue
		}

		quantity_precision = strconv.Itoa(v.QuantityPrecision)

		if run_http("/fapi/v1/order", "new_order") {

			price_precision = strconv.Itoa(v.PricePrecision)

			run_http("/fapi/v1/order", "stop_order")
		}

	}
}

func set_minimum_quantity_per_order(quantity_precision int) {

	if quantity_precision == 3 {
		minimum_quantity_per_order = 0.001

	} else if quantity_precision == 2 {
		minimum_quantity_per_order = 0.01

	} else if quantity_precision == 1 {
		minimum_quantity_per_order = 0.1

	} else {
		minimum_quantity_per_order = 1
	}
}

func run_http(endpoint string, identifier string) bool {

	var response *http.Response

	var err error

	if identifier == "exchange" || identifier == "ticker" || identifier == "klines" {
		response, err = http.Get(base_url + endpoint)
	}

	if identifier == "position" {

		query_string := "symbol=" + symbol + "&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)

		mac := hmac.New(sha256.New, []byte(api_secret))

		mac.Write([]byte(query_string))

		signature := "&signature=" + hex.EncodeToString(mac.Sum(nil))

		client := &http.Client{}

		req, err := http.NewRequest("GET", base_url+endpoint+"?"+query_string+signature, nil)

		if err != nil {
			fmt.Println(err)
		}

		req.Header.Set("X-MBX-APIKEY", api_key)

		response, err = client.Do(req)
	}

	if identifier == "new_order" {

		var query_string string

		decimal_format := "%." + quantity_precision + "f"

		quantity = fmt.Sprintf(decimal_format, quantity_after_per_trade_divide_by_price)

		if long {
			query_string = "symbol=" + symbol + "&side=BUY&type=MARKET&quantity=" + quantity + "&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)
		}

		if short {
			query_string = "symbol=" + symbol + "&side=SELL&type=MARKET&quantity=" + quantity + "&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)
		}

		mac := hmac.New(sha256.New, []byte(api_secret))

		mac.Write([]byte(query_string))

		signature := "&signature=" + hex.EncodeToString(mac.Sum(nil))

		client := &http.Client{}

		req, err := http.NewRequest("POST", base_url+endpoint+"?"+query_string+signature, nil)

		if err != nil {
			fmt.Println(err)
		}

		req.Header.Set("X-MBX-APIKEY", api_key)

		response, err = client.Do(req)
	}

	if identifier == "close_order" {

		var side string

		if long {

			side = "SELL"
		}

		if short {

			side = "BUY"
		}

		query_string := "symbol=" + symbol + "&side=" + side + "&type=MARKET&quantity=" + quantity + "&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)

		fmt.Print(query_string)

		mac := hmac.New(sha256.New, []byte(api_secret))

		mac.Write([]byte(query_string))

		signature := "&signature=" + hex.EncodeToString(mac.Sum(nil))

		client := &http.Client{}

		req, err := http.NewRequest("POST", base_url+endpoint+"?"+query_string+signature, nil)

		if err != nil {
			fmt.Println(err)
		}

		req.Header.Set("X-MBX-APIKEY", api_key)

		response, err = client.Do(req)
	}

	if identifier == "cancel_order" {

		query_string := "symbol=" + symbol + "&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)

		fmt.Print(query_string)

		mac := hmac.New(sha256.New, []byte(api_secret))

		mac.Write([]byte(query_string))

		signature := "&signature=" + hex.EncodeToString(mac.Sum(nil))

		client := &http.Client{}

		req, err := http.NewRequest("DELETE", base_url+endpoint+"?"+query_string+signature, nil)

		if err != nil {
			fmt.Println(err)
		}

		req.Header.Set("X-MBX-APIKEY", api_key)

		response, err = client.Do(req)
	}

	if identifier == "stop_order" {

		var query_string string

		var decimal_format string

		if long {

			decimal_format = "%." + price_precision + "f"

			stopPrice = fmt.Sprintf(decimal_format, ticker_price*(1-stop_loss_percentage))

			query_string = "symbol=" + symbol + "&stopPrice=" + stopPrice + "&closePosition=true&side=SELL&type=STOP_MARKET&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)

		}

		if short {

			decimal_format = "%." + price_precision + "f"

			stopPrice = fmt.Sprintf(decimal_format, ticker_price*(1+stop_loss_percentage))

			query_string = "symbol=" + symbol + "&stopPrice=" + stopPrice + "&closePosition=true&side=BUY&type=STOP_MARKET&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)

		}

		// fmt.Println(query_string)

		mac := hmac.New(sha256.New, []byte(api_secret))

		mac.Write([]byte(query_string))

		signature := "&signature=" + hex.EncodeToString(mac.Sum(nil))

		client := &http.Client{}

		req, err := http.NewRequest("POST", base_url+endpoint+"?"+query_string+signature, nil)

		if err != nil {
			fmt.Println(err)
		}

		req.Header.Set("X-MBX-APIKEY", api_key)

		response, err = client.Do(req)
	}

	if identifier == "account" {

		query_string := "symbol=" + symbol + "&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)

		mac := hmac.New(sha256.New, []byte(api_secret))

		mac.Write([]byte(query_string))

		signature := "&signature=" + hex.EncodeToString(mac.Sum(nil))

		client := &http.Client{}

		req, err := http.NewRequest("GET", base_url+endpoint+"?"+query_string+signature, nil)

		if err != nil {
			fmt.Println(err)
		}

		req.Header.Set("X-MBX-APIKEY", api_key)

		response, err = client.Do(req)
	}

	if err != nil {

		fmt.Println(err.Error())

		return true
	}

	responseData, err := ioutil.ReadAll(response.Body)

	if err != nil {
		log.Fatal(err)
	}

	if identifier == "exchange" {
		json.Unmarshal(responseData, &exchange)
	}

	if identifier == "ticker" {
		json.Unmarshal(responseData, &ticker)
	}

	if identifier == "klines" {
		json.Unmarshal(responseData, &klines)
	}

	if identifier == "new_order" {

		json.Unmarshal(responseData, &new_order)

		if new_order.Symbol == "" {

			return false
		}

		fmt.Println("New order for " + new_order.Symbol + "  " + time.Now().Format("2006.01.02 15"))

		return true
	}

	if identifier == "stop_order" {

		json.Unmarshal(responseData, &stop_order)

		if stop_order.Symbol == "" {

			fmt.Println(symbol + " " + string(responseData))

			fmt.Println(ticker_price)

			fmt.Println(stopPrice)

			return false
		}

		fmt.Println("Short order for " + stop_order.Symbol + " " + time.Now().Format("2006.01.02 15"))

		fmt.Println(stop_order.StopPrice)
	}

	if identifier == "account" {
		json.Unmarshal(responseData, &account)
	}

	if identifier == "close_order" {

		fmt.Println(string(responseData))

		return true
	}
	if identifier == "cancel_order" {

		fmt.Println(string(responseData))
	}

	return false
}

func set_current_candles_length_and_longest_candles_length() {

	var current_candle = true

	for i := len(klines) - 1; i >= 0; i-- {

		high, _ := strconv.ParseFloat(klines[i][2], 32)
		low, _ := strconv.ParseFloat(klines[i][3], 32)

		if current_candle {
			the_longest_candle = math.Abs(high - low)
			current_candle_length = the_longest_candle
			current_candle = false
		}

		other_candles_length := math.Abs(high - low)

		if other_candles_length > the_longest_candle {
			the_longest_candle = other_candles_length
		}
	}
}

func current_candle_length_is_the_longest() bool {

	current_candle := true

	var current_candle_length float64

	for i := len(klines) - 1; i >= 0; i-- {

		high, _ := strconv.ParseFloat(klines[i][2], 32)
		low, _ := strconv.ParseFloat(klines[i][3], 32)

		if current_candle {

			open, _ := strconv.ParseFloat(klines[i][1], 32)

			if ticker_price > open {

				current_candle_length = math.Abs(ticker_price - low)
			}

			if ticker_price < open {

				current_candle_length = math.Abs(ticker_price - high)
			}

			current_candle = false

			continue
		}

		other_candles_high_vs_low := math.Abs(high - low)

		if current_candle_length < other_candles_high_vs_low {
			return false
		}
	}

	return true
}

func current_candle_is_the_longest_and_highest_lowest_and_set_long_or_short() bool {

	var current_candle_high float64

	var current_candle_low float64

	var current_candle_open float64

	current_candle := true

	for i := len(klines) - 1; i >= 0; i-- {

		high, _ := strconv.ParseFloat(klines[i][2], 32)

		low, _ := strconv.ParseFloat(klines[i][3], 32)

		if current_candle {

			current_candle_open, _ = strconv.ParseFloat(klines[i][1], 32)

			if ticker_price > current_candle_open {

				current_candle_length = math.Abs(ticker_price - low)

				current_candle_high = high

			}

			if ticker_price < current_candle_open {

				current_candle_length = math.Abs(ticker_price - high)

				current_candle_low = low

			}

			current_candle = false

			continue
		}

		other_candles_high := high

		other_candles_low := low

		other_candles_length := math.Abs(high - low)

		if current_candle_length < other_candles_length {

			return false
		}

		if ticker_price > current_candle_open {

			if other_candles_high > current_candle_high {

				return false
			}

			short = true
		}

		if ticker_price < current_candle_open {

			if other_candles_low < current_candle_low {

				return false
			}

			long = true
		}

	}

	return true
}

func slope_pattern_found_and_set_long_or_short() bool {

	open_of_first_candle, _ := strconv.ParseFloat(klines[0][1], 32)

	open_of_current_candle, _ := strconv.ParseFloat(klines[len(klines)-1][1], 32)

	// If green candle and line sloping up we go short
	if ticker_price > open_of_current_candle {

		if open_of_first_candle < open_of_current_candle {

			short = true

			return true
		}
	}

	// If red candle and line slopping down we go long
	if ticker_price < open_of_current_candle {

		if open_of_first_candle > open_of_current_candle {

			long = true

			return true
		}
	}

	return false
}

func check_symbol_already_has_open_position_and_consider_closing_position(symbol string) bool {

	for _, position := range account.Positions {

		position_amount, _ := strconv.ParseFloat(position.PositionAmt, 32)

		if position_amount != 0.0 && symbol == position.Symbol {

			// consider_closing_this_position(position.Symbol, position.UpdateTime, position.PositionAmt)

			close_this_position_if_next_hour(position.Symbol, position.UpdateTime, position.PositionAmt)

			return true
		}
	}

	return false
}

func consider_closing_this_position(symbol string, update_time int64, amount string) {

	update_time = update_time / 1000

	time_diff := time.Now().Unix() - int64(update_time)

	if time_diff >= close_position_hours_passed {

		if string(amount[0]) == "-" {

			short = true

			quantity = amount[1:]

		} else {

			long = true

			quantity = amount
		}

		if run_http("/fapi/v1/order", "close_order") {

			run_http("/fapi/v1/allOpenOrders", "cancel_order")
		}

	}
}

func close_this_position_if_next_hour(symbol string, update_time int64, amount string) {

	update_time = update_time / 1000

	position_open_time := (time.Unix(update_time, 0)).String()

	time_now := time.Now().Format("2006.01.02 15")

	if position_open_time[11:13] != time_now[11:13] {

		if string(amount[0]) == "-" {

			short = true

			quantity = amount[1:]

		} else {

			long = true

			quantity = amount
		}

		if run_http("/fapi/v1/order", "close_order") {

			run_http("/fapi/v1/allOpenOrders", "cancel_order")
		}
	}
}
