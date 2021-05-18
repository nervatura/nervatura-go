package app

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/joho/godotenv/autoload" // load .env file automatically
	db "github.com/nervatura/nervatura-go/pkg/database"
	nt "github.com/nervatura/nervatura-go/pkg/nervatura"
	srv "github.com/nervatura/nervatura-go/pkg/service"
	ut "github.com/nervatura/nervatura-go/pkg/utils"
	"golang.org/x/sync/errgroup"
)

// App - Nervatura Application
type App struct {
	version   string
	services  map[string]srv.APIService
	defConn   nt.DataDriver
	infoLog   *log.Logger
	errorLog  *log.Logger
	tokenKeys map[string]map[string]string
}

var services = make(map[string]srv.APIService)

func registerService(name string, server srv.APIService) {
	services[name] = server
}

func New(version string) (app *App, err error) {
	app = &App{
		version:   version,
		services:  services,
		tokenKeys: make(map[string]map[string]string),
	}

	app.infoLog = log.New(os.Stdout, "INFO: ", log.LstdFlags)
	app.errorLog = log.New(os.Stdout, "ERROR: ", log.LstdFlags)
	if os.Getenv("NT_APP_LOG_FILE") != "" {
		f, err := os.OpenFile(os.Getenv("NT_APP_LOG_FILE"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			app.errorLog.Printf(ut.GetMessage("error_opening_log"), err)
		} else {
			app.infoLog = log.New(f, "INFO: ", log.LstdFlags)
			app.errorLog = log.New(f, "ERROR: ", log.LstdFlags)
		}
		defer f.Close()
	}

	err = app.checkDefaultConn()
	if err != nil {
		app.errorLog.Printf(ut.GetMessage("error_checking_def_db"), err)
		return nil, err
	}

	err = app.setPrivateKey()
	if err != nil {
		app.errorLog.Printf(ut.GetMessage("error_private_key"), err)
		return nil, err
	}

	err = app.startService("cli")
	if err != nil {
		app.errorLog.Printf(ut.GetMessage("error_starting_cli"), err)
		return nil, err
	}

	if services["cli"].Results() == "server" {
		app.startServer()
	}

	return app, err
}

func (app *App) setPrivateKey() error {
	pkey := os.Getenv("NT_TOKEN_PRIVATE_KEY")
	kid := ut.ToString(os.Getenv("NT_TOKEN_KID"), "private")
	ktype := ut.ToString(os.Getenv("NT_TOKEN_PRIVATE_KEY_TYPE"), "KEY")
	if pkey != "" {
		if _, err := os.Stat(pkey); err == nil {
			content, err := ioutil.ReadFile(pkey)
			if err != nil {
				return err
			}
			pkey = string(content)
		}
		app.tokenKeys[kid] = nt.SM{
			"type":  "private",
			"ktype": ktype,
			"value": pkey,
		}
	}
	return nil
}

func (app *App) startServer() {
	app.infoLog.Println(ut.GetMessage("skipping_cli"))
	app.infoLog.Printf(ut.GetMessage("enabled_drivers"), strings.Join(db.Drivers, ","))

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	g, ctx := errgroup.WithContext(ctx)

	if _, found := services["http"]; found && ut.GetEnvValue("bool", os.Getenv("NT_HTTP_ENABLED")).(bool) {
		g.Go(func() error {
			return app.startService("http")
		})
	} else {
		app.infoLog.Println(ut.GetMessage("http_disabled"))
	}

	if _, found := services["grpc"]; found && ut.GetEnvValue("bool", os.Getenv("NT_GRPC_ENABLED")).(bool) {
		g.Go(func() error {
			return app.startService("grpc")
		})
	} else {
		app.infoLog.Println(ut.GetMessage("grpc_disabled"))
	}

	select {
	case <-interrupt:
		break
	case <-ctx.Done():
		break
	}

	app.infoLog.Println(ut.GetMessage("shutdown_signal"))

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = services["http"].StopService(shutdownCtx)
	_ = services["grpc"].StopService(nil)

	err := g.Wait()
	if err != nil {
		app.errorLog.Printf(ut.GetMessage("application_error"), err)
		os.Exit(2)
	}
}

func (app *App) startService(name string) error {
	services[name].ConnectApp(app)
	return services[name].StartService()
}

func (app *App) checkDefaultConn() (err error) {
	connStr := ""
	alias := ""
	if os.Getenv("NT_ALIAS_DEFAULT") != "" {
		connStr = os.Getenv("NT_ALIAS_" + strings.ToUpper(os.Getenv("NT_ALIAS_DEFAULT")))
		alias = strings.ToLower(os.Getenv("NT_ALIAS_DEFAULT"))
	}
	if connStr != "" {
		app.defConn = &db.SQLDriver{}
		return app.defConn.CreateConnection(alias, connStr)
	}
	return nil
}

func (app *App) GetNervaStore(database string) *nt.NervaStore {
	if app.defConn != nil {
		if app.defConn.Connection().Alias == database {
			return nt.New(app.defConn)
		}
	}
	return nt.New(&db.SQLDriver{})
}

func (app *App) GetResults() string {
	return app.services["cli"].Results()
}
