package service

import (
	"errors"
	"os"

	ntura "github.com/nervatura/nervatura-go/pkg/nervatura"
)

// CLIService implements the Nervatura API service
type CLIService struct {
	GetNervaStore func(database string) *ntura.NervaStore
}

func respondData(code int, data interface{}, errCode int, err error) string {
	if err != nil {
		data = ntura.IM{"code": errCode, "message": err.Error()}
	}
	if data == nil {
		data = ntura.IM{"code": code, "message": "OK"}
	}
	jdata, err := ntura.ConvertToByte(data)
	if err != nil {
		return `{"code":0,"error": "` + err.Error() + `"}`
	}
	return string(jdata)
}

func (srv *CLIService) TokenDecode(token string) string {
	claim, err := ntura.TokenDecode(token)
	return respondData(200, claim, 400, err)
}

func (srv *CLIService) TokenLogin(token string) (*ntura.API, string) {
	claim, err := ntura.TokenDecode(token)
	if err != nil {
		return nil, respondData(0, nil, 401, errors.New("Unauthorized"))
	}
	database := ntura.ToString(claim["database"], "")
	nstore := srv.GetNervaStore(database)
	if nstore == nil {
		return nil, respondData(0, nil, 401, errors.New("Unauthorized"))
	}
	api := (&ntura.API{NStore: nstore})
	err = api.TokenLogin(ntura.IM{"token": token})
	return api, respondData(200, api.NStore.User, 401, err)
}

func (srv *CLIService) UserLogin(options ntura.IM) string {
	if _, found := options["database"]; !found {
		return respondData(0, nil, 400, errors.New(ntura.GetMessage("missing_database")))
	}
	nstore := srv.GetNervaStore(options["database"].(string))
	token, engine, err := (&ntura.API{NStore: nstore}).UserLogin(options)
	return respondData(200, ntura.SM{"token": token, "engine": engine}, 400, err)
}

func (srv *CLIService) UserPassword(api *ntura.API, options ntura.IM) string {
	if _, found := options["username"]; found {
		if api.NStore.User.Scope != "admin" {
			return respondData(0, nil, 401, errors.New("Unauthorized"))
		}
	}
	if _, found := options["custnumber"]; found {
		if api.NStore.User.Scope != "admin" {
			return respondData(0, nil, 401, errors.New("Unauthorized"))
		}
	}
	if _, found := options["username"]; !found {
		if _, found := options["custnumber"]; !found {
			if api.NStore.Customer != nil {
				options["custnumber"] = api.NStore.Customer["custnumber"]
			} else {
				options["username"] = api.NStore.User.Username
			}
		}
	}
	err := api.UserPassword(options)
	return respondData(204, nil, 400, err)
}

func (srv *CLIService) TokenRefresh(api *ntura.API) string {
	token, err := api.TokenRefresh()
	return respondData(200, ntura.SM{"token": token}, 400, err)
}

func (srv *CLIService) Get(api *ntura.API, options ntura.IM) string {
	results, err := api.Get(options)
	return respondData(200, results, 400, err)
}

func (srv *CLIService) View(api *ntura.API, data []ntura.IM) string {
	results, err := api.View(data)
	return respondData(200, results, 400, err)
}

func (srv *CLIService) Function(api *ntura.API, options ntura.IM) string {
	results, err := api.Function(options)
	return respondData(200, results, 400, err)
}

func (srv *CLIService) Update(api *ntura.API, nervatype string, data []ntura.IM) string {
	results, err := api.Update(nervatype, data)
	return respondData(200, results, 400, err)
}

func (srv *CLIService) Delete(api *ntura.API, options ntura.IM) string {
	err := api.Delete(options)
	return respondData(204, nil, 400, err)
}

func (srv *CLIService) DatabaseCreate(apiKey string, options ntura.IM) string {
	if os.Getenv("NT_API_KEY") != apiKey {
		return respondData(0, nil, 401, errors.New("Unauthorized"))
	}
	log, err := (&ntura.API{NStore: srv.GetNervaStore("")}).DatabaseCreate(options)
	return respondData(200, log, 400, err)
}

func (srv *CLIService) ReportList(api *ntura.API, options ntura.IM) string {
	if api.NStore.User.Scope != "admin" {
		return respondData(0, nil, 401, errors.New("Unauthorized"))
	}
	results, err := api.ReportList(options)
	return respondData(200, results, 400, err)
}

func (srv *CLIService) ReportInstall(api *ntura.API, options ntura.IM) string {
	if api.NStore.User.Scope != "admin" {
		return respondData(0, nil, 401, errors.New("Unauthorized"))
	}
	results, err := api.ReportInstall(options)
	return respondData(200, results, 400, err)
}

func (srv *CLIService) ReportDelete(api *ntura.API, options ntura.IM) string {
	if api.NStore.User.Scope != "admin" {
		return respondData(0, nil, 401, errors.New("Unauthorized"))
	}
	err := api.ReportDelete(options)
	return respondData(204, nil, 400, err)
}

func (srv *CLIService) Report(api *ntura.API, options ntura.IM) string {
	if _, found := options["output"]; !found || (options["output"] != "xml") {
		options["output"] = "base64"
	}
	results, err := api.Report(options)
	return respondData(200, results, 400, err)
}
