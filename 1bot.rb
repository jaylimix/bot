require 'net/http'
require 'json'
require 'open-uri'
require 'openssl'
require 'csv'
require_relative 'coin_quantity_cap'

END{

loop do

    $interval = '1h'

    $extra = ''

    for cqc in $coin_quantity_cap

        $pair = cqc[0]

        $quantity_size = cqc[1]

        $cap = cqc[2]

        # if $pair != 'CELR'

        #     next

        # else

        #     puts ''
        #     puts $pair

        # end

        ############
        # Get Klines
        ############

        $type = 'GET'

        $end_point = '/fapi/v1/klines'

        $extra = '&interval=' + $interval + '&limit=155'

        klines = execute()

        if klines.empty? || klines == 'error' || klines.include?('code')
            next
        end

        if klines.count != 155
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

        counter = 0

        until counter == 150 do

            close_of_previous_bar = klines[counter][4]

            total_1 += close_of_previous_bar.to_f

            counter += 1

        end

        counter = 1

        until counter == 151 do

            close_of_previous_bar = klines[counter][4]

            total_2 += close_of_previous_bar.to_f

            counter += 1

        end

        counter = 2

        until counter == 152 do

            close_of_previous_bar = klines[counter][4]

            total_3 += close_of_previous_bar.to_f

            counter += 1

        end

        counter = 3

        until counter == 153 do

            close_of_previous_bar = klines[counter][4]

            total_4 += close_of_previous_bar.to_f

            counter += 1

        end

        counter = 4

        until counter == 154 do

            close_of_previous_bar = klines[counter][4]

            total_5 += close_of_previous_bar.to_f

            counter += 1

        end

        if total_1 > total_2 && total_2 > total_3 && total_3 > total_4 && total_4 > total_5

            # puts $pair + ' has a downward 100 moving average for the past 5 hours'

        else

            next

        end

        ############################################
        # Go next when two consecutive green candles
        ############################################

        key_of_previous_bar = klines.count - 2

        open_of_previous_bar = klines[key_of_previous_bar][1]

        close_of_previous_bar = klines[key_of_previous_bar][4]

        if close_of_previous_bar.to_f > open_of_previous_bar.to_f

            key_of_previous_2x_bar = klines.count - 3

            open_of_previous_2x_bar = klines[key_of_previous_2x_bar][1]

            close_of_previous_2x_bar = klines[key_of_previous_2x_bar][4]

            if close_of_previous_2x_bar.to_f > open_of_previous_2x_bar.to_f

                next

            end

        end

        ##########################
        # Get Position Information
        ##########################

        $type = 'GET'

        $end_point = '/fapi/v2/positionRisk'

        position_risk = execute()

        if position_risk.include?('code') || position_risk == 'error' || position_risk.empty?
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
            next
        end

        ##################
        # Get Ticker Price
        ##################

        $type = 'GET'

        $end_point = '/fapi/v1/ticker/price'

        ticker_array = execute()

        if ticker_array.empty? || ticker_array == 'error' || ticker_array.include?('code')
            next
        end

        $ticker_price = (ticker_array['price']).to_f

        ##################
        # If no position 
        # If have position
        ##################

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

            $type = 'GET'

            $end_point = '/fapi/v1/time'

            result = execute()

            if $open_orders.count == 2

                #####################################
                # Compare server time with order time
                #####################################

                if Time.at(result['serverTime'].to_i / 1000).to_s[0, 13] != Time.at($open_orders[0]['time'] / 1000).to_s[0, 13]

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

            $type = 'GET'

            $end_point = '/fapi/v1/allOrders'

            $all_orders = execute()

            if $all_orders == 'error' || $all_orders.include?('code')
                next
            end

            if $all_orders != []

                already_loss = false

                last_order_index = $all_orders.count - 1

                until last_order_index == 0 do

                    if $all_orders[last_order_index]['status'] == 'FILLED' || $all_orders[last_order_index]['status'] == 'NEW'

                        if Time.at(result['serverTime'].to_i / 1000).to_s[0, 13] == Time.at($all_orders[last_order_index]['updateTime'] / 1000).to_s[0, 13]
                            already_loss = true
                            break
                        end

                    end
                    
                    last_order_index -= 1
                end

                if already_loss
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

            if green_candle
                
                diff_of_high_vs_close = high_of_previous_bar.to_f - close_of_previous_bar.to_f

                diff_of_close_vs_low = close_of_previous_bar.to_f - low_of_previous_bar.to_f

                if diff_of_high_vs_close / diff_of_close_vs_low > 1

                    is_hammer = true

                end

            end

            if red_candle

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

            #####################################
            # Pass criterias, open a new position
            #####################################
        
            $type = 'POST'
        
            $end_point = '/fapi/v1/order'

            $entry_quantity = (50 / $ticker_price).to_s

            $price_after_x_percent = close_of_previous_bar * 1.005

            print_out($pair)
            
            result = open_new_limit_order()

            if result.include?('orderId')

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

                print_out($pair)

                $stop_price = $ticker_price + $average_range

                $stop_price = $stop_price.to_s[0, $cap]

                create_stop_loss()

            end

            if take_profit_does_not_exist

                print_out($pair)

                entry_price = position_risk[0]['entryPrice'].to_f

                position_amount = position_risk[0]['positionAmt'].to_f.abs / 10

                $multiplier = 1

                start = 0

                until start == 10 do

                    $tp_price = entry_price - $average_range * $multiplier

                    if start == 9

                        $quantity = (position_amount * 2).to_s[0, $quantity_size]

                    else

                        $quantity = position_amount.to_s[0, $quantity_size]

                    end

                    create_take_profit()

                    $multiplier += 1
            
                    start += 1
            
                end
            end
            
            ##############################################
            # Go to next when first limit order is not hit
            # Go to next when order just got triggered
            ##############################################

            if $open_orders.count == 11 || $open_orders.count == 1
                
                next

            end

            ####################################
            # Adjust the stop loss to break even
            ####################################

            for open_order in $open_orders

                if open_order['type'] == 'STOP_MARKET'

                    stop_price = open_order['stopPrice'][0, $cap]

                    $position_entry_price = position_risk[0]['entryPrice'][0, $cap]

                    if stop_price != $position_entry_price

                        $old_order_id = open_order['orderId']

                        adjust_stop_loss()

                    end
                end
            end
        end
    end
end
}

