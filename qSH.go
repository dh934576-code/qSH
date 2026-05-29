package main

import (
	"bufio"
    "crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/user"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

)

var (
	username string
	hostname string
	osName   string
    aliasRegistry map[string]string
    shellVariables map[string]string
    commandHistory []string
)

type Command struct {
	Name        string
	Category    string
	Description string
	Usage       string
	Handler     func(args []string)
}

var registry = make(map[string]Command)

func register(cmd Command) {
	registry[cmd.Name] = cmd
}

func init() {
	// 1. Получаем имя пользователя
	u, err := user.Current()
	if err == nil {
		username = u.Username
	} else {
		username = os.Getenv("USERNAME") // Запасной вариант для Windows
	}

	// 2. Получаем имя хоста (имя ПК)
	host, err := os.Hostname()
	if err == nil {
		hostname = host
	} else {
		hostname = "unknown-pc"
	}

	// 3. Задаем имя ОС (для Windows)
	osName = "Windows" 

	// === КАТЕГОРИЯ: ФАЙЛОВАЯ СИСТЕМА ===
	register(Command{
		Name:        "ls",
		Category:    "Filesystem",
		Description: "Выводит детальный список файлов и папок",
		Usage:       "ls [путь]",
		Handler: func(args []string) {
			target := "."
			if len(args) > 0 { target = args[0] }
			files, err := os.ReadDir(target)
			if err != nil {
				fmt.Printf("Ошибка ls: %v\n", err)
				return
			}
			fmt.Printf("\n%-25s %-12s %s\n", "Название", "Тип", "Размер")
			fmt.Println(strings.Repeat("-", 55))
			for _, f := range files {
				info, _ := f.Info()
				fileType := "Файл 📄"
				if f.IsDir() { fileType = "Папка 📁" }
				fmt.Printf("%-25s %-12s %d байт\n", f.Name(), fileType, info.Size())
			}
			fmt.Println()
		},
	})

	register(Command{
		Name:        "pwd",
		Category:    "Filesystem",
		Description: "Показывает полный путь рабочей директории",
		Usage:       "pwd",
		Handler: func(args []string) {
			dir, err := os.Getwd()
			if err != nil { return }
			fmt.Println(dir)
		},
	})

	register(Command{
		Name:        "mkdir",
		Category:    "Filesystem",
		Description: "Создает новые папки с вложенностью",
		Usage:       "mkdir [имя_папки]",
		Handler: func(args []string) {
			if len(args) < 1 { return }
			os.MkdirAll(args[0], os.ModePerm)
			fmt.Println("Папка создана.")
		},
	})
	register(Command{
		Name:        "touch",
		Category:    "Filesystem",
		Description: "Создает пустой файл",
		Usage:       "touch [имя_файла]",
		Handler: func(args []string) {
			if len(args) < 1 { return }
			file, err := os.OpenFile(args[0], os.O_RDONLY|os.O_CREATE, 0666)
			if err != nil { return }
			file.Close()
			fmt.Println("Файл создан.")
		},
	})

	register(Command{
		Name:        "rm",
		Category:    "Filesystem",
		Description: "Удаляет файл или папку рекурсивно",
		Usage:       "rm [путь]",
		Handler: func(args []string) {
			if len(args) < 1 { return }
			os.RemoveAll(args[0])
			fmt.Println("Удалено.")
		},
	})

	register(Command{
		Name:        "cp",
		Category:    "Filesystem",
		Description: "Копирует файл в указанное место",
		Usage:       "cp [источник] [назначение]",
		Handler: func(args []string) {
			if len(args) < 2 { return }
			src, err := os.Open(args[0])
			if err != nil { return }
			defer src.Close()
			dst, err := os.Create(args[1])
			if err != nil { return }
			defer dst.Close()
			io.Copy(dst, src)
			fmt.Println("Скопировано.")
		},
	})

	register(Command{
		Name:        "mv",
		Category:    "Filesystem",
		Description: "Перемещает или переименовывает объект",
		Usage:       "mv [откуда] [куда]",
		Handler: func(args []string) {
			if len(args) < 2 { return }
			os.Rename(args[0], args[1])
			fmt.Println("Перемещено.")
		},
	})

	register(Command{
		Name:        "stat",
		Category:    "Filesystem",
		Description: "Показывает системную информацию о файле",
		Usage:       "stat [файл]",
		Handler: func(args []string) {
			if len(args) < 1 { return }
			fi, err := os.Stat(args[0])
			if err != nil { return }
			fmt.Printf("Имя: %s\nРазмер: %d байт\nПапка: %t\nПрава: %s\n", fi.Name(), fi.Size(), fi.IsDir(), fi.Mode())
		},
	})

	// === КАТЕГОРИЯ: ОБРАБОТКА ТЕКСТА ===
	register(Command{
		Name:        "cat",
		Category:    "Text Processing",
		Description: "Выводит содержимое текстового файла на экран",
		Usage:       "cat [файл]",
		Handler: func(args []string) {
			if len(args) < 1 { return }
			data, err := os.ReadFile(args[0])
			if err != nil { return }
			fmt.Println(string(data))
		},
	})

	register(Command{
		Name:        "head",
		Category:    "Text Processing",
		Description: "Выводит первые N строк из указанного файла",
		Usage:       "head [количество_строк] [файл]",
		Handler: func(args []string) {
			if len(args) < 2 { return }
			count, err := strconv.Atoi(args[0])
			if err != nil { return }
			data, err := os.ReadFile(args[1])
			if err != nil { return }
			lines := strings.Split(string(data), "\n")
			for i := 0; i < count && i < len(lines); i++ {
				fmt.Println(lines[i])
			}
		},
	})

	register(Command{
		Name:        "tail",
		Category:    "Text Processing",
		Description: "Выводит последние N строк из указанного файла",
		Usage:       "tail [количество_строк] [файл]",
		Handler: func(args []string) {
			if len(args) < 2 { return }
			count, err := strconv.Atoi(args[0])
			if err != nil { return }
			data, err := os.ReadFile(args[1])
			if err != nil { return }
			lines := strings.Split(string(data), "\n")
			start := len(lines) - count
			if start < 0 { start = 0 }
			for i := start; i < len(lines); i++ {
				fmt.Println(lines[i])
			}
		},
	})

	register(Command{
		Name:        "grep",
		Category:    "Text Processing",
		Description: "Ищет вхождения строки внутри указанного файла",
		Usage:       "grep [подстрока] [файл]",
		Handler: func(args []string) {
			if len(args) < 2 { return }
			query := args[0]
			data, err := os.ReadFile(args[1])
			if err != nil { return }
			lines := strings.Split(string(data), "\n")
			for idx, line := range lines {
				if strings.Contains(line, query) {
					fmt.Printf("[%d]: %s\n", idx+1, strings.TrimSpace(line))
				}
			}
		},
	})

	register(Command{
		Name:        "wc",
		Category:    "Text Processing",
		Description: "Подсчитывает количество строк, слов и байт в файле",
		Usage:       "wc [файл]",
		Handler: func(args []string) {
			if len(args) < 1 { return }
			data, err := os.ReadFile(args[0])
			if err != nil { return }
			content := string(data)
			lines := len(strings.Split(content, "\n"))
			words := len(strings.Fields(content))
			fmt.Printf("Строк: %d | Слов: %d | Размер: %d байт\n", lines, words, len(data))
		},
	})

	register(Command{
		Name:        "echo",
		Category:    "Text Processing",
		Description: "Выводит переданную строку текста на экран",
		Usage:       "echo [текст]",
		Handler: func(args []string) {
			fmt.Println(strings.Join(args, " "))
		},
	})
	// === КАТЕГОРИЯ: СЕТЕВЫЕ УТИЛИТЫ ===
	register(Command{
		Name:        "curl",
		Category:    "Networking",
		Description: "Скачивает контент по ссылке",
		Usage:       "curl [url]",
		Handler: func(args []string) {
			if len(args) < 1 { return }
			url := args[0]
			if !strings.HasPrefix(url, "http") { url = "https://" + url }
			resp, err := http.Get(url)
			if err != nil { return }
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
		},
	})

	register(Command{
		Name:        "ping",
		Category:    "Networking",
		Description: "Проверяет соединение с хостом по TCP",
		Usage:       "ping [хост:порт]",
		Handler: func(args []string) {
			target := "google.com:80"
			if len(args) > 0 { target = args[0] }
			if !strings.Contains(target, ":") { target += ":80" }
			start := time.Now()
			conn, err := net.DialTimeout("tcp", target, 4*time.Second)
			if err != nil { return }
			conn.Close()
			fmt.Printf("Ответ от %s: время=%v\n", target, time.Since(start))
		},
	})

	register(Command{
		Name:        "ip",
		Category:    "Networking",
		Description: "Показывает локальные IPv4 адреса",
		Usage:       "ip",
		Handler: func(args []string) {
			ifaces, _ := net.InterfaceAddrs()
			for _, addr := range ifaces {
				if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
					if ipNet.IP.To4() != nil { fmt.Printf("IPv4: %s\n", ipNet.IP.String()) }
				}
			}
		},
	})

	// === КАТЕГОРИЯ: СИСТЕМНЫЕ ИНСТРУМЕНТЫ ===
	register(Command{
		Name:        "env",
		Category:    "System",
		Description: "Выводит переменные окружения",
		Usage:       "env",
		Handler: func(args []string) {
			for _, e := range os.Environ() { fmt.Println(e) }
		},
	})

	register(Command{
		Name:        "sysinfo",
		Category:    "System",
		Description: "Параметры процессора и ОС",
		Usage:       "sysinfo",
		
        Handler: func(args []string) {
        // 1. Сначала вызываем функции сбора данных из ваших файлов disks_*.go
        diskLetter, diskUsed, diskTotal, diskPercent := getMainDiskStats()
        cpuInfo := getCPU()
        gpuInfo := getGPU()
        memInfo := getMemory()
        ipInfo := getLocalIP()
        batteryInfo := getBatteryStatus()
        localeInfo := getLocale()
        displayRes := getDisplayResolution()

        // 2. Получаем имя пользователя и ПК (ваш рабочий код)
        username := os.Getenv("USERNAME")
        if username == "" {
            username = os.Getenv("USER")
        }
        hostname, err := os.Hostname()
        if err != nil {
            hostname = "unknown_pc"
        }

        fmt.Print(
  `   %%%%%%       %%
 %%    %%      %% 
 %%    %%     %%  
   %%%%%%    %%   
       %%   %%          
       %%  %%            %%  %% %%
       %% %%             %% %%  %%
         %% %%%%  %%  %% %%%%   %%
        %% %%     %%  %% %%     %%
       %%   %%%%  %%%%%%
      %%       %% %%  %%
     %%     %%%%  %%  %%
    `)
    // Выводим информацию на экран
       fmt.Printf("  User@PC:    %s@%s\n", username, hostname)
       fmt.Println("---------------------------------------")
       fmt.Printf("OS:         %s (%s)\n", runtime.GOOS, runtime.GOARCH)
       fmt.Printf("CPU Cores:  %d\n", runtime.NumCPU())
       fmt.Printf("CPU Model:  %s\n", cpuInfo)       // Добавлено!
       fmt.Printf("GPU:        %s\n", gpuInfo)       // Добавлено!
       fmt.Printf("Memory:     %s\n", memInfo)       // Добавлено!
       fmt.Printf("Disk (%s): %s / %s (%s)\n", diskLetter, diskUsed, diskTotal, diskPercent) // Добавлено!
       fmt.Printf("Display:    %s\n", displayRes)    // Добавлено!
       fmt.Printf("Local IP:   %s\n", ipInfo)        // Добавлено!
       fmt.Printf("Battery:    %s\n", batteryInfo)   // Добавлено!
       fmt.Printf("Locale:     %s\n", localeInfo)    // Добавлено!
       fmt.Printf("Go Version: %s\n", runtime.Version())
       fmt.Printf("Shell Time: %s\n", time.Now().Format("2006-01-02 15:04:05 Monday"))
    },
})

	register(Command{
		Name:        "date",
		Category:    "System",
		Description: "Выводит текущую дату и время",
		Usage:       "date",
		Handler: func(args []string) {
			fmt.Println(time.Now().Format("2006-01-02 15:04:05 Monday"))
		},
	})

	register(Command{
		Name:        "clear",
		Category:    "System",
		Description: "Очищает экран терминала",
		Usage:       "clear",
		Handler: func(args []string) {
			cmd := exec.Command("cmd", "/c", "cls")
			cmd.Stdout = os.Stdout
			cmd.Run()
		},
    })

	// === КАТЕГОРИЯ: КРИПТОГРАФИЯ ===
	register(Command{
		Name:        "md5",
		Category:    "Crypto",
		Description: "Хэширует строку в MD5",
		Usage:       "md5 [текст]",
		Handler: func(args []string) {
			if len(args) < 1 { return }
			hash := md5.Sum([]byte(strings.Join(args, " ")))
			fmt.Println(hex.EncodeToString(hash[:]))
		},
	})

	register(Command{
		Name:        "sha256",
		Category:    "Crypto",
		Description: "Хэширует строку в SHA-256",
		Usage:       "sha256 [текст]",
		Handler: func(args []string) {
			if len(args) < 1 { return }
			hash := sha256.Sum256([]byte(strings.Join(args, " ")))
			fmt.Println(hex.EncodeToString(hash[:]))
		},
	})

	register(Command{
		Name:        "sha1",
		Category:    "Crypto",
		Description: "Хэширует строку в SHA-1",
		Usage:       "sha1 [текст]",
		Handler: func(args []string) {
			if len(args) < 1 { return }
			h := sha1.New()
			h.Write([]byte(strings.Join(args, " ")))
			fmt.Println(hex.EncodeToString(h.Sum(nil)))
		},
	})

	register(Command{
		Name:        "b64encode",
		Category:    "Crypto",
		Description: "Кодирует строку в Base64",
		Usage:       "b64encode [текст]",
		Handler: func(args []string) {
			if len(args) < 1 { return }
			fmt.Println(base64.StdEncoding.EncodeToString([]byte(strings.Join(args, " "))))
		},
	})

	register(Command{
		Name:        "b64decode",
		Category:    "Crypto",
		Description: "Декодирует строку из Base64",
		Usage:       "b64decode [base64_строка]",
		Handler: func(args []string) {
			if len(args) < 1 { return }
			dec, err := base64.StdEncoding.DecodeString(args[0])
			if err != nil { return }
			fmt.Println(string(dec))
		},
	})

	// === КАТЕГОРИЯ: МАТЕМАТИКА ===
	register(Command{
		Name:        "rand",
		Category:    "Math",
		Description: "Генерирует случайное число",
		Usage:       "rand [максимум]",
		Handler: func(args []string) {
			maxVal := 100
			if len(args) > 0 { if p, err := strconv.Atoi(args[0]); err == nil { maxVal = p } }
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			fmt.Println(r.Intn(maxVal))
		},
	})

	register(Command{
		Name:        "math",
		Category:    "Math",
		Description: "Функции: sqrt, sin, cos, log",
		Usage:       "math [функция] [число]",
		Handler: func(args []string) {
			if len(args) < 2 { return }
			val, err := strconv.ParseFloat(args[1], 64)
			if err != nil { return }
			switch args[0] {
			case "sqrt": fmt.Println(math.Sqrt(val))
			case "sin":  fmt.Println(math.Sin(val))
			case "cos":  fmt.Println(math.Cos(val))
			case "log":  fmt.Println(math.Log(val))
			}
		},
	})
	// === СЛУЖЕБНАЯ СПРАВКА ===
	register(Command{
		Name:        "help",
		Category:    "System",
		Description: "Показывает структурированное меню справки и параметры команд",
		Usage:       "help [имя_команды]",
		Handler: func(args []string) {
			if len(args) > 0 {
				target := args[0]
				if cmd, ok := registry[target]; ok {
					fmt.Printf("\nКоманда:      %s\n", cmd.Name)
					fmt.Printf("Категория:    %s\n", cmd.Category)
					fmt.Printf("Описание:     %s\n", cmd.Description)
					fmt.Printf("Использование: %s\n\n", cmd.Usage)
				} else {
					fmt.Printf("Команда '%s' не найдена.\n", target)
				}
				return
			}
			categories := make(map[string][]string)
			for name, cmd := range registry {
				categories[cmd.Category] = append(categories[cmd.Category], name)
			}
			fmt.Println("\n=================== ИНТЕРФЕЙС УПРАВЛЕНИЯ qSH ===================")
			for cat, cmds := range categories {
				sort.Strings(cmds)
				fmt.Printf("\n⚡ [%s]:\n", cat)
				for _, name := range cmds {
					fmt.Printf("  %-12s - %s\n", name, registry[name].Description)
				}
			}
			fmt.Println("\nДля подробной информации наберите: help [имя_команды]")
		},
	})
}

func main() {
    reader := bufio.NewReader(os.Stdin)

    for {
        // ваш цикл обработки команд...
		currentDir, err := os.Getwd()
		if err != nil {
			currentDir = "qSH"
		}
		fmt.Printf("%s %% ", filepath.Base(currentDir))
		os.Stdout.Sync()

		input, err := reader.ReadString('\n')
		if err != nil {
			continue
		}
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Проверка встроенных псевдонимов (Aliases)
		if translated, exists := aliasRegistry[input]; exists {
			input = translated
		}

		args := strings.Fields(input)
		if len(args) == 0 {
			continue
		}
		cmdName := args[0]

		if cmdName == "exit" {
			fmt.Println("Завершение работы оболочки. До свидания!")
			return
		}

		if cmdName == "cd" {
			target := os.Getenv("USERPROFILE")
			if len(args) > 1 {
				target = args[1]
			}
			if err := os.Chdir(target); err != nil {
				fmt.Printf("cd: директория не найдена: %s\n", target)
			}
			continue
		}

		// Поиск кастомных фич из блоков 5-9
		if interceptAndExecute(cmdName, args[1:]) {
			continue
		}

		if cmd, exists := registry[cmdName]; exists {
			cmd.Handler(args[1:])
			continue
		}

		cmd := exec.Command("cmd", "/c", input)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		_ = cmd.Run()
	    }
    }

// Инициализация базовых переменных и дефолтных алиасов
func initZshFeatures() {
	// Дефолтные алиасы (сокращения) как в реальном Zsh
    aliasRegistry = make(map[string]string)
    shellVariables = make(map[string]string)
	aliasRegistry["ll"] = "ls"
	aliasRegistry["la"] = "ls"
	aliasRegistry["g"] = "grep"
	aliasRegistry["md"] = "mkdir"
	aliasRegistry["rd"] = "rm"

	// Начальные системные переменные нашей оболочки
	shellVariables["SHELL_NAME"] = "qSH"
	shellVariables["SHELL_VERSION"] = "3.5.0"
	shellVariables["ZSH_THEME"] = "robbyrussell"
}
// Функция interceptAndExecute проверяет, относится ли команда к расширенному синтаксису
func interceptAndExecute(cmdName string, args []string) bool {
	// Автоматически сохраняем команду в историю перед выполнением
	fullCommand := cmdName
	if len(args) > 0 {
		fullCommand += " " + strings.Join(args, " ")
	}
	commandHistory = append(commandHistory, fullCommand)

	// Перенаправляем на соответствующие обработчики из следующих блоков
	switch cmdName {
	case "alias":
		handleAliasCommand(args)
		return true
	case "history":
		handleHistoryCommand(args)
		return true
	case "set", "export":
		handleVariableCommand(cmdName, args)
		return true
	}
	return false
}
// handleAliasCommand позволяет пользователю создавать сокращения на лету, например: alias c=clear
func handleAliasCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("--- Текущие алиасы (сокращения) ---")
		for short, long := range aliasRegistry {
			fmt.Printf("  %s -> '%s'\n", short, long)
		}
		return
	}

	// Разбираем строку вида name=command
	pair := strings.Join(args, " ")
	parts := strings.SplitN(pair, "=", 2)
	if len(parts) < 2 {
		fmt.Println("Использование: alias [сокращение]=[полная_команда]")
		return
	}

	short := strings.TrimSpace(parts[0])
	long := strings.TrimSpace(parts[1])
	aliasRegistry[short] = long
	fmt.Printf("Алиас создан: %s теперь запускает '%s'\n", short, long)
}

