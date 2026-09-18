package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Запускает процесс обработки данных и создания .xlsx
func xlsx(filesFolder, orderName string) string {
	capturedFiles, err := readFiles(filesFolder)
	if err != nil {
		log.Print(err)
		return err.Error()
	}

	for i := range capturedFiles {
		readCSVData(filesFolder, &capturedFiles[i])
	}

	results := calculateFiles(capturedFiles)
	return createFile(capturedFiles, results, filesFolder, orderName)
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

	s = append(s,
		[]string{"P3-34 measurement log"},
		[]string{"Started at", time.Now().Format("2006-01-02"), time.Now().Format("15:04:05")},
		[]string{"Port", D.Port},
		[]string{"Interval min", D.Interval},
		[]string{"Frequency", D.Frequency},
		[]string{"Device ID", D.ID},
		[]string{"Device Name", D.Name},
		[]string{"Device Number", D.Number},
		[]string{"Device Soft", D.Soft},
		[]string{"Battery", D.Battery},
		[]string{"Order", D.Order},
		[]string{"Sample", sampleName},
		[]string{"Time", "Avr meas", "Avr err", "Max meas", "Max err"})

	if err = w.WriteAll(s); err != nil {
		return nil, err
	}

	log.Print("started new CSV: ", fileName)
	w.Flush()
	return fileCSV, nil
}

// Добавляет измерение в CSV файл
func addToCSV(w io.Writer, data []string) {
	// 0.764±0.421 uW/cm2
	// 0.827±0.456 uW/cm2

	var result = make([]string, 0, len(data)+1)
	result = append(result, time.Now().Format("15:04:05"))

	for _, value := range data {
		result = append(result, strings.Split(strings.TrimSpace(strings.TrimSuffix(value, " uW/cm2")), "±")...)
	}

	//0.7640.421
	//0.8270.456

	writer := csv.NewWriter(w)
	writer.Comma = ';'

	if err := writer.Write(result); err != nil {
		log.Print(err)
	}
	log.Print("added to CSV: ", result)
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
	//result.WriteString(time.Now().Format("15:04:05"))
	result.WriteString(strings.Replace(split[2], "(", "", 1))
	result.WriteString("±")
	result.WriteString(strings.Replace(split[4], ")", "", 1))
	result.WriteString(" ")
	result.WriteString(strings.Replace(strings.Replace(split[5], "(P=", "", 1), ")\r\nready >>>", "", 1))
	//result.WriteString("\r\n")

	/*
		log.Printf("Длина %s: %v", split, len(split))

		for i := range split {
			log.Printf("№%v содержит: %s", i, split[i])
		}

		2026/09/10 16:04:12 №0 содержит:
		Smax
		2026/09/10 16:04:12 №1 содержит: =
		2026/09/10 16:04:12 №2 содержит: (0.827
		2026/09/10 16:04:12 №3 содержит:
		2026/09/10 16:04:12 №4 содержит: 0.456)
		2026/09/10 16:04:12 №5 содержит: uW/cm2
		2026/09/10 16:04:12 №6 содержит:
		2026/09/10 16:04:12 №7 содержит: (P=0.95)

		log.Printf("result: %v", result.String())
	*/

	return result.String()
}

