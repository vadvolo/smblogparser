package types

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type SmbLogItem struct {
	Login     string
	Path      string
	File      string
	Action    string
	Device    string
	Timestamp time.Time
	Created   time.Time
}

func NewSmbLogItem() *SmbLogItem {
	return &SmbLogItem{
		Created: time.Now(),
	}
}

func (item *SmbLogItem) Print() {
	date, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		fmt.Sprintf("%e", err)
	}

	fmt.Println(string(date))
}

type Logger struct {
	Device string
	Items  []*SmbLogItem
}

func NewLogger(device string) *Logger {
	return &Logger{
		Device: device,
	}
}

func (l *Logger) ExportCVS() {
	for _, item := range l.Items {
		fmt.Printf("%s,%s,%s,%s,%s", item.Login, item.Device, item.Timestamp, item.File, item.Action)
	}
}

func (l *Logger) ReadBytes() {
	input := bufio.NewReader(os.Stdin)
	buff := [][]byte{}
	for {

		c, err := input.ReadBytes(10)
		if err == io.EOF {
			break
		}

		fmt.Println(string(c))

		buff = append(buff, c)
		if len(buff) == 2 {
			logItem := l.ParseData(buff)
			logItem.Print()
			l.Items = append(l.Items, logItem)
			buff = [][]byte{}
		}
	}
}

func (l *Logger) Write(w *io.Writer) {

}

func (l *Logger) ParseData(input [][]byte) *SmbLogItem {
	if strings.Contains(string(input[0]), "open_file") {
		logItem := l.firstLineParse(input[0])
		parseSecondLine(input[1], logItem)
		logItem.Action = "open_file"
		return logItem
	}
	if strings.Contains(string(input[0]), "close_normal_file") {
		logItem := l.firstLineParse(input[0])
		parseSecondLine(input[1], logItem)
		logItem.Action = "close_file"
		return logItem
	}
	return nil
}

func (l *Logger) firstLineParse(input []byte) *SmbLogItem {
	fmt.Println("Parging first line", string(input))
	// hostname := new(strings.Builder)
	timestamp := new(strings.Builder)
	offset := 1
	for i := offset; i < len(input); i++ {
		// if hostname.Cap() == 0 {
		// 	for input[i+1] != 91 {
		// 		hostname.WriteByte(input[i])
		// 		i++
		// 		if i == len(input) {
		// 			break
		// 		}
		// 	}
		// }

		// i += 2

		if timestamp.Cap() == 0 {
			for input[i] != 46 {
				switch input[i] {
				case 47:
					timestamp.WriteByte(45)
				default:
					timestamp.WriteByte(input[i])
				}
				i++
				if i == len(input) {
					break
				}
			}
		}
		break
	}

	t, err := time.Parse(time.DateTime, timestamp.String())
	if err != nil {
		fmt.Println(err)
	}

	logItem := NewSmbLogItem()
	logItem.Device = l.Device
	logItem.Timestamp = t
	return logItem
}

func parseSecondLine(input []byte, logItem *SmbLogItem) {
	fmt.Println("Parging second line", string(input))
	offset := 0
	login := new(strings.Builder)
	filePath := new(strings.Builder)
	for i := offset; i < len(input); i++ {
		if input[i] == 32 {
			continue
		}
		for input[i] != 32 {
			login.WriteByte(input[i])
			i++
			if i == len(input) {
				break
			}
		}

		fmt.Println(login)

		for string(input[i-5:i]) != "file " {
			i++
			if i == len(input) {
				break
			}
		}
		for string(input[i:i+5]) != " read" && string(input[i:i+5]) != " (num" {
			filePath.WriteByte(input[i])
			i++
			if i == len(input) {
				break
			}
		}

		break
	}
	logItem.Login = login.String()
	logItem.File = filePath.String()
}