// handleHistoryCommand выводит список всех ранее введенных команд
func handleHistoryCommand(args []string) {
	if len(args) > 0 && args[0] == "-c" {
		commandHistory = []string{}
		fmt.Println("История команд успешно очищена.")
		return
	}

	fmt.Println("--- История команд ---")
	for idx, cmd := range commandHistory {
		fmt.Printf("  %3d  %s\n", idx+1, cmd)
	}
}
// handleVariableCommand обрабатывает создание и вывод локальных переменных и переменных окружения
func handleVariableCommand(mode string, args []string) {
	if len(args) == 0 {
		fmt.Println("--- Переменные оболочки qSH ---")
		for key, val := range shellVariables {
			fmt.Printf("  %s=%s\n", key, val)
		}
		return
	}

	// Разбираем строку вида KEY=VALUE
	pair := strings.Join(args, " ")
	parts := strings.SplitN(pair, "=", 2)
	if len(parts) < 2 {
		fmt.Println("Использование: set/export [КЛЮЧ]=[ЗНАЧЕНИЕ]")
		return
	}

	key := strings.TrimSpace(parts[0])
	val := strings.TrimSpace(parts[1])

	shellVariables[key] = val

	// Если выбран экспорт, прокидываем переменную на уровень всей ОС Windows
	if mode == "export" {
		os.Setenv(key, val)
		fmt.Printf("Переменная экспортирована в систему: %s=%s\n", key, val)
	} else {
		fmt.Printf("Локальная переменная сохранена: %s=%s\n", key, val)
	}
}
// Структура для отслеживания фоновых задач
type BackgroundJob struct {
	ID      int
	CmdStr  string
	Process *os.Process
}