// Создаёт файл .xlsx
func createFile(listOfFiles []FileWData, results []FinalData, filesFolder, order string) string {

	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("Ошибка при создании файла .xlsx: %v", err)
		}
	}()

	boldStyle, err := f.NewStyle(&excelize.Style{
		NumFmt: 2, // 2 = 0.00
		Font: &excelize.Font{
			Bold: true,
		},
	})
	numberStyle, err := f.NewStyle(&excelize.Style{
		NumFmt: 2,
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
	})

	if err != nil {
		fmt.Println("Ошибка при создании стиля:", err)
		return err.Error()
	}

	for idx := range listOfFiles { // создание листов с сырыми данными
		index, err := f.NewSheet(listOfFiles[idx].sample)
		if err != nil {
			log.Printf("Ошибка создания листа %s: %v", listOfFiles[idx].sample, err)
			continue
		}
		csvFile, err := os.Open(filepath.Join(filesFolder, listOfFiles[idx].fileName))
		if err != nil {
			log.Printf("Ошибка открытия файла %s: %v", listOfFiles[idx].fileName, err)
			continue
		}

		reader := csv.NewReader(csvFile)
		reader.Comma = ';'
		reader.FieldsPerRecord = -1
		reader.LazyQuotes = true

		rows, err := reader.ReadAll()
		if err != nil {
			log.Printf("Ошибка чтения файла %s: %v", listOfFiles[idx].fileName, err)
			continue
		}

		for rowIdx, row := range rows {
			rowNum := rowIdx + 1 // номера строк и столбцов в Excel начинаются с 1
			for colIdx, cellValue := range row {
				colName, err := excelize.CoordinatesToCellName(colIdx+1, rowNum)
				if err != nil {
					log.Printf("Ошибка конвертации координат: %v", err)
					continue
				}
				err = f.SetCellStr(listOfFiles[idx].sample, colName, cellValue)
				if err != nil {
					log.Print(err)
				}
			}
		}
		//err = f.SetCellValue(listOfFiles[idx].sample, "F1", order) // вставка номера заказа
		//if err != nil {
		//	log.Print(err)
		//}
		//err = f.SetCellValue(listOfFiles[idx].sample, "G1", listOfFiles[idx].sample) // вставка имени образца
		//if err != nil {
		//	log.Print(err)
		//}

		f.SetActiveSheet(index) // активация листа

		err = csvFile.Close()
		if err != nil {
			log.Print(err)
		}
	}

	for _, sheetName := range f.GetSheetList() { // перевод данных из текста в числа
		if err != nil {
			fmt.Println("Ошибка при применении стиля к столбцу:", err)
			return err.Error()
		}
		for row := 10; row <= 42; row++ {
			for col := 4; col <= 7; col++ {
				cell, _ := excelize.CoordinatesToCellName(col, row)

				if val, _ := f.GetCellValue(sheetName, cell); val != "" {
					if num, err := strconv.ParseFloat(strings.ReplaceAll(val, ",", "."), 64); err == nil {
						err = f.SetCellValue(sheetName, cell, num)
						if err != nil {
							log.Print(err)
						}
						err = f.SetCellStyle(sheetName, "D12", "G42", numberStyle)
						if err != nil {
							log.Print(err)
						}

					}
				}
			}
		}
		err = f.SetCellStyle(sheetName, "D12", "G42", numberStyle)
		err = f.SetColStyle(sheetName, "D", boldStyle)
		err = f.SetColStyle(sheetName, "F", boldStyle)
	}

	index, err := f.NewSheet("Результат")
	if err != nil {
		log.Printf("ОШИБКА: %v", err)
		return err.Error()
	}
	err = f.SetCellValue("Результат", "A1", "Группа")
	err = f.SetCellValue("Результат", "B1", "average value мкВт/см2")
	err = f.SetCellValue("Результат", "D1", "Группа")
	err = f.SetCellValue("Результат", "E1", "max value мкВт/см2")
	//err = f.SetCellValue("Результат", "G1", order)
	if err != nil {
		log.Print(err)
	}

	for i, data := range results {
		row := i + 2 // начинаем со 2-й строки (1-я строка - заголовки)

		cellA := fmt.Sprintf("A%d", row)
		cellB := fmt.Sprintf("B%d", row)
		cellD := fmt.Sprintf("D%d", row)
		cellE := fmt.Sprintf("E%d", row)

		err = f.SetCellValue("Результат", cellA, data.sampleName)
		err = f.SetCellValue("Результат", cellB, data.avgValue)
		err = f.SetCellValue("Результат", cellD, data.sampleName)
		err = f.SetCellValue("Результат", cellE, data.maxValue)
		if err != nil {
			log.Print(err)
		}
	}
	styleID, err := f.NewStyle(&excelize.Style{
		CustomNumFmt: &[]string{"0.00"}[0], // формат 0.00
		Alignment: &excelize.Alignment{
			Horizontal: "center", // выравнивание по центру
		},
	})
	styleTxt, err := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "left", // выравнивание слева
		},
	})
	if err != nil {
		fmt.Println("Ошибка создания стиля:", err)
		return err.Error()
	}

	if err := f.SetColStyle("Результат", "B", styleID); err != nil {
		fmt.Println("Ошибка применения стиля:", err)
		return err.Error()
	}
	if err := f.SetColStyle("Результат", "E", styleID); err != nil {
		fmt.Println("Ошибка применения стиля:", err)
		return err.Error()
	}
	if err := f.SetColStyle("Результат", "A", styleTxt); err != nil {
		fmt.Println("Ошибка применения стиля:", err)
		return err.Error()
	}
	if err := f.SetColStyle("Результат", "D", styleTxt); err != nil {
		fmt.Println("Ошибка применения стиля:", err)
		return err.Error()
	}
	f.SetActiveSheet(index)       // активация листа
	err = f.DeleteSheet("Sheet1") // удаление лишнего листа появившегося при создании файла
	if err != nil {
		log.Print(err)
	}
	err = f.SaveAs("Результаты " + order + " за " + time.Now().Format("02012006") + ".xlsx") // сохранение файла
	if err != nil {
		log.Print(err)
	}

	return "Имя файла в " + filesFolder + "\r\nЗаказ " + order
}