def print_out(msg)
    puts ''
    puts msg.to_s + ' ' + Time.now.strftime('%d/%m') + ' ' + Time.now.strftime('%H:%M')
end

def execute()

    secret = ENV['SHORT_SECRET']

    key = ENV['SHORT_KEY']

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

    $extra = '&side=SELL&type=LIMIT' + '&price=' + $price_after_x_percent.to_s[0, $cap] + '&quantity=' + $entry_quantity[0, $quantity_size] + '&timeInForce=GTC'

    result = execute()

    if result.empty?

        puts 'Empty open new limit order'
    
    elsif result == 'error'

        puts 'Error open new limit order'

    elsif result.has_key?('code')

        puts result

    else

        puts 'Create sell limit order'

        return result
        
    end
end

def create_take_profit()

    $type = 'POST'

    $end_point = '/fapi/v1/order'

    $extra = '&side=BUY&type=LIMIT' + '&price=' + $tp_price.to_s[0, $cap] + '&quantity=' + $quantity + '&timeInForce=GTC' + '&reduceOnly=true'

    result = execute()

    if result.empty?

        puts 'Empty create take profit'
    
    elsif result == 'error'

        puts 'Error create take profit'

    elsif result.has_key?('code')

        puts result

    else

        puts $tp_price.to_s[0, $cap]
        
    end

end

def create_stop_loss()
    
    $type = 'POST'

    $end_point = '/fapi/v1/order'

    $extra = '&stopPrice=' + $stop_price.to_s[0, $cap] + '&side=BUY&type=STOP_MARKET' + '&closePosition=true'

    result = execute()

    if result.empty?

        puts 'Empty create stop loss'
    
    elsif result == 'error'

        puts 'Error create stop loss'

    elsif result.has_key?('code')

        puts result

    else

        puts 'Stop Loss created'
        
    end
end

def limit_entry_create_stop_loss()

    $stop_price = $price_after_x_percent + $average_range

    $type = 'POST'

    $end_point = '/fapi/v1/order'

    $extra = '&stopPrice=' + $stop_price.to_s[0, $cap] + '&side=BUY&type=STOP_MARKET' + '&closePosition=true'

    result = execute()

    if result.empty?

        puts 'Empty create stop loss'
    
    elsif result == 'error'

        puts 'Error create stop loss'

    elsif result.has_key?('code')

        puts result

    else

        puts $stop_price.to_s[0, $cap]
        
    end

end

def adjust_stop_loss()

    $type = 'POST'

    $end_point = '/fapi/v1/order'

    $extra = '&stopPrice=' + $position_entry_price + '&side=BUY&type=STOP_MARKET' + '&closePosition=true'

    result = execute()

    if result.empty?
        
        print_out($pair)
        puts 'Adjust stop loss empty'

    elsif result == 'error'

        print_out($pair)
        puts 'Adjust stop loss error'

    elsif result.has_key?('code')

        print_out($pair)
        puts result
        puts 'Adjust stop loss'

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