/*
Nervatura Demo Application
*/

package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	_ "github.com/nervatura/nervatura-client"
	_ "github.com/nervatura/nervatura-demo"
	_ "github.com/nervatura/nervatura-docs"
	ntura "github.com/nervatura/nervatura-go"
	driver "github.com/nervatura/nervatura-go/driver"
	_ "github.com/nervatura/report-templates"
	"github.com/spf13/viper"
	"github.com/unrolled/secure"
)

var config ntura.Settings
var defConn ntura.DataDriver

func getNStore(options ntura.IM) *ntura.NervaStore {
	var database string
	if _, found := options["database"]; found {
		database = options["database"].(string)
	}
	if defConn != nil {
		if defConn.Connection().Alias == database {
			return ntura.New(config, defConn)
		}
	}
	if database != "" {
		return ntura.New(config, &driver.SQLDriver{})
	}
	if defConn == nil {
		return nil
	}
	return ntura.New(config, defConn)
}

func apiAuthLogin(c echo.Context) error {
	data := ntura.IM{}
	if err := c.Bind(&data); err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	nstore := getNStore(data)
	if nstore == nil {
		return echo.ErrUnauthorized
	}
	tokenStr, err := (&ntura.API{NStore: nstore}).AuthUserLogin(data)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, ntura.SM{"token": tokenStr})
}

func apiAuthPassword(c echo.Context) error {
	data := ntura.IM{}
	if err := c.Bind(&data); err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	if _, found := data["username"]; found {
		if c.Get("nstore").(*ntura.NervaStore).User["scope"] != "admin" {
			return echo.ErrUnauthorized
		}
	}
	if _, found := data["custnumber"]; found {
		if c.Get("nstore").(*ntura.NervaStore).User["scope"] != "admin" {
			return echo.ErrUnauthorized
		}
	}
	if _, found := data["username"]; !found {
		if _, found := data["custnumber"]; !found {
			if c.Get("nstore").(*ntura.NervaStore).Customer != nil {
				data["custnumber"] = c.Get("nstore").(*ntura.NervaStore).Customer["custnumber"]
			} else {
				data["username"] = c.Get("nstore").(*ntura.NervaStore).User["username"]
			}
		}
	}
	err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).AuthPassword(data)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func apiAuthRefresh(c echo.Context) error {
	tokenStr, err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).AuthToken()
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, ntura.SM{"token": tokenStr})
}

func apiGetFilter(c echo.Context) error {
	params := ntura.IM{"nervatype": c.ParamValues()[0],
		"metadata": c.QueryParam("metadata")}
	query := strings.Split(c.QueryString(), "&")
	for index := 0; index < len(query); index++ {
		if strings.HasPrefix(query[index], "filter=") {
			params["filter"] = query[index][7:]
		}
	}
	results, err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).APIGet(params)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, results)
}

func apiGetIds(c echo.Context) error {
	params := ntura.IM{"nervatype": c.ParamValues()[0],
		"metadata": c.QueryParam("metadata"), "ids": c.ParamValues()[1]}
	results, err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).APIGet(params)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, results)
}

func apiView(c echo.Context) error {
	data := make([]ntura.IM, 0)
	if err := json.NewDecoder(c.Request().Body).Decode(&data); err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	results, err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).APIView(data)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, results)
}

func apiFunction(c echo.Context) error {
	data := ntura.IM{}
	if err := c.Bind(&data); err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	results, err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).APIFunction(data)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, results)
}

func apiPost(c echo.Context) error {
	data := make([]ntura.IM, 0)
	if err := json.NewDecoder(c.Request().Body).Decode(&data); err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	results, err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).APIPost(c.ParamValues()[0], data)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, results)
}

func apiDelete(c echo.Context) error {
	data := ntura.IM{}
	if err := c.Bind(&data); err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).APIDelete(data)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func report(c echo.Context) error {
	options := ntura.IM{"filters": ntura.IM{}}
	for key, value := range c.QueryParams() {
		if strings.HasPrefix(key, "filters[") {
			fkey := key[8 : len(key)-1]
			options["filters"].(ntura.IM)[fkey] = value[0]
		} else {
			switch key {
			case "report_id":
				reportID, err := strconv.Atoi(value[0])
				if err == nil {
					options["report_id"] = reportID
				}
			case "output":
				if value[0] == "data" {
					options["output"] = "tmp"
				} else {
					options["output"] = value[0]
				}
			default:
				options[key] = value[0]
			}
		}
	}
	results, err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).Report(options)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	if options["output"] == "tmp" {
		return c.JSON(http.StatusOK, results)
	}
	if results["filetype"] == "xlsx" {
		return c.Blob(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", results["template"].([]uint8))
	}
	if results["filetype"] == "ntr" && options["output"] == "xml" {
		return c.XML(http.StatusOK, results["template"])
	}
	return c.Blob(http.StatusOK, "application/pdf", results["template"].([]uint8))
}

