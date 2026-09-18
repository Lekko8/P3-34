package main

import (
	"io"
	"log"
	"os"
	"os/user"
	"strings"
	"time"
)

type Data struct {
	Order     string
	Port      string
	Interval  string
	Frequency string
	ID        string
	Name      string
	Number    string
	Soft      string
	Battery   string
}

var (
	D           = &Data{}
	filesFolder = "C:\\Users\\" + getUserName() + "\\Desktop\\p3-34\\P3-34 measurements\\" + time.Now().Format("02012006")
	fileLog     *os.File
)

//rsrc -manifest P3-34.manifest -o rsrc.syso
//rsrc -ico icon.ico -manifest P3-34.manifest -o rsrc_windows_amd64.syso

func main() {
	logging() // если нет папки log у файла при запуске, то логи не будут выписываться

	log.Print("Открыли)")

	// Гарантируем закрытие лог-файла при выходе
	defer func() {
		if err := fileLog.Close(); err != nil {
			log.Printf("Ошибка закрытия лог-файла: %v", err)
		}
		log.Print("Закрыли(")
	}()

	// Гарантируем закрытие порта при выходе
	defer func() {
		if err := closePort(); err != nil {
			log.Printf("Ошибка закрытия порта: %v", err)
		}
	}()

	// Открываем порт
	err := openPort()
	if err != nil {
		log.Printf("Ошибка открытия порта: %v", err)
		if errGUI(err) {
			GUI()
		}
	} else {
		log.Printf("Порт успешно открыт: %s", D.Port)
		GUI()
	}
}

func getUserName() string {
	corUser, err := user.Current()
	if err != nil {
		log.Print(err.Error())
		return ""
	}
	return strings.Split(corUser.Username, "\\")[1]
}

func logging() {

	var err error

	fileLog, err = os.OpenFile("log/"+time.Now().Format("02.01.2006_15.04_")+"P3-34.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Print("Не удалось открыть файл логов:", err)
		return
	}

	_, _ = fileLog.Write([]byte{0xEF, 0xBB, 0xBF})

	multiWriter := io.MultiWriter(os.Stdout, fileLog)

	log.SetOutput(multiWriter)
	log.Print("started logging")
}
