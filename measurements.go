package main

import (
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Проводит измерение
func getMeas(fileSCV *os.File, borderNum float32) string {

	var (
		result = strings.Builder{}
		ans    []string
		meas   []string
	)

	//ans = append(ans, time.Now().Format("15:04:05"))
	measAvr := convMeasData(get("MEAS:AVR"))
	tempAvg, err := strconv.ParseFloat(strings.Split(measAvr, "±")[0], 32)
	if err != nil {
		log.Println("Parsing error: ", err)
	}

	// запрашиваем значения, пока полученное Avg не будет выше порога
	for float32(tempAvg) < borderNum {
		log.Println("tempAvg < borderNum:", tempAvg)

		measAvr = convMeasData(get("MEAS:AVR"))

		tempAvg, err = strconv.ParseFloat(strings.Split(measAvr, "±")[0], 32)
		if err != nil {
			log.Println("Parsing error: ", err)
		}
	}

	ans = append(ans, measAvr)
	meas = append(meas, measAvr)
	ans = append(ans, "\r\n")

	measMax := convMeasData(get("MEAS:MAX"))

	ans = append(ans, measMax)
	meas = append(meas, measMax)

	go addToCSV(fileSCV, meas)

	ans = append(ans, get("NEWMEAS"))

	for _, res := range ans {
		result.WriteString(strings.TrimSuffix(res, "ready >>>"))
	}

	return result.String()
}

// Запрашивает информацию перед измерением и устанавливает частоту
func getInfo(freq string) string {

	var (
		result = strings.Builder{}
		ans    []string
	)

	ans = append(ans, time.Now().Format("02.01.2006 15:04:05"))

	log.Print("TESTing niggers: ", get("TEST"))

	/*
		ID: 26
		Name: P3_34_UHF
		#: 54826
		Soft: V2.0
	*/

	info := get("?")
	infoSplit := strings.Split(info, "\r\n")
	D.ID = strings.Split(infoSplit[1], " ")[1]
	ans = append(ans, infoSplit[1])
	D.Name = strings.Split(infoSplit[2], " ")[1]
	ans = append(ans, infoSplit[2])
	D.Number = strings.Split(infoSplit[3], " ")[1]
	ans = append(ans, infoSplit[3])
	D.Soft = strings.Split(infoSplit[4], " ")[1]
	ans = append(ans, infoSplit[4])
	D.Battery = infoSplit[5]
	ans = append(ans, infoSplit[5])
	/*
		for i, t := range infoSplit {
			if i == 0 || i == 7 {
				continue
			}
			log.Printf("№%v: %v", i, strings.Split(t, " ")[1])
		}

			2026/09/07 16:49:09 №0:
			2026/09/07 16:49:09 №1: ID: 26
			2026/09/07 16:49:09 №2: Name: P3_34_UHF
			2026/09/07 16:49:09 №3: #: 54826
			2026/09/07 16:49:09 №4: Soft: V2.0
			2026/09/07 16:49:09 №5: UBATT = 8.6 V Normal
			2026/09/07 16:49:09 №6: ready >>>
			2026/09/07 16:49:09 №7:

			ans = append(ans, D.ID)
			log.Print("result.WriteString(get(\"?\")) - ", D.ID)

			D.Battery = get("UBATT")
			ans = append(ans, D.Battery)
			log.Print("result.WriteString(get(\"UBATT\")) - ", D.Battery)
	*/
	D.Frequency = " "
	log.Print("freq in meas func: ", freq)
	if freq != "0" {
		D.Frequency = freq
		log.Printf("Кладу в D.Frequency: %v", D.Frequency)
		frequency := get("FREQ:" + strings.Replace(freq, ",", ".", 1))
		ans = append(ans, strings.TrimSuffix(frequency, "ready >>>"))
		log.Print("result.WriteString(get(\"FREQ\")) - ", frequency)
	}

	ans = append(ans, get("NEWMEAS"))

	result.WriteString("\r\n")

	for _, res := range ans {
		result.WriteString(res)
		result.WriteString("\r\n")
	}

	return strings.TrimSuffix(result.String(), "ready >>>\r\n") + "\r\n"
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

	ans := strings.TrimSuffix(result.String(), "\r\n")

	log.Print("get [", this, "]: ", ans)
	return "\r\n" + ans
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
