package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/tickets-dao/logs-saver/internal/config"
	"github.com/tickets-dao/logs-saver/pkg/logger"
)

func main() {
	loc := time.FixedZone("Moscow", 3*60*60)
	time.Local = loc

	conf := &config.Config{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := conf.ParseConfiguration()
	if err != nil {
		log.Fatal("Failed to parse configuration: ", err)
	}

	err = logger.InitLogger(conf.Logger)
	if err != nil {
		log.Fatal("Failed to create logger: ", err)

		return
	}

	f, err := os.Create("tmp/" + time.Now().Format("2006-01-02T15:04:05") + ".csv")
	if err != nil {
		log.Fatal("failed to create new csv file:", err)
	}

	defer f.Close()

	csvWriter := csv.NewWriter(f)

	r := chi.NewRouter()
	r.Use(cors.AllowAll().Handler)
	r.Post("/", saveLog(csvWriter))

	logger.Infof(ctx, "starting mux at '%s:%s'", conf.Listen.BindIP, conf.Listen.Port)

	srv := &http.Server{Addr: fmt.Sprintf("%s:%s", conf.Listen.BindIP, conf.Listen.Port), Handler: r}
	if err = srv.ListenAndServe(); err != nil {
		logger.Fatalf(ctx, "failed to run server: %v", err)
	}
}

func saveLog(db *csv.Writer) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		startedAt := time.Now()
		ctx := req.Context()

		body, err := io.ReadAll(req.Body)
		if err != nil {
			erString := fmt.Sprintf("failed to read body: %v", err)
			logger.ErrorKV(ctx, erString)
			http.Error(rw, erString, 400)
			return
		}

		logTime := req.Header.Get("time")
		if logTime == "" {
			logger.Warnf(ctx, "no log time provided, going to use time.Now()")
			logTime = time.Now().Format(time.RFC3339Nano)
		}

		err = db.Write([]string{
			req.Header.Get("level"),
			req.Header.Get("module"),
			string(body),
			logTime,
		})
		if err != nil {
			erString := fmt.Sprintf("failed to save log: %v", err)
			logger.ErrorKV(ctx, erString)
			http.Error(rw, erString, 400)
			return
		}
		db.Flush()

		logID := uuid.New()

		rw.Header().Set("id", logID.String())

		logger.Infof(ctx, "saved log '%s' in %s", logID, time.Since(startedAt))
	}
}
