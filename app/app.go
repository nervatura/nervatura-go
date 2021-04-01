package app

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/joho/godotenv/autoload" // load .env file automatically
	db "github.com/nervatura/nervatura-go/pkg/database"
	ntura "github.com/nervatura/nervatura-go/pkg/nervatura"
	srv "github.com/nervatura/nervatura-go/pkg/service"
	"golang.org/x/sync/errgroup"
)

// App - Nervatura Application
type App struct {
	services    map[string]srv.APIService
	defConn     ntura.DataDriver
	infoLog     *log.Logger
	errorLog    *log.Logger
	signingKeys ntura.IM
}

var services = make(map[string]srv.APIService)

func registerService(name string, server srv.APIService) {
	services[name] = server
}

func New() (app *App, err error) {
	app = &App{
		services:    services,
		signingKeys: make(ntura.IM),
	}

	app.infoLog = log.New(os.Stdout, "INFO: ", log.LstdFlags)
	app.errorLog = log.New(os.Stdout, "ERROR: ", log.LstdFlags)
	if os.Getenv("NT_APP_LOG_FILE") != "" {
		f, err := os.OpenFile(os.Getenv("NT_APP_LOG_FILE"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			app.errorLog.Printf("error opening log file: %v\n", err)
		} else {
			app.infoLog = log.New(f, "INFO: ", log.LstdFlags)
			app.errorLog = log.New(f, "ERROR: ", log.LstdFlags)
		}
		defer f.Close()
	}

	err = app.checkDefaultConn()
	if err != nil {
		app.errorLog.Printf("error checking default database connection: %v\n", err)
		return nil, err
	}

	/*
		err = app.checkExternalToken()
		if err != nil {
			app.errorLog.Printf("error loading external token: %v\n", err)
			return nil, err
		}
	*/

	err = app.startService("cli")
	if err != nil {
		app.errorLog.Printf("error starting cli service: %v\n", err)
		return nil, err
	}

	if services["cli"].Results() == "server" {
		app.startServer()
	}

	return app, err
}

func (app *App) startServer() {
	app.infoLog.Println("skipping cli, start Nervatura server")
	app.infoLog.Printf("enabled database driver(s): %s.\n", strings.Join(db.Drivers, ","))

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	g, ctx := errgroup.WithContext(ctx)

	if _, found := services["http"]; found && ntura.GetEnvValue("bool", os.Getenv("NT_HTTP_ENABLED")).(bool) {
		g.Go(func() error {
			return app.startService("http")
		})
	} else {
		app.infoLog.Println("http rest api is disabled")
	}

	if _, found := services["grpc"]; found && ntura.GetEnvValue("bool", os.Getenv("NT_GRPC_ENABLED")).(bool) {
		g.Go(func() error {
			return app.startService("grpc")
		})
	} else {
		app.infoLog.Println("grpc api is disabled")
	}

	select {
	case <-interrupt:
		break
	case <-ctx.Done():
		break
	}

	app.infoLog.Println("received shutdown signal")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = services["http"].StopService(shutdownCtx)
	_ = services["grpc"].StopService(nil)

	err := g.Wait()
	if err != nil {
		app.errorLog.Printf("application returning an error: %v\n", err)
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

/*
func (app *App) checkExternalToken() error {
	if os.Getenv("NT_TOKEN_KID") != "" && os.Getenv("NT_TOKEN_KEY") != "" {
		app.signingKeys[os.Getenv("NT_TOKEN_KID")] = os.Getenv("NT_TOKEN_KEY")
	}
	if ntura.GetEnvValue("bool", os.Getenv("NT_TOKEN_EXTERNAL_ENABLED")).(bool) && (os.Getenv("NT_TOKEN_EXTERNAL_URL") != "") {
		res, err := http.Get(os.Getenv("NT_TOKEN_EXTERNAL_URL"))
		if err != nil {
			return err
		}
		defer res.Body.Close()
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		err = ntura.ConvertFromByte(data, &app.signingKeys)
		if err != nil {
			return err
		}
	}
	return nil
}
*/

func (app *App) GetNervaStore(database string) *ntura.NervaStore {
	if app.defConn != nil {
		if app.defConn.Connection().Alias == database {
			return ntura.New(app.defConn)
		}
	}
	return ntura.New(&db.SQLDriver{})
}

func (app *App) GetResults() string {
	return app.services["cli"].Results()
}
