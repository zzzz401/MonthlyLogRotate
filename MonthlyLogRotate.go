package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func getLastDayOfMonth() time.Time {
	year, month, _ := time.Now().Date()
	// Neat Trick returns the 0 day of next month which is = to the last day of this month
	return time.Date(year, time.Month(month+1), 0, 23, 59, 59, 0, time.Local)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func openLog(path string) *os.File {
	_, err := os.Stat(path)

	var logFile *os.File

	if err == nil {
		logFile, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	} else {
		logFile, err = os.Create(path)
	}
	checkError(err)

	return logFile
}

func generateLogFilePath(logDir string, logName string) string {
	year := strconv.Itoa(time.Now().Year())
	month := fmt.Sprintf("%02d", int(time.Now().Month()))
	return logDir + logName + "-" + year + "-" + month + ".log"
}

var gracefulStop = make(chan os.Signal, 1)

func main() {
	logDirPtr := flag.String("logDir", "", "Path to log directory with trailing slash")
	logNamePtr := flag.String("logName", "", "Name of each log file")
	flag.Parse()

	stat, err := os.Stdin.Stat()

	checkError(err)

	if (stat.Mode() & os.ModeCharDevice) != 0 {
		fmt.Println("No data being piped in from StdIn")
		os.Exit(1)
	}

	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	logFilePath := generateLogFilePath(*logDirPtr, *logNamePtr)
	logFile := openLog(logFilePath)

	scanner := bufio.NewScanner(os.Stdin)
	endOfMonthUnix := getLastDayOfMonth().Unix()

	for {
		select {
		case <-gracefulStop:
			logFile.Sync()
			logFile.Close()
			fmt.Println("Stopping Log Rotation")
			os.Exit(0)
		default:
			stillReading := scanner.Scan()

			if stillReading == false {
				fmt.Println("Nothing to read from StdIn")
				gracefulStop <- syscall.SIGTERM
				continue
			}

			text := scanner.Text()

			if err := scanner.Err(); err != nil {
				checkError(err)
			}

			if time.Now().Unix() > endOfMonthUnix {
				logFile.Sync()
				logFile.Close()
				logFilePath = generateLogFilePath(*logDirPtr, *logNamePtr)
				logFile = openLog(logFilePath)
				endOfMonthUnix = getLastDayOfMonth().Unix()
			}

			if text != "" {
				_, err := logFile.WriteString(text + "\r\n")
				checkError(err)
				logFile.Sync()
			}
		}
	}
}
