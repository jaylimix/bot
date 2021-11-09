package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
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

var green_candle bool

var red_candle bool

var the_number_of_times_the_current_candle_is_longer_than_others int

func main() {

	run_http("get", "/fapi/v1/exchangeInfo", "exchange")

	for _, v := range exchange.Symbols {

		if v.Symbol == "BTCSTUSDT" {
			continue
		}
		// fmt.Println(v.Symbol, v.PricePrecision, v.QuantityPrecision)

		run_http("get", "/fapi/v1/ticker/price?symbol="+v.Symbol, "ticker")

		ticker_price, _ = strconv.ParseFloat(ticker.Price, 32)

		run_http("get", "/fapi/v1/klines?limit=100&interval=1h&symbol="+v.Symbol, "klines")

		parse_ohlc_then_compare_current_hours_candle_length_with_the_rest()

		if the_number_of_times_the_current_candle_is_longer_than_others >= 95 {

			// fmt.Println("Longer than others by 90 times")
			fmt.Println(the_number_of_times_the_current_candle_is_longer_than_others)
			fmt.Println(v.Symbol)
			dt := time.Now()
			fmt.Println(dt.Format("2006.01.02 15"))

			if green_candle {
				fmt.Println("LONG LONG LONG")
			}

			if red_candle {
				fmt.Println("SHORT SHORT SHORT")
			}

			fmt.Println()
		}

		// RESET COUNTER TO 0 FOR NEXT PAIR
		// RESET GREEN AND RED CANDLES
		the_number_of_times_the_current_candle_is_longer_than_others = 0
		green_candle = false
		red_candle = false

		// break
	}

}

func run_http(http_type string, endpoint string, identifier string) {

	var response *http.Response

	var err error

	if http_type == "get" {
		response, err = http.Get(base_url + endpoint)
	}

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
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
		// fmt.Println(res)
		json.Unmarshal([]byte(res), &ticker)
	}
	if identifier == "klines" {
		json.Unmarshal([]byte(res), &klines)
	}
}

func parse_ohlc_then_compare_current_hours_candle_length_with_the_rest() {

	current_candle := true

	var other_candles_high_vs_low float64

	// fmt.Println(len(klines))
	// os.Exit(1)

	for i := len(klines) - 1; i >= 0; i-- {
		// for index, candle_info := range klines {

		open, _ := strconv.ParseFloat(klines[i][1], 32)
		high, _ := strconv.ParseFloat(klines[i][2], 32)
		low, _ := strconv.ParseFloat(klines[i][3], 32)
		// close, _ := strconv.ParseFloat(candle_info[4], 32)

		// fmt.Println(ticker_price)

		// fmt.Println(close)

		// fmt.Println(index)

		// continue

		if current_candle {

			// fmt.Println("Current candle")
			// fmt.Println(open)
			// fmt.Println(high)
			// fmt.Println(low)

			if ticker_price > open {
				// CURRENT CANDLE IS GREEN
				// COMPARE LOW WITH TICKER PRICE

				green_candle = true

				current_candle_length = math.Abs(ticker_price - low)
			}
			if ticker_price < open {
				// CURRENT CANDLE IS RED
				// COMPARED HIGH WITH TICKER PRICE

				red_candle = true

				current_candle_length = math.Abs(ticker_price - high)
			}

			// fmt.Println(current_candle_length)

			current_candle = false

			// fmt.Println()

		} else {

			// fmt.Println("Not current candle")
			// fmt.Println(open)
			// fmt.Println(high)
			// fmt.Println(low)

			other_candles_high_vs_low = math.Abs(high - low)

			// fmt.Println(high_vs_low)
		}

		if current_candle_length > other_candles_high_vs_low {

			the_number_of_times_the_current_candle_is_longer_than_others++
		}

	}

}
