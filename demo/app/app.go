package app

import (
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	ntura "github.com/nervatura/nervatura-go"
	driver "github.com/nervatura/nervatura-go/driver"
	"github.com/spf13/viper"
)

// App is a Nervatura Server Application
type App struct {
	config      ntura.Settings
	defConn     ntura.DataDriver
	settings    *viper.Viper
	signingKeys ntura.IM
	server      *echo.Echo
}

// New - create a new Nervatura Server App
func New() (err error) {
	app := new(App)
	err = app.init()
	if err != nil {
		return err
	}

	app.server = echo.New()
	app.setMiddleware()
	app.parseRequests()

	hostname := app.settings.GetString("host")
	if app.settings.GetInt("port") > 0 {
		hostname += ":" + app.settings.GetString("port")
	}

	app.server.Logger.Fatal(app.server.Start(hostname))

	return nil
}

func (app *App) init() error {
	err := app.readConfig()
	if err != nil {
		return err
	}
	app.config, err = ntura.ReadConfig("config")
	if err != nil {
		return err
	}
	if _, found := app.config.Alias["default"]; found {
		connStr := app.config.Alias[app.config.Alias["default"]]
		app.defConn = &driver.SQLDriver{}
		err = app.defConn.CreateConnection(app.config.Alias["default"], connStr, app.config)
		if err != nil {
			return err
		}
	}

	app.signingKeys = ntura.IM{}
	if app.config.APIEndpoint != "" {
		app.signingKeys, err = getCertificates(app.config.APIEndpoint)
		if err != nil {
			return err
		}
	}
	app.signingKeys[app.config.TokenKid] = app.config.TokenKey

	return err
}

func (app *App) readConfig() error {
	app.settings = viper.New()
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	app.settings.AddConfigPath("config")
	app.settings.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	app.settings.SetEnvPrefix("NT")
	app.settings.AutomaticEnv()
	app.settings.SetConfigName("server")
	return app.settings.ReadInConfig()
}

func (app *App) setMiddleware() {

	//app.server.Use(middleware.Logger())
	app.server.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `${time_rfc3339} | ${status} | ${latency_human} | ${remote_ip} | ${method} | ${uri}` + "\n",
		Output: os.Stdout,
	}))
	//app.server.Use(middleware.Recover())

	app.server.Use(middleware.CORSWithConfig(getCORSConfig(app.settings)))
	//secure := secure.New(getSecureOptions(app.settings))
	//app.server.Use(echo.WrapMiddleware(secure.Handler))

}

// parseRequests - Parse requests from REST API
func (app *App) parseRequests() {

	app.server.Static("/docs/*", "../../nervatura-docs/docs")
	app.server.Static("/report/*", "../../nervatura-demo/docs")
	app.server.Static("/client/*", "../../nervatura-client/build")

	app.server.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/docs/")
	})

	app.server.POST("/api/auth/login*", app.apiAuthLogin)
	app.server.POST("/api/auth/password*", app.apiAuthPassword, app.tokenAuth)
	app.server.GET("/api/auth/refresh*", app.apiAuthRefresh, app.tokenAuth)

	app.server.POST("/api/database*", app.apiDatabase)
	app.server.POST("/api/function*", app.apiFunction, app.tokenAuth)
	app.server.POST("/api/view*", app.apiView, app.tokenAuth)

	app.server.GET("/api/report*", app.report, app.tokenAuth)
	app.server.GET("/api/report/list*", app.reportList, app.tokenAuth)
	app.server.POST("/api/report/install*", app.reportInstall, app.tokenAuth)
	app.server.DELETE("/api/report/delete*", app.reportDelete, app.tokenAuth)

	app.server.GET("/api/:nervatype", app.apiGetFilter, app.tokenAuth)
	app.server.GET("/api/:nervatype/:ids", app.apiGetIds, app.tokenAuth)
	app.server.POST("/api/:nervatype", app.apiPost, app.tokenAuth)
	app.server.DELETE("/api/:nervatype", app.apiDelete, app.tokenAuth)

	app.server.POST("/npi/token/login*", app.npiTokenLogin)
	app.server.POST("/npi/token/", app.npi, app.tokenAuth)
	app.server.POST("/npi/", app.npiTokenLogin)
	app.server.POST("/npi", app.npiTokenLogin)

}
