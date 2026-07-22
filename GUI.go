package main

import (
	"log"
	"strconv"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func GUI() {

	var (
		inputData    *walk.LineEdit
		inputCount   *walk.LineEdit
		inputLong    *walk.LineEdit
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

			HSplitter{
				Children: []Widget{

					Label{Text: "Количество циклов:"},
					LineEdit{AssignTo: &inputCount, Text: "6", MaxSize: Size{Width: 60, Height: 0}},
					HSpacer{},
				},
			},

			HSplitter{
				Children: []Widget{

					Label{Text: "Длительность цикла:"},
					LineEdit{AssignTo: &inputLong, Text: "10", MaxSize: Size{Width: 15, Height: 0}},
					HSpacer{},
				},
			},

			HSplitter{
				Children: []Widget{

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

					PushButton{
						Text: "Создать .xlsx",
						OnClicked: func() {
							//err := statusLabel.SetText("Файл " + createFile() + " успешно создан")
							//if err != nil {
							//	log.Panic(err.Error())
							//}
						},
					},
				},
			},

			Label{Text: "Сохранять в:"},
			LineEdit{AssignTo: &inputData, Text: filesFolder},

			Label{AssignTo: &statusSearch, Text: "Найденные файлы:"},
			TextEdit{AssignTo: &statusFiles, ReadOnly: true, VScroll: true},

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
