package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
	"golang.org/x/sync/errgroup"
)

var (
	port   serial.Port
	portMu sync.RWMutex
)

const (
	testMessage  = "TEST"
	testResponse = "OK" // Предполагаемый ответ на тестовое сообщение
	openTimeout  = 3 * time.Second
	readTimeout  = 500 * time.Millisecond
	writeTimeout = 1 * time.Second
)

// Отправляет сообщение по открытому коннекту
func btWrite(message string) error {
	portMu.RLock()
	currentPort := port
	portMu.RUnlock()

	if currentPort == nil {
		return errors.New("port not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		_, err := currentPort.Write([]byte(message + "\r\n"))
		errChan <- err
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return errors.New("timeout of writing in port")
	}
}

// Общая функция чтения ответа от устройства
func readResponse(p serial.Port) (string, error) {
	var answer strings.Builder
	buffer := make([]byte, 2048)
	const EOT = 0x04

	for {
		if err := p.SetReadTimeout(readTimeout); err != nil {
			return answer.String(), err
		}

		n, err := p.Read(buffer)
		if err != nil {
			if err.Error() == "i/o timeout" || err == io.EOF {
				return answer.String(), nil
			}
			return answer.String(), err
		}

		if n > 0 {
			// Ищем EOT в полученных данных
			data := buffer[:n]
			if idx := strings.IndexByte(string(data), EOT); idx >= 0 {
				// Нашли EOT - добавляем данные до него и выходим
				answer.Write(data[:idx])
				return answer.String(), nil
			}

			// EOT не найден - добавляем все данные
			answer.Write(data)
		}
	}
}

// Читает сообщение
func btRead() (string, error) {
	portMu.RLock()
	currentPort := port
	portMu.RUnlock()

	if currentPort == nil {
		return "", errors.New("port not initialized")
	}

	//start := time.Now()
	//defer log.Print("btRead()", time.Since(start))

	return readResponse(currentPort)
}

// Проверяет, является ли порт целевым устройством
func isTargetDevice(p serial.Port, portName string) bool {
	// Отправляем тестовое сообщение
	if _, err := p.Write([]byte(testMessage + "\r\n")); err != nil {
		log.Printf("Write TEST to %s: %v", portName, err)
		return false
	}

	// Читаем весь ответ
	response, err := readResponse(p)
	if err != nil {
		log.Printf("Read response from %s: %v", portName, err)
		return false
	}

	log.Printf("Port %s response: %q", portName, response)

	// Проверяем ответ
	return response != "" &&
		(strings.Contains(response, "ready >>>") || strings.Contains(response, testResponse))
}

// Открытие порта с поиском целевого устройства
func openPort() error {
	portMu.Lock()
	defer portMu.Unlock()

	// Закрываем текущий порт, если он открыт
	if port != nil {
		log.Printf("Closing existing port before reopening")
		if err := port.Close(); err != nil {
			log.Printf("Error closing existing port: %v", err)
		}
		port = nil
	}

	// Пробуем открыть сохраненный порт
	if savedPort, err := readSavedPort(); err == nil && savedPort != "" {
		log.Printf("Trying saved port: %s", savedPort)
		if p, err := tryOpenPort(savedPort); err == nil {
			if isTargetDevice(p, savedPort) {
				port = p
				D.Port = savedPort
				log.Printf("Successfully connected to saved port: %s", savedPort)
				return nil
			}
			if err = p.Close(); err != nil {
				log.Printf("Error closing saved port: %v", err)
			}
			log.Printf("Saved port %s is not target device", savedPort)
		}
	}

	// Сканируем все порты
	return scanPorts()
}

// Пытается открыть конкретный порт
func tryOpenPort(portName string) (serial.Port, error) {
	mode := &serial.Mode{
		BaudRate: 9600,
		DataBits: 8,
		StopBits: serial.OneStopBit,
		Parity:   serial.NoParity,
	}

	ctx, cancel := context.WithTimeout(context.Background(), openTimeout)
	defer cancel()

	resultCh := make(chan struct {
		p   serial.Port
		err error
	}, 1)

	go func() {
		p, err := serial.Open(portName, mode)
		resultCh <- struct {
			p   serial.Port
			err error
		}{p, err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout opening port %s", portName)
	case result := <-resultCh:
		return result.p, result.err
	}
}

// Сканирует все доступные порты
func scanPorts() error {
	ports, err := serial.GetPortsList()
	if err != nil {
		return fmt.Errorf("getting ports list: %w", err)
	}

	if len(ports) == 0 {
		return errors.New("no serial ports found")
	}

	log.Printf("Found %d ports: %v", len(ports), ports)

	ctx, cancel := context.WithTimeout(context.Background(), openTimeout*2)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)
	foundCh := make(chan struct {
		p        serial.Port
		portName string
	}, 1)

	// Канал для отслеживания найденного порта
	found := make(chan struct{})
	var foundOnce sync.Once

	for _, portName := range ports {

		g.Go(func() error {
			// Проверяем, не найден ли уже порт
			select {
			case <-ctx.Done():
				return nil
			case <-found:
				return nil
			default:
			}

			log.Printf("Scanning port: %s", portName)
			p, err := tryOpenPort(portName)
			if err != nil {
				log.Printf("Cannot open port %s: %v", portName, err)
				return nil
			}

			// Гарантируем закрытие порта, если он не целевой
			defer func() {
				if p != nil {
					if err := p.Close(); err != nil {
						log.Printf("Error closing port: %v", err)
					}
					log.Printf("Closed port %s after scanning", portName)
				}
			}()

			if isTargetDevice(p, portName) {
				foundOnce.Do(func() {
					close(found)
					select {
					case foundCh <- struct {
						p        serial.Port
						portName string
					}{p, portName}:
						log.Printf("Target device found on port: %s", portName)
						p = nil // Предотвращаем закрытие в defer
					default:
						log.Printf("Found channel is full, closing port %s", portName)
					}
				})
			}

			return nil
		})
	}

	// Ждем завершения всех горутин
	go func() {
		if err := g.Wait(); err != nil {
			log.Printf("Error scanning ports: %v", err)
		}
		close(foundCh)
	}()

	// Ждем результат
	select {
	case result, ok := <-foundCh:
		if !ok {
			return errors.New("target device not found")
		}
		port = result.p
		D.Port = result.portName

		// Сохраняем найденный порт
		//if err := savePortName(result.portName); err != nil {
		//	log.Printf("Cannot save port name: %v", err)
		//}

		log.Printf("Successfully connected to port: %s", result.portName)
		return nil

	case <-ctx.Done():
		return errors.New("timeout scanning ports")
	}
}

// Читает сохраненное имя порта
//func readSavedPort() (string, error) {
//
//	if fileExists(settingsFileName) {
//		log.Printf("Saved port not found")
//		data, err := os.ReadFile(settingsFileName)
//		if err != nil {
//			return "", err
//		}
//
//		portName := strings.TrimSpace(string(data))
//		if portName == "" {
//			return "", errors.New("empty port name")
//		}
//
//		return portName, nil
//	}
//
//	return "", nil
//}

func readSavedPort() (string, error) {
	f, err := os.Open(settingsFileName)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("Error closing saved port: %v", err)
		}
	}()

	scanner := bufio.NewScanner(f)
	scanner.Scan()
	portName := strings.TrimSpace(scanner.Text())
	if portName == "" {
		return "", errors.New("empty port name")
	}
	return portName, nil
}

