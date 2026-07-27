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

			progress := int((elapsed.Seconds() / s.Seconds()) * 100)
			if progress > 100 {
				progress = 100
			}
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
						log.Print(err.Error())
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

// Тестово пингует соединение
func getTestPing() string {

	if bt() == nil {
		err := bt().Close()
		if err != nil {
			log.Print(err.Error())
		}
		return "Ошибка соединения"
	}

	result := make(chan []byte, 1)

	_, err := bt().Write([]byte("TEST"))
	if err != nil {
		log.Print(err.Error())
	}

	go func() {

		data := make([]byte, 1024)
		result = make(chan []byte, 1)

		n, err := bt().Read(data)
		if err != nil {
			result <- []byte("Ошибка чтения")
			log.Print(err.Error())
		}
		result <- data[:n]
	}()

	select {
	case data := <-result:
		_ = bt().Close()
		return string(data)
	case <-time.After(2 * time.Second):
		_ = bt().Close()
		return "таймаут соединения"
	}
}
