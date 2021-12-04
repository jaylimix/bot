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

var stop_loss_percentage = 0.01 * 5

var usd_per_trade = 1.00 * 20

var overextended_percent = 0.01 * 10

var close_position_hours_passed = int64(60 * 60 * 10)

var limit = "100"

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
	UpdateTime  int
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

var current_candle_length float64

var long bool

var short bool

var symbol string

var price_precision string

var quantity_precision string

var quantity_after_per_trade_divide_by_price float64

var quantity string

var new_order NewOrder

var stop_order StopOrder

var stopPrice string

var netout = false

func main() {
	lambda.Start(handleRequest)
}

func handleRequest() {

	run_http("/fapi/v1/exchangeInfo", "exchange")

	run_http("/fapi/v2/account", "account")

	for _, v := range exchange.Symbols {

		symbol = v.Symbol

		// fmt.Println(symbol)

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

		price_precision = strconv.Itoa(v.PricePrecision)

		quantity_precision = strconv.Itoa(v.QuantityPrecision)

		if this_symbol_already_has_open_position(symbol) {

			continue
		}

		if run_http("/fapi/v1/ticker/price?symbol="+symbol, "ticker") {
			continue
		}

		ticker_price, _ = strconv.ParseFloat(ticker.Price, 32)

		quantity_after_per_trade_divide_by_price = usd_per_trade / ticker_price

		minimum_quantity_per_order := 0.00

		if v.QuantityPrecision == 3 {
			minimum_quantity_per_order = 0.001

		} else if v.QuantityPrecision == 2 {
			minimum_quantity_per_order = 0.01

		} else if v.QuantityPrecision == 1 {
			minimum_quantity_per_order = 0.1

		} else {
			minimum_quantity_per_order = 1
		}

		if quantity_after_per_trade_divide_by_price < minimum_quantity_per_order {
			continue
		}

		if run_http("/fapi/v1/klines?limit="+limit+"&interval=1h&symbol="+symbol, "klines") {
			continue
		}

		current_candle_is_not_longer_than_most := parse_ohlc_then_compare_current_hours_candle_length_with_the_rest()

		if current_candle_is_not_longer_than_most {
			reset_variables_for_next_pair()
			continue
		}

		current_candle_is_overextended := is_the_current_candle_overextended()

		if current_candle_is_overextended {
			reset_variables_for_next_pair()
			continue
		}

		if long || short {

			// fmt.Println("Symbol is " + symbol)

			if symbol_already_has_open_position(symbol) {
				continue
			}

			new_order_created := run_http("/fapi/v1/order", "new_order")

			if new_order_created {
				run_http("/fapi/v1/order", "stop_order")
			}

		}

		fmt.Println()

		reset_variables_for_next_pair()

	}
}

func reset_variables_for_next_pair() {
	long = false
	short = false
	new_order.Symbol = ""
	stop_order.Symbol = ""
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

		if netout {

			quantity_in_float, _ := strconv.ParseFloat(quantity, 32)

			quantity_in_float *= 2

			quantity = fmt.Sprintf(decimal_format, quantity_in_float)
		}

		if long {
			// fmt.Println("LONG LONG LONG")
			query_string = "symbol=" + symbol + "&side=BUY&type=MARKET&quantity=" + quantity + "&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)
		}

		if short {
			// fmt.Println("SHORT SHORT SHORT")
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

	if identifier == "stop_order" {

		var query_string string

		var decimal_format string

		if long {

			fmt.Println("LONG Stop Order")

			decimal_format = "%." + price_precision + "f"

			stopPrice = fmt.Sprintf(decimal_format, ticker_price*(1-stop_loss_percentage))

			query_string = "symbol=" + symbol + "&stopPrice=" + stopPrice + "&closePosition=true&side=SELL&type=STOP_MARKET&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)

		}

		if short {

			fmt.Println("SHORT Stop Order")

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
		// os.Exit(1)
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

			// fmt.Println(symbol + "  " + string(responseData))

			return false
		}

		// fmt.Println(string(responseData))

		fmt.Println(new_order.Symbol + "  " + time.Now().Format("2006.01.02 15"))

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

		// fmt.Println(string(responseData))

		fmt.Println(stop_order.Symbol + " " + time.Now().Format("2006.01.02 15"))

		fmt.Println(stop_order.StopPrice)
	}

	if identifier == "account" {
		json.Unmarshal(responseData, &account)
	}

	if identifier == "close_order" {

		fmt.Println(string(responseData))

		fmt.Println()
	}

	return false
}

func parse_ohlc_then_compare_current_hours_candle_length_with_the_rest() bool {

	current_candle := true

	var other_candles_high_vs_low float64

	var number_of_times_current_candle_is_longer_than_the_rest int

	for i := len(klines) - 1; i >= 0; i-- {

		high, _ := strconv.ParseFloat(klines[i][2], 32)
		low, _ := strconv.ParseFloat(klines[i][3], 32)

		if current_candle {

			length_of_ticker_to_high := math.Abs(ticker_price - high)

			length_of_low_to_ticker := math.Abs(ticker_price - low)

			if length_of_low_to_ticker > length_of_ticker_to_high {

				long = true

				current_candle_length = length_of_low_to_ticker
			}

			if length_of_ticker_to_high > length_of_low_to_ticker {

				short = true

				current_candle_length = length_of_ticker_to_high
			}

			if !long && !short {

				return true
			}

			current_candle = false

			continue

		}

		other_candles_high_vs_low = math.Abs(high - low)

		if current_candle_length > other_candles_high_vs_low {

			number_of_times_current_candle_is_longer_than_the_rest++
		}
	}

	if number_of_times_current_candle_is_longer_than_the_rest < 95 {

		return true
	}

	return false
}

func is_the_current_candle_overextended() bool {

	current_candle_open, _ := strconv.ParseFloat(klines[len(klines)-1][1], 32)

	open_of_last_x_candle, _ := strconv.ParseFloat(klines[len(klines)-20][1], 32)

	if long && (current_candle_open-open_of_last_x_candle)/open_of_last_x_candle >= overextended_percent {

		return true
	}

	if short && (open_of_last_x_candle-current_candle_open)/open_of_last_x_candle >= overextended_percent {

		return true
	}

	return false
}

func this_symbol_already_has_open_position(symbol string) bool {

	for _, position := range account.Positions {

		position_amount, _ := strconv.ParseFloat(position.PositionAmt, 32)

		if position_amount != 0.0 && symbol == position.Symbol {

			consider_closing_this_position(position.Symbol, position.UpdateTime, position.PositionAmt)

			return true
		}
	}

	return false
}

func symbol_already_has_open_position(symbol string) bool {

	for _, position := range account.Positions {

		if symbol == position.Symbol {

			position_amount, _ := strconv.ParseFloat(position.PositionAmt, 32)

			if position_amount > 0 && long || position_amount < 0 && short {

				return true
			}

			if position_amount > 0 && short || position_amount < 0 && long {

				fmt.Println("Open opposing direction for " + symbol)

				netout = true

				return false
			}

			consider_closing_this_position(position.Symbol, position.UpdateTime, position.PositionAmt)
		}
	}

	return false
}

func consider_closing_this_position(symbol string, update_time int, amount string) {

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

		run_http("/fapi/v1/order", "close_order")
	}
}
