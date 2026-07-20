package main

import (
	"log"
	"strconv"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func main() {

	GUI()
	//rsrc -manifest P3-34.manifest -o rsrc.syso
	//rsrc -ico icon.ico -manifest P3-34.manifest -o rsrc_windows_amd64.syso
}

var (
	filesFolder = "C:\\Users\\admin\\Desktop\\p3-34\\P3-34 measurements"

	countOfFiles int
)

func GUI() {

	var (
		inputData    *walk.LineEdit
		statusFiles  *walk.TextEdit
		statusSearch *walk.Label
		statusLabel  *walk.Label
		mw           *walk.MainWindow
	)

	appIcon, _ := walk.NewIconFromResourceId(2)

	mainWindow := MainWindow{
		AssignTo: &mw,
		Title:    "P3-34",
		Icon:     appIcon,
		Size:     Size{Width: 350, Height: 400},
		Layout:   VBox{},
		Children: []Widget{

			Label{Text: "Путь к папке:"},
			LineEdit{AssignTo: &inputData, Text: filesFolder},

			PushButton{
				Text: "Повторить поиск",
				OnClicked: func() {
					countOfFiles = 0

					filesFolder = inputData.Text()

					err := statusLabel.SetText("Прочитаны файлы из " + filesFolder)
					if err != nil {
						log.Panic(err.Error())
					}

					//err = statusFiles.SetText(readFileList(filesFolder))
					//if err != nil {
					//	log.Panic(err.Error())
					//}

					err = statusSearch.SetText("Найденные файлы: " + strconv.Itoa(countOfFiles))
					if err != nil {
						log.Panic(err.Error())
					}
				},
			},

			Label{AssignTo: &statusSearch, Text: "Найденные файлы:"},
			TextEdit{AssignTo: &statusFiles, ReadOnly: true, VScroll: true},

			PushButton{
				Text: "Создать .xlsx",
				OnClicked: func() {
					//err := statusLabel.SetText("Файл " + createFile() + " успешно создан")
					//if err != nil {
					//	log.Panic(err.Error())
					//}
				},
			},

			Label{AssignTo: &statusLabel, Text: "Лёша лох"},
		},
	}

	if err := mainWindow.Create(); err != nil {
		log.Fatal(err)
	}

	err := statusLabel.SetText("Прочитаны файлы из " + filesFolder) //запуск поиска на старте
	if err != nil {
		log.Panic(err.Error())
	}
	//err = statusFiles.SetText(readFileList(filesFolder))
	//if err != nil {
	//	log.Panic(err.Error())
	//}

	mw.Run()

}
