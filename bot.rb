require 'net/http'
require 'json'
require 'open-uri'
require 'openssl'
require 'csv'
require_relative 'coin_quantity_cap'

END{

$long = get_terminal_input()

loop do

    $interval = '1h'

    $extra = ''

    for cqc in $coin_quantity_cap

        $pair = cqc[0]

        quantity = cqc[1]

        $cap = cqc[2]

        # if $pair != 'OMG'
            # next
        # end

        $time_now = Time.now.strftime('%Y-%m-%d %H')

        if $long

            $file_name = 'long_positions/' + $pair + '.csv'

        else

            $file_name = 'short_positions/' + $pair + '.csv'

        end

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

        $extra = '&interval=' + $interval + '&limit=100'

        klines = execute()

        if klines.empty? || klines == 'error' || klines.include?('code')
            # print_out('cannot get klines for ' + $pair)
            next
        end

        if klines.count != 100
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
        
        number_of_candles = klines.count - 50

        until previous_bar_key == number_of_candles do
            high = klines[previous_bar_key][2].to_f
            low = klines[previous_bar_key][3].to_f
            diff = high - low
            total_diff += diff
            previous_bar_key -= 1
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

        start = 0

        $multiplier = 0.5
        
        if $long

            $stop_price = $ticker_price - $average_range * $multiplier

        else

            $stop_price = $ticker_price + $average_range * $multiplier

        end

        $stop_price = $stop_price.to_s[0, $cap]

        $quantity = (quantity / 10.0)

        if position_amount == 0

            #############################################
            # When no position, delete hanging stop order
            #############################################
            
            if $open_orders.count == 1

                $type = 'DELETE'

                $end_point = '/fapi/v1/allOpenOrders'
                
                print_out( $pair )

                puts execute()

            end

            ##############################################################################
            # When no position, delete entry order and stop order when one hour has passed
            ##############################################################################

            if $open_orders.count == 2

                #####################################
                # Compare server time with order time
                #####################################

                $type = 'GET'

                $end_point = '/fapi/v1/time'

                result = execute()

                time_diff = result['serverTime'].to_i - $open_orders[0]['time'].to_i

                if time_diff > 60*60*1000

                    $type = 'DELETE'

                    $end_point = '/fapi/v1/allOpenOrders'

                    print_out( $pair )

                    puts execute()

                    puts 'More than an hour'

                end
            end

            ###################################################
            # Check whether position is opened in the same hour
            ###################################################

            if File.exist?($file_name)
            
                row = CSV.read($file_name)

                if Time.now.strftime('%Y-%m-%d %H') == row[0][0].to_s

                    next

                end
        
            end

            ##################################
            # CHECK IF HAMMER OR SHOOTING STAR
            ##################################

            is_hammer = false

            key_of_previous_bar = klines.count - 2

            open_of_previous_bar = klines[key_of_previous_bar][1]

            high_of_previous_bar = klines[key_of_previous_bar][2]

            low_of_previous_bar = klines[key_of_previous_bar][3]

            close_of_previous_bar = klines[key_of_previous_bar][4]

            green_candle = false

            red_candle = false

            if close_of_previous_bar.to_f > open_of_previous_bar.to_f

                green_candle = true

            else

                red_candle = true

            end

            if $long && green_candle
                
                diff_of_open_vs_low = open_of_previous_bar.to_f - low_of_previous_bar.to_f

                diff_of_high_vs_open = high_of_previous_bar.to_f - open_of_previous_bar.to_f

                if diff_of_open_vs_low / diff_of_high_vs_open > 1

                    is_hammer = true
                    
                end

            end

            if $long && red_candle

                diff_of_close_vs_low = close_of_previous_bar.to_f - low_of_previous_bar.to_f

                diff_of_high_vs_close = high_of_previous_bar.to_f - close_of_previous_bar.to_f

                if diff_of_close_vs_low / diff_of_high_vs_close > 1

                    is_hammer = true

                end

            end

            if !$long && green_candle
                
                diff_of_high_vs_close = high_of_previous_bar.to_f - close_of_previous_bar.to_f

                diff_of_close_vs_low = close_of_previous_bar.to_f - low_of_previous_bar.to_f

                if diff_of_high_vs_close / diff_of_close_vs_low > 1

                    is_hammer = true

                end

            end

            if !$long && red_candle

                diff_of_high_vs_open = high_of_previous_bar.to_f - open_of_previous_bar.to_f

                diff_of_open_vs_low = open_of_previous_bar.to_f - low_of_previous_bar.to_f

                if diff_of_high_vs_open / diff_of_open_vs_low > 1

                    is_hammer = true

                end

            end

            if !is_hammer

                next

            end

            #############################################
            # Check higher or lower than previous candles
            #############################################

            key_of_previous_bar = klines.count - 2

            high_of_previous_bar = (klines[key_of_previous_bar][2]).to_f

            low_of_previous_bar = (klines[key_of_previous_bar][3]).to_f

            close_of_previous_bar = (klines[key_of_previous_bar][4]).to_f

            count_compare_highest_lowest = 0

            until key_of_previous_bar == 0 do

                key_of_previous_bar -= 1

                if $long

                    side = 'BUY'

                    low_of_previous_previous_bar = klines[key_of_previous_bar][3].to_f

                    if low_of_previous_bar < low_of_previous_previous_bar

                        count_compare_highest_lowest += 1

                    else

                        break

                    end

                else

                    side = 'SELL'

                    high_of_previous_previous_bar = klines[key_of_previous_bar][2].to_f

                    if high_of_previous_bar > high_of_previous_previous_bar

                        count_compare_highest_lowest += 1

                    else

                        break

                    end

                end
                
            end

            if count_compare_highest_lowest < 30 || count_compare_highest_lowest > 80
                
                next

            end

            #####################################
            # Pass criterias, open a new position
            #####################################
        
            $type = 'POST'
        
            $end_point = '/fapi/v1/order'
        
            $entry_side = side

            $entry_quantity = quantity.to_s

            $price_after_zero_point_five_percent = close_of_previous_bar * 0.997
            
            result = open_new_limit_order()

            if result.include?('orderId')

                CSV.open($file_name, "wb") do |csv|

                    csv << [ Time.now.strftime('%Y-%m-%d %H') ]
            
                end

                limit_entry_create_stop_loss()
       
            end

        end
        
        if position_amount != 0

            #######################################################################
            # Create stop loss and take profit if not found in open orders response
            #######################################################################

            stop_loss_does_not_exist = true

            take_profit_does_not_exist = true

            for open_order in $open_orders

                if open_order['type'] == 'STOP_MARKET'

                    stop_loss_does_not_exist = false

                end

                if open_order['type'] == 'LIMIT'

                    take_profit_does_not_exist = false

                end

            end

            if stop_loss_does_not_exist

                create_stop_loss()

            end

            if take_profit_does_not_exist

                entry_price = position_risk[0]['entryPrice'].to_f

                until start == 10 do

                    if $long

                        $tp_price = entry_price + $average_range * $multiplier
            
                    else
            
                        $tp_price = entry_price - $average_range * $multiplier
            
                    end

                    create_take_profit()

                    $multiplier += 1
            
                    start += 1
            
                end
            end
            
            #############################################
            # Go to next when open orders is still 11
            # Go to next when an order just got triggered
            #############################################

            if $open_orders.count >= 11 || $open_orders.count == 1
                
                next

            end

            ####################################
            # Adjust the stop loss to break even
            ####################################

            for open_order in $open_orders

                if open_order['type'] == 'STOP_MARKET'

                    stop_price = open_order['stopPrice'].to_f

                    $position_entry_price = position_risk[0]['entryPrice'].to_f

                    the_difference = (stop_price - $position_entry_price).abs

                    if the_difference / $position_entry_price > 0.005

                        $old_order_id = open_order['orderId']

                        # puts 'OPEN ORDERS COUNT'

                        # puts $open_orders.count
                        
                        adjust_stop_loss()

                    end
                end
            end
        end
    end
end
}