func reportList(c echo.Context) error {
	params := ntura.IM{"label": c.QueryParam("label")}
	results, err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).ReportList(params)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, results)
}

func reportInstall(c echo.Context) error {
	params := ntura.IM{"reportkey": c.QueryParam("reportkey")}
	results, err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).ReportInstall(params)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, results)
}

func reportDelete(c echo.Context) error {
	params := ntura.IM{"reportkey": c.QueryParam("reportkey")}
	err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).ReportDelete(params)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func apiDatabase(c echo.Context) error {
	apiKey := c.Request().Header.Get("X-Api-Key")
	if config.APIKey != apiKey {
		return echo.ErrUnauthorized
	}
	data := ntura.IM{"database": c.QueryParam("alias"), "demo": c.QueryParam("demo")}
	log, err := (&ntura.API{NStore: ntura.New(config, &driver.SQLDriver{})}).DatabaseCreate(data)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, log)
}

func npiTokenLogin(c echo.Context) error {
	data := ntura.JSONData{}
	if err := c.Bind(&data); err != nil {
		data.Error = ntura.IM{"code": "json_decode", "message": err.Error(), "data": ""}
		return c.JSON(http.StatusBadRequest, data)
	}
	nstore := getNStore(data.Params)
	if nstore == nil {
		return echo.ErrUnauthorized
	}
	result, _ := (&ntura.Npi{NStore: nstore}).GetLogin(data.Params)
	data.Result = result
	return c.JSON(http.StatusOK, data)
}

func npi(c echo.Context) error {
	data := ntura.JSONData{}
	if err := c.Bind(&data); err != nil {
		data.Error = ntura.IM{"code": "json_decode", "message": err.Error(), "data": ""}
		return c.JSON(http.StatusBadRequest, data)
	}
	result := (&ntura.Npi{NStore: c.Get("nstore").(*ntura.NervaStore)}).GetAPI(data)
	return c.JSON(http.StatusOK, result)
}

func readConfig() (*viper.Viper, error) {
	v := viper.New()
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	v.AddConfigPath("../config")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("NT")
	v.AutomaticEnv()
	v.SetConfigName("server")
	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}
	return v, nil
}

func getCertificates(APIEndpoint string) (certs ntura.IM, err error) {
	res, err := http.Get(APIEndpoint)
	if err != nil {
		return
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	json.Unmarshal(data, &certs)
	return
}

func tokenAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tokenStr := ""
		auth := c.Request().Header.Get("Authorization")
		l := len("Bearer")
		if len(auth) > l+1 && auth[:l] == "Bearer" {
			tokenStr = auth[l+1:]
		}
		if tokenStr == "" {
			return echo.ErrUnauthorized
		}
		claim, err := ntura.TokenDecode(tokenStr)
		if err != nil {
			return echo.ErrUnauthorized
		}
		nstore := getNStore(claim)
		if nstore == nil {
			return echo.ErrUnauthorized
		}
		err = (&ntura.API{NStore: nstore}).AuthTokenLogin(ntura.IM{"token": tokenStr})
		if err != nil {
			return echo.ErrUnauthorized
		}
		c.Set("nstore", nstore)
		return next(c)
	}
}

func getCORSConfig(settings *viper.Viper) middleware.CORSConfig {
	return middleware.CORSConfig{
		AllowOrigins:     settings.GetStringSlice("cors.allow_origins"),
		AllowMethods:     settings.GetStringSlice("cors.allow_methods"),
		AllowHeaders:     settings.GetStringSlice("cors.allow_headers"),
		AllowCredentials: settings.GetBool("cors.allow_credentials"),
		ExposeHeaders:    settings.GetStringSlice("cors.expose_headers"),
		MaxAge:           settings.GetInt("cors.max_age"),
	}
}

