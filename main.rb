require 'net/http'
require 'json'
require 'open-uri'
require 'openssl'




    base_url = 'https://testnet.binancefuture.com'

    end_point = '/fapi/v1/order'

    secret = "a6d2fabd26dbe982d0b104e41e115352dc24dfda6726725f153c05aaa6440ca3"

    key = "14b417a306cd837d3c3ec9cee6f6c4ca2468b0b06a6028c3978ba8a6287ac5c2"

    micro_time = (Time.new.strftime('%s').to_i * 1000).to_s

    query_string = 'symbol=BTCUSDT&side=BUY&type=MARKET&quantity=0.05&timestamp=' + micro_time

    signature = OpenSSL::HMAC.hexdigest('SHA256', secret, query_string)

    uri = URI(base_url + end_point + '?' + query_string + '&signature=' + signature)

    req = Net::HTTP::Post.new(uri)

    req['X-MBX-APIKEY'] = key

    begin
        res = Net::HTTP.start(uri.hostname, uri.port, :use_ssl => true) do |http|
            http.request(req)
        end

        puts JSON.parse(res.body)

        if res.code == '200' || res.code == '400'
            puts JSON.parse(res.body)
        else
            return 'error'
        end
    rescue Exception
        
    end
