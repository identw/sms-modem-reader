# Sms modem reader
The application reads sms from the gsm huawei modem (tested on models: E173, E352b) using [AT commands](http://download-c.huawei.com/download/downloadCenter?downloadId=51047&version=120450&siteCode) and sends these sms to the specified webhook using the post request. It also periodically checks the balance and exports it to metrics on uri /metrics in prometheus format.

## Setup

## webhook
You can specify url using the environment variable `WEBHOOK_URL`. For example: `export WEBHOOK_URL=https://example.com/sms`

Requst type: `POST`  
Content-type: `application/json`  
Body: `{"message": "text sms"}`  

## Check balance
The balance is checked using a USSD request, which can be specified through the environment variable `BALANCE_USSD`. The balance is read using a regular expression, which is specified using the environment variable `BALANCE_REGEXP` in the [golang regex syntax](https://golang.org/pkg/regexp/syntax/). The application tries to calculate the balance both from the response to the USSD request and from incoming sms. In regular expression, the first captured group will be considered the balance.

## Environment variables

  * `METRIC_PORT` (default: "40111") - port for prometheus metrics
  * `METRIC_LISTEN_IP` (default: "0.0.0.0") - ip address of the interface for prometheus metrics
  * `BALANCE_USSD` (default: "*122#") - USSD request to check balance
  * `BALANCE_REGEXP` (default: "^.*составляет ([0-9]+(\.[0-9]+)?) руб.*$") - regex to pull the current balance out of the message. The current balance will be the first captured group in the regular expression (that is, the first parentheses). The syntax used is https://golang.org/pkg/regexp/syntax/
  * `INTERVAL_READ_SMS` (default: "10") - interval in seconds with which you want to check incoming sms
  * `INTERVAL_CHECK_BALANCE` (default: "86400") - interval in seconds with which to check the current balance
  * `WEBHOOK_URL` (default: "http://127.0.0.1/webhook") - url where sms messages will be sent
  * `SERIAL_FILE` (default: "/dev/ttyUSB0") - serial port file
  * `SERIAL_BAUD` (default: "9600") - serial port baud rate (bps)
  * `SERIAL_STOP_BITS` (default: "1") - the number of stop bits. Possible values: "1", "2", "15"
  * `SERIAL_PARITY` (default: "N") - indicates the presence and type of parity bit. Possible values: "N", "O", "E", "M", "S"
  * `SERIAL_SIZE` (default: "8") - the number of bits that frame start and stop bits

## prometheus метрики
Metrics can be obtained at url: http://METRIC_LISTEN_IP:METRIC_PORT/metrics


| metric                         | type  | description                                                                  |
|--------------------------------|-------|------------------------------------------------------------------------------|
| sms_reader_balance             | guage | current balance                                                              |
| sms_reader_balance_last_check  | guage | unix timestamp last check                                                    |
| sms_reader_sms_last_status     | guage | sms last read status, 0 - successful, 1 - failed to read                     |
| sms_reader_webhook_last_status | guage | status of the last submission to webhook, 0 - successful, 1 - failed to send |

Example metrics:
```
# HELP sms_reader_balance current balance
# TYPE sms_reader_balance gauge
sms_reader_balance 350.36
# HELP sms_reader_balance_last_check unix timestamp last check balance
# TYPE sms_reader_balance_last_check gauge
sms_reader_balance_last_check 1.597837689e+09
# HELP sms_reader_sms_last_status read sms correct, 1 = fail, 0 = success
# TYPE sms_reader_sms_last_status gauge
sms_reader_sms_last_status 0
# HELP sms_reader_webhook_last_status send webhook last send fail, 1 = fail, 0 = success
# TYPE sms_reader_webhook_last_status gauge
sms_reader_webhook_last_status 1
```