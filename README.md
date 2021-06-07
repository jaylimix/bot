**Steps**
1. Create a binance account.
2. Set up for Futures.
3. Set up API and get the secrets and keys.
4. Set up environment variables with the secrets and keys.
5. Transfer 100 USDT from Spot Wallet to Futures Wallet.
6. Open terminal and cd to Ruby code directory and run -> ruby short_bot.rb

**About**

short_bot.rb - Opens new positions and creates 1 stop loss and 10 take profits. When first take profit is hit, bot adjust stop loss to entry price.

coin_quantity_cap - Lets you define the pairs you want to trade, their quantity, and price decimals. Must provide correct decimals, otherwise API rejects order.