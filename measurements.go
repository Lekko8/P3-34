package main

import (
	"log"
	"strings"
	"time"
)

// Проводит измерение
func getMeas() string {

	ans := get("MEAS:AVR") + get("MEAS:MAX") + get("NEWMEAS")
	//+ "\r\n"
	addToCSV(ans)

	return ans
}

// Запрашивает информацию перед измерением и устанавливает частоту
func getInfo(freq string) string {

	ans := get("?") + get("UBATT")

	if freq != "" {
		ans += get("FREQ:" + strings.Replace(freq, ",", ".", 1))
	}

	ans += get("NEWMEAS")

	addToCSV(ans)

	return ans
}

// Отправляет и читает по запросу
func get(this string) string {

	err := btWrite(this)
	if err != nil {
		log.Print("get btWrite(): ", err)
	}

	ans, err := btRead()
	if err != nil {
		log.Print("get btRead(): ", err)
	}

	log.Print("get [", this, "]: ", ans)
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
