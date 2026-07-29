package main

import (
	"log"
	"strconv"
	"time"

	"go.bug.st/serial"
)

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
