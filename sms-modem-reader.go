package main

import (
	"log"
	"fmt"
	"time"
	"net/http"
	"sync"
	"regexp"
	"strconv"
	"encoding/json"
	"bytes"
	"os"
	"github.com/identw/sms-modem-reader/sms"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MetricPort (env: METRIC_PORT) - metric port
	MetricPort = "40111"
	// MetricListenIP (env: METRIC_LISTEN_IP)  - metric interface
	MetricListenIP = "0.0.0.0"
	// BalanceUSSD (env: BALANCE_USSD) - ussd for check balance - check balance (defailt for rostelecom)
	BalanceUSSD = "*122#"
	// BalanceParse (env: BALANCE_PARSE) - regex for find balance from sms (default for rostelecom)
	BalanceParse = regexp.MustCompile(`^.*составляет ([0-9]+(\.[0-9]+)?) руб.*$`)
	// IntervalReadSms (env: INTERVAL_READ_SMS) - Interval read sms in seconds
	IntervalReadSms int64 = 10
	// IntervalCheckBalance (env: INTERVAL_CHECK_BALANCE) - interval check balance in seconds
	IntervalCheckBalance int64 = 24 * 3600
	// WebhookURL (env: WEBHOOK_URL) - webhook url
	WebhookURL = "http://127.0.0.1/webhook"
	// SerialFile (env: SERIAL_FILE) - serial file
	SerialFile = "/dev/ttyUSB0"
	// SerialBaud (env: SERIAL_BAUD) - serial baud rate
	SerialBaud = 9600
	// SerialStopBits (env: SERIAL_STOP_BITS) - serial stop bits
	SerialStopBits byte = 1
	// SerialParity (env: SERIAL_PARITY)- serial parity
	SerialParity byte = 'N'
	// SerialSize (env: SERIAL_SIZE) - serial bits  
	SerialSize byte = 8

	promReadSmsLastStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sms_reader_sms_last_status",
		Help: "read sms correct, 1 = fail, 0 = success",
	})

	promBalance = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sms_reader_balance",
		Help: "current balance",
	})

	promBalanceLastCheck = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sms_reader_balance_last_check",
		Help: "unix timestamp last check balance",
	})

	promWebhookLastStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sms_reader_webhook_last_status",
		Help: "send webhook last send fail, 1 = fail, 0 = success",
	})
	mu sync.Mutex
)

// WebhookMessage - message for webhook
type WebhookMessage struct {
	Message string `json:"message"`
}

