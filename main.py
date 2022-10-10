
import time
from requests import Request, Session
import hmac
import time
import json
from datetime import datetime
import os
import sys

API_KEY = os.environ.get('API_KEY')

API_SECRET = os.environ.get('API_SECRET')

NAME = str(sys.argv[1]).upper() + '-PERP'

f = open(str(sys.argv[1]) + '.txt')

lines = f.readlines()

f.close()

ENTRY_SIZE = float(lines[0].strip())

STOP_BUY_SELL_PERCENTAGE_AWAY = float(lines[1].strip()) * 0.01

TRAILING_STOP_PERCENTAGE_AWAY = float(lines[2].strip()) * 0.01

DECIMALS = int(lines[3].strip())

LIMIT_PERCENTAGE_AWAY = 0.01

SPLIT_TIMES = 5

LIMIT_SIZE = ENTRY_SIZE / SPLIT_TIMES

BASE_URL = "https://ftx.com/api"

trailing_stop_payload = {}

stop_buy_sell_payload = {}

delete_all_orders_payload = {}

limit_order_payload = {}


def need_auth(endpoint):

    if endpoint == 'positions':

        request = Request('GET', BASE_URL + '/' + 'positions')

    if endpoint == 'get_stop_buy_sell_orders':
        request = Request('GET', BASE_URL + '/' +
                          'conditional_orders?market=' + NAME + '&type=stop')

    if endpoint == 'create_stop_buy_sell':
        request = Request('POST', BASE_URL + '/' +
                          'conditional_orders', data=stop_buy_sell_payload)

    if endpoint == 'create_limit_order':
        request = Request('POST', BASE_URL + '/' +
                          'orders', data=limit_order_payload)

    if endpoint == 'get_trailing_stop':
        request = Request('GET', BASE_URL + '/' +
                          'conditional_orders?market=' + NAME + '&type=trailing_stop')

    if endpoint == 'create_trailing_stop':
        request = Request('POST', BASE_URL + '/' +
                          'conditional_orders', data=trailing_stop_payload)

    if endpoint == 'delete_orders':
        request = Request('DELETE', BASE_URL + '/' + 'orders',
                          data=delete_all_orders_payload)

    prepared = request.prepare()

    ts = int(time.time() * 1000)

    signature_payload = f'{ts}{prepared.method}{prepared.path_url}'.encode(
    )

    if endpoint == 'create_trailing_stop':
        signature_payload = f'{ts}{prepared.method}{prepared.path_url}{trailing_stop_payload}'.encode(
        )

    if endpoint == 'create_stop_buy_sell':
        signature_payload = f'{ts}{prepared.method}{prepared.path_url}{stop_buy_sell_payload}'.encode(
        )

    if endpoint == 'create_limit_order':
        signature_payload = f'{ts}{prepared.method}{prepared.path_url}{limit_order_payload}'.encode(
        )

    if endpoint == 'delete_orders':
        signature_payload = f'{ts}{prepared.method}{prepared.path_url}{delete_all_orders_payload}'.encode(
        )

    signature = hmac.new(API_SECRET.encode(),
                         signature_payload, 'sha256').hexdigest()

    prepared.headers['Content-Type'] = 'application/json'

    prepared.headers['FTX-KEY'] = API_KEY

    prepared.headers['FTX-SIGN'] = signature

    prepared.headers['FTX-TS'] = str(ts)

    s = Session()

    response = s.send(prepared, timeout=10)

    return response.json()


def no_need_auth(endpoint):

    if endpoint == 'futures':
        url = '/futures/' + NAME

    if endpoint == 'ohlc':
        url = '/markets/' + NAME + '/candles?resolution=3600' + \
            '&start_time=' + str(int(time.time() - 3600))

    request = Request('GET', BASE_URL + url)

    # print(request.url)

    prepared = request.prepare()

    s = Session()

    response = s.send(prepared, timeout=10)

    return response.json()


def create_limit_orders(stop_buy_sell_price, side):

    if side == 'buy':
        limit_price = stop_buy_sell_price * (1 + LIMIT_PERCENTAGE_AWAY)
        limit_side = 'sell'
    else:
        limit_price = stop_buy_sell_price * (1 - LIMIT_PERCENTAGE_AWAY)
        limit_side = 'buy'

    for x in range(SPLIT_TIMES):

        global limit_order_payload

        limit_order_payload = json.dumps({
            'market': NAME,
            'side': limit_side,
            'price': limit_price,
            'size': LIMIT_SIZE,
            'type': 'limit'
        })

        if side == 'buy':
            limit_price = limit_price * (1 + LIMIT_PERCENTAGE_AWAY)
            limit_side = 'sell'
        else:
            limit_price = limit_price * (1 - LIMIT_PERCENTAGE_AWAY)
            limit_side = 'buy'

        # print(limit_order_payload)

        create_limit_order = need_auth('create_limit_order')

        if create_limit_order['success']:

            print('Limit order created\n')

        # print(create_limit_order)


print()

# print('####### ' + NAME + ' ' + datetime.now().strftime("%d/%m/%Y %H:%M:%S") +' #######\n')
print('####### ' + NAME + ' #######\n')
print('Size:', ENTRY_SIZE)
print('Entry Percentage:', STOP_BUY_SELL_PERCENTAGE_AWAY)
print('Trailing Stop Percentage:', TRAILING_STOP_PERCENTAGE_AWAY)
print('Decimals', DECIMALS)