def get_terminal_input()

    print 'Long or Short? (l or s) '

    long_or_short = gets.chomp

    if long_or_short == 'l'

        return true

    elsif long_or_short == 's'

        return false

    else
        
        get_terminal_input()

    end

end

def print_out(msg)
    puts ''
    puts msg.to_s + ' ' + Time.now.strftime('%d/%m') + ' ' + Time.now.strftime('%H:%M')
end

def execute()

    if $long
        secret = ENV['LONG_SECRET']
        key = ENV['LONG_KEY']
    else
        secret = ENV['SHORT_SECRET']
        key = ENV['SHORT_KEY']
    end

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

def open_new_limit_order()

    $type = 'POST'
        
    $end_point = '/fapi/v1/order'

    $extra = '&side=' + $entry_side + '&type=LIMIT' + '&price=' + $price_after_zero_point_five_percent.to_s[0, $cap] + '&quantity=' + $entry_quantity + '&timeInForce=GTC'

    result = execute()

    if !result.empty? && result.has_key?('code')

        print_out($pair)

        puts result

        $cap -= 1

        open_new_limit_order()

    else

        return result

    end
end

def create_take_profit()

    if $long

        side = 'SELL'

    else

        side = 'BUY'

    end

    $type = 'POST'

    $end_point = '/fapi/v1/order'

    $extra = '&side=' + side + '&type=LIMIT' + '&price=' + $tp_price.to_s[0, $cap] + '&quantity=' + $quantity.to_s + '&timeInForce=GTC' + '&reduceOnly=true'

    result = execute()

    if !result.empty? && result.has_key?('code')

        print_out($pair)

        puts $extra

        puts result

        # $cap -= 1

        # create_take_profit()

    end

