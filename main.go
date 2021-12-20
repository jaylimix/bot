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

const STOP_LOSS_PERCENTAGE = 0.10

const USD_PER_TRADE = 50.00

const CLOSE_POSITION_HOURS_PASSED = int64(60 * 60 * 5)

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

var klines [][]string

var account Account

var exchange Exchange

var ticker Ticker

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

		// if symbol != "CELOUSDT" {
		// 	continue
		// }

		if check_symbol_already_has_open_position_and_consider_closing_position(symbol) {
			continue
		}

		// if total_number_of_positions() >= CHANGE_CONDITION_NUMBER {
		// 	continue
		// }

		if TURN_OFF_OPENING_NEW_POSITIONS {
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

		if total_number_of_positions() < CHANGE_CONDITION_NUMBER {

			if !beat_other_candles_highest_or_lowest_and_is_longest_and_set_long_or_short() {
				continue
			}
		}

		if total_number_of_positions() >= CHANGE_CONDITION_NUMBER {

			if !ticker_is_halfway_and_is_longest_and_set_long_or_short() {
				continue
			}
		}

		quantity_precision = strconv.Itoa(v.QuantityPrecision)

		if run_http_and_return_false_if_error("/fapi/v1/order", "new_order") {

			price_precision = strconv.Itoa(v.PricePrecision)

			run_http_and_return_false_if_error("/fapi/v1/order", "stop_order")
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

		json.Unmarshal(response_data, &stop_order)

		if stop_order.Symbol == "" {

			fmt.Println(symbol + " " + string(response_data))

			fmt.Println(ticker_price)

			fmt.Println(stopPrice)
		}

		fmt.Println("Short order for " + stop_order.Symbol + " " + time.Now().Format("2006.01.02 15"))

		fmt.Println(stop_order.StopPrice)
	}

	if identifier == "account" {

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

		json.Unmarshal(response_data, &account)

		return true
	}

	return false
}

func beat_other_candles_highest_or_lowest_and_is_longest_and_set_long_or_short() bool {

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

		// Check that current candle length is the longest //

		if current_candle_length < other_candles_length {

			return false
		}

		// Check that ticker is highest //

		if ticker_price > current_candle_open {

			if other_candles_high > current_candle_high {

				return false
			}

			short = true
		}

		// Check that ticker is lowest //

		if ticker_price < current_candle_open {

			if other_candles_low < current_candle_low {

				return false
			}

			long = true
		}

	}

	return true
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

func close_this_position_if_next_hour(symbol string, update_time int64, amount string) {

	update_time = update_time / 1000

	position_open_time := (time.Unix(update_time, 0)).String()

	time_now := time.Now().Format("2006.01.02 15")

	// position_open_hour, _ := strconv.Atoi(position_open_time[11:13])

	// time_now_hour, _ := strconv.Atoi(time_now[11:13])

	// if time_now_hour-position_open_hour >= 3 {

	if position_open_time[11:13] != time_now[11:13] {

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

func ticker_is_halfway_and_is_longest_and_set_long_or_short() bool {

	var current_candle_high float64

	var current_candle_low float64

	var current_candle_open float64

	current_candle := true

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

		if current_candle_length < other_candles_length {

			return false
		}

		other_candles_high := high

		other_candles_low := low

		// If green candle check that ticker is highest //

		if ticker_price > current_candle_open {

			if other_candles_high > current_candle_high {

				return false
			}
		}

		// If red candle check that ticker is lowest //

		if ticker_price < current_candle_open {

			if other_candles_low < current_candle_low {

				return false
			}
		}
	}

	// Check that ticker price is halfway between high and low //

	halfway_price := (current_candle_high + current_candle_low) / 2

	// Green candle //

	if ticker_price > current_candle_open {

		if ticker_price <= halfway_price {

			short = true

			return true
		}
	}

	// Red candle //

	if ticker_price < current_candle_open {

		if ticker_price >= halfway_price {

			long = true

			return true
		}
	}

	return false
}

func candle_is_long_and_ticker_is_halfway_and_set_long_or_short() bool {

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

				return false
			}
		}

		// If red candle returns false if ticker is not the lowest //

		if ticker_price < current_candle_open {

			if other_candles_low < current_candle_low {

				return false
			}
		}
	}

	// Return false if current cannot beat most other candles, for example 34 < 35 //

	if count_beat_other_candles < len(klines)-5 {

		return false
	}

	// Check that ticker price is halfway between high and low //

	halfway_price := (current_candle_high + current_candle_low) / 2

	// Green candle //

	if ticker_price > current_candle_open {

		if ticker_price <= halfway_price {

			short = true

			return true
		}
	}

	// Red candle //

	if ticker_price < current_candle_open {

		if ticker_price >= halfway_price {

			long = true

			return true
		}
	}

	return false
}
