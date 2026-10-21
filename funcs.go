package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

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
	statusMeas  *walk.TextEdit
	progressBar *walk.ProgressBar
	initButton  *walk.PushButton
	inputName   *walk.LineEdit
	filesFolder string
	mw          *walk.MainWindow
}

func NewMeasRun(
	statusLabel *walk.Label,
	statusMeas *walk.TextEdit,
	progressBar *walk.ProgressBar,
	initButton *walk.PushButton,
	inputName *walk.LineEdit,
	filesFolder string,
	mw *walk.MainWindow) *MeasRun {

	return &MeasRun{
		statusLabel: statusLabel,
		statusMeas:  statusMeas,
		progressBar: progressBar,
		initButton:  initButton,
		inputName:   inputName,
		filesFolder: filesFolder,
		mw:          mw,
	}
}

// Останавливает измерение
func (r *MeasRun) stop() {
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

		err = r.statusLabel.SetText("Измерение прервано")
		if err != nil {
			log.Print(err)
		}
		//if message != "" {
		//	err = r.statusLabel.SetText(message)
		//	if err != nil {
		//		log.Print(err)
		//	}
		//}
		r.progressBar.SetValue(0)
	})
}

// Запускает измерение
func (r *MeasRun) run(d *Data, totalCycles, cycleMinutes int, freq string, borderNum float32) {
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
		if r.statusMeas != nil {

		}
		r.progressBar.SetValue(0)
	})

	go r.runLoop(d, freq, borderNum)
}

// Запускает цикл
func (r *MeasRun) runLoop(d *Data, freq string, borderNum float32) {
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

	r.statusMeas.AppendText(getInfo(d, freq)) // получаем мета-данные прибора и устанавливаем частоту

	fileCSV, err := newCSV(d, r.inputName.Text())
	if err != nil {
		log.Print("Не удалось создать CSV файл: ", err)
	}
	defer func() {
		if err := fileCSV.Close(); err != nil {
			log.Print(err)
		}
	}()

	for i := 1; i <= r.totalCycles; i++ {
		// Проверка отмены
		select {
		case <-r.ctx.Done():
			r.updateStatus(
				fmt.Sprintf(
					"Прервано на цикле %d/%d", i, r.totalCycles))
			r.mw.Synchronize(func() {
				r.progressBar.SetValue(0)
			})
			return
		default:
		}

		// Обновление цикла
		r.updateCycle(i, r.totalCycles)

		// Выполнение измерения
		if err := r.doMeas(fileCSV, borderNum); err != nil {
			r.updateStatus(
				fmt.Sprintf(
					"Прервано на цикле %d/%d", i, r.totalCycles))
			r.mw.Synchronize(func() {
				r.progressBar.SetValue(0)
			})
			return
		}
	}

	// Завершено
	r.updateStatus(
		fmt.Sprintf(
			"Измерение завершено: %d/%d", r.totalCycles, r.totalCycles))
	r.mw.Synchronize(func() {
		r.progressBar.SetValue(100)
	})
}

// Выполняет измерение
func (r *MeasRun) doMeas(fileCSV *os.File, borderNum float32) error {
	done := make(chan struct{})
	var result string
	var isValid = false

	go func() {
		for !isValid {
			result, isValid = r.meas(r.ctx, r.cycleMinutes, fileCSV, borderNum)
		}

		select {
		case <-r.ctx.Done():
			close(done)
			return
		default:
			r.mw.Synchronize(func() {
				if r.statusMeas != nil {
					r.statusMeas.AppendText(result + "\r\n")
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

// Запускает новый цикл измерений
func (r *MeasRun) meas(ctx context.Context, m int, fileCSV *os.File, borderNum float32) (string, bool) {
	s := time.Duration(m) * time.Minute

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			return "Измерение прервано", false

		case <-ticker.C:
			elapsed := time.Since(start)

			progress := max(min(int((elapsed.Seconds()/s.Seconds())*100), 100), 0)
			r.updateProgress(progress)
			if elapsed >= s {
				r.updateProgress(100)
				return getMeas(fileCSV, borderNum) // выполняем измерение
			}
			if progress%10 == 0 {
				r.mw.Synchronize(func() {
					err := r.statusLabel.
						SetText(
							fmt.Sprintf("Текущий цикл: %d/%d",
								r.currentCycle, r.totalCycles))
					if err != nil {
						log.Print(err)
					}
				})
			}
		}
	}
}

// Сменяет цикл
func (r *MeasRun) updateCycle(current, total int) {
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

// Обновляет статус
func (r *MeasRun) updateStatus(message string) {
	r.mw.Synchronize(func() {
		err := r.statusLabel.SetText(message)
		if err != nil {
			log.Print(err)
		}
	})
}

// Обновляет прогресс бар
func (r *MeasRun) updateProgress(percent int) {

	r.mu.Lock()
	r.progress = percent
	r.mu.Unlock()

	r.mw.Synchronize(func() {
		if r.progressBar != nil {
			r.progressBar.SetValue(percent)
		}
	})
}

// Проверяет соединение
func (r *MeasRun) testPingBt() bool {
	result, ping := getTestPing()
	r.statusMeas.AppendText(result)
	return ping
}