var backgroundJobs []BackgroundJob
var jobCounter = 1

// handleBackgroundCommand запускает любую утилиту Windows скрытно в фоне
func handleBackgroundCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Использование: bgrun [команда] [аргументы...]")
		return
	}

	cmd := exec.Command(args[0], args[1:]...)
	err := cmd.Start() // Запуск без ожидания окончания (.Run ждет, а .Start нет)
	if err != nil {
		fmt.Printf("Ошибка запуска фонового процесса: %v\n", err)
		return
	}

	job := BackgroundJob{
		ID:      jobCounter,
		CmdStr:  strings.Join(args, " "),
		Process: cmd.Process,
	}
	backgroundJobs = append(backgroundJobs, job)
	fmt.Printf("[%d] Запущен фоновый процесс с PID %d (%s)\n", job.ID, cmd.Process.Pid, job.CmdStr)
	jobCounter++
}

// handleJobsList выводит список всех запущенных вами фоновых утилит
func handleJobsList() {
	fmt.Println("--- Список активных фоновых задач ---")
	if len(backgroundJobs) == 0 {
		fmt.Println("  Фоновые задачи отсутствуют.")
		return
	}

	for _, job := range backgroundJobs {
		// Проверяем, жив ли процесс в Windows
		err := job.Process.Signal(os.Interrupt)
		status := "Работает"
		if err != nil {
			status = "Завершен"
		}
		fmt.Printf("  [%d] PID: %d | Статус: %s | Команда: %s\n", job.ID, job.Process.Pid, status, job.CmdStr)
	}
}

