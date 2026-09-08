package main

import (
	"encoding/csv"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Создаёт файл для начала записи циклов
func createFile(filesFolder, order string) string {
	return "Имя файла в " + filesFolder + "\r\nЗаказ " + order
}

// Создаёт новый CSV
func newCSV(sampleName string) (*os.File, error) {
	fileName := "P3_34_" + time.Now().Format("20060102_150405_") + sampleName + ".csv"

	folderPath := "P3-34 measurements/" + time.Now().Format("02012006")
	err := os.MkdirAll(folderPath, 0755)
	if err != nil {
		log.Print(err)
		return nil, err
	}

	filePath := filepath.Join(folderPath, fileName)
	fileCSV, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Print(err)
		return nil, err
	}

	w := csv.NewWriter(fileCSV)
	w.Comma = ';'
	var s [][]string

	s = append(s, []string{"P3-34 measurement log"},
		[]string{"Started at", time.Now().Format("2006-01-02"), time.Now().Format("15:04:05")},
		[]string{"Port", D.Port},
		[]string{"Interval min", D.Interval},
		[]string{"Frequency", D.Frequency},
		[]string{"Device ID", D.ID},
		[]string{"Device Name", D.Name},
		[]string{"Device Number", D.Number},
		[]string{"Device Soft", D.Soft},
		[]string{"Battery", D.Battery})

	if err = w.WriteAll(s); err != nil {
		return nil, err
	}

	log.Print("started new CSV: ", fileName)
	w.Flush()
	return fileCSV, nil
}

// Добавляет измерение в CSV файл
func addToCSV(w io.Writer, data []string) {
	writer := csv.NewWriter(w)
	writer.Comma = ';'

	if err := writer.Write(data); err != nil {
		log.Print(err)
	}
	log.Print("added to CSV: ", data)
	writer.Flush()
}

func convMeasData(data string) string {
	/*
		структура данных:
		 <S> = (1.53 � 0.84) uW/cm2  (P=0.95)
		ready >>>
		либо:
		Smax = (0.000 � 0.000) uW/cm2  (P=0.95)
		ready >>>
		если разделим по пробелам, то получим
		<S>|=|(1.53|�|0.84)|uW/cm2||(P=0.95)
		ready >>>
		нужно брать: [2], [4], [5], [7]?
	*/
	split := strings.Split(data, " ")

	result := strings.Builder{}
	result.WriteString(strings.Replace(split[2], "(", "", 1))
	result.WriteString("±")
	result.WriteString(strings.Replace(split[4], ")", "", 1))
	result.WriteString(" ")
	result.WriteString(strings.Replace(strings.Replace(split[5], "(P=", "", 1), ")\r\nready >>>", "", 1))
	//result.WriteString("\r\n")

	log.Printf("Длина %s: %v", split, len(split))

	for i := range split {
		log.Printf("№%v содержит: %s", i, split[i])
	}

	log.Printf("result: %v", result.String())

	return result.String()
}
