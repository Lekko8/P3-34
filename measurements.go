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

	for {
		select {
		case <-ctx.Done():
			return "Измерение прервано"

		case <-ticker.C:
			elapsed := time.Since(start)

			progress := max(min(int((elapsed.Seconds()/s.Seconds())*100), 100), 0)
			runner.UpdateProgress(progress)
			if elapsed >= s {
				runner.UpdateProgress(100)
				return getMeas()
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

// Проводит измерение
func getMeas() string {

	ans := get("MEAS:AVR") + "\r\n" + get("MEAS:MAX") + "\r\n" + get("NEWMEAS")

	addToCSV(ans)

	return ans
}

// Запрашивает информацию перед измерением
func getInfo() string {

	ans := get("?") + get("UBATT") + get("NEWMEAS")

	addToCSV(ans)

	return ans
}

// Отправляет и читает по запросу
func get(this string) string {

	err := btWrite(this)
	if err != nil {
		log.Print(err)
	}

	ans, err := btRead()
	if err != nil {
		log.Print(err)
	}

	return "\n" + ans
}

// Устанавливает частоту
func setFreq(v string) string {

	err := btWrite("FREQ" + v)
	if err != nil {
		log.Print(err)
	}

	ans, err := btRead()
	if err != nil {
		log.Print(err)
	}

	return "\n" + ans
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
