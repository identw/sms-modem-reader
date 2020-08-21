# Sms modem reader
Приложение считывает sms из gsm модема huawei (протестировано на моделях: E173, E352b) c помощью [AT комманд](http://download-c.huawei.com/download/downloadCenter?downloadId=51047&version=120450&siteCode) и отправляет эти sms на указанный webhook c помощью post запроса. А также периодически проверяет баланс и экспортирует его в метрики на uri /metrics в формате prometheus.

## Установка
```bash
curl -L https://github.com/identw/sms-modem-reader/releases/latest/download/sms-modem-reader-amd64 -o /usr/local/bin/sms-modem-reader
chmod +x /usr/local/bin/sms-modem-reader
```
Для запуска выполните:
```bash
/usr/local/bin/sms-modem-reader
```
Если нужно поменять опции по умолчанию, то экспортируйте нужные [переменные среды](#переменные-среды). Например:
```bash
export WEBHOOK_URL=https://example.com/webhook
export BALANCE_USSD="*105#"
/usr/local/bin/sms-modem-reader
```

Для удобства можно создать systemd сервис:
`/etc/systemd/system/sms-modem-reader.service`:
```
[Unit]
Description=Sms modem reader

[Install]
WantedBy=multi-user.target

[Service]
EnvironmentFile=-/etc/default/sms-modem-reader
ExecStart=/usr/local/bin/sms-modem-reader
Restart=on-failure
```
Включаем сервис:
```bash
systemctl daemon-reload
systemctl enable sms-modem-reader
```

В этом файле мы можете указать любые доступные [переменные среды](#переменные-среды)
`/etc/default/sms-modem-reader`:
```
 WEBHOOK_URL=https://example.com/webhook
 BALANCE_USSD="*105#"
```


Если ваш модем не работает в режиме модема, то его можно переключить в этот режим с помощью программы `usb_modeswitch` (пакет `usb-modeswitch` в ubuntu/debian). Для этого можно добавить еще одно udev правило:

`/etc/udev/rules.d/99-sms-modem-reader.rules`:
```
ATTRS{idVendor}=="12d1",ATTRS{idProduct}=="1506",RUN+="/usr/sbin/usb_modeswitch -v 12d1 -p 1506",TAG+="systemd"
```
Все udev аттрибуты можно посмотреть с помощью команды `udevadm info -a /dev/ttyUSB0`

`idVendor` и `idProduct` можно узнать с помощью `lsusb`. Например:
```
# lsusb
Bus 004 Device 001: ID 1d6b:0001 Linux Foundation 1.1 root hub
Bus 003 Device 002: ID 12d1:1506 Huawei Technologies Co., Ltd. Modem/Networkcard
Bus 003 Device 001: ID 1d6b:0002 Linux Foundation 2.0 root hub
...
```
`12d1` - idVendor  
`1506` - idProduct  


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
  * `WEBHOOK_BASIC_AUTH` (умолчание: "false") - включает/отключает http аутентификацию для отправки вебхуков
  * `WEBHOOK_BASIC_AUTH_USER` (умолчание: "admin") - при включенной http аутентификации, используется этот пользователь
  * `WEBHOOK_BASIC_AUTH_PASS` (умолчание: "") -  при включенной http аутентификации, используется этот пароль
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