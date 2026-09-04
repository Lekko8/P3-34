package main

import (
	"io"
	"log"
	"os"
	"strings"
	"time"
)

// Проводит измерение
func getMeas(fileSCV *os.File) string {

	var (
		result = strings.Builder{}
		ans    []string
	)

	ans = append(ans, time.Now().Format("15:04:05"))
	ans = append(ans, convMeasData(get("MEAS:AVR")))
	ans = append(ans, convMeasData(get("MEAS:MAX")))
	go addToCSV(fileSCV, ans)
	ans = append(ans, get("NEWMEAS"))

	for _, res := range ans {
		result.WriteString(res)
	}

	return result.String()
}

// Запрашивает информацию перед измерением и устанавливает частоту
func getInfo(fileSCV *os.File, freq string) string {

	var (
		result = strings.Builder{}
		ans    []string
	)

	ans = append(ans, time.Now().String())

	D.ID = get("?")
	ans = append(ans, D.ID)
	log.Print("result.WriteString(get(\"?\"))", result.String())
	//time.Sleep(10 * time.Millisecond)

	D.Battery = get("UBATT")
	ans = append(ans, D.Battery)
	log.Print("result.WriteString(get(\"UBATT\"))", result.String())
	//time.Sleep(10 * time.Millisecond)

	D.Frequency = ""
	if freq != "" {
		D.Frequency = get("FREQ:" + strings.Replace(freq, ",", ".", 1))
		ans = append(ans, D.Frequency)
		log.Print("result.WriteString(get(\"FREQ\"))", result.String())
		//time.Sleep(10 * time.Millisecond)
	}

	ans = append(ans, get("NEWMEAS"))

	result.WriteString(time.Now().Format("20060102_150405") + "\r\n")

	for _, res := range ans {
		result.WriteString(res)
	}

	go addToCSV(fileSCV, ans)

	return result.String()
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
	return "\r\n" + ans // TODO: проверить перенос строки
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
