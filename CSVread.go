package main

import (
	"encoding/csv"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type FileWData struct {
	fileName   string
	sample     string
	sampleTime time.Time
	data       []CSVData
}

type CSVData struct {
	avgValue float32
	maxValue float32
	isValid  bool
}

type FinalData struct {
	sampleName string
	avgValue   float32
	maxValue   float32
}

// Считает все данные и выводит полный список
func calculateFiles(capturedFiles []FileWData) []FinalData {
	var calculatedFiles []FinalData

	for _, file := range capturedFiles {
		fileData, err := calculateData(file)
		if err != nil {
			log.Print(err)
			continue
		}

		for _, data := range fileData {
			calculatedFiles = append(calculatedFiles, FinalData{
				sampleName: file.sample,
				avgValue:   data.avgValue,
				maxValue:   data.maxValue,
			})
		}
	}
	return calculatedFiles
}

// Подсчёт данных для R, выводит массив средних значений
func calculateData(datafile FileWData) ([]FinalData, error) {
	if len(datafile.data) == 0 {
		return nil, errors.New("no data to calculate")
	}

	const blockSize = 5
	dataLen := len(datafile.data)

	numBlocks := dataLen / blockSize
	if dataLen%blockSize != 0 {
		numBlocks++
	}

	calculatedData := make([]FinalData, numBlocks)

	for a := 0; a < numBlocks; a++ {
		var sumAvg, sumMax float32
		var count int

		start := a * blockSize
		end := min(start+blockSize, dataLen)

		for i := start; i < end; i++ {
			if !datafile.data[i].isValid {
				continue
			}
			sumAvg += datafile.data[i].avgValue
			sumMax += datafile.data[i].maxValue
			count++
		}

		if count > 0 {
			calculatedData[a].avgValue = sumAvg / float32(count)
			calculatedData[a].maxValue = sumMax / float32(count)
		}
	}

	return calculatedData, nil
}

// Читает файлы и собирает из них данные (кроме измерений)
func readFiles(filesFolder string) ([]FileWData, error) {
	files, err := os.ReadDir(filesFolder) // читаем все файлы в папке
	if err != nil {
		log.Print(err)
		return nil, err
	}

	if len(files) == 0 {
		return nil, errors.New("no files to read")
	}

	sorting(files) // сортируем

	capturedFiles := make([]FileWData, len(files))
	for i, file := range files {
		capturedFiles[i] = createFileWData(file) // заполняем список файлов файлами
	}

	return capturedFiles, nil
}

func createFileWData(file os.DirEntry) FileWData {
	return FileWData{
		fileName:   file.Name(),
		sampleTime: fileDateTime(file.Name()),
		sample:     getSampleName(file.Name()),
	}
}

// Берём имя образца
func getSampleName(filename string) string {
	base := strings.TrimSuffix(filename, ".csv")
	parts := strings.Split(base, "_")

	if len(parts) < 5 {
		log.Printf("Предупреждение: неожиданный формат имени файла: %s", filename)
		return base
	}
	return strings.Join(parts[4:], "_")
}

// Берём дату и время файла (из имени)
func fileDateTime(filename string) time.Time {
	layout := "20060102150405"
	DateTime, err := time.Parse(layout, strings.Split(strings.Split(filename, "_")[2]+strings.Split(filename, "_")[3], ".")[0])
	if err != nil {
		log.Println("Ошибка чтения даты и времени из имени файла: ", err)
	}
	return DateTime
}

// Сортировка по дате
func sorting(files []os.DirEntry) {
	sort.Slice(files, func(i, j int) bool {
		info1 := fileDateTime(files[i].Name())
		info2 := fileDateTime(files[j].Name())
		return info1.Before(info2)
	})
}

// Читает данные из .csv и заполняет FileWData.data
func readCSVData(filesFolder string, file *FileWData) {
	csvPath := filepath.Join(filesFolder, file.fileName)
	csvFile, err := os.Open(csvPath)
	if err != nil {
		log.Printf("ОШИБКА: не удалось открыть файл %s: %v", file.fileName, err)
		return
	}
	defer func() {
		if err := csvFile.Close(); err != nil {
			log.Printf("ОШИБКА: не удалось закрыть файл %s: %v", file.fileName, err)
		}
	}()

	reader := csv.NewReader(csvFile)
	reader.Comma = ';'
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	var dataList []CSVData

	for {
		record, err := reader.Read()
		if err == io.EOF {
			log.Printf("ПРЕДУПРЕЖДЕНИЕ: файл %s не содержит данных", file.fileName)
			return
		}
		if err != nil {
			log.Printf("ОШИБКА: %v", err)
			return
		}
		if len(record) > 0 && strings.Contains(strings.ToLower(record[0]), "time") {
			break
		}
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("ОШИБКА при чтении файла %s: %v", file.fileName, err)
			continue
		}

		if isEmptyRecord(record) {
			dataList = append(dataList, CSVData{
				avgValue: -1,
				maxValue: -1,
				isValid:  false,
			})
			continue
		}

		//if len(record) < 6 || !strings.Contains(record[0], "202") {
		//	continue
		//}

		avgVal, errAvg := strconv.ParseFloat(strings.TrimSpace(record[1]), 64)
		maxVal, errMax := strconv.ParseFloat(strings.TrimSpace(record[3]), 64)

		if errAvg == nil && errMax == nil {
			dataList = append(dataList, CSVData{
				avgValue: float32(avgVal),
				maxValue: float32(maxVal),
				isValid:  true,
			})
		} else {
			dataList = append(dataList, CSVData{
				avgValue: -1,
				maxValue: -1,
				isValid:  false,
			})
		}
	}

	file.data = dataList
}

// Проверяем, что запись пустая
func isEmptyRecord(record []string) bool {
	if len(record) == 0 {
		return true
	}
	for _, field := range record {
		if strings.TrimSpace(field) != "" {
			return false
		}
	}
	return true
}
