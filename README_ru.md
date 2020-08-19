# Sms modem reader
Приложение считывает sms из gsm модема huawei (протестировано на моделях: E173, E352b) c помощью [AT комманд](http://download-c.huawei.com/download/downloadCenter?downloadId=51047&version=120450&siteCode) и отправляет эти sms на указанный webhook c помощью post запроса. А также периодически проверяет  баланс и экспортирует его в метрики на uri /metrics в формате prometheus.

## Установка

## webhook
Указать url можно с помощью переменной среды `WEBHOOK_URL`. Например: `export WEBHOOK_URL=https://example.com/sms`

Тип запроса: `POST`  
Content-type: `application/json`  
Тело: `{"message": "text sms"}`  

## Проверка баланса
Баланс проверяется с помощью USSD запроса, который можно указать через переменную среды `BALANCE_USSD`. Считывается баланс с помощью регулярного выражения, которое указывается с помощью переменной среды `BALANCE_REGEXP` в синтаксисе [golang](https://golang.org/pkg/regexp/syntax/). Приложение пытается считать баланс как с ответа на USSD запрос, так и в приходящих sms. В регулярном выражении, балансом будет считаться первая захваченная группа.

## Переменные среды

  * `METRIC_PORT` (умолчание: "40111") - порт для prometheus метрик
  * `METRIC_LISTEN_IP` (умолчание: "0.0.0.0") - ip адрес интерфейса для prometheus метрик
  * `BALANCE_USSD` (умолчание: "*122#") - ussd запрос для проверки баланса
  * `BALANCE_REGEXP` (умолчание: "^.*составляет ([0-9]+(\.[0-9]+)?) руб.*$") - регулярное выражение, чтобы вытащить из сообщения текущий баланс. Текущим балансом будет считаться первая захваченная группа в регулярном выражении (то есть первые скобки). Используемый синтаксис - https://golang.org/pkg/regexp/syntax/
  * `INTERVAL_READ_SMS` (умолчание: "10") - интервал в секундах, с которым требуется проверять приходящие sms
  * `INTERVAL_CHECK_BALANCE` (умолчание: "86400") - интервал в секундах, с которым следует проверять текущий баланс
  * `WEBHOOK_URL` (умолчание: "http://127.0.0.1/webhook") - url куда будут отправлены sms сообщения
  * `SERIAL_FILE` (умолчание: "/dev/ttyUSB0") - файл последовательного порта
  * `SERIAL_BAUD` (умолчание: "9600") - скорость передачи данных последовательного порта (bps)
  * `SERIAL_STOP_BITS` (умолчание: "1") - количество стоповых битов. Возможные значения: "1", "2", "15"
  * `SERIAL_PARITY` (умолчание: "N") - обозначение наличия и типа бита четности. Возможные значения: "N", "O", "E", "M", "S"
  * `SERIAL_SIZE` (умолчание: "8") - количество битов, которые обрамляют стартовый и стоповый биты

## prometheus метрики
Метрики можно получить по url: http://METRIC_LISTEN_IP:METRIC_PORT/metrics.

| метрика                        | тип   | описание                                                                    |
|--------------------------------|-------|-----------------------------------------------------------------------------|
| sms_reader_balance             | guage | текущий баланс                                                              |
| sms_reader_balance_last_check  | guage | unix timestamp последней проверки                                           |
| sms_reader_sms_last_status     | guage | статус последнего чтения sms, 0 - успешно, 1 - не удалось прочитать         |
| sms_reader_webhook_last_status | guage | статус последней отправки на webhook, 0 - успешно, 1 - не удалось отправить |

Пример метрик:
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