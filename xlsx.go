package main

import (
	"log"
)

// Создаёт новый CSV
func newCSV() {
	log.Print("starting CSV")
}

// Добавляет измерение в CSV файл
func addToCSV(meas string) {
	log.Print("add to CSV: ", meas)
}
