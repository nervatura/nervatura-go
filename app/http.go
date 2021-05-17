//+build http all

package app

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	srv "github.com/nervatura/nervatura-go/pkg/service"
	ut "github.com/nervatura/nervatura-go/pkg/utils"
	"github.com/unrolled/secure"
)

type httpServer struct {
	app        *App
	mux        *chi.Mux
	service    srv.HTTPService
	admin      srv.AdminService
	client     srv.ClientService
	result     string
	server     *http.Server
	tlsEnabled bool
}

func init() {
	registerService("http", &httpServer{})
}

func (s *httpServer) StartService() error {
	s.mux = chi.NewRouter()
	s.service = srv.HTTPService{
		GetNervaStore: s.app.GetNervaStore,
		GetParam: func(req *http.Request, name string) string {
			return chi.URLParam(req, name)
		},
		GetTokenKeys: func() map[string]map[string]string {
			return s.app.tokenKeys
		},
	}

	s.admin = srv.AdminService{
		GetNervaStore: s.app.GetNervaStore,
		GetTokenKeys: func() map[string]map[string]string {
			return s.app.tokenKeys
		},
	}
	err := s.admin.LoadTemplates()
	if err != nil {
		return err
	}

	/*
		s.client = srv.ClientService{
			Path: "client",
		}
		s.client.LoadManifest()
		if err != nil {
			return err
		}
	*/

	s.setPublicKeys()
	s.setMiddleware()
	s.setRoutes()

	// Start API server.
	return s.startServer()
}

func (s *httpServer) setPublicKeys() {
	publicUrl := os.Getenv("NT_TOKEN_PUBLIC_KEY_URL")
	ktype := ut.ToString(os.Getenv("NT_TOKEN_PUBLIC_KEY_TYPE"), "KEY")
	if publicUrl != "" {
		res, err := http.Get(publicUrl)
		if err != nil {
			s.app.errorLog.Printf("external token loading: %v\n", err)
			return
		}
		defer res.Body.Close()
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			s.app.errorLog.Printf("external token loading: %v\n", err)
			return
		}
		var tokenKeys map[string]string
		err = ut.ConvertFromByte(data, &tokenKeys)
		if err != nil {
			s.app.errorLog.Printf("external token loading: %v\n", err)
		}
		for key, value := range tokenKeys {
			s.app.tokenKeys[key] = map[string]string{
				"type":  "public",
				"ktype": ktype,
				"value": value,
			}
		}
	}
}

func (s *httpServer) startServer() error {
	s.server = &http.Server{
		Handler:      s.mux,
		Addr:         fmt.Sprintf(":%d", ut.GetEnvValue("int", os.Getenv("NT_HTTP_PORT")).(int)),
		ReadTimeout:  ut.GetEnvValue("duration", os.Getenv("NT_HTTP_READ_TIMEOUT")).(time.Duration) * time.Second,
		WriteTimeout: ut.GetEnvValue("duration", os.Getenv("NT_HTTP_WRITE_TIMEOUT")).(time.Duration) * time.Second,
	}
	s.tlsEnabled = ut.GetEnvValue("bool", os.Getenv("NT_HTTP_TLS_ENABLED")).(bool) &&
		os.Getenv("NT_TLS_CERT_FILE") != "" && os.Getenv("NT_TLS_KEY_FILE") != ""

	s.app.infoLog.Printf("HTTP server serving at: %s. SSL/TLS authentication: %v.\n",
		os.Getenv("NT_HTTP_PORT"), s.tlsEnabled)
	if s.tlsEnabled {
		if err := s.server.ListenAndServeTLS(os.Getenv("NT_TLS_CERT_FILE"), os.Getenv("NT_TLS_KEY_FILE")); err != http.ErrServerClosed {
			return err
		}
	} else {
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			return err
		}
	}

	return nil
}

func (s *httpServer) StopService(ctx interface{}) error {
	if s.server != nil {
		s.app.infoLog.Println("stopping HTTP server")
		return s.server.Shutdown(ctx.(context.Context))
	}
	return nil
}

func (s *httpServer) Results() string {
	return s.result
}

func (s *httpServer) ConnectApp(app interface{}) {
	s.app = app.(*App)
}

