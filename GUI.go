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
		inputFreq   *walk.LineEdit
		statusMeas  *walk.TextEdit
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
								AssignTo: &inputFreq,
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

					Label{Text: "Имя текущего образца:"},
					LineEdit{
						AssignTo: &inputName,
						MaxSize:  Size{Width: 50, Height: 0},
						Text:     "И",
					},
					Label{Text: "Номер заказа:"},
					LineEdit{
						AssignTo: &inputOrder,
						MaxSize:  Size{Width: 150, Height: 0},
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
							defer log.Print("Начать измерение: ", time.Since(start))
							log.Print("КНОПКА: Начать измерение")

							filesFolder = inputWay.Text()
							freq := inputFreq.Text()
							//log.Print("FREQ: ", freq)

							statusMeas.AppendText("Начать измерение\r\n")

							if runner == nil {
								runner = NewMeasRun(
									statusLabel, statusMeas,
									progressBar, initButton,
									inputName, mw)
							}

							if runner.isWorking {
								runner.stop()
								return
							}

							totalCycles, err := strconv.Atoi(inputCount.Text())
							if err != nil {
								log.Print(err)
								return
							}

							D.Interval = inputLong.Text()
							cycleMinutes, err := strconv.Atoi(D.Interval)
							if err != nil {
								log.Print(err)
								return
							}

							if runner.testPingBt() {
								runner.run(totalCycles, cycleMinutes, freq)
							}

						},
					},

					PushButton{
						Text:    "Собрать .xlsx за сегодня",
						MaxSize: Size{Width: 150, Height: 0},
						OnClicked: func() {
							log.Print("КНОПКА: Собрать .xlsx за сегодня")
							filesFolder = inputWay.Text()
							order := inputOrder.Text()

							err := statusLabel.SetText(
								"Записано в " + createFile(filesFolder, order))
							if err != nil {
								log.Print(err)
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

			TextEdit{AssignTo: &statusMeas, ReadOnly: true, VScroll: true},
		},
	}

	if err := mainWindow.Create(); err != nil {
		log.Panic(err)
	}

	mw.Run()

	defer func() {
		err := closePort()
		if err != nil {
			log.Print(err)
		}
	}()

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
					start := time.Now()
					defer log.Print("Повторить проверку: ", time.Since(start))
					log.Print("КНОПКА: Повторить проверку")

					err := reconnectPort()
					if err != nil {
						log.Print(err)
					}

					err = statusConnect.SetText(getPortsList())
					if err != nil {
						log.Print(err)
					}

					test, _ := getTestPing()
					statusConnect.AppendText("\r\n" + test)

				},
			},
		},
	}
	if err := dialog.Create(owner); err != nil {
		log.Panic(err)
	}

	err := statusConnect.SetText(getPortsList())
	if err != nil {
		log.Print(err)
	}

	dig.Run()
}

func errGUI(inErr error) bool {
	var (
		statusCon *walk.TextEdit
		//setPortButton   *walk.PushButton
		reconnectButton *walk.PushButton
		doneButton      *walk.PushButton
		dlg             *walk.Dialog
	)
	var mainGUI = false

	appIcon, _ := walk.NewIconFromResourceId(2)
	_, err := Dialog{
		AssignTo: &dlg,
		Title:    "P3-34 ошибка Bluetooth",
		Icon:     appIcon,
		MinSize:  Size{Height: 300, Width: 400},
		Layout:   VBox{},
		Children: []Widget{
			PushButton{
				AssignTo: &reconnectButton,
				Text:     "Переподключиться",
				OnClicked: func() {
					start := time.Now()
					defer log.Print("Переподключиться: ", time.Since(start))
					log.Print("КНОПКА: Переподключиться")

					err := statusCon.SetText("Переподключаюсь")
					if err != nil {
						log.Print(err)
					}

					err = statusCon.SetText(getPortsList())
					if err != nil {
						log.Print(err)
					}

					err = reconnectPort()
					if err != nil {
						log.Print(err)
					}

					test, _ := getTestPing()
					statusCon.AppendText("\r\n" + test)

				},
			},
			PushButton{
				AssignTo: &doneButton,
				Text:     "Должно работать",
				OnClicked: func() {
					log.Print("КНОПКА: Должно работать")
					mainGUI = true
					dlg.Cancel()
				},
			},
			TextEdit{

				AssignTo: &statusCon,
				ReadOnly: true,
				VScroll:  true,
				Text: "Ошибка подключения к Bluetooth: " + inErr.Error() + "\r\nПроверьте, " +
					"в настройках должно быть написано P3-34 подключён",
			},
		},
	}.Run(nil)
	if err != nil {
		log.Panic(err)
	}
	return mainGUI
}
