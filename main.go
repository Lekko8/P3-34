package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"
)

type Data struct {
	Order     string // номер заказа
	Port      string // номер порта (COM<цифра>)
	Interval  string // промежуток между измерениями
	Frequency string // частота
	ID        string // 26
	Name      string // P3_34_UHF
	Number    string // 54826
	Soft      string // V2.0
	Battery   string // заряд батареи
}

type Settings struct {
	savedPort    string
	totalCycles  int
	cycleMinutes int
	freq         string
	borderNum    string
}

const settingsFileName = "settings.txt"

var TitleName = "P3-34 версия от 22.10.2026"

//rsrc -manifest P3-34.manifest -o rsrc.syso
//rsrc -ico icon.ico -manifest P3-34.manifest -o rsrc_windows_amd64.syso

func main() {

	var (
		d = &Data{}
		s = &Settings{
			savedPort:    "COM",
			totalCycles:  30,
			cycleMinutes: 2,
			freq:         "2,2",
			borderNum:    "0,5",
		}
		fileLog     *os.File
		filesFolder = fmt.Sprintf(
			"C:\\Users\\%s\\Desktop\\p3-34\\P3-34 measurements\\%s",
			getUserName(), time.Now().Format("02012006"))
	)

	logging(fileLog) // если нет папки log у файла при запуске, то логи не будут выписываться

	defer func() {
		log.Print("Закрыли(")
		if fileLog != nil {
			_ = fileLog.Close()
		}
	}()

	defer func() {
		if err := closePort(); err != nil {
			log.Printf("Ошибка закрытия порта: %v", err)
		}
	}()

	defer func() {
		if err := s.SaveSettings(); err != nil {
			log.Print(err)
		}
	}()

	log.Print("Открыли)")

	// Открываем порт
	err := openPort(d, s)

	if err != nil {
		log.Printf("Ошибка открытия порта: %v", err)
		if errGUI(d, s, err) {
			initGUI(s, d, filesFolder)
		}
	} else {
		log.Printf("Порт успешно открыт: %s", d.Port)
		initGUI(s, d, filesFolder)
	}
}

func initGUI(s *Settings, d *Data, filesFolder string) {
	err := s.LoadSettings(d)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%+v\n", *s)
	GUI(s, d, filesFolder)
}

func getUserName() string {
	corUser, err := user.Current()
	if err != nil {
		log.Print(err.Error())
		return "бро"
	}
	return strings.Split(corUser.Username, "\\")[1]
}

func (s *Settings) LoadSettings(d *Data) error {
	log.Println("Loading settings")
	//if _, err := os.Stat(settingsFileName); os.IsNotExist(err) {
	//	if err := s.SaveSettings(d); err != nil {
	//		return err
	//	}
	//	return nil
	//}

	f, err := os.Open(settingsFileName)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Print(err)
		}
	}()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	if len(lines) < 5 {
		return fmt.Errorf("settings.txt: expected 5 lines, got %d", len(lines))
	}

	d.Port = lines[0]
	s.freq = lines[3]
	s.borderNum = lines[4]

	if s.totalCycles, err = strconv.Atoi(lines[1]); err != nil {
		return fmt.Errorf("totalCycles: %w", err)
	}
	if s.cycleMinutes, err = strconv.Atoi(lines[2]); err != nil {
		return fmt.Errorf("cycleMinutes: %w", err)
	}
	return nil
}

func (s *Settings) SaveSettings() error {
	f, err := os.Create(settingsFileName)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Print(err)
		}
	}()

	_, err = fmt.Fprintf(f, "%s\n%d\n%d\n%s\n%s\n",
		s.savedPort, s.totalCycles, s.cycleMinutes, s.freq, s.borderNum)
	return err
}

func logging(fileLog *os.File) {
	var writers []io.Writer

	if fi, err := os.Stdout.Stat(); err == nil && fi != nil {
		writers = append(writers, os.Stdout)
	}

	if st, err := os.Stat("log"); err == nil && st.IsDir() {
		name := fmt.Sprintf("log/%s_P3-34.log", time.Now().Format("02.01.2006_15.04"))
		f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			fileLog = f
			if info, e := f.Stat(); e == nil && info.Size() == 0 {
				_, _ = f.Write([]byte{0xEF, 0xBB, 0xBF}) // BOM только в новый файл
			}
			writers = append(writers, f)
		}
	}

	switch len(writers) {
	case 0:
		log.SetOutput(io.Discard)
	case 1:
		log.SetOutput(writers[0])
	default:
		log.SetOutput(io.MultiWriter(writers...))
	}
	log.Print("started logging")
}
