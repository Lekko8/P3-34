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

type Settings struct {
	port         string
	totalCycles  int
	cycleMinutes int
	freq         string
	borderNum    string
}

const settingsFileName = "settings.txt"

var (
	D           = &Data{}
	filesFolder = fmt.Sprintf("C:\\Users\\%s\\Desktop\\p3-34\\P3-34 measurements\\%s",
		getUserName(), time.Now().Format("02012006"))
	fileLog   *os.File
	TitleName = "P3-34 версия от 15.10.2026"
)

//rsrc -manifest P3-34.manifest -o rsrc.syso
//rsrc -ico icon.ico -manifest P3-34.manifest -o rsrc_windows_amd64.syso

func main() {
	s := &Settings{
		port:         D.Port,
		totalCycles:  30,
		cycleMinutes: 2,
		freq:         "2,2",
		borderNum:    "0,5",
	}
	defer func() {
		if err := s.SaveSettings(); err != nil {
			log.Print(err)
		}
	}()

	logging() // если нет папки log у файла при запуске, то логи не будут выписываться

	log.Print("Открыли)")

	// Гарантируем закрытие лог-файла при выходе
	defer func() {
		log.Print("Закрыли(")
		if fileLog != nil {
			_ = fileLog.Close()
		}
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
			err := s.LoadSettings()
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("%+v\n", *s)
			GUI(s)
		}
	} else {
		log.Printf("Порт успешно открыт: %s", D.Port)
		err := s.LoadSettings()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%+v\n", *s)
		GUI(s)
	}
}

func getUserName() string {
	corUser, err := user.Current()
	if err != nil {
		log.Print(err.Error())
		return "бро"
	}
	return strings.Split(corUser.Username, "\\")[1]
}

func (s *Settings) LoadSettings() error {
	log.Println("Loading settings")
	if _, err := os.Stat(settingsFileName); os.IsNotExist(err) {
		//s := &Settings{
		//	port:         D.Port,
		//	totalCycles:  30,
		//	cycleMinutes: 2,
		//	freq:         "2,2",
		//	borderNum:    "0,5",
		//}
		if err := s.SaveSettings(); err != nil {
			return err
		}
		return nil
	}

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

	s.port = lines[0]
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
		D.Port, s.totalCycles, s.cycleMinutes, s.freq, s.borderNum)
	return err
}

func logging() {
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