// Register middleware.
func (s *httpServer) setMiddleware() {

	s.mux.Use(middleware.Logger)
	s.mux.Use(middleware.RequestID)
	s.mux.Use(middleware.Recoverer)

	s.mux.Use(middleware.CleanPath)
	s.mux.Use(middleware.StripSlashes)

	if ut.GetEnvValue("bool", os.Getenv("NT_CORS_ENABLED")).(bool) {
		s.mux.Use(cors.Handler(cors.Options{
			AllowedOrigins:   ut.GetEnvValue("slice", os.Getenv("NT_CORS_ALLOW_ORIGINS")).([]string),
			AllowedMethods:   ut.GetEnvValue("slice", os.Getenv("NT_CORS_ALLOW_METHODS")).([]string),
			AllowedHeaders:   ut.GetEnvValue("slice", os.Getenv("NT_CORS_ALLOW_HEADERS")).([]string),
			ExposedHeaders:   ut.GetEnvValue("slice", os.Getenv("NT_CORS_EXPOSE_HEADERS")).([]string),
			AllowCredentials: ut.GetEnvValue("bool", os.Getenv("NT_CORS_ALLOW_CREDENTIALS")).(bool),
			MaxAge:           ut.GetEnvValue("int", os.Getenv("NT_CORS_MAX_AGE")).(int),
		}))
	}

	if ut.GetEnvValue("bool", os.Getenv("NT_SECURITY_ENABLED")).(bool) {
		s.mux.Use(secure.New(secure.Options{
			AllowedHosts:            ut.GetEnvValue("slice", os.Getenv("NT_SECURITY_ALLOWED_HOSTS")).([]string),
			AllowedHostsAreRegex:    ut.GetEnvValue("bool", os.Getenv("NT_SECURITY_ALLOWED_HOSTS_ARE_REGEX")).(bool),
			HostsProxyHeaders:       ut.GetEnvValue("slice", os.Getenv("NT_SECURITY_HOSTS_PROXY_HEADERS")).([]string),
			SSLRedirect:             ut.GetEnvValue("bool", os.Getenv("NT_SECURITY_SSL_REDIRECT")).(bool),
			SSLTemporaryRedirect:    ut.GetEnvValue("bool", os.Getenv("NT_SECURITY_SSL_TEMPORARY_REDIRECT")).(bool),
			SSLHost:                 ut.GetEnvValue("slice", os.Getenv("NT_SECURITY_SSL_HOST")).(string),
			STSSeconds:              ut.GetEnvValue("int64", os.Getenv("NT_SECURITY_STS_SECONDS")).(int64),
			STSIncludeSubdomains:    ut.GetEnvValue("bool", os.Getenv("NT_SECURITY_STS_INCLUDE_SUBDOMAINS")).(bool),
			STSPreload:              ut.GetEnvValue("bool", os.Getenv("NT_SECURITY_STS_PRELOAD")).(bool),
			ForceSTSHeader:          ut.GetEnvValue("bool", os.Getenv("NT_SECURITY_FORCE_STS_HEADER")).(bool),
			FrameDeny:               ut.GetEnvValue("bool", os.Getenv("NT_SECURITY_FRAME_DENY")).(bool),
			CustomFrameOptionsValue: ut.GetEnvValue("slice", os.Getenv("NT_SECURITY_CUSTOM_FRAME_OPTIONS_VALUE")).(string),
			ContentTypeNosniff:      ut.GetEnvValue("bool", os.Getenv("NT_SECURITY_CONTENT_TYPE_NOSNIFF")).(bool),
			BrowserXssFilter:        ut.GetEnvValue("bool", os.Getenv("NT_SECURITY_BROWSER_XSS_FILTER")).(bool),
			ContentSecurityPolicy:   ut.GetEnvValue("slice", os.Getenv("NT_SECURITY_CONTENT_SECURITY_POLICY")).(string),
			PublicKey:               ut.GetEnvValue("slice", os.Getenv("NT_SECURITY_PUBLIC_KEY")).(string),
			ReferrerPolicy:          ut.GetEnvValue("slice", os.Getenv("NT_SECURITY_REFERRER_POLICY")).(string),
			FeaturePolicy:           ut.GetEnvValue("slice", os.Getenv("NT_SECURITY_FEATURE_POLICY")).(string),
			ExpectCTHeader:          ut.GetEnvValue("slice", os.Getenv("NT_SECURITY_EXPECT_CT_HEADER")).(string),
			IsDevelopment:           ut.GetEnvValue("bool", os.Getenv("NT_SECURITY_DEVELOPMENT")).(bool),
		}).Handler)
	}

}

func (s *httpServer) tokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := s.service.TokenLogin(w, r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Register API routes.
func (s *httpServer) setRoutes() {
	// Register static dirs.
	var publicFS, _ = fs.Sub(ut.Public, "static")
	FileServer(s.mux, "/", http.FS(publicFS))

	s.mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		home := ut.ToString(os.Getenv("NT_HTTP_HOME"), "/")
		if home != "/" {
			http.Redirect(w, r, home, http.StatusSeeOther)
		}
	})

	//s.mux.Get("/"+s.client.Path, s.client.Render)

	s.mux.Route("/admin", func(r chi.Router) {
		r.Get("/", s.admin.Home)
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			switch r.PostFormValue("formID") {
			case "database":
				s.admin.Database(w, r)
			case "menu":
				if r.PostFormValue("menu") == "client" {
					http.Redirect(w, r, "/client", http.StatusSeeOther)
					return
				}
				if r.PostFormValue("menu") == "docs" {
					http.Redirect(w, r, "https://nervatura.github.io/nervatura/", http.StatusSeeOther)
					return
				}
				s.admin.Menu(w, r)
			case "admin":
				s.admin.Admin(w, r)
			default:
				s.admin.Login(w, r)
			}
		})
	})
	s.mux.Route("/api", func(r chi.Router) {
		r.Post("/database", s.service.DatabaseCreate)
		r.Get("/config", s.service.Config)
		r.Group(func(r chi.Router) {
			r.Use(s.tokenAuth)
			r.Post("/function", s.service.Function)
			r.Post("/view", s.service.View)
		})

		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", s.service.UserLogin)
			r.Group(func(r chi.Router) {
				r.Use(s.tokenAuth)
				r.Post("/password", s.service.UserPassword)
				r.Get("/refresh", s.service.TokenRefresh)
			})
		})

		r.Route("/{nervatype}", func(r chi.Router) {
			r.Use(s.tokenAuth)
			r.Get("/", s.service.GetFilter)
			r.Get("/{IDs}", s.service.GetIds)
			r.Post("/", s.service.Update)
			r.Delete("/", s.service.Delete)
		})

		r.Route("/report", func(r chi.Router) {
			r.Use(s.tokenAuth)
			r.Get("/", s.service.Report)
			r.Post("/", s.service.Report)
			r.Get("/list", s.service.ReportList)
			r.Post("/install", s.service.ReportInstall)
			r.Delete("/delete", s.service.ReportDelete)
		})

	})

}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {

	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
