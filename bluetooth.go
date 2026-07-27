package main

import (
	"log"

	"go.bug.st/serial"
)

// Открывает коннект по стандартному порту
func bt() serial.Port {

	mode := &serial.Mode{
		BaudRate: 9600,
	}

	port, err := serial.Open("COM4", mode)
	if err != nil {
		log.Print(err)
		err = port.Close()
		if err != nil {
			log.Print(err.Error())
		}
	}

	return port
}

func getPortsList() int {
	ports, err := serial.GetPortsList()
	if err != nil {
		log.Print(err)
	}
	if len(ports) != 0 {
		for _, port := range ports {
			log.Println(port)
		}
	}
	return len(ports)
}
