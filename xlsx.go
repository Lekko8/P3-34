package main

import (
	"encoding/csv"
	"io"
	"log"
	"os"
	"time"
)

// Создаёт файл для начала записи циклов
func createFile(filesFolder, order string) string {
	return "Имя файла в " + filesFolder + "\r\nЗаказ " + order
}

// Создаёт новый CSV
func newCSV(sampleName string) (*os.File, error) {
	fileName := "P3_34_" + time.Now().Format("20060102_150405_") + sampleName + ".csv"

	fileCSV, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Print(err)
		return nil, err
	}

	w := csv.NewWriter(fileCSV)
	w.Comma = ';'
	var s [][]string

	s = append(s, []string{"P3-34 measurement log"},
		[]string{"Started at", time.Now().Format("2006-01-02 15:04:05")},
		[]string{"Port", D.Port},
		[]string{"Interval min", D.Interval},
		[]string{"Frequency", D.Frequency},
		[]string{"Device ID", D.ID},
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
