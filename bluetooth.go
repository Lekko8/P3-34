package main

import (
	"log"
	"strconv"
	"time"

	"go.bug.st/serial"
)

// Открывает коннект по стандартному порту
func bt() serial.Port {
	start := time.Now()

	mode := &serial.Mode{
		BaudRate: 9600,
	}

	log.Print("bt до открытия: ", time.Since(start))
	port, err := serial.Open("COM4", mode)
	if err != nil {
		log.Print("bt до ошибки: ", time.Since(start))
		log.Print(err)
		//err = port.Close()
		//if err != nil {
		//	log.Print(err.Error())
		//}
	}
	log.Print("bt: ", time.Since(start))
	return port
}

func getPortsList() string {
	start := time.Now()
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
		log.Print(time.Since(start))
		return "Портов: " + strconv.Itoa(len(ports)) + "\n\r" + portList
	}
	log.Print(time.Since(start))
	return "Порты не найдены, проверьте соедение с P3-34"
}
