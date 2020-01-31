package app

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	ntura "github.com/nervatura/nervatura-go"
	driver "github.com/nervatura/nervatura-go/driver"
	"github.com/spf13/viper"
	"github.com/unrolled/secure"
)

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

func (app *App) getNStore(options ntura.IM) *ntura.NervaStore {
	var database string
	if _, found := options["database"]; found {
		database = options["database"].(string)
	}
	if app.defConn != nil {
		if app.defConn.Connection().Alias == database {
			return ntura.New(app.config, app.defConn)
		}
	}
	if database != "" {
		return ntura.New(app.config, &driver.SQLDriver{})
	}
	if app.defConn == nil {
		return nil
	}
	return ntura.New(app.config, app.defConn)
}

func (app *App) tokenAuth(next echo.HandlerFunc) echo.HandlerFunc {
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
		nstore := app.getNStore(claim)
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
