package logging

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/op/go-logging"
)

type HttpBackend struct {
	cli      *http.Client
	minLevel logging.Level
}

func (h *HttpBackend) Log(level logging.Level, i int, record *logging.Record) error {
	fmt.Printf("customLog: %s %s %s\n", record.Time, level, record.Formatted(i+1))

	req, err := http.NewRequest("POST", "http://5.101.179.223:12345/", strings.NewReader(record.Formatted(i+1)))
	if err != nil {
		err = fmt.Errorf("failed to make request: %v", err)

		fmt.Println(err.Error())
		return err
	}

	req.Header.Set("level", level.String())
	req.Header.Set("module", record.Module)
	req.Header.Set("time", record.Time.Format(time.RFC3339Nano))

	_, err = h.cli.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to perform post request: %v", err)
		fmt.Println(err.Error())

		return err
	}

	return nil
}

func (h *HttpBackend) GetLevel(s string) logging.Level {
	return h.minLevel
}

func (h *HttpBackend) SetLevel(level logging.Level, s string) {
}

func (h *HttpBackend) IsEnabledFor(level logging.Level, s string) bool {
	return true
}

func NewHTTPBackend() logging.LeveledBackend {
	formatStr := "[%{id:03x}] %{shortfile} -> %{message}"
	format := logging.MustStringFormatter(formatStr)

	backend := &HttpBackend{
		cli: &http.Client{
			Timeout: time.Second,
		},
	}

	formatted := logging.NewBackendFormatter(backend, format)
	leveled := logging.AddModuleLevel(formatted)
	return leveled
}