func main() {

	// read from env
	var err error
	if os.Getenv("METRIC_PORT") != "" {
		MetricPort = os.Getenv("METRIC_PORT")
	}
	if os.Getenv("METRIC_LISTEN_IP") != "" {
		MetricListenIP = os.Getenv("METRIC_LISTEN_IP")
	}
	if os.Getenv("BALANCE_USSD") != "" {
		BalanceUSSD = os.Getenv("BALANCE_USSD")
	}
	if os.Getenv("BALANCE_REGEXP") != "" {
		BalanceParse = regexp.MustCompile(os.Getenv("BALANCE_REGEXP"))
	}
	if os.Getenv("INTERVAL_READ_SMS") != "" {
		IntervalReadSms, err = strconv.ParseInt(os.Getenv("INTERVAL_READ_SMS"), 10, 64)
		if err != nil {
			log.Fatalf("ParseInt error env 'INTERVAL_READ_SMS': %s", err)
		}
	}
	if os.Getenv("INTERVAL_CHECK_BALANCE") != "" {
		IntervalCheckBalance, err = strconv.ParseInt(os.Getenv("INTERVAL_CHECK_BALANCE"), 10, 64)
		if err != nil {
			log.Fatalf("ParseInt error env 'INTERVAL_CHECK_BALANCE': %s", err)
		}
	}
	if os.Getenv("WEBHOOK_URL") != "" {
		WebhookURL = os.Getenv("WEBHOOK_URL")
	}
	if os.Getenv("SERIAL_FILE") != "" {
		SerialFile = os.Getenv("SERIAL_FILE")
	}
	if os.Getenv("SERIAL_BAUD") != "" {
		tmp, err := strconv.ParseInt(os.Getenv("SERIAL_BAUD"), 10, 64)
		if err != nil {
			log.Fatalf("ParseInt error env 'SERIAL_BAUD': %s", err)
		}
		SerialBaud = int(tmp)
	}
	if os.Getenv("SERIAL_STOP_BITS") != "" {
		tmp, err := strconv.ParseInt(os.Getenv("SERIAL_STOP_BITS"), 10, 64)
		if err != nil {
			log.Fatalf("ParseInt error env 'SERIAL_STOP_BITS': %s", err)
		}
		if tmp != 1 && tmp != 2 && tmp != 15 {
			log.Fatal("SERIAL_STOP_BITS only supports the following values: '1', '2', '15'")
		}
		SerialStopBits = byte(tmp)
	}
	if os.Getenv("SERIAL_PARITY") != "" {
		tmp := os.Getenv("SERIAL_PARITY")
		if 	tmp[0] != 'N' && 
			tmp[0] != 'O' &&
			tmp[0] != 'E' &&
			tmp[0] != 'M' &&
			tmp[0] != 'S' {

			log.Fatal("SERIAL_PARITY only supports the following values: 'N', 'O', 'E', 'M', 'S'")
		}
		SerialParity = byte(tmp[0])
	}
	if os.Getenv("SERIAL_SIZE") != "" {
		tmp, err := strconv.ParseInt(os.Getenv("SERIAL_SIZE"), 10, 64)
		if err != nil {
			log.Fatalf("ParseInt error env 'SERIAL_SIZE': %s", err)
		}
		SerialSize = byte(tmp)
	}
	
	// init sms operator
	smsOperator := sms.SmsOperator{
		SerialFile: SerialFile,
		SerialBaud: SerialBaud,
		SerialStopBits: SerialStopBits,
		SerialParity: SerialParity,
		SerialSize: SerialSize,
	}
	smsOperator.Init()

	// Init prom
	promBalance.Set(1000.0)
	promBalanceLastCheck.Set(float64(time.Now().Unix()))
	promReadSmsLastStatus.Set(0)
	promWebhookLastStatus.Set(0)
	
	// run check balance by interval
	go checkBalance(&smsOperator)
	// run sms reader by interval
	go readSmsByTimer(&smsOperator)

	// expose metrics
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(fmt.Sprintf("%s:%s", MetricListenIP, MetricPort), nil)
}

func readSmsByTimer(so *sms.SmsOperator) {
	for {
		mu.Lock()
		err := so.ReadAllSms()
		if err != nil {
			promReadSmsLastStatus.Set(1)
			log.Print(err)
			time.Sleep(10 * time.Second)
			so.Init()
			continue
		}
		
		for k, v := range so.Messages {
			log.Print(v)

			// if check balance
			if BalanceParse.MatchString(v.Text) {
				b, _ := strconv.ParseFloat(string(BalanceParse.FindSubmatch([]byte(v.Text))[1]), 64)
				promBalance.Set(b)
				promBalanceLastCheck.Set(float64(time.Now().Unix()))
			}
			
			// Send message to webhook
			message := WebhookMessage{
				Message: v.Text,
			}
			jsonStr, _ := json.Marshal(message)
			req, err := http.NewRequest("POST", WebhookURL, bytes.NewBuffer(jsonStr))
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil || resp.Status != "200" {
				promWebhookLastStatus.Set(1)
				continue
			}
			defer resp.Body.Close()
			promWebhookLastStatus.Set(0)
			
			// remove sms
			so.RemoveSms(k)
		}
		mu.Unlock()
		time.Sleep(time.Duration(IntervalReadSms) * time.Second)
	}
}

func checkBalance(so *sms.SmsOperator) {
	for {
		time.Sleep(11 * time.Second)
		mu.Lock()
		log.Print("check balance")
		balance, err := so.CUSD(BalanceUSSD)
		if err != nil {
			log.Printf("check balance error ussd: %s", err)
			promBalance.Set(0)
			continue
		}
		log.Printf("Balance ussd answer: %s", balance)
		if BalanceParse.MatchString(balance) {
			b, _ := strconv.ParseFloat(string(BalanceParse.FindSubmatch([]byte(balance))[1]), 64)
			promBalance.Set(b)
			promBalanceLastCheck.Set(float64(time.Now().Unix()))
		}
		mu.Unlock()
		time.Sleep(time.Duration(IntervalCheckBalance) * time.Second + 13 * time.Second)
	}
}