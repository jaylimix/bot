require 'net/http'
require 'json'
require 'open-uri'
require 'openssl'
require 'csv'
require_relative 'coin_quantity_cap'

END{

loop do

    true_and_false = [true, false]

    $extra = ''

    for taf in true_and_false

        $long = taf

        $interval = '1h'

        for cqc in $coin_quantity_cap

            $pair = cqc[0]

            quantity = cqc[1]

            $cap = cqc[2]

            # if $pair != 'OMG'
                # next
            # end

            if $long

                $file_name = 'long_positions/' + $pair + '.csv'

            else

                $file_name = 'short_positions/' + $pair + '.csv'

            end

            #################
            # Get Open Orders
            #################

            $type = 'GET'

            $end_point = '/fapi/v1/openOrders'

            $open_orders = execute()

            if $open_orders == 'error' || $open_orders.include?('code') || $open_orders.empty?
                # print_out('cannot get open orders for ' + $pair)
                next
            end

            ########################################
            # Adjust stop loss to become entry price
            ########################################

            if $open_orders.count >= 11 # API often returns wrong value causes adjustment of the stop loss too soon

                next

            end

            for open_order in $open_orders

                if open_order['type'] == 'STOP_MARKET'

                    if File.exist?($file_name)
        
                        row = CSV.read($file_name)

                        if row[0][1].to_s == ''

                            $old_order_id = open_order['orderId'].to_s

                        else

                            $old_order_id = row[0][1].to_s

                            if $old_order_id == open_order['orderId'].to_s

                                break

                            end

                        end

                    end

                    ###############
                    # Get Positions
                    ###############

                    $type = 'GET'

                    $end_point = '/fapi/v2/positionRisk'

                    position_risk = execute()

                    if position_risk.include?('code') || position_risk == 'error' || position_risk.empty?
                        # print_out('cannot get position risk for ' + $pair)
                        next
                    end

                    if position_risk[0]['positionAmt'].to_f == 0.0
                        next
                    end

                    $position_entry_price = position_risk[0]['entryPrice']

                    new_order_id = adjust_stop_loss()

                    if new_order_id == 'empty' || new_order_id == 'error'

                        print_out(new_order_id)

                        break

                    end

                    CSV.open($file_name, "wb") do |csv|

                        csv << [ Time.now.strftime('%Y-%m-%d %H'), new_order_id ]
                
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
        
        return 'empty'

    elsif result == 'error'

        return 'error'

    elsif result.has_key?('code')

        print_out($pair)

        puts result

        puts ''

        puts 'open orders count is: ' + $open_orders.count.to_s

        $cap -= 1

        adjust_stop_loss()

    else
        
        print_out( $pair )

        puts 'Stop Loss is now the Entry Price'

        $end_point = '/fapi/v1/openOrder'

        $type = 'DELETE'

        $end_point = '/fapi/v1/order'

        $extra = '&orderId=' + $old_order_id.to_s

        execute()

        return result['orderId']

    end
end
