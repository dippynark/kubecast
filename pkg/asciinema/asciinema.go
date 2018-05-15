package asciinema

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dippynark/kubepf/pkg/server"
)

type header struct {
	Version   int   `json:"version"`
	Width     int   `json:"width"`
	Height    int   `json:"height"`
	Timestamp int64 `json:"timestamp"`
}

func Init(ttyWrite *server.TtyWrite, file *os.File) error {
	h := header{
		Version:   2,
		Width:     80,
		Height:    80,
		Timestamp: int64(ttyWrite.Timestamp / 1000000000),
	}

	b, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %s", err)
	}

	bytesWritten, err := file.Write(b)
	if err != nil {
		return fmt.Errorf("write failed: %s", err)
	}
	if bytesWritten != len(b) {
		return fmt.Errorf("failed to write all bytes")
	}

	return nil
}

func Append(ttyWrite *server.TtyWrite, file *os.File) error {

	file.Seek(0, 0)
	scanner := bufio.NewScanner(file)
	scanner.Scan()

	var h header
	err := json.Unmarshal([]byte(scanner.Text()), &h)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %s", err)
	}

	var entry []interface{}
	entry = append(entry, ((float64(ttyWrite.Timestamp))/1000000000)-(float64(h.Timestamp)))
	entry = append(entry, "o")
	entry = append(entry, strings.Replace(string(ttyWrite.Buffer[0:ttyWrite.Count]), "\n", "\r\n", -1))

	b, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %s", err)
	}

	b = append([]byte{'\n'}, b...)

	bytesWritten, err := file.Write(b)
	if err != nil {
		return fmt.Errorf("write failed: %s", err)
	}
	if bytesWritten != len(b) {
		return fmt.Errorf("failed to write all bytes")
	}

	return nil
}
