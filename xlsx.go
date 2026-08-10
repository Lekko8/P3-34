package main

import (
	"log"
)

// Создаёт файл для начала записи циклов
func createFile(filesFolder, order string) string {
	return "Имя файла в " + filesFolder + "\r\nЗаказ " + order
}

// Создаёт новый CSV
func newCSV() {
	log.Print("starting new CSV")
}

// Добавляет измерение в CSV файл
func addToCSV(meas string) {
	log.Print("add to CSV: ", meas)
}
