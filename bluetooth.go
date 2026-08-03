package main

import (
	"errors"
	"log"
	"strconv"
	"time"

	"go.bug.st/serial"
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
		log.Print("btWrite: ", err)
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
		return string(data[:n]), nil
	case <-time.After(1200 * time.Millisecond):
		return "Таймаут чтения", errors.New("timeout of reading from port")
	}
}

// Открывает коннект по стандартному порту
//func bt(ctx context.Context) serial.Port {
//
//	result := make(chan serial.Port, 1)
//
//	go func() {
//		log.Print("Открываю порт")
//
//		mode := &serial.Mode{
//			BaudRate: 9600,
//		}
//
//		port, err := serial.Open("COM4", mode)
//		if err != nil {
//			log.Print("Ошибка открытия порта: ", err)
//			close(result)
//			return
//		}
//
//		select {
//		case <-ctx.Done():
//			err := port.Close()
//			if err != nil {
//				log.Print(err)
//			}
//			close(result)
//			return
//		default:
//			result <- port
//			log.Print("Выключаю go открыть порт")
//		}
//	}()
//
//	select {
//	case port, ok := <-result:
//		if !ok {
//			log.Print("канал почему-то закрыт")
//			return nil
//		}
//		return port
//	case <-ctx.Done():
//		log.Print("контекст закрыт")
//		return nil
//	case <-time.After(time.Millisecond * 500):
//		log.Print("таймаут")
//		return nil
//	}
//}

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
