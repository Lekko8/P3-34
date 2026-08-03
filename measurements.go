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

	err := btWrite("MEAS:AVR")
	if err != nil {
		log.Print(err)
	}

	ans, err := btRead()
	if err != nil {
		log.Print(err)
	}

	return "\n" + ans
}

// Запрашивает метаданные
func getMetaData() string {

	err := btWrite("?")
	if err != nil {
		log.Print(err)
	}

	ans, err := btRead()
	if err != nil {
		log.Print(err)
	}

	return ans + "\r\n"
}

// Запрашивает уровень батареи
func getUBatt() string {

	err := btWrite("UBATT")
	if err != nil {
		log.Print(err)
	}

	ans, err := btRead()
	if err != nil {
		log.Print(err)
	}

	return ans + "\r\n"
}

// Проверяет соединение
func getTestPing() (string, bool) {
	start := time.Now()
	defer log.Print("getTestPing: ", time.Since(start))

	err := btWrite("TEST")
	if err != nil {
		log.Print("getTestPing - btWrite: ", err)
		return "Ошибка отправки сообщения", false
	}

	answer, err := btRead()
	if err != nil {
		log.Print("getTestPing - btRead: ", err)
		return "Ошибка чтения", false
	}

	return answer, true
}
