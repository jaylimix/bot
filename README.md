Steps
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