position_size = 0.0

positions = need_auth('positions')

if not positions['success']:

    print(positions)
    exit()

for position in positions['result']:

    if position['future'] == NAME:

        position_size = position['size']

print("\nGet and set the last price")

print()

futures = no_need_auth('futures')

if not futures['success']:

    print(futures)
    exit()

last_price = futures['result']['last']

print("Checking whether to create a stop buy/sell")

print()

ohlc = no_need_auth('ohlc')

if not ohlc['success']:

    print(ohlc)
    exit()

open = ohlc['result'][0]['open']
high = ohlc['result'][0]['high']
low = ohlc['result'][0]['low']

if position_size == 0:

    stop_orders = need_auth('get_stop_buy_sell_orders')

    # print(stop_orders)

    if not stop_orders['success']:

        print(stop_orders)
        exit()

    if not stop_orders['result']:

        print("Create a stop buy/sell order\n")

        if last_price > open:
            print("GREEN CANDLE")
            print()
            stop_buy_sell_price = low * \
                (1 + STOP_BUY_SELL_PERCENTAGE_AWAY)
            side = "buy"

        else:
            print("RED CANDLE")
            print()
            stop_buy_sell_price = high * \
                (1 - STOP_BUY_SELL_PERCENTAGE_AWAY)
            side = "sell"

        # print(stop_buy_sell_price)

        stop_buy_sell_payload = json.dumps({
            "market": NAME,
            "side": side,
            "size": ENTRY_SIZE,
            "type": "stop",
            "triggerPrice": stop_buy_sell_price
        })

        # print(stop_buy_sell_payload)

        create_stop_buy_sell = need_auth('create_stop_buy_sell')

        if not create_stop_buy_sell['success']:
            print(create_stop_buy_sell)
            exit()

        print("Create limit orders\n")

        create_limit_orders = create_limit_orders(
            stop_buy_sell_price, side)

    else:

        print("Stop buy/sell already exist\n")

        for stop_order in stop_orders['result']:

            exchange_stop_trigger_price = stop_order['triggerPrice']

        print(
            "Compare and see whether an adjustment to the stop order is needed\n")

        stop_buy_sell_price = 0.0

        if last_price > open:

            print("GREEN CANDLE\n")

            stop_buy_sell_price = low * \
                (1 + STOP_BUY_SELL_PERCENTAGE_AWAY)

            side = "buy"

        else:
            print("RED CANDLE\n")

            stop_buy_sell_price = high * \
                (1 - STOP_BUY_SELL_PERCENTAGE_AWAY)

            side = "sell"

        print('Exchange stop ' + side + ' price:',
              round(exchange_stop_trigger_price, DECIMALS))

        print('Computer stop ' + side + ' price:',
              round(stop_buy_sell_price, DECIMALS))

        print()

        if round(exchange_stop_trigger_price, DECIMALS) != round(stop_buy_sell_price, DECIMALS):

            print("Delete all orders\n")

            delete_all_orders_payload = json.dumps({"market": NAME})

            delete_orders = need_auth('delete_orders')

            if delete_orders['success']:

                print("All orders deleted\n")

                print("Add new stop buy/sell\n")

                stop_buy_sell_payload = json.dumps({
                    "market": NAME,
                    "side": side,
                    "size": ENTRY_SIZE,
                    "type": "stop",
                    "triggerPrice": stop_buy_sell_price
                })

                # print(stop_buy_sell_payload)

                create_stop_buy_sell = need_auth('create_stop_buy_sell')

                if not create_stop_buy_sell['success']:
                    print(create_stop_buy_sell)
                    exit()

                print("Create limit orders\n")

                create_limit_orders(stop_buy_sell_price, side)

            else:

                print(delete_orders)
                exit()

        else:
            print("No changes needed\n")

######################################################################

print('Checking whether there is any trailing stop orders')

print()

get_trailing_stop = need_auth('get_trailing_stop')

if not get_trailing_stop['success']:

    print(get_trailing_stop)
    exit()

if get_trailing_stop['result']:

    print('Found Trailing stop order\n')

    exit()

print('No trailing_stop orders, bot will open a reduce only trailing stop order\n')

side = ''

trail_value = 0.0

if last_price > open:

    price_away = last_price * (1 - TRAILING_STOP_PERCENTAGE_AWAY)

    trail_value = -1 * (last_price - price_away)

    side = 'sell'

else:

    price_away = last_price * (1 + TRAILING_STOP_PERCENTAGE_AWAY)

    trail_value = price_away - last_price

    side = 'buy'

trailing_stop_payload = json.dumps({
    "reduceOnly": True,
    "market": NAME,
    "side": side,
    "size": ENTRY_SIZE,
    "trailValue": trail_value,
    "type": "trailingStop"
})

print("Creating trailing stop order\n")

# print()

# print(trailing_stop_payload)

# print()

create_trailing_stop = need_auth('create_trailing_stop')

# print(create_trailing_stop)

if not create_trailing_stop['success']:

    print(create_trailing_stop)

print('Trailing stop created\n')
