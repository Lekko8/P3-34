package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/lxn/walk"
)

type MeasRun struct {
	ctx          context.Context
	cancel       context.CancelFunc
	isWorking    bool
	totalCycles  int
	cycleMinutes int
	currentCycle int
	progress     int
	mu           sync.Mutex
	wg           sync.WaitGroup

	statusLabel *walk.Label
	statusFiles *walk.TextEdit
	progressBar *walk.ProgressBar
	initButton  *walk.PushButton
	mw          *walk.MainWindow
}

func NewMeasRun(
	statusLabel *walk.Label,
	statusFiles *walk.TextEdit,
	progressBar *walk.ProgressBar,
	initButton *walk.PushButton,
	mw *walk.MainWindow) *MeasRun {

	return &MeasRun{
		statusLabel: statusLabel,
		statusFiles: statusFiles,
		progressBar: progressBar,
		initButton:  initButton,
		mw:          mw,
	}
}

// Stop останавливает измерение
func (r *MeasRun) Stop(message string) {
	r.mu.Lock()

	if r.cancel != nil {
		r.cancel()
	}
	r.isWorking = false
	r.mu.Unlock()
	r.wg.Wait()

	r.mw.Synchronize(func() {
		err := r.initButton.SetText("Начать измерение")
		if err != nil {
			log.Print(err)
		}
		if message != "" {
			err = r.statusLabel.SetText(message)
			if err != nil {
				log.Print(err)
			}
		}
		r.progressBar.SetValue(0)
	})
}

// UpdateStatus обновляет статус
func (r *MeasRun) UpdateStatus(message string) {
	r.mw.Synchronize(func() {
		err := r.statusLabel.SetText(message)
		if err != nil {
			log.Print(err)
		}
	})
}

// UpdateCycle сменяет цикл
func (r *MeasRun) UpdateCycle(current, total int) {
	r.mu.Lock()
	r.currentCycle = current
	r.mu.Unlock()
	r.mw.Synchronize(func() {
		err := r.statusLabel.SetText(
			fmt.Sprintf("Текущий цикл: %d/%d", current, total))
		if err != nil {
			log.Print(err)
		}
		r.progressBar.SetValue(0)
	})
}

// UpdateProgress обновляет прогресс бар
func (r *MeasRun) UpdateProgress(percent int) {

	r.mu.Lock()
	r.progress = percent
	r.mu.Unlock()

	r.mw.Synchronize(func() {
		if r.progressBar != nil {
			r.progressBar.SetValue(percent)
		}
	})
}

// DoMeas выполняет измерение
func (r *MeasRun) DoMeas() error {
	done := make(chan struct{})
	var result string
	go func() {
		result = startMeas(r.ctx, r.cycleMinutes, r)

		select {
		case <-r.ctx.Done():
			close(done)
			return
		default:
			r.mw.Synchronize(func() {
				if r.statusFiles != nil {
					r.statusFiles.AppendText(result + "\r\n")
				}
				r.progressBar.SetValue(0)
			})
		}
		close(done)
	}()

	select {
	case <-r.ctx.Done():
		return r.ctx.Err()
	case <-done:
		return nil
	}
}

// Run запускает измерения
func (r *MeasRun) Run(totalCycles, cycleMinutes int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isWorking {
		log.Println("Измерение уже запущено")
		return
	}

	r.totalCycles = totalCycles
	r.cycleMinutes = cycleMinutes
	r.currentCycle = 0
	r.progress = 0
	r.isWorking = true
	if r.ctx != nil {
		r.cancel()
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())

	r.wg.Add(1)
	r.mw.Synchronize(func() {
		err := r.initButton.SetText("Остановить")
		if err != nil {
			log.Print(err)
		}
		if r.statusFiles != nil {
			r.statusFiles.AppendText(getMetaData())
			r.statusFiles.AppendText(getUBatt())
		}
		r.progressBar.SetValue(0)
	})

	go r.runLoop()
}

// runLoop запускает цикл
func (r *MeasRun) runLoop() {
	defer r.wg.Done()
	defer func() {
		r.mu.Lock()
		r.isWorking = false
		r.currentCycle = 0
		r.mu.Unlock()
		r.mw.Synchronize(func() {
			err := r.initButton.SetText("Начать измерение")
			if err != nil {
				log.Print(err)
			}
			r.progressBar.SetValue(0)
		})
	}()

	for i := 1; i <= r.totalCycles; i++ {
		// Проверка отмены
		select {
		case <-r.ctx.Done():
			r.UpdateStatus(
				fmt.Sprintf(
					"Прервано на цикле %d/%d", i, r.totalCycles))
			r.mw.Synchronize(func() {
				r.progressBar.SetValue(0)
			})
			return
		default:
		}

		// Обновление цикла
		r.UpdateCycle(i, r.totalCycles)

		// Выполнение измерения
		if err := r.DoMeas(); err != nil {
			r.UpdateStatus(
				fmt.Sprintf(
					"Прервано на цикле %d/%d", i, r.totalCycles))
			r.mw.Synchronize(func() {
				r.progressBar.SetValue(0)
			})
			return
		}
	}

	// Завершено
	r.UpdateStatus(
		fmt.Sprintf(
			"Измерение завершено: %d/%d", r.totalCycles, r.totalCycles))
	r.mw.Synchronize(func() {
		r.progressBar.SetValue(100)
	})
}

func (r *MeasRun) testPingBt() bool {
	result, ping := getTestPing()
	r.statusFiles.AppendText(result)
	return ping
}

// createFile создаёт файл для начала записи циклов
func createFile(filesFolder string) string {
	return "Имя файла в " + filesFolder
}
