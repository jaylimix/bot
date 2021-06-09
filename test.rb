require 'net/http'
require 'json'
require 'open-uri'
require 'openssl'
require 'csv'
require_relative 'coin_quantity_cap'

END{

    $interval = '1h'

    $extra = ''

    $pair = 'ZRX'

    quantity = 50

    $cap = 6


    $time_now = Time.now.strftime('%Y-%m-%d %H')

    ##################
    # Get Ticker Price
    ##################

    $type = 'GET'

    $end_point = '/fapi/v1/ticker/price'

    ticker_array = execute()

    if ticker_array.empty? || ticker_array == 'error' || ticker_array.include?('code')
        # print_out('cannot get ticker price for ' + $pair)
        next
    end

    $ticker_price = (ticker_array['price']).to_f

    max_position_size = 100

    min_position_size = 20

    if $ticker_price * quantity >= max_position_size

        print_out($pair + '  MORE than $' + max_position_size.to_s)

        next

    end

    if $ticker_price * quantity < min_position_size

        print_out($pair + ' LESS than $' + min_position_size.to_s)

    end

    ############
    # Get Klines
    ############

    $type = 'GET'

    $end_point = '/fapi/v1/klines'

    $extra = '&interval=' + $interval + '&limit=205'

    klines = execute()

    if klines.empty? || klines == 'error' || klines.include?('code')
        # print_out('cannot get klines for ' + $pair)
        next
    end

    if klines.count != 205
        next
    end

    ########################################
    # Calculate and Compare Moving Averages
    ########################################

    total_1 = 0

    total_2 = 0

    total_3 = 0

    total_4 = 0

    total_5 = 0

    total_6 = 0

    kc1 = klines.count-1

    kc2 = klines.count-2

    kc3 = klines.count-3

    kc4 = klines.count-4

    kc5 = klines.count-5

    until kc1 == 5 do

        kc1 -= 1

        close_of_previous_bar = klines[kc1][4]

        total_1 += close_of_previous_bar.to_f
        
    end

    ma_1 = total_1 / 200

    ########################

    until kc2 == 4 do

        kc2 -= 1

        close_of_previous_bar = klines[kc2][4]

        total_2 += close_of_previous_bar.to_f
        
    end

    ma_2 = total_2 / 200

    #######################

    until kc3 == 3 do

        kc3 -= 1

        close_of_previous_bar = klines[kc3][4]

        total_3 += close_of_previous_bar.to_f
        
    end

    ma_3 = total_3 / 200

    #######################

    until kc4 == 2 do

        kc4 -= 1

        close_of_previous_bar = klines[kc4][4]

        total_4 += close_of_previous_bar.to_f
        
    end

    ma_4 = total_4 / 200

    #######################

    until kc5 == 1 do

        kc5 -= 1

        close_of_previous_bar = klines[kc5][4]

        total_5 += close_of_previous_bar.to_f
        
    end

    ma_5 = total_5 / 200

    if ma_1 < ma_2 && ma_2 < ma_3 && ma_4 < ma_5

        # puts $pair + ' has a downward 200 moving average for the past 5 hours'

    else

        next

    end

    ##########################
    # Get Position Information
    ##########################

    $type = 'GET'

    $end_point = '/fapi/v2/positionRisk'

    position_risk = execute()

    if position_risk.include?('code') || position_risk == 'error' || position_risk.empty?
        # print_out('cannot get position risk for ' + $pair)
        next
    end

    previous_bar_key = klines.count - 2

    total_diff = 0
    
    number_of_candles = 0

    until previous_bar_key == 100 do
        high = klines[previous_bar_key][2].to_f
        low = klines[previous_bar_key][3].to_f
        diff = high - low
        total_diff += diff
        previous_bar_key -= 1
        number_of_candles += 1
    end

    $average_range =  (total_diff / number_of_candles).to_f

    position_amount = (position_risk[0]['positionAmt']).to_f

    ###########################
    # CHECK FOR ANY OPEN ORDERS
    ###########################

    $type = 'GET'

    $end_point = '/fapi/v1/openOrders'

    $open_orders = execute()

    if $open_orders == 'error' || $open_orders.include?('code') # API returns empty when no open orders
        # print_out('cannot get open orders for ' + $pair)
        next
    end

    #############################################
    # Set Stop Loss, Take Profit, Global Quantity
    #############################################

    $stop_price = $ticker_price + $average_range

    $stop_price = $stop_price.to_s[0, $cap]

    $quantity = (quantity / 10.0)


    #############################################
    # Check higher or lower than previous candles
    #############################################

    key_of_previous_bar = klines.count - 2

    high_of_previous_bar = (klines[key_of_previous_bar][2]).to_f

    close_of_previous_bar = (klines[key_of_previous_bar][4]).to_f

    count_compare_highest = 0

    until key_of_previous_bar == 0 do

        key_of_previous_bar -= 1

        high_of_previous_previous_bar = klines[key_of_previous_bar][2].to_f

        if high_of_previous_bar > high_of_previous_previous_bar

            count_compare_highest += 1

        else

            break

        end
        
    end

    if count_compare_highest <= 10  
        next
    end

    puts $pair

    puts count_compare_highest

}

