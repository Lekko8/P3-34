package main

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

var (
	port   serial.Port
	portMu sync.Mutex
)

// Отправляет сообщение по открытому коннекту
func btWrite(message string) error {

	errChan := make(chan error, 1)
	go func() {
		_, err := port.Write([]byte(message + "\r\n"))
		errChan <- err
	}()

	select {
	case err := <-errChan:
		//log.Print("btWrite: ", err)
		return err
	case <-time.After(1200 * time.Millisecond):
		log.Print("btWrite: timeout")
		return errors.New("timeout of writing in port")
	}
}

// Читает сообщение
func btRead() (string, error) {

	err := port.SetReadTimeout(3 * time.Second)
	if err != nil {
		log.Print("connectBt.SetReadTimeout: ", err)
	}

	data := make([]byte, 1024)
	done := make(chan struct{})
	var n int
	var goerr error

	go func() {
		n, goerr = port.Read(data)
		close(done)
	}()

	select {
	case <-done:
		if goerr != nil {
			return "Ошибка чтения", goerr
		}
		return strings.Split(string(data[:n]), "EOT")[0], nil
	case <-time.After(1200 * time.Millisecond):
		return "Таймаут чтения", errors.New("timeout of reading from port")
	}
}

// Открытие порта
func openPort() error {
	mode := &serial.Mode{BaudRate: 9600}
	p, err := serial.Open("COM4", mode)
	if err != nil {
		log.Print(err)
	}
	port = p
	return err
}

// Закрытие порта
func closePort() error {
	portMu.Lock()
	defer portMu.Unlock()
	if port != nil {
		err := port.Close()
		if err != nil {
			log.Print(err)
		}
		return err
	}
	return nil
}

// Перезапуск порта
func reconnectPort() error {
	portMu.Lock()
	defer portMu.Unlock()
	log.Print("reconnecting port: ", port)
	if port != nil {
		err := port.Close()
		if err != nil {
			log.Print("port.Close(): ", err)
		}
		port = nil
	}
	portMu.Unlock()
	time.Sleep(800 * time.Millisecond)
	portMu.Lock()
	err := openPort()
	log.Print("reconnected port: ", port)
	if err != nil {
		log.Print("openPort(): ", err)
		return err
	}
	return nil
}

// Читает список доступных портов и отдаёт их список
func getPortsList() string {
	//start := time.Now()
	//defer log.Print(time.Since(start))

	ports, err := serial.GetPortsList()
	if err != nil {
		log.Print(err)
	}
	if len(ports) != 0 {
		var portList string
		for _, port := range ports {
			portList = portList + "\r\n" + port
			log.Println(port)
		}

		return "Портов: " + strconv.Itoa(len(ports)) + "\n\r" + portList
	}
	return "Порты не найдены, проверьте соединение с P3-34"
}
