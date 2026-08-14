package main

import (
	"io"
	"log"
	"strings"
	"time"
)

// Проводит измерение
func getMeas() string {

	ans := get("MEAS:AVR") + get("MEAS:MAX") + get("NEWMEAS")
	//+ "\r\n"
	addToCSV(ans)

	return time.Now().String() + "\r\n" + ans
}

// Запрашивает информацию перед измерением и устанавливает частоту
func getInfo(freq string) string {

	var result = strings.Builder{}

	result.WriteString(get("?"))
	log.Print("result.WriteString(get(\"?\"))", result.String())
	time.Sleep(10 * time.Millisecond)

	result.WriteString(get("UBATT"))
	log.Print("result.WriteString(get(\"UBATT\"))", result.String())
	time.Sleep(10 * time.Millisecond)

	if freq != "" {
		result.WriteString(get("FREQ:" + strings.Replace(freq, ",", ".", 1)))
		log.Print("result.WriteString(get(\"FREQ\"))", result.String())
		time.Sleep(10 * time.Millisecond)
	}

	result.WriteString(get("NEWMEAS"))

	ans := time.Now().String() + "\r\n" + result.String()

	addToCSV(ans)

	return ans
}

// Отправляет и читает по запросу
func get(this string) string {

	err := btWrite(this)
	if err != nil {
		log.Print("get btWrite(): ", err)
	}

	var result = strings.Builder{}

	if wr, err := btRead(); err == nil {
		result.WriteString(wr)
	}

	if err != nil {
		if err != io.EOF {
			log.Print("get btRead(): ", err)
		}
	}

	ans := result.String()

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

	var result = strings.Builder{}

	if wr, err := btRead(); err == nil {
		result.WriteString(wr)
	} else {
		log.Print("getTestPing - btRead: ", err)
		return "Ошибка чтения", false
	}

	ans := result.String()

	return ans, true
}
