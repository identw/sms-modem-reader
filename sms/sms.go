package sms

import (
	"log"
	"fmt"
	"os"
	"bytes"
	"time"
	"regexp"
	"github.com/tarm/serial"
)

var (
	regexUCS2 = regexp.MustCompile(`^[0-9ABCDEF]+$`)
)

type SmsOperator struct {
	Messages map[string]SmsMessage
	Port *serial.Port
	SerialFile string
	SerialBaud int
	SerialStopBits byte
	SerialParity byte
	SerialSize byte
	SerialReadTimeout time.Duration
}

type SmsMessage struct {
	Id string
	Ids []string
	Status string
	Number string
	Unknown string
	Time string
	Text string
}

func (so *SmsOperator) Init() {
	c := &serial.Config{
		Name: so.SerialFile, 
		Baud: so.SerialBaud, 
		StopBits: serial.StopBits(so.SerialStopBits),
		Parity: serial.Parity(so.SerialParity),
		Size: so.SerialSize,
		ReadTimeout: 10 * time.Second,
	}
	
	for {
		var err error
		so.Port, err = serial.OpenPort(c)
		if err != nil {
			log.Print(err)
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}
	so.Messages = make(map[string]SmsMessage)
}

func (so *SmsOperator) SimCommand(command string) ([]byte, error) {
	answer := make([]byte, 0)
	nbyte := 0
	end := []byte("OK\r\n")
	simError := []byte("ERROR\r\n")
	buf := make([]byte, 256)

	_, err := so.Port.Write([]byte(command + "\r\n"))
	if err != nil {
		log.Printf("%s\n", err)
		return nil, err
	}
	for {
		n, err := so.Port.Read(buf)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Read error:")
			fmt.Fprintln(os.Stderr, err)
			return nil, err
		}
		nbyte += n
		for _, v := range buf[:n] {
			answer = append(answer, v)
		}
		if nbyte >= 4 && bytes.Equal(answer[nbyte-4:nbyte], end) {
			break
		}
		if nbyte >= 7 && bytes.Equal(answer[nbyte-7:nbyte], simError) {
			return nil, fmt.Errorf("sim command error")
		}
	}
	return answer, nil
}

func (so *SmsOperator) CUSD(command string) (string, error) {
	answer := make([]byte, 0)
	nbyte := 0
	simError := []byte("ERROR\r\n")
	buf := make([]byte, 256)
	_, err := so.Port.Write([]byte(fmt.Sprintf("AT+CUSD=1,\"%02X\",15\r\n", Encode7Bit(command))))
	if err != nil {
		log.Printf("%s\n", err)
		return "", err
	}
	for {
		n, err := so.Port.Read(buf)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Read error:")
			fmt.Fprintln(os.Stderr, err)
			return "", err
		}
		nbyte += n
		for _, v := range buf[:n] {
			answer = append(answer, v)
		}
		if regexp.MustCompile(`(?s)\r\n\+CUSD:.*\r\n`).MatchString(string(answer)) {
			textByte := regexp.MustCompile(`(?s)\r\n\+CUSD:.*"([^"]+)".*\r\n`).FindSubmatch(answer)[1]
			text, err := DecodeUcs2(bytesToHex(textByte), false)
			if err != nil {
				return "", fmt.Errorf("cusd read error: %s", err)
			}
			return text, nil
		}
		if nbyte >= 7 && bytes.Equal(answer[nbyte-7:nbyte], simError) {
			return "", fmt.Errorf("cusd command error")
		}
	}
	return "", fmt.Errorf("not found cusd answer")
}


func (so *SmsOperator) ReadAllSms() (error) {
	so.SimCommand("AT+CMGF=1")
	allSmsBytes, err := so.SimCommand("AT+CMGL=\"ALL\"")
	if err != nil {
		return err
	}
	splitSms := bytes.Split(allSmsBytes, []byte("\r\n"))

	for i := 0; i < len(splitSms); i++ {
		if len(splitSms[i]) > 0 && string(splitSms[i][0]) == "+" {
			split := bytes.Split(splitSms[i], []byte(","))
			id := (bytes.Split(split[0], []byte(": ")))[1]
			var text string
			if (regexUCS2.MatchString(string(splitSms[i+1]))) {
				text, _ = DecodeUcs2(bytesToHex(splitSms[i+1]), false)
			} else {
				text = string(splitSms[i+1])
			}
			message := SmsMessage{
				Id: string(id),
				Status: string(split[1]),
				Number: string(split[2]),
				Unknown: string(split[3]),
				Time: string(split[4]) + " " + string(split[5]),
				Text: text,
			}
			if _, ok := so.Messages[message.Time]; ok { 
				// if id exist in messages
				if so.checkExistId(message.Time, message.Id){
					continue
				}
				m := so.Messages[message.Time]
				m.Text += text
				if len(m.Ids) > 0 {
					m.Ids = append(m.Ids, message.Id)
				} else {
					m.Ids = make([]string, 0)
					m.Ids = append(m.Ids, message.Id)
				}
				so.Messages[message.Time] = m
			} else {
				so.Messages[message.Time] = message
			}
		}
	}
	return nil
}

func (so *SmsOperator) RemoveSms(key string) {
	if _, ok := so.Messages[key]; ok { 
		if len(so.Messages[key].Ids) > 0 {
			for _, id := range so.Messages[key].Ids {
				so.SimCommand("AT+CMGD=" + id)
			}
		} else {
			so.SimCommand( "AT+CMGD=" + so.Messages[key].Id)
		}
		delete(so.Messages, key)
	}
}

func (so *SmsOperator) checkExistId(key string, id string) bool {
	for _, v := range so.Messages[key].Ids {
		if id == v {
			return true
		}
	}
	return false
}