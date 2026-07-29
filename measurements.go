package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Запускает новый цикл измерений
func startMeas(ctx context.Context, m int, runner *MeasRun) string {
	s := time.Duration(m) * time.Second //TODO: заменить на минуты перед релизом

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	start := time.Now()
	var currentValue string

	for {
		select {
		case <-ctx.Done():
			return "Измерение прервано"

		case <-ticker.C:
			elapsed := time.Since(start)

			progress := max(min(int((elapsed.Seconds()/s.Seconds())*100), 100), 0)
			runner.UpdateProgress(progress)
			if elapsed >= s {
				currentValue = getCurrentValue()
				runner.UpdateProgress(100)
				return currentValue
			}
			if progress%10 == 0 {
				runner.mw.Synchronize(func() {
					err := runner.statusLabel.
						SetText(
							fmt.Sprintf("Текущий цикл: %d/%d",
								runner.currentCycle, runner.totalCycles))
					if err != nil {
						log.Print(err)
					}
				})
			}
		}
	}
}

// Запрашивает текущее значение
func getCurrentValue() string {

	return "\n" + "Текущее значение"
}

// Запрашивает метаданные
func getMetaData() string {
	return "Мета данные" + "\r\n"
}

// Проверяет соединение
func getTestPing() (string, bool) {
	start := time.Now()

	connectBt := bt()

	defer func() {
		if connectBt != nil {
			err := connectBt.Close()
			if err != nil {
				log.Print("getTestPing - connectBt.Close(): ", err)
			}
		}
		log.Print("getTestPing: ", time.Since(start))
	}()

	if connectBt == nil {
		return "Ошибка соединения\r\n", false
	}

	result := make(chan []byte, 1)

	_, err := connectBt.Write([]byte("TEST\r\n"))
	if err != nil {
		log.Print("connectBt.Write: ", err)
	} //TODO: зависает на этом моменте
	log.Print("Коннект")
	go func() {

		data := make([]byte, 1024)
		//result = make(chan []byte, 1)

		n, err := connectBt.Read(data)
		if err != nil {
			result <- []byte("Ошибка чтения")
			log.Print("Ошибка чтения: ", err)
		}
		result <- data[:n]
	}()

	select {
	case data := <-result:
		return string(data), true

	case <-time.After(2 * time.Second):
		return "Таймаут соединения", false
	}
}