// parseShellVariables заменяет конструкции вида $VAR на их реальные значения из памяти
func parseShellVariables(input string) string {
	for key, val := range shellVariables {
		placeholder := "$" + key
		if strings.Contains(input, placeholder) {
			input = strings.ReplaceAll(input, placeholder, val)
		}
	}
	return input
}
// handleScriptExecute позволяет запускать кастомные файлы со списками команд (.qsh)
func handleScriptExecute(args []string) {
	if len(args) < 1 {
		fmt.Println("Использование: source [путь_к_скрипту.qsh]")
		return
	}

	file, err := os.Open(args[0])
	if err != nil {
		fmt.Printf("Ошибка открытия скрипта: %v\n", err)
		return
	}
	defer file.Close()

	fmt.Printf("[qSH] Запуск скрипта %s...\n", args[0])
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Игнорируем пустые строки и комментарии
		}
		fmt.Printf(">> Выполнение: %s\n", line)
		// В реальном шелле здесь вызывается внутренняя функция парсинга строки
	}
}

// handleBenchmark измеряет точное время выполнения любой команды
func handleBenchmark(args []string) {
	if len(args) < 1 {
		fmt.Println("Использование: time [команда] [аргументы...]")
		return
	}

	start := time.Now()
	cmdName := args[0]
	if cmd, exists := registry[cmdName]; exists {
		cmd.Handler(args[1:])
	} else {
		cmd := exec.Command("cmd", "/c", strings.Join(args, " "))
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		_ = cmd.Run()
	}
	fmt.Printf("\n[Бенчмарк] Время выполнения: %v\n", time.Since(start))
}

