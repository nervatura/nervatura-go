package app

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	ntura "github.com/nervatura/nervatura-go/pkg/nervatura"
	srv "github.com/nervatura/nervatura-go/pkg/service"
)

type cliServer struct {
	app     *App
	service srv.CLIService
	args    ntura.SM
	result  string
}

func init() {
	registerService("cli", &cliServer{})
}

// StartService - Start Nervatura CLI server
func (s *cliServer) StartService() error {
	s.service = srv.CLIService{
		GetNervaStore: s.app.GetNervaStore,
	}
	err := s.parseFlags()
	if err != nil {
		flag.Usage()
		return err
	}

	if s.args["cmd"] == "server" {
		s.result = "server"
		return nil
	}

	err = s.checkRequired()
	if err != nil {
		flag.Usage()
		return err
	}

	s.result, err = s.parseCommand()
	if err != nil {
		return err
	}
	fmt.Println(s.result)
	return nil
}

func (s *cliServer) Results() string {
	return s.result
}

func (s *cliServer) ConnectApp(app interface{}) {
	s.app = app.(*App)
}

func (s *cliServer) StopService(interface{}) error {
	return nil
}

func (s *cliServer) parseFlags() (err error) {
	s.args = make(ntura.SM)
	var cmds = []string{"server", "Delete", "Function", "Get", "Update", "View",
		"UserPassword", "TokenLogin", "TokenRefresh", "UserLogin", "DatabaseCreate",
		"Report", "ReportDelete", "ReportInstall", "ReportList", "TokenDecode"}

	var help bool
	flag.BoolVar(&help, "help", false, "Program usage")
	var cmd string
	flag.StringVar(&cmd, "c", "server", `Available commands:
`+strings.Join(cmds[:10], ", ")+`,
`+strings.Join(cmds[10:], ", "))
	var token string
	flag.StringVar(&token, "t", "",
		`Bearer token parameter. Required for the following command: Delete, 
		 Function, Get, Update, View, UserPassword, TokenLogin, TokenRefresh,
		 Report, ReportDelete, ReportInstall, ReportList, TokenDecode`)
	var options string
	flag.StringVar(&options, "o", "",
		`Options: JSON Object string. Required for the following commands:
Delete, Function, Get, Update, UserPassword, UserLogin,
DatabaseCreate, Report, ReportDelete, ReportInstall, ReportList`)
	var ntype string
	flag.StringVar(&ntype, "nt", "", "Command nervatype parameter. Required for the following command: Update")
	var data string
	flag.StringVar(&data, "d", "",
		`Data: JSON Array string. Required for the following commands: Update, View`)
	var key string
	flag.StringVar(&key, "k", "",
		`API key. Required for the following commands: DatabaseCreate`)

	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()

	s.args["cmd"] = cmd
	if help {
		s.args["cmd"] = "help"
	}
	if token != "" {
		s.args["token"] = token
	}
	if options != "" {
		s.args["options"] = options
	}
	if data != "" {
		s.args["data"] = data
	}
	if ntype != "" {
		s.args["nervatype"] = ntype
	}
	if key != "" {
		s.args["key"] = key
	}

	if len(flag.Args()) > 1 {
		return errors.New("invalid parameters: " + strings.Join(flag.Args(), ","))
	}

	return nil
}

func (s *cliServer) checkRequired() (err error) {
	if _, found := s.args["token"]; !found && s.args["cmd"] != "UserLogin" && s.args["cmd"] != "DatabaseCreate" {
		return errors.New("missing required parameter: token(-t)")
	}
	switch s.args["cmd"] {
	case "Delete", "Function", "Get", "UserPassword",
		"UserLogin", "Report", "ReportDelete", "ReportInstall", "ReportList":
		if _, found := s.args["options"]; !found {
			return errors.New("missing required parameter: options(-o)")
		}

	case "DatabaseCreate":
		if _, found := s.args["options"]; !found {
			return errors.New("missing required parameter: options(-o)")
		}
		if _, found := s.args["key"]; !found {
			return errors.New("missing required parameter: API key(-k)")
		}

	case "View":
		if _, found := s.args["data"]; !found {
			return errors.New("missing required parameter: data(-d)")
		}

	case "Update":
		if _, found := s.args["nervatype"]; !found {
			return errors.New("missing required parameter: nervatype(-nt)")
		}
		if _, found := s.args["data"]; !found {
			return errors.New("missing required parameter: data(-d)")
		}

	}
	return nil
}

// parseCommand - Parse s.args from command line parameters
func (s *cliServer) parseCommand() (result string, err error) {
	if s.args["cmd"] == "help" {
		flag.Usage()
		return "", nil
	}

	var api *ntura.API
	if _, found := s.args["token"]; found {
		if s.args["cmd"] == "TokenDecode" {
			return s.service.TokenDecode(s.args["token"]), nil
		}
		api, result = s.service.TokenLogin(s.args["token"])
		if api == nil || s.args["cmd"] == "TokenLogin" {
			return result, nil
		}
	}

	var options ntura.IM
	if _, found := s.args["options"]; found {
		if ntura.ConvertFromByte([]byte(s.args["options"]), &options); err != nil {
			return "", errors.New("invalid options json")
		}
	}

	var data []ntura.IM
	if _, found := s.args["data"]; found {
		if ntura.ConvertFromByte([]byte(s.args["data"]), &data); err != nil {
			return "", errors.New("invalid data json")
		}
	}

	apiMap := map[string]func(api *ntura.API) string{
		"UserLogin": func(api *ntura.API) string {
			return s.service.UserLogin(options)
		},
		"UserPassword": func(api *ntura.API) string {
			return s.service.UserPassword(api, options)
		},
		"TokenRefresh": func(api *ntura.API) string {
			return s.service.TokenRefresh(api)
		},
		"Get": func(api *ntura.API) string {
			return s.service.Get(api, options)
		},
		"View": func(api *ntura.API) string {
			return s.service.View(api, data)
		},
		"Function": func(api *ntura.API) string {
			return s.service.Function(api, options)
		},
		"Update": func(api *ntura.API) string {
			return s.service.Update(api, s.args["nervatype"], data)
		},
		"Delete": func(api *ntura.API) string {
			return s.service.Delete(api, options)
		},
		"DatabaseCreate": func(api *ntura.API) string {
			return s.service.DatabaseCreate(s.args["key"], options)
		},
		"Report": func(api *ntura.API) string {
			return s.service.Report(api, options)
		},
		"ReportList": func(api *ntura.API) string {
			return s.service.ReportList(api, options)
		},
		"ReportInstall": func(api *ntura.API) string {
			return s.service.ReportInstall(api, options)
		},
		"ReportDelete": func(api *ntura.API) string {
			return s.service.ReportDelete(api, options)
		},
	}
	if _, found := apiMap[s.args["cmd"]]; !found {
		return "", errors.New("Invalid program command: " + s.args["cmd"] + " (-c)")
	}
	return apiMap[s.args["cmd"]](api), nil
}