end

def create_stop_loss()
    
    if $long
    
        side = 'SELL'

    else

        side = 'BUY'

    end

    $type = 'POST'

    $end_point = '/fapi/v1/order'

    $extra = '&stopPrice=' + $stop_price.to_s[0, $cap] + '&side=' + side + '&type=STOP_MARKET' + '&closePosition=true'

    puts $extra

    result = execute()

    if !result.empty? && result.has_key?('code')

        print_out($pair)

        puts $stop_price.to_s

        $cap -= 1

        create_stop_loss()

    else

        print_out($pair + ' SL: ' + $stop_price.to_s[0, $cap])

    end

end

def limit_entry_create_stop_loss()
    
    if $long

        $stop_price = $price_after_zero_point_five_percent - $average_range * $multiplier

        side = 'SELL'

    else

        $stop_price = $price_after_zero_point_five_percent + $average_range * $multiplier

        side = 'BUY'

    end

    $type = 'POST'

    $end_point = '/fapi/v1/order'

    $extra = '&stopPrice=' + $stop_price.to_s[0, $cap] + '&side=' + side + '&type=STOP_MARKET' + '&closePosition=true'

    result = execute()

    if !result.empty? && result.has_key?('code')

        print_out($pair)

        puts $stop_price.to_s

        $cap -= 1

        create_stop_loss()

    else

        print_out($pair + ' SL: ' + $stop_price.to_s[0, $cap])
        
    end

end

def adjust_stop_loss()

    $type = 'POST'

    $end_point = '/fapi/v1/order'

    if $long

        side = 'SELL'

    else

        side = 'BUY'

    end

    $extra = '&stopPrice=' + $position_entry_price.to_s[0, $cap] + '&side=' + side + '&type=STOP_MARKET' + '&closePosition=true'

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

        puts $pair

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