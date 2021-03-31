**Steps**
1. Create two binance accounts, one for long only and one for short only.
2. Set up the for Futures.
3. Set up API and get the secrets and keys.
4. Set up environment variables with the secrets and keys.
5. Transfer 100 USDT from Spot Wallet to Futures Wallet, do for both Binance accounts.
6. Create two folders in the same directory, /long_positions and /short_positions.
7. Open Ruby code and decide whether to $long = true or $long = false. You can run both long and short at the same time but my experience is only one makes money at X time depending on market direction.
8. Open terminal and cd to Ruby code directory and run
    while true ; do ruby bot.rb ; done ;
9 . Open another terminal and run
    while true ; do ruby adjust.rb ; done ;

**About**

bot.rb opens new positions and creates 1 stop loss and 10 take profits.

adjust.rb move the stop loss to entry price when first take profit is hit.

coin_quantity_cap lets you define the pairs you want to trade, their quantity, and price decimals. Must provide correct decimals, otherwise API rejects order.

long_positions and short_positions folder ensures that pairs that lost money in the same hour do not get trigger again within the same hour.



