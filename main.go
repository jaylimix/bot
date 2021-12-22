package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

// const API_KEY = "294254234f002b644ad82c7e1fbc444b28a022518e7b624c9eb66a4d986f94c4"

// const API_SECRET = "9b1cfa1c87be05be52e26dfc1ea6bbdaa3a255971c248bca19e435931b12178b"

// const BASE_URL = "https://testnet.binancefuture.com"

const API_KEY = "Iv49dUKHcJ8rqypuu4SW9Xa0nLYgv75b2QtdvQtcIeP7EnhTkmRanZxtA8yQMMi7"

const API_SECRET = "u5ASQxwwYC4b1TJqUvLGZsqwXSXdqdIsj7uKf8X8nkXZ13xAe8gPVzc1Bq4mGF0L"

const BASE_URL = "https://fapi.binance.com"

const STOP_LOSS_PERCENTAGE = 0.02

const USD_PER_TRADE = 50.00

const CLOSE_POSITION_HOURS_PASSED = int64(3600)

const LIMIT = "40"

const CHANGE_CONDITION_NUMBER = 10

const TURN_OFF_OPENING_NEW_POSITIONS = false

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

type AllOrders struct {
	Symbol     string
	UpdateTime int
}

var klines [][]string

var account Account

var exchange Exchange

var ticker Ticker

var all_orders []AllOrders

var long bool

var short bool

var stopPrice string

var side string

var symbol string

var price_precision string

var quantity_precision string

var quantity string

var ticker_price float64

var quantity_after_per_trade_divide_by_price float64

var current_candle_length float64

var minimum_quantity_per_order float64

var new_order NewOrder

var stop_order StopOrder

func main() {
	lambda.Start(handle_request)
}

func handle_request() {

	if !run_http_and_return_false_if_error("/fapi/v1/exchangeInfo", "exchange") {
		os.Exit(1)
	}

	if !run_http_and_return_false_if_error("/fapi/v2/account", "account") {
		os.Exit(1)
	}

	for _, v := range exchange.Symbols {

		long = false
		short = false
		side = ""
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

		// if symbol != "GTCUSDT" {
		// 	continue
		// }

		if check_symbol_already_has_open_position_and_consider_closing_position(symbol) {
			continue
		}

		if !run_http_and_return_false_if_error("/fapi/v1/allOrders", "all_orders") {
			continue
		}

		if TURN_OFF_OPENING_NEW_POSITIONS {
			continue
		}

		if previous_order_is_the_same_hour() {
			continue
		}

		if !run_http_and_return_false_if_error("/fapi/v1/ticker/price?symbol="+symbol, "ticker") {
			continue
		}

		ticker_price, _ = strconv.ParseFloat(ticker.Price, 32)

		quantity_after_per_trade_divide_by_price = USD_PER_TRADE / ticker_price

		set_minimum_quantity_per_order(v.QuantityPrecision)

		if quantity_after_per_trade_divide_by_price < minimum_quantity_per_order {
			continue
		}

		if !run_http_and_return_false_if_error("/fapi/v1/klines?LIMIT="+LIMIT+"&interval=1h&symbol="+symbol, "klines") {
			continue
		}

		set_long_or_short_when_candle_is_long_and_ticker_is_one_third_and_is_highest_or_lowest()

		if !long && !short {
			continue
		}

		price_precision = strconv.Itoa(v.PricePrecision)

		if run_http_and_return_false_if_error("/fapi/v1/order", "stop_order") {

			quantity_precision = strconv.Itoa(v.QuantityPrecision)

			run_http_and_return_false_if_error("/fapi/v1/order", "new_order")
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

func run_http_and_return_false_if_error(endpoint string, identifier string) bool {

	if identifier == "exchange" || identifier == "ticker" || identifier == "klines" {

		response, err := http.Get(BASE_URL + endpoint)

		if err != nil {

			fmt.Println(err.Error())

			return false
		}

		response_data, err := ioutil.ReadAll(response.Body)

		if err != nil {

			fmt.Println(err.Error())

			return false
		}

		if identifier == "exchange" {

			json.Unmarshal(response_data, &exchange)
		}

		if identifier == "ticker" {

			json.Unmarshal(response_data, &ticker)
		}

		if identifier == "klines" {

			json.Unmarshal(response_data, &klines)
		}

		return true
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

		mac := hmac.New(sha256.New, []byte(API_SECRET))

		mac.Write([]byte(query_string))

		signature := "&signature=" + hex.EncodeToString(mac.Sum(nil))

		client := &http.Client{}

		req, err := http.NewRequest("POST", BASE_URL+endpoint+"?"+query_string+signature, nil)

		if err != nil {

			fmt.Println(err)

			return false
		}

		req.Header.Set("X-MBX-APIKEY", API_KEY)

		response, err := client.Do(req)

		if err != nil {

			fmt.Println(err)

			return false
		}

		response_data, err := ioutil.ReadAll(response.Body)

		if err != nil {

			fmt.Println(err)

			return false
		}

		json.Unmarshal(response_data, &new_order)

		if new_order.Symbol == "" {

			return false
		}

		fmt.Println("New order for " + new_order.Symbol + "  " + time.Now().Format("2006.01.02 15"))

		return true
	}

	if identifier == "close_order" {

		query_string := "symbol=" + symbol + "&side=" + side + "&type=MARKET&quantity=" + quantity + "&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)

		mac := hmac.New(sha256.New, []byte(API_SECRET))

		mac.Write([]byte(query_string))

		signature := "&signature=" + hex.EncodeToString(mac.Sum(nil))

		client := &http.Client{}

		req, err := http.NewRequest("POST", BASE_URL+endpoint+"?"+query_string+signature, nil)

		if err != nil {

			fmt.Println(err)

			return false
		}

		req.Header.Set("X-MBX-APIKEY", API_KEY)

		response, err := client.Do(req)

		if err != nil {

			fmt.Println(err)

			return false
		}

		response_data, err := ioutil.ReadAll(response.Body)

		if err != nil {

			fmt.Println(err)

			return false
		}

		fmt.Println(string(response_data))

		return true
	}

	if identifier == "cancel_order" {

		query_string := "symbol=" + symbol + "&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)

		fmt.Print(query_string)

		mac := hmac.New(sha256.New, []byte(API_SECRET))

		mac.Write([]byte(query_string))

		signature := "&signature=" + hex.EncodeToString(mac.Sum(nil))

		client := &http.Client{}

		req, err := http.NewRequest("DELETE", BASE_URL+endpoint+"?"+query_string+signature, nil)

		if err != nil {

			fmt.Println(err)
		}

		req.Header.Set("X-MBX-APIKEY", API_KEY)

		response, err := client.Do(req)

		if err != nil {

			fmt.Println(err)
		}

		response_data, err := ioutil.ReadAll(response.Body)

		if err != nil {

			fmt.Println(err)
		}

		fmt.Println(string(response_data))
	}

	if identifier == "stop_order" {

		var decimal_format string

		var query_string string

		if long {

			decimal_format = "%." + price_precision + "f"

			stopPrice = fmt.Sprintf(decimal_format, ticker_price*(1-STOP_LOSS_PERCENTAGE))

			query_string = "symbol=" + symbol + "&stopPrice=" + stopPrice + "&closePosition=true&side=SELL&type=STOP_MARKET&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)

		}

		if short {

			decimal_format = "%." + price_precision + "f"

			stopPrice = fmt.Sprintf(decimal_format, ticker_price*(1+STOP_LOSS_PERCENTAGE))

			query_string = "symbol=" + symbol + "&stopPrice=" + stopPrice + "&closePosition=true&side=BUY&type=STOP_MARKET&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)

		}

		mac := hmac.New(sha256.New, []byte(API_SECRET))

		mac.Write([]byte(query_string))

		signature := "&signature=" + hex.EncodeToString(mac.Sum(nil))

		client := &http.Client{}

		req, err := http.NewRequest("POST", BASE_URL+endpoint+"?"+query_string+signature, nil)

		if err != nil {

			fmt.Println(err)

			return false
		}

		req.Header.Set("X-MBX-APIKEY", API_KEY)

		response, err := client.Do(req)

		if err != nil {

			fmt.Println(err)

			return false
		}

		response_data, err := ioutil.ReadAll(response.Body)

		if err != nil {

			fmt.Println(err)

			return false
		}

		json.Unmarshal(response_data, &stop_order)

		if stop_order.Symbol == "" {

			fmt.Println(symbol + " " + string(response_data))

			return false
		}

		fmt.Println("Short order " + stop_order.StopPrice + " for " + stop_order.Symbol + " " + time.Now().Format("2006.01.02 15"))

		return true
	}

	if identifier == "account" || identifier == "all_orders" {

		query_string := "symbol=" + symbol + "&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)

		mac := hmac.New(sha256.New, []byte(API_SECRET))

		mac.Write([]byte(query_string))

		signature := "&signature=" + hex.EncodeToString(mac.Sum(nil))

		client := &http.Client{}

		req, err := http.NewRequest("GET", BASE_URL+endpoint+"?"+query_string+signature, nil)

		if err != nil {

			fmt.Println(err)

			return false
		}

		req.Header.Set("X-MBX-APIKEY", API_KEY)

		response, err := client.Do(req)

		if err != nil {

			fmt.Println(err)

			return false
		}

		response_data, err := ioutil.ReadAll(response.Body)

		if err != nil {

			fmt.Println(err)

			return false
		}

		if identifier == "account" {

			json.Unmarshal(response_data, &account)
		}

		if identifier == "all_orders" {

			json.Unmarshal(response_data, &all_orders)
		}

		return true
	}

	return false
}

func check_symbol_already_has_open_position_and_consider_closing_position(symbol string) bool {

	for _, position := range account.Positions {

		position_amount, _ := strconv.ParseFloat(position.PositionAmt, 32)

		if symbol == position.Symbol && position_amount != 0.0 {

			consider_closing_this_position(position.Symbol, position.UpdateTime, position.PositionAmt)

			// close_this_position_if_next_hour(position.Symbol, position.UpdateTime, position.PositionAmt)

			return true
		}
	}

	return false
}

func previous_order_is_the_same_hour() bool {

	// fmt.Println(all_orders[len(all_orders)-1].UpdateTime)

	update_time := all_orders[len(all_orders)-1].UpdateTime / 1000

	// fmt.Println(update_time)

	previous_order_time := (time.Unix(int64(update_time), 0)).String()

	// fmt.Println(previous_order_time)

	time_now := time.Now().Format("2006.01.02 15")

	// fmt.Println(time_now)

	previous_order_hour, _ := strconv.Atoi(previous_order_time[11:13])

	// fmt.Println(previous_order_hour)

	time_now_hour, _ := strconv.Atoi(time_now[11:13])

	// fmt.Println(time_now_hour)

	if previous_order_hour == time_now_hour {
		return true
	}

	return false
}

func set_long_or_short_when_candle_is_long_and_ticker_is_one_third_and_is_highest_or_lowest() {

	var current_candle_high float64

	var current_candle_low float64

	var current_candle_open float64

	current_candle := true

	var count_beat_other_candles int

	for i := len(klines) - 1; i >= 0; i-- {

		high, _ := strconv.ParseFloat(klines[i][2], 32)

		low, _ := strconv.ParseFloat(klines[i][3], 32)

		if current_candle {

			current_candle_open, _ = strconv.ParseFloat(klines[i][1], 32)

			current_candle_high = high

			current_candle_low = low

			current_candle_length = math.Abs(current_candle_high - current_candle_low)

			current_candle = false

			continue
		}

		// Check that current candle length is the longest //

		other_candles_length := math.Abs(high - low)

		if current_candle_length > other_candles_length {

			count_beat_other_candles++
		}

		// Set other candles high and low

		other_candles_high := high

		other_candles_low := low

		// If green candle returns false if ticker is not the highest //

		if ticker_price > current_candle_open {

			if other_candles_high > current_candle_high {

				return
			}
		}

		// If red candle returns false if ticker is not the lowest //

		if ticker_price < current_candle_open {

			if other_candles_low < current_candle_low {

				return
			}
		}
	}

	// Current candle length beats all or is second place: 40 >= 39 or 39 >= 39 //

	if count_beat_other_candles >= len(klines)-1 {

		// Check that ticker price is one third between high and low //

		// Green candle //

		if ticker_price > current_candle_open {

			one_third_price := current_candle_high - ((current_candle_high - current_candle_low) / 3)

			if ticker_price <= one_third_price {

				short = true

				return
			}
		}

		// Red candle //

		if ticker_price < current_candle_open {

			one_third_price := current_candle_low + ((current_candle_high - current_candle_low) / 3)

			if ticker_price >= one_third_price {

				long = true

				return
			}
		}
	}
}

func consider_closing_this_position(symbol string, update_time int64, amount string) {

	update_time = update_time / 1000

	time_diff := time.Now().Unix() - int64(update_time)

	if time_diff >= CLOSE_POSITION_HOURS_PASSED {

		if string(amount[0]) == "-" {

			side = "BUY"

			quantity = amount[1:]

		} else {

			side = "SELL"

			quantity = amount
		}

		if run_http_and_return_false_if_error("/fapi/v1/order", "close_order") {

			run_http_and_return_false_if_error("/fapi/v1/allOpenOrders", "cancel_order")
		}

	}
}

func total_number_of_positions() int {

	var number_of_positions int

	for _, position := range account.Positions {

		position_amount, _ := strconv.ParseFloat(position.PositionAmt, 32)

		if position_amount != 0.0 {

			number_of_positions++
		}
	}

	return number_of_positions
}
