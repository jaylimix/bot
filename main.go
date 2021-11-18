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
)

// var base_url string = "https://fapi.binance.com"
var base_url string = "https://testnet.binancefuture.com"

var klines [][]string

type Symbols struct {
	Symbol            string
	PricePrecision    int
	QuantityPrecision int
}

type Exchange struct {
	Symbols []Symbols
}

var exchange Exchange

type Ticker struct {
	Price string
}

var ticker Ticker

var ticker_price float64

var current_candle_length float64

var long bool

var short bool

var limit string = "100"

var symbol string

var usd_per_trade = 60.00

var minimum_quantity_per_order float64

var price_precision string

var quantity_precision string

var stop_loss_percentage = 0.01 * 3

var quantity_after_per_trade_divide_by_price float64

var quantity string

func main() {

	run_http("get", "/fapi/v1/exchangeInfo", "exchange")

	for _, v := range exchange.Symbols {

		symbol = v.Symbol

		// if symbol == "BTCSTUSDT" || symbol == "XRPBUSD" || symbol == "BTCBUSD" || symbol == "ETHBUSD" {
		// 	continue
		// }

		// if symbol != "BTCUSDT" {
		// 	continue
		// }

		// fmt.Println(v.Symbol, v.PricePrecision, v.QuantityPrecision)

		// os.Exit(1)

		fmt.Println(v.Symbol, v.PricePrecision, v.QuantityPrecision)

		if run_http("get", "/fapi/v1/ticker/price?symbol="+symbol, "ticker") {
			continue
		}

		ticker_price, _ = strconv.ParseFloat(ticker.Price, 32)

		quantity_after_per_trade_divide_by_price = usd_per_trade / ticker_price

		// fmt.Println(quantity_after_per_trade_divide_by_price)

		// fmt.Println()

		minimum_quantity_per_order := 0.00

		// price_precision = strconv.Itoa(v.PricePrecision)

		// fmt.Println(v.QuantityPrecision)

		// fmt.Println(quantity_precision)

		// break

		if v.QuantityPrecision == 3 {
			minimum_quantity_per_order = 0.001

		} else if v.QuantityPrecision == 2 {
			minimum_quantity_per_order = 0.01

		} else if v.QuantityPrecision == 1 {
			minimum_quantity_per_order = 0.1

		} else {
			minimum_quantity_per_order = 1
		}

		quantity_precision = strconv.Itoa(v.QuantityPrecision)

		if quantity_after_per_trade_divide_by_price < minimum_quantity_per_order {
			fmt.Println("Skip " + symbol)
			fmt.Println()
			continue
		}

		// quantity = strconv.ParseFloat(quantity_after_per_trade_divide_by_price, 64)

		long = true

		run_http("post", "/fapi/v1/order", "new_order")

		fmt.Println()

		run_http("post", "/fapi/v1/order", "stop_order")

		break

		if run_http("get", "/fapi/v1/klines?limit="+limit+"&interval=1h&symbol="+symbol, "klines") {
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

		fmt.Println(symbol)
		dt := time.Now()
		fmt.Println(dt.Format("2006.01.02 15"))
		fmt.Println(ticker_price)

		if long || short {

			run_http("post", "/fapi/v1/order", "new_order")

			run_http("post", "/fapi/v1/order", "stop_order")
		}

		fmt.Println()

		reset_variables_for_next_pair()

	}

	// main()
}

func reset_variables_for_next_pair() {
	long = false
	short = false
}

func run_http(http_type string, endpoint string, identifier string) bool {

	var response *http.Response

	var err error

	if http_type == "get" {
		response, err = http.Get(base_url + endpoint)
	}

	if http_type == "post" && identifier == "new_order" {

		api_key := "14b417a306cd837d3c3ec9cee6f6c4ca2468b0b06a6028c3978ba8a6287ac5c2"

		api_secret := "a6d2fabd26dbe982d0b104e41e115352dc24dfda6726725f153c05aaa6440ca3"

		var query_string string

		decimal_format := "%." + quantity_precision + "f"

		quantity = fmt.Sprintf(decimal_format, quantity_after_per_trade_divide_by_price)

		if long {
			fmt.Println("LONG LONG LONG")
			query_string = "symbol=" + symbol + "&side=BUY&type=MARKET&quantity=" + quantity + "&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)
		}

		if short {
			fmt.Println("SHORT SHORT SHORT")
			query_string = "symbol=" + symbol + "&side=SELL&type=MARKET&quantity=" + quantity + "&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)
		}

		fmt.Println(query_string)

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

	if http_type == "post" && identifier == "stop_order" {

		api_key := "14b417a306cd837d3c3ec9cee6f6c4ca2468b0b06a6028c3978ba8a6287ac5c2"

		api_secret := "a6d2fabd26dbe982d0b104e41e115352dc24dfda6726725f153c05aaa6440ca3"

		var query_string string

		var stopPrice string

		var decimal_format string

		fmt.Println(stop_loss_percentage)

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

		fmt.Println(query_string)

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

	if err != nil {

		fmt.Println(err.Error())
		// os.Exit(1)
		return true
	}

	responseData, err := ioutil.ReadAll(response.Body)

	if err != nil {
		log.Fatal(err)
	}

	res := string(responseData)

	if identifier == "exchange" {
		json.Unmarshal([]byte(res), &exchange)
	}
	if identifier == "ticker" {
		json.Unmarshal([]byte(res), &ticker)
	}
	if identifier == "klines" {
		json.Unmarshal([]byte(res), &klines)
	}
	if identifier == "new_order" {
		fmt.Println(res)
	}
	if identifier == "stop_order" {
		fmt.Println(res)
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

		// fmt.Println(symbol)
		// fmt.Println(number_of_times_current_candle_is_longer_than_the_rest)
		return true
	}

	return false
}

func is_the_current_candle_overextended() bool {

	// THIS SECTION CHECKS WHETHER THE CURRENT CANDLE HAS OVEREXTENDED
	// COMPARE TO THE LAST 10TH CANDLE, IF IT HAS ALREADY PUMP 10% THEN ALGO WILL PREVENT LONG
	// COMPARE TO THE LAST 10TH CANDLE, IF IT HAS ALREADY DUMP 10% THEN ALGO WILL PREVENT SHORT
	// CRITERIA IS THAT CURRENT CANDLE OPEN IS >= 10% COMPARED TO LAST 10TH CANDLE

	current_candle_open, _ := strconv.ParseFloat(klines[len(klines)-1][1], 32)

	open_of_last_x_candle, _ := strconv.ParseFloat(klines[len(klines)-20][1], 32)

	if long && (current_candle_open-open_of_last_x_candle)/open_of_last_x_candle >= 0.1 {

		// fmt.Println(symbol)

		// dt := time.Now()

		// fmt.Println(dt.Format("2006.01.02 15"))

		// fmt.Println("Cannot long because the current open compared with the last 10th candle open is already 10%")

		return true
	}

	if short && (open_of_last_x_candle-current_candle_open)/open_of_last_x_candle >= 0.1 {

		// fmt.Println(symbol)

		// dt := time.Now()

		// fmt.Println(dt.Format("2006.01.02 15"))

		// fmt.Println("Cannot short because the current open caompared with the last 10th candle open is already 10%")

		return true
	}

	return false
}
