package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func main() {

	var err error

	base_url := "https://testnet.binancefuture.com"

	endpoint := "/fapi/v1/order"

	query_string := "symbol=BCHUSDT" + "&stopPrice=500.00" + "&closePosition=true&side=SELL&type=STOP_MARKET&timestamp=" + strconv.FormatInt(time.Now().Unix()*1000, 10)

	api_key := "14b417a306cd837d3c3ec9cee6f6c4ca2468b0b06a6028c3978ba8a6287ac5c2"

	api_secret := "a6d2fabd26dbe982d0b104e41e115352dc24dfda6726725f153c05aaa6440ca3"

	h := hmac.New(sha256.New, []byte(api_secret))

	h.Write([]byte(query_string))

	signature := "&signature=" + hex.EncodeToString(h.Sum(nil))

	client := &http.Client{}

	req, err := http.NewRequest("POST", base_url+endpoint+"?"+query_string+signature, nil)

	if err != nil {
		fmt.Println("#1")
	}

	req.Header.Set("X-MBX-APIKEY", api_key)

	resp, err := client.Do(req)

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Println("#2")
	}

	fmt.Println(string(body))

}
