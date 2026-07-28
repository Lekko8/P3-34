package main

import (
	"log"
	"strconv"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func GUI() {
	var (
		inputWay    *walk.LineEdit
		inputCount  *walk.LineEdit
		inputLong   *walk.LineEdit
		inputName   *walk.LineEdit
		inputOrder  *walk.LineEdit
		inputHz     *walk.LineEdit
		statusFiles *walk.TextEdit
		statusLabel *walk.Label
		progressBar *walk.ProgressBar
		initButton  *walk.PushButton
		mw          *walk.MainWindow
		runner      *MeasRun
	)

	appIcon, _ := walk.NewIconFromResourceId(2)

	mainWindow := MainWindow{
		AssignTo: &mw,
		Title:    "P3-34",
		Icon:     appIcon,
		Size:     Size{Width: 500, Height: 600},
		Layout:   VBox{},
		Children: []Widget{

			HSplitter{
				Children: []Widget{

					HSplitter{
						Children: []Widget{

							Label{Text: "Количество циклов:"},
							LineEdit{
								AssignTo: &inputCount,
								MinSize:  Size{Width: 50, Height: 0},
								Text:     "30",
							},
							Label{Text: "Длительность цикла (мин):"},
							LineEdit{
								AssignTo: &inputLong,
								MinSize:  Size{Width: 30, Height: 0},
								Text:     "2",
							},
							Label{Text: "Частота:"},
							LineEdit{
								AssignTo: &inputHz,
								MinSize:  Size{Width: 50, Height: 0},
								Text:     "2,2",
							},
						},
					},
				},
			},
			// Путь сохранения
			HSplitter{
				Children: []Widget{

					Label{Text: "Сохранять в:"},
					LineEdit{
						AssignTo: &inputWay,
						MaxSize:  Size{Width: 400, Height: 0},
						Text:     filesFolder,
					},
				},
			},

			HSplitter{
				Children: []Widget{

					Label{Text: "Имя текущего бразца:"},
					LineEdit{
						AssignTo: &inputName,
						MaxSize:  Size{Width: 50, Height: 0},
						Text:     "И",
					},
					Label{Text: "Номер заказа:"},
					LineEdit{
						AssignTo: &inputOrder,
						MaxSize:  Size{Width: 150, Height: 0},
						//Text:     "000z",
					},
					PushButton{
						Text:    "Проверить соединение",
						MaxSize: Size{Width: 100, Height: 0},
						OnClicked: func() {
							connectGUI(mw, appIcon)
						},
					},
				},
			},

			HSplitter{
				Children: []Widget{

					PushButton{
						Text:     "Начать измерение",
						MaxSize:  Size{Width: 150, Height: 0},
						AssignTo: &initButton,
						OnClicked: func() {
							start := time.Now()

							filesFolder = inputWay.Text()

							if runner == nil {
								runner = NewMeasRun(
									statusLabel, statusFiles,
									progressBar, initButton, mw)
							}

							if runner.isWorking {
								runner.Stop("Измерение прервано")
								return
							}

							totalCycles, err := strconv.Atoi(inputCount.Text())
							if err != nil {
								log.Print(err.Error())
								return
							}

							cycleMinutes, err := strconv.Atoi(inputLong.Text())
							if err != nil {
								log.Print(err.Error())
								return
							}

							if runner.testPingBt() {
								runner.Run(totalCycles, cycleMinutes)
							}
							log.Print("Кнопка: ", time.Since(start))
						},
					},

					PushButton{
						Text:    "Собрать .xlsx за сегодня",
						MaxSize: Size{Width: 150, Height: 0},
						OnClicked: func() {
							filesFolder = inputWay.Text()
							err := statusLabel.SetText(
								"Записано в " + createFile(filesFolder))
							if err != nil {
								log.Print(err.Error())
							}
						},
					},
				},
			},

			HSplitter{
				Children: []Widget{

					//HSpacer{},
					Label{AssignTo: &statusLabel, Text: "Готов к работе"},
					//HSpacer{},
					ProgressBar{
						AssignTo: &progressBar,
						MinValue: 0,
						MaxValue: 100,
						Value:    0,
						MinSize:  Size{Width: 50, Height: 10},
						MaxSize:  Size{Width: 400, Height: 10},
					},
				},
			},

			TextEdit{AssignTo: &statusFiles, ReadOnly: true, VScroll: true},
		},
	}

	if err := mainWindow.Create(); err != nil {
		log.Fatal(err)
	}

	mw.Run()
}

func connectGUI(owner walk.Form, appIcon *walk.Icon) {
	var (
		dig           *walk.Dialog
		statusConnect *walk.TextEdit
	)

	dialog := Dialog{
		AssignTo: &dig,
		Title:    "Bluetooth соединение",
		Icon:     appIcon,
		MinSize:  Size{Width: 200, Height: 150},
		Layout:   VBox{},
		Children: []Widget{
			TextEdit{
				AssignTo: &statusConnect,
			},
			PushButton{
				Text:    "Повторить проверку",
				MaxSize: Size{Width: 100, Height: 0},
				OnClicked: func() {
					err := statusConnect.SetText(getPortsList())
					if err != nil {
						log.Print(err.Error())
					}
				},
			},
		},
	}
	if err := dialog.Create(owner); err != nil {
		log.Fatal(err)
	}

	err := statusConnect.SetText(getPortsList())
	if err != nil {
		log.Print(err.Error())
	}

	dig.Run()
}
