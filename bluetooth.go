package main

import (
	"errors"
	"log"
	"os"
	"strconv"
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

	if port == nil {
		return errors.New("port not initialized")
	}

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
	start := time.Now()
	defer log.Print("btRead()", time.Since(start))

	var answer []byte // Используем []byte для эффективности
	buffer := make([]byte, 2048)
	const EOT = 0x04

	for {
		if err := port.SetReadTimeout(100 * time.Millisecond); err != nil {
			return "", err
		}

		n, err := port.Read(buffer)
		if err != nil {
			if err.Error() == "i/o timeout" {
				return string(answer), nil
			}
			return "", err
		}

		if n > 0 {
			// Ищем EOT в полученных данных
			eotIndex := -1
			for i := range n {
				if buffer[i] == EOT {
					eotIndex = i
					break
				}
			}

			if eotIndex >= 0 {
				// Нашли EOT - добавляем данные до него и выходим
				answer = append(answer, buffer[:eotIndex]...)
				return string(answer), nil
			}

			// EOT не найден - добавляем все данные
			answer = append(answer, buffer[:n]...)
		}
	}
}

//func btRead() (string, error) {
//
//	start := time.Now()
//	defer log.Print("btRead()", time.Since(start))
//
//	//err := port.SetReadTimeout(2 * time.Second)
//	//if err != nil {
//	//	log.Print("connectBt.SetReadTimeout: ", err)
//	//}
//
//	var answer strings.Builder
//	ticker := time.NewTicker(100 * time.Millisecond)
//	defer ticker.Stop()
//
//	for {
//		data := make([]byte, 2048)
//		done := make(chan struct{})
//		var n int
//		var goerr error
//
//		go func() {
//			n, goerr = port.Read(data)
//			close(done)
//		}()
//
//		select {
//		case <-done:
//			if goerr != nil {
//				return answer.String(), goerr
//			}
//			// strings.Split(string(data[:n]), "EOT")[0]
//			answer.Write(data[:n])
//			//return answer.String(), nil
//		case <-ticker.C:
//			log.Print("btRead: timeout")
//			return answer.String(), nil
//			//return "Таймаут чтения", errors.New("timeout of reading from port")
//		}
//	}
//}

// Открытие порта
func openPort() error {
	filePort, err := os.OpenFile("Port.txt", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Print("opening filePort", err)
		return err
	}
	defer func() {
		if err := filePort.Close(); err != nil {
			log.Print("closing filePort: ", err)
		}
	}()

	var portNameB = make([]byte, 4)

	_, err = filePort.Read(portNameB)
	if err != nil {
		log.Print("Reading filePort: ", err)
	}

	mode := &serial.Mode{BaudRate: 9600}
	ch := make(chan serial.Port)
	errChan := make(chan error, 1)

	if portNameB != nil {
		portName := string(portNameB)
		log.Print("Opening port ", portName)
		if err != nil {
			return err
		}
		go func(mode *serial.Mode) {
			p, err := serial.Open(portName, mode)
			if err != nil {
				log.Print("go openPort - serial.Open("+portName+", mode): ", err)
				errChan <- err
			}
			ch <- p
		}(mode)

		select {
		case err := <-errChan:
			return err
		case p := <-ch:
			port = p
			D.Port = portName
			return err
		case <-time.After(1000 * time.Millisecond):
			return errors.New("timeout opening port")
		}
	} else {
		portNameCh := make(chan string)
		ports, err := serial.GetPortsList()
		if err != nil {
			return err
		}

		for _, port := range ports {
			go func(mode *serial.Mode) {
				p, err := serial.Open(port, mode)
				if err != nil {
					log.Print("openPort - serial.Open("+port+", mode): ", err)
				}
				ch <- p
				portNameCh <- port
			}(mode)
		}

		select {
		case p := <-ch:
			portName := <-portNameCh
			port = p // port - глобальная переменная
			D.Port = portName

			_, err = filePort.Write([]byte(portName))

			return err
		case <-time.After(1000 * time.Millisecond):
			return errors.New("timeout opening port")
		}
	}
}

// Закрытие порта
func closePort() error {
	portMu.Lock()
	defer portMu.Unlock()
	if port != nil {
		err := port.Close()
		if err != nil {
			log.Print("closePort: ", err)
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
	start := time.Now()
	defer log.Print("getPortsList: ", time.Since(start))

	ports, err := serial.GetPortsList()
	if err != nil {
		log.Print("getPortsList: ", err)
	}
	if len(ports) != 0 {
		var portList string
		for _, port := range ports {
			portList = portList + "\r\n" + port
			log.Println("getPortsList: ", port)
		}

		return "Портов: " + strconv.Itoa(len(ports)) + "\n\r" + portList
	}
	return "Порты не найдены, проверьте соединение с P3-34"
}