// Сохраняет имя порта
//func savePortName(portName string) error {
//	return os.WriteFile(portFile, []byte(portName), 0644)
//}

// Закрытие порта
func closePort() error {
	portMu.Lock()
	defer portMu.Unlock()

	if port != nil {
		err := port.Close()
		port = nil
		if err != nil {
			log.Print("closePort: ", err)
		}
		return err
	}
	return nil
}

// Перезапуск порта
func reconnectPort() error {
	log.Print("Starting port reconnection...")

	// Закрываем текущий порт
	if err := closePort(); err != nil {
		log.Printf("Error closing port during reconnect: %v", err)
	}

	// Ждем, пока порт полностью освободится
	time.Sleep(1 * time.Second)

	// Открываем порт заново
	if err := openPort(); err != nil {
		log.Printf("Error opening port during reconnect: %v", err)
		return err
	}

	log.Printf("Port reconnected successfully: %s", D.Port)
	return nil
}

// Читает список доступных портов и отдаёт их список
func getPortsList() string {
	start := time.Now()
	defer log.Print("getPortsList: ", time.Since(start))

	ports, err := serial.GetPortsList()
	if err != nil {
		log.Print("getPortsList: ", err)
		return "Ошибка получения списка портов"
	}

	if len(ports) == 0 {
		return "Порты не найдены, проверьте соединение с P3-34"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Портов: %d\r\n", len(ports)))

	for _, portName := range ports {
		builder.WriteString("\r\n")
		builder.WriteString(portName)
		log.Println("getPortsList: ", portName)
	}

	return builder.String()
}
