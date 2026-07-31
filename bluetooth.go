package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"go.bug.st/serial"
)

// Отправляет сообщение по открытому коннекту
func btWrite(connectBt serial.Port, message string) error {

	errChan := make(chan error)

	go func() {
		_, err := connectBt.Write([]byte(message + "\r\n"))
		if err != nil {
			log.Print("connectBt.Write: ", err)
		}

		log.Print("Коннект Write: ", message)
		errChan <- err
	}()
	return <-errChan
}

// Читает сообщение
func btRead(ctx context.Context, connectBt serial.Port, result chan []byte) {

	data := make([]byte, 1024)

	n, err := connectBt.Read(data)
	if err != nil {
		result <- []byte("Ошибка чтения ответа")
		log.Print("Ошибка чтения: ", err)
		ctx.Done()
	}

	result <- data[:n]
	ctx.Done()
}

// Открывает коннект по стандартному порту
func bt() serial.Port {
	result := make(chan serial.Port, 1)

	go func() {
		defer close(result)

		mode := &serial.Mode{
			BaudRate: 9600,
		}

		port, err := serial.Open("COM4", mode)
		if err != nil {
			log.Print("Ошибка открытия порта: ", err)
			if port != nil {
				result <- nil
				err = port.Close()
				if err != nil {
					log.Print("Ошибка закрытия порта: ", err)
				}
				return
			}
		}
		result <- port
	}()
	select {
	case port := <-result:
		return port
	case <-time.After(time.Millisecond * 500):
		return nil
	}
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