// parseQuotes обрабатывает аргументы в кавычках типа: echo "Привет мир"
func parseQuotes(input string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false

	for _, r := range input {
		switch r {
		case '"', '\'':
			inQuotes = !inQuotes
		case ' ':
			if inQuotes {
				current.WriteRune(r)
			} else if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// logSessionActivity автоматически записывает историю сессии в текстовый файл логов
func logSessionActivity() {
	logFile, err := os.OpenFile("qsh_session.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer logFile.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	_, _ = logFile.WriteString(fmt.Sprintf("[%s] Сессия запущена пользователем\n", timestamp))
}

// checkSyntaxError выполняет базовую проверку на незакрытые кавычки перед запуском
func checkSyntaxError(input string) bool {
	quotesCount := strings.Count(input, "\"") + strings.Count(input, "'")
	if quotesCount%2 != 0 {
		fmt.Println("qSH Синтаксическая ошибка: обнаружена незакрытая кавычка!")
		return true
	}
	return false
}

// handleVariableDeletion удаляет переменную из локальной памяти (команда unset)
func handleVariableDeletion(args []string) {
	if len(args) < 1 {
		fmt.Println("Использование: unset [ИМЯ_ПЕРЕМЕННОЙ]")
		return
	}
	delete(shellVariables, args[0])
	fmt.Printf("Переменная %s успешно удалена из памяти.\n", args[0])
}

// handleAliasDeletion удаляет созданный ранее алиас (команда unalias)
func handleAliasDeletion(args []string) {
	if len(args) < 1 {
		fmt.Println("Использование: unalias [ИМЯ_АЛИАСА]")
		return
	}
	delete(aliasRegistry, args[0])
	fmt.Printf("Алиас %s успешно удален.\n", args[0])
}

// handleSystemUptime показывает, сколько времени запущена сама оболочка qSH
var shellStartTime = time.Now()
func handleSystemUptime() {
	fmt.Printf("Оболочка qSH активна уже: %v\n", time.Since(shellStartTime))
}

// handleKillJob принудительно завершает фоновый процесс по его ID
func handleKillJob(args []string) {
	if len(args) < 1 {
		fmt.Println("Использование: kill [ID_задачи]")
		return
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("Неверный ID задачи.")
		return
	}
	for i, job := range backgroundJobs {
		if job.ID == id {
			_ = job.Process.Kill()
			fmt.Printf("Фоновый процесс [%d] с PID %d успешно завершен.\n", job.ID, job.Process.Pid)
			backgroundJobs = append(backgroundJobs[:i], backgroundJobs[i+1:]...)
			return
		}
	}
	fmt.Println("Задача с таким ID не найдена.")
}

// initExtendedCommands дорегистрирует утилиты в глобальное хелп-меню
func initExtendedCommands() {
	register(Command{
		Name: "time", Category: "System", Description: "Замеряет скорость работы команды", Usage: "time [команда]",
		Handler: handleBenchmark,
	})
	register(Command{
		Name: "uptime", Category: "System", Description: "Показывает время аптайма шелла", Usage: "uptime",
		Handler: func(args []string) { handleSystemUptime() },
	})
	register(Command{
		Name: "unset", Category: "System", Description: "Удаляет переменную из памяти", Usage: "unset [имя]",
		Handler: handleVariableDeletion,
	})
}