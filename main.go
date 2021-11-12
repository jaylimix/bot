package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"reflect"
	"strconv"
	"time"
)

var base_url string = "https://fapi.binance.com"

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

var same_pair_same_hour = [][]string{}

// var same_pair_same_hour_found bool

func main() {

	run_http("get", "/fapi/v1/exchangeInfo", "exchange")

	for _, v := range exchange.Symbols {

		if v.Symbol == "BTCSTUSDT" || v.Symbol == "XRPBUSD" || v.Symbol == "BTCBUSD" || v.Symbol == "ETHBUSD" {
			continue
		}

		// if v.Symbol != "BTCUSDT" {
		// 	continue
		// }

		if same_pair_same_hour_found(v.Symbol) {
			continue
		}

		// fmt.Println(v.Symbol, v.PricePrecision, v.QuantityPrecision)

		if run_http("get", "/fapi/v1/ticker/price?symbol="+v.Symbol, "ticker") {
			continue
		}

		ticker_price, _ = strconv.ParseFloat(ticker.Price, 32)

		if run_http("get", "/fapi/v1/klines?limit="+limit+"&interval=1h&symbol="+v.Symbol, "klines") {
			continue
		}

		current_candle_is_not_longer_than_most := parse_ohlc_then_compare_current_hours_candle_length_with_the_rest(v.Symbol)

		if current_candle_is_not_longer_than_most {
			reset_variables_for_next_pair()
			continue
		}

		current_candle_is_overextended := is_the_current_candle_overextended(v.Symbol)

		if current_candle_is_overextended {
			reset_variables_for_next_pair()
			continue
		}

		fmt.Println(v.Symbol)
		dt := time.Now()
		fmt.Println(dt.Format("2006.01.02 15"))
		fmt.Println(ticker_price)

		if long {

			same_pair_same_hour = append(same_pair_same_hour, []string{v.Symbol, dt.Format("2006.01.02 15"), "LONG"})
			fmt.Println("LONG LONG LONG")
		}

		if short {
			same_pair_same_hour = append(same_pair_same_hour, []string{v.Symbol, dt.Format("2006.01.02 15"), "SHORT"})
			fmt.Println("SHORT SHORT SHORT")
		}

		fmt.Println()

		reset_variables_for_next_pair()
	}

	main()
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
	return false
}

func parse_ohlc_then_compare_current_hours_candle_length_with_the_rest(symbol string) bool {

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

func is_the_current_candle_overextended(symbol string) bool {

	// THIS SECTION CHECKS WHETHER THE CURRENT CANDLE HAS OVEREXTENDED
	// COMPARE TO THE LAST 10TH CANDLE, IF IT HAS ALREADY PUMP 10% THEN ALGO WILL PREVENT LONG
	// COMPARE TO THE LAST 10TH CANDLE, IF IT HAS ALREADY DUMP 10% THEN ALGO WILL PREVENT SHORT
	// CRITERIA IS THAT CURRENT CANDLE OPEN IS >= 10% COMPARED TO LAST 10TH CANDLE

	current_candle_open, _ := strconv.ParseFloat(klines[len(klines)-1][1], 32)

	open_of_last_10th_candle, _ := strconv.ParseFloat(klines[len(klines)-10][1], 32)

	if long && (current_candle_open-open_of_last_10th_candle)/open_of_last_10th_candle >= 0.1 {

		// fmt.Println(symbol)

		// dt := time.Now()

		// fmt.Println(dt.Format("2006.01.02 15"))

		// fmt.Println("Cannot long because the current open compared with the last 10th candle open is already 10%")

		return true
	}

	if short && (open_of_last_10th_candle-current_candle_open)/open_of_last_10th_candle >= 0.1 {

		// fmt.Println(symbol)

		// dt := time.Now()

		// fmt.Println(dt.Format("2006.01.02 15"))

		// fmt.Println("Cannot short because the current open caompared with the last 10th candle open is already 10%")

		return true
	}

	return false
}

func same_pair_same_hour_found(symbol string) bool {

	dt := time.Now()

	var long_or_short_found bool

	if len(same_pair_same_hour) > 0 {

		for _, spsh := range same_pair_same_hour {

			if reflect.DeepEqual(spsh, []string{symbol, dt.Format("2006.01.02 15"), "LONG"}) {
				fmt.Println("LONG FOUND for " + symbol)
				long_or_short_found = true
				continue
			}

			if reflect.DeepEqual(spsh, []string{symbol, dt.Format("2006.01.02 15"), "SHORT"}) {
				fmt.Println("SHORT FOUND for " + symbol)
				long_or_short_found = true
				continue
			}

		}
	}

	if long_or_short_found {
		return true
	}

	return false
}