def print_out(msg)
    puts ''
    puts msg.to_s + ' ' + Time.now.strftime('%d/%m') + ' ' + Time.now.strftime('%H:%M')
end

def execute()

    secret = ENV['LONG_SECRET']

    key = ENV['LONG_KEY']

    micro_time = (Time.new.strftime('%s').to_i * 1000).to_s

    query_string = 'symbol=' + $pair + 'USDT' + '&timestamp=' + micro_time + $extra

    signature = OpenSSL::HMAC.hexdigest('SHA256', secret, query_string)

    uri = URI('https://fapi.binance.com' + $end_point + '?' + query_string + '&signature=' + signature)

    case $type
    when 'GET'
        req = Net::HTTP::Get.new(uri)
    when 'POST'
        req = Net::HTTP::Post.new(uri)
    when 'DELETE'
        req = Net::HTTP::Delete.new(uri)
    end

    req['X-MBX-APIKEY'] = key

    begin
        res = Net::HTTP.start(uri.hostname, uri.port, :use_ssl => true) do |http|
            http.request(req)
        end

        if res.code == '200' || res.code == '400'
            return JSON.parse(res.body)
        else
            return 'error'
        end
    rescue Exception
        # print_out('no internet')
        sleep 10
        execute()
    end
end

def create_take_profit()

    $type = 'POST'

    $end_point = '/fapi/v1/order'

    $extra = '&side=BUY&type=LIMIT' + '&price=' + $tp_price.to_s[0, $cap] + '&quantity=' + $quantity.to_s + '&timeInForce=GTC' + '&reduceOnly=true'

    result = execute()

    if !result.empty? && result.has_key?('code')

        print_out($pair)

        puts result

        $cap -= 1

        create_take_profit()

    end

end

def create_stop_loss()
    
    $type = 'POST'

    $end_point = '/fapi/v1/order'

    $extra = '&stopPrice=' + $stop_price.to_s[0, $cap] + '&side=BUY&type=STOP_MARKET' + '&closePosition=true'

    result = execute()

    if !result.empty? && result.has_key?('code')

        print_out($pair)

        puts result

        puts $extra

    else

        print_out($pair)

        puts 'SL: ' + $stop_price.to_s[0, $cap]

    end

end

def adjust_stop_loss()

    $type = 'POST'

    $end_point = '/fapi/v1/order'

    $extra = '&stopPrice=' + $position_entry_price.to_s[0, $cap] + '&side=BUY&type=STOP_MARKET' + '&closePosition=true'

    result = execute()

    if result.empty?
        
        print_out($pair)
        puts 'empty'

    elsif result == 'error'

        print_out($pair)
        puts 'error'

    elsif result.has_key?('code')

        print_out($pair)
        puts result
        puts $extra
        puts 'Open orders count: ' + $open_orders.count.to_s

    else

        print_out($pair)

        puts 'Stop Loss is now the Entry Price'

        #######################
        # Delete Previous Order
        #######################

        $end_point = '/fapi/v1/openOrder'

        $type = 'DELETE'

        $end_point = '/fapi/v1/order'

        $extra = '&orderId=' + $old_order_id.to_s

        execute()

    end
end