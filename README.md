This is a bot looks at the 1 hour chart and will:
- trigger with market when conditions met.
- create a stop loss at the same time with 3% distance (adjustable).
- will close position after 5 hours (adjustable).

This bot will not:
- open BUSD pair.
- open pair that has minimum quantity more than usd_per_trade (adjustable), see line 23.

Preparation:
- Set up Binance Futures, copy API key and secret and paste it on line 30 and 32.
- Transfer 100 USDT to Binance Futures.
- Download and install VSCODE editor https://code.visualstudio.com/download
- Download and install Golang https://go.dev/doc/install

In VSCODE:
- Copy paste this code into a new file, named it as: main.go
- Click Terminal.
- cd to the directory of the this file and write: go run main.go
- Enter to run the bot.

Keep this terminal process running 24/7.
Observe this terminal, it will let you know when an order is triggered.