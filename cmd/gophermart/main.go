package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app"
)

func main() {
	addrPtr := flag.String("a", app.AddresDef, "адрес и порт запуска сервиса")
	dbURIPtr := flag.String("d", "", "строка с адресом подключения к БД")
	accAddrPtr := flag.String("r", app.AccAddresDef, "адрес системы расчёта начислений")
	flag.Parse()

	opts := []app.FuncOpt{
		app.SetAddr(*addrPtr),
		app.SetDBURI(*dbURIPtr),
		app.SetAccNetAddr(*accAddrPtr),
	}

	if addrENV, ok := os.LookupEnv("RUN_ADDRESS"); ok {
		opts = append(opts, app.SetAddr(addrENV))
	}

	if dbConnENV, ok := os.LookupEnv("DATABASE_URI"); ok {
		opts = append(opts, app.SetDBURI(dbConnENV))
	}

	if accAddrENV, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); ok {
		opts = append(opts, app.SetAccNetAddr(accAddrENV))
	}

	cfg, err := app.NewConfig(opts...)
	if err != nil {
		log.Printf("parse config: %v\n", err)
	}

	app := app.New(cfg)

	ctx := context.Background()

	chErr := make(chan error)
	go func(ce chan<- error) {
		defer close(ce)
		ce <- app.Start()
	}(chErr)

	ctxSinal, stopSignal := signal.NotifyContext(ctx, os.Interrupt)
	select {
	case <-ctxSinal.Done():
		log.Println("signal")
	case err := <-chErr:
		stopSignal()
		if err != nil {
			log.Printf("app start err %v\n", err)
		}
	}

	if err := app.Stop(ctx); err != nil {
		log.Printf("app stop err: %v\n", err)
	} else {
		log.Println("all services stopped")
	}

	log.Println("app stopped")
}
