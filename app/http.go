//+build http all

package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	ntura "github.com/nervatura/nervatura-go/pkg/nervatura"
	srv "github.com/nervatura/nervatura-go/pkg/service"
	"github.com/unrolled/secure"
)

type httpServer struct {
	app        *App
	mux        *chi.Mux
	service    srv.HTTPService
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
	}

	s.setMiddleware()
	s.setStatic()
	s.setRoutes()

	// Start API server.
	//return s.startServerWithGracefulShutdown()
	return s.startServer()
}

func (s *httpServer) startServer() error {
	s.server = &http.Server{
		Handler:      s.mux,
		Addr:         fmt.Sprintf(":%d", ntura.GetEnvValue("int", os.Getenv("NT_HTTP_PORT")).(int)),
		ReadTimeout:  ntura.GetEnvValue("duration", os.Getenv("NT_HTTP_READ_TIMEOUT")).(time.Duration) * time.Second,
		WriteTimeout: ntura.GetEnvValue("duration", os.Getenv("NT_HTTP_WRITE_TIMEOUT")).(time.Duration) * time.Second,
	}
	s.tlsEnabled = ntura.GetEnvValue("bool", os.Getenv("NT_HTTP_TLS_ENABLED")).(bool) &&
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

	if ntura.GetEnvValue("bool", os.Getenv("NT_CORS_ENABLED")).(bool) {
		s.mux.Use(cors.Handler(cors.Options{
			AllowedOrigins:   ntura.GetEnvValue("slice", os.Getenv("NT_CORS_ALLOW_ORIGINS")).([]string),
			AllowedMethods:   ntura.GetEnvValue("slice", os.Getenv("NT_CORS_ALLOW_METHODS")).([]string),
			AllowedHeaders:   ntura.GetEnvValue("slice", os.Getenv("NT_CORS_ALLOW_HEADERS")).([]string),
			ExposedHeaders:   ntura.GetEnvValue("slice", os.Getenv("NT_CORS_EXPOSE_HEADERS")).([]string),
			AllowCredentials: ntura.GetEnvValue("bool", os.Getenv("NT_CORS_ALLOW_CREDENTIALS")).(bool),
			MaxAge:           ntura.GetEnvValue("int", os.Getenv("NT_CORS_MAX_AGE")).(int),
		}))
	}

	if ntura.GetEnvValue("bool", os.Getenv("NT_SECURITY_ENABLED")).(bool) {
		s.mux.Use(secure.New(secure.Options{
			AllowedHosts:            ntura.GetEnvValue("slice", os.Getenv("NT_SECURITY_ALLOWED_HOSTS")).([]string),
			AllowedHostsAreRegex:    ntura.GetEnvValue("bool", os.Getenv("NT_SECURITY_ALLOWED_HOSTS_ARE_REGEX")).(bool),
			HostsProxyHeaders:       ntura.GetEnvValue("slice", os.Getenv("NT_SECURITY_HOSTS_PROXY_HEADERS")).([]string),
			SSLRedirect:             ntura.GetEnvValue("bool", os.Getenv("NT_SECURITY_SSL_REDIRECT")).(bool),
			SSLTemporaryRedirect:    ntura.GetEnvValue("bool", os.Getenv("NT_SECURITY_SSL_TEMPORARY_REDIRECT")).(bool),
			SSLHost:                 ntura.GetEnvValue("slice", os.Getenv("NT_SECURITY_SSL_HOST")).(string),
			STSSeconds:              ntura.GetEnvValue("int64", os.Getenv("NT_SECURITY_STS_SECONDS")).(int64),
			STSIncludeSubdomains:    ntura.GetEnvValue("bool", os.Getenv("NT_SECURITY_STS_INCLUDE_SUBDOMAINS")).(bool),
			STSPreload:              ntura.GetEnvValue("bool", os.Getenv("NT_SECURITY_STS_PRELOAD")).(bool),
			ForceSTSHeader:          ntura.GetEnvValue("bool", os.Getenv("NT_SECURITY_FORCE_STS_HEADER")).(bool),
			FrameDeny:               ntura.GetEnvValue("bool", os.Getenv("NT_SECURITY_FRAME_DENY")).(bool),
			CustomFrameOptionsValue: ntura.GetEnvValue("slice", os.Getenv("NT_SECURITY_CUSTOM_FRAME_OPTIONS_VALUE")).(string),
			ContentTypeNosniff:      ntura.GetEnvValue("bool", os.Getenv("NT_SECURITY_CONTENT_TYPE_NOSNIFF")).(bool),
			BrowserXssFilter:        ntura.GetEnvValue("bool", os.Getenv("NT_SECURITY_BROWSER_XSS_FILTER")).(bool),
			ContentSecurityPolicy:   ntura.GetEnvValue("slice", os.Getenv("NT_SECURITY_CONTENT_SECURITY_POLICY")).(string),
			PublicKey:               ntura.GetEnvValue("slice", os.Getenv("NT_SECURITY_PUBLIC_KEY")).(string),
			ReferrerPolicy:          ntura.GetEnvValue("slice", os.Getenv("NT_SECURITY_REFERRER_POLICY")).(string),
			FeaturePolicy:           ntura.GetEnvValue("slice", os.Getenv("NT_SECURITY_FEATURE_POLICY")).(string),
			ExpectCTHeader:          ntura.GetEnvValue("slice", os.Getenv("NT_SECURITY_EXPECT_CT_HEADER")).(string),
			IsDevelopment:           ntura.GetEnvValue("bool", os.Getenv("NT_SECURITY_DEVELOPMENT")).(bool),
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

// Register static dirs.
func (s *httpServer) setStatic() {

	/*
		workDir, _ := os.Getwd()
		clientDir := http.Dir(filepath.Join(workDir, "..", "nervatura-client", "build"))
		FileServer(app.mux, "/", clientDir)
	*/
	s.mux.Get("/web", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://nervatura.github.io/nervatura-client/",
			http.StatusTemporaryRedirect)
	})

}

// Register API routes.
func (s *httpServer) setRoutes() {

	s.mux.Route("/api", func(r chi.Router) {

		r.Post("/database", s.service.DatabaseCreate)
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
