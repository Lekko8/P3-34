package main

import (
	"io"
	"log"
	"os"
	"os/user"
	"strings"
	"time"
)

var filesFolder = "C:\\Users\\" + getUserName() + "\\Desktop\\p3-34\\P3-34 measurements"
var file *os.File

//rsrc -manifest P3-34.manifest -o rsrc.syso
//rsrc -ico icon.ico -manifest P3-34.manifest -o rsrc_windows_amd64.syso

func main() {

	loging()

	log.Print("Открыли)")

	err := openPort()
	if err != nil {
		if errGUI(err) {
			GUI()
		}
	} else {
		GUI()
	}

	log.Print("Закрыли(")

	err = file.Close()
	if err != nil {
		log.Print(err)
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

func loging() {

	var err error

	file, err = os.OpenFile(time.Now().Format("02.01.2006_15.04_")+"P3-34.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Panic("Не удалось открыть файл логов:", err)
	}

	_, _ = file.Write([]byte{0xEF, 0xBB, 0xBF})

	multiWriter := io.MultiWriter(os.Stdout, file)

	log.SetOutput(multiWriter)
}
