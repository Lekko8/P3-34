package main

import (
	"log"
	"sync"
	"time"

	"go.bug.st/serial"
)

var (
	port   serial.Port
	portMu sync.Mutex
)

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
