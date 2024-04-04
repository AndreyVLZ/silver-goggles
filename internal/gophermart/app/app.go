package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/http"
	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/http/handler"
	m "github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/middle"
	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/pkg/bcrypt"
	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/pkg/hash"
	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/repository/accrual"
	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/repository/postgres"
	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/service/auth"
	ordSrv "github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/service/order"
)

const (
	AddresDef    string = "localhost:8080"
	AccAddresDef string = "localhost:8081"
)

type Config struct {
	Addr   string
	DBConn string
	AccURL *url.URL
}

type FuncOpt func(*Config) error

func NewConfig(opts ...FuncOpt) (*Config, error) {
	cfg := &Config{}
	for i := range opts {
		if err := opts[i](cfg); err != nil {
			return nil, fmt.Errorf("newConfig: %w", err)
		}
	}

	return cfg, nil
}

type api interface {
	Start() error
	Stop(context.Context) error
}

type service interface {
	Start() error
	Stop() error
	Name() string
}

type Services []service

type app struct {
	api api
	Services
	log *slog.Logger
	cfg *Config
}

func (srvs *Services) add(arrSrvs ...service) { *srvs = append(*srvs, arrSrvs...) }

func New(cfg *Config) *app {
	log := initLog()
	updateInterval := 6

	pqStore := postgres.New(postgres.Config{ConnDB: cfg.DBConn})

	auther := handler.NewAuthBycookie("cookieKey")

	bcHashe := hash.NewHashe(bcrypt.Hash, bcrypt.Compare)
	authSrv := auth.NewAuthService(pqStore, bcHashe) // service
	userHandler := handler.NewUserHandler(authSrv, auther.Set, log)

	accRepo := accrual.New(cfg.AccURL)

	saveSrv := ordSrv.NewSaveService(accRepo, pqStore) // service save
	orderSrv := ordSrv.NewOrderService(pqStore)        // service get
	orderHandler := handler.NewOrderHandler(saveSrv, orderSrv, log)

	uSrv := ordSrv.NewUService(accRepo, pqStore, updateInterval, log) // service update

	r := http.NewRouter()

	// группа маршрутов logger
	r.Group("/api/user", func(r http.Router) { //prefix
		r.Use(m.Logger(log)) // middle group

		r.Handle("/register", // pattern
			m.Use(
				userHandler.Register(), // handler
				m.Post(),               // check method
				m.AppJSON(),            // check contentType
			),
		)

		r.Handle("/login",
			m.Use(
				userHandler.Login(),
				m.Post(),
				m.AppJSON(),
			),
		)

		// группа маршрутов auth
		r.Group("/", func(r http.Router) {
			r.Use(m.AuthByFunc(handler.GetContextKey(), auther.Check)) // auth

			r.Handle("/orders",
				m.Select{
					Get: orderHandler.GetOrders(),
					Post: m.Use(
						orderHandler.LoadHandle(),
						m.TextPlain(),
					),
				},
			)

			r.Handle("/balance",
				m.Use(
					orderHandler.Balance(),
					m.Get(),
				),
			)

			r.Handle("/balance/withdraw",
				m.Use(
					orderHandler.Withdraw(),
					m.Post(),
					m.AppJSON(),
				),
			)

			r.Handle("/withdrawals",
				m.Use(
					orderHandler.Withdrawals(),
					m.Get(),
				),
			)
		})
	})

	httpServer := http.NewServer(
		http.ServerConfig{Addr: cfg.Addr}, *r,
	)

	app := &app{
		api: httpServer,
		log: log,
		cfg: cfg,
	}

	app.Services.add(
		uSrv,
		pqStore,
	)

	return app
}

func (app *app) Start() error {
	app.log.Info("start server",
		"addr", app.cfg.Addr,
		"dbConn", app.cfg.DBConn,
		"accURL", app.cfg.AccURL.String(),
	)

	for i := range app.Services {
		if err := app.Services[i].Start(); err != nil {
			return err
		}
		app.log.Info("start service", "name", app.Services[i].Name())
	}

	return app.api.Start()
}

func (app *app) Stop(ctx context.Context) error {
	ctxTimeout, stopTimeout := context.WithTimeout(ctx, 5*time.Second)
	defer stopTimeout()

	errs := make([]error, 0, len(app.Services)+1)
	if err := app.api.Stop(ctxTimeout); err != nil {
		errs = append(errs, err)
	}

	for _, srv := range app.Services {
		if err := srv.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("service [%s] err: %w", srv.Name(), err))
			continue
		}
		app.log.Info("stop service", "name", srv.Name())
	}

	return errors.Join(errs...)
}

func initLog() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

func SetAddr(addr string) FuncOpt {
	return func(s *Config) error {
		s.Addr = addr
		return nil
	}
}

func SetDBURI(dbConn string) FuncOpt {
	return func(s *Config) error {
		s.DBConn = dbConn
		return nil
	}
}

func SetAccNetAddr(accAddr string) FuncOpt {
	return func(s *Config) error {
		u, err := url.Parse(accAddr)
		if err != nil {
			return err
		}
		s.AccURL = u

		return nil
	}
}
