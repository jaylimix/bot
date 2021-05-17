**Steps**
1. Create two binance accounts, one for long only and one for short only.
2. Set up the for Futures.
3. Set up API and get the secrets and keys.
4. Set up environment variables with the secrets and keys.
5. Transfer 100 USDT from Spot Wallet to Futures Wallet, do for both Binance accounts.
6. Create two folders in the same directory, /long_positions and /short_positions.
7. Open terminal and cd to Ruby code directory and run -> ruby bot.rb

**About**

bot.rb - Opens new positions and creates 1 stop loss and 10 take profits. When first take profit is hit, bot adjust stop loss to entry price.

coin_quantity_cap - Lets you define the pairs you want to trade, their quantity, and price decimals. Must provide correct decimals, otherwise API rejects order.

long_positions and short_positions - CSV files that ensures that pairs that lost money in the same hour do not get trigger again within the same hour. This is like a database.