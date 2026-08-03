package main

import (
	"log"
	"os/user"
	"strings"
)

func getUserName() string {
	corUser, err := user.Current()
	if err != nil {
		log.Print(err.Error())
		return ""
	}
	return strings.Split(corUser.Username, "\\")[1]
}

var filesFolder = "C:\\Users\\" + getUserName() + "\\Desktop\\p3-34\\P3-34 measurements"

//rsrc -manifest P3-34.manifest -o rsrc.syso
//rsrc -ico icon.ico -manifest P3-34.manifest -o rsrc_windows_amd64.syso

func main() {

	err := openPort()
	if err != nil {
		if errGUI(err) {
			GUI()
		}
	} else {
		GUI()
	}

	log.Print("Закрыли(")

}