func getSecureOptions(settings *viper.Viper) secure.Options {
	return secure.Options{
		AllowedHosts:            settings.GetStringSlice("security.allowed_hosts"),
		AllowedHostsAreRegex:    settings.GetBool("security.allowed_hosts_are_regex"),
		HostsProxyHeaders:       settings.GetStringSlice("security.hosts_proxy_headers"),
		SSLRedirect:             settings.GetBool("security.ssl_redirect"),
		SSLTemporaryRedirect:    settings.GetBool("security.ssl_temporary_redirect"),
		SSLHost:                 settings.GetString("security.ssl_host"),
		SSLProxyHeaders:         settings.GetStringMapString("security.ssl_proxy_headers"),
		STSSeconds:              settings.GetInt64("security.sts_seconds"),
		STSIncludeSubdomains:    settings.GetBool("security.sts_include_subdomains"),
		STSPreload:              settings.GetBool("security.sts_preload"),
		ForceSTSHeader:          settings.GetBool("security.force_sts_header"),
		FrameDeny:               settings.GetBool("security.frame_deny"),
		CustomFrameOptionsValue: settings.GetString("security.custom_frame_options_value"),
		ContentTypeNosniff:      settings.GetBool("security.content_type_nosniff"),
		BrowserXssFilter:        settings.GetBool("security.browser_xss_filter"),
		ContentSecurityPolicy:   settings.GetString("security.content_security_policy"),
		PublicKey:               settings.GetString("security.public_key"),
		ReferrerPolicy:          settings.GetString("security.referrer_policy"),
		FeaturePolicy:           settings.GetString("security.feature_policy"),
		ExpectCTHeader:          settings.GetString("security.expect_ct_header"),
		IsDevelopment:           settings.GetBool("development"),
	}
}

func main() {
	var err error
	settings, err := readConfig()
	if err != nil {
		panic(err)
	}
	config, err = ntura.ReadConfig("../config")
	if err != nil {
		panic(err)
	}
	if _, found := config.Alias["default"]; found {
		connStr := config.Alias[config.Alias["default"]]
		defConn = &driver.SQLDriver{}
		err = defConn.CreateConnection(config.Alias["default"], connStr, config)
		if err != nil {
			panic(err)
		}
	}

	e := echo.New()
	//e.Use(middleware.Logger())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `${time_rfc3339} | ${status} | ${latency_human} | ${remote_ip} | ${method} | ${uri}` + "\n",
		Output: os.Stdout,
	}))
	//e.Use(middleware.Recover())

	e.Use(middleware.CORSWithConfig(getCORSConfig(settings)))
	//secure := secure.New(getSecureOptions(settings))
	//e.Use(echo.WrapMiddleware(secure.Handler))

	signingKeys := ntura.IM{}
	if config.APIEndpoint != "" {
		signingKeys, err = getCertificates(config.APIEndpoint)
		if err != nil {
			panic(err)
		}
	}
	signingKeys[config.TokenKid] = config.TokenKey

	e.Static("/docs/*", "../../../nervatura-docs/docs")
	e.Static("/report/*", "../../../nervatura-demo/docs")
	e.Static("/client/*", "../../../nervatura-client/build")

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/docs/")
	})

	e.POST("/api/auth/login*", apiAuthLogin)
	e.POST("/api/auth/password*", apiAuthPassword, tokenAuth)
	e.GET("/api/auth/refresh*", apiAuthRefresh, tokenAuth)

	e.POST("/api/database*", apiDatabase)
	e.POST("/api/function*", apiFunction, tokenAuth)
	e.POST("/api/view*", apiView, tokenAuth)

	e.GET("/api/report*", report, tokenAuth)
	e.GET("/api/report/list*", reportList, tokenAuth)
	e.POST("/api/report/install*", reportInstall, tokenAuth)
	e.DELETE("/api/report/delete*", reportDelete, tokenAuth)

	e.GET("/api/:nervatype", apiGetFilter, tokenAuth)
	e.GET("/api/:nervatype/:ids", apiGetIds, tokenAuth)
	e.POST("/api/:nervatype", apiPost, tokenAuth)
	e.DELETE("/api/:nervatype", apiDelete, tokenAuth)

	e.POST("/npi/token/login*", npiTokenLogin)
	e.POST("/npi/token/", npi, tokenAuth)
	e.POST("/npi/", npiTokenLogin)
	e.POST("/npi", npiTokenLogin)

	hostname := settings.GetString("host")
	if settings.GetInt("port") > 0 {
		hostname += ":" + settings.GetString("port")
	}

	e.Logger.Fatal(e.Start(hostname))
}
