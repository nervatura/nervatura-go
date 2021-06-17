//+build http all

package service

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	db "github.com/nervatura/nervatura-service/pkg/database"
	nt "github.com/nervatura/nervatura-service/pkg/nervatura"
	ut "github.com/nervatura/nervatura-service/pkg/utils"
)

// HTTPService implements the Nervatura API service
type HTTPService struct {
	Config        map[string]interface{}
	GetNervaStore func(database string) *nt.NervaStore
	GetParam      func(req *http.Request, name string) string
	GetTokenKeys  func() map[string]map[string]string
}

const contentKey = "Content-Type"

// respondMessage write json response format
func (srv *HTTPService) respondMessage(w http.ResponseWriter, code int, payload interface{}, errCode int, err error) {
	var response []byte
	var jerr error
	if err != nil || payload != nil {
		w.Header().Set(contentKey, "application/json")
		if err != nil {
			w.WriteHeader(errCode)
			response, jerr = ut.ConvertToByte(nt.SM{"code": strconv.Itoa(errCode), "message": err.Error()})
		} else {
			w.WriteHeader(code)
			response, jerr = ut.ConvertToByte(payload)
		}
		if jerr == nil {
			w.Write(response)
		}
	} else {
		w.WriteHeader(code)
	}
}

func (srv *HTTPService) ClientConfig(w http.ResponseWriter, r *http.Request) {
	config := nt.IM{}
	if srv.Config["NT_CLIENT_CONFIG"] != "" {
		if content, err := ioutil.ReadFile(srv.Config["NT_CLIENT_CONFIG"].(string)); err == nil {
			_ = ut.ConvertFromByte(content, &config)
		}
	}
	srv.respondMessage(w, http.StatusOK, config, http.StatusBadRequest, nil)
}

func (srv *HTTPService) TokenLogin(w http.ResponseWriter, r *http.Request) (ctx context.Context, err error) {
	tokenStr := ""
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER" {
		tokenStr = bearer[7:]
	}
	if tokenStr == "" {
		return ctx, errors.New(ut.GetMessage("error_unauthorized"))
	}
	claim, err := ut.TokenDecode(tokenStr)
	if err != nil {
		return ctx, err
	}
	database := ut.ToString(claim["database"], "")
	nstore := srv.GetNervaStore(database)
	if nstore == nil {
		return ctx, errors.New(ut.GetMessage("error_unauthorized"))
	}
	err = (&nt.API{NStore: nstore}).TokenLogin(nt.IM{"token": tokenStr, "keys": srv.GetTokenKeys()})
	if err != nil {
		return ctx, err
	}
	ctx = context.WithValue(r.Context(), NstoreCtxKey, nstore)
	return ctx, nil
}

func (srv *HTTPService) UserLogin(w http.ResponseWriter, r *http.Request) {
	data := nt.IM{}
	err := ut.ConvertFromReader(r.Body, &data)
	if err != nil {
		srv.respondMessage(w, 0, nil, http.StatusBadRequest, err)
	}
	if _, found := data["database"]; !found {
		srv.respondMessage(w, 0, nil, http.StatusBadRequest, errors.New(ut.GetMessage("missing_database")))
	}
	nstore := srv.GetNervaStore(data["database"].(string))
	if nstore == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}
	token, engine, err := (&nt.API{NStore: nstore}).UserLogin(data)
	srv.respondMessage(w, http.StatusOK, nt.SM{"token": token, "engine": engine, "version": srv.Config["version"].(string)}, http.StatusBadRequest, err)
}

func (srv *HTTPService) UserPassword(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}
	nstore := r.Context().Value(NstoreCtxKey).(*nt.NervaStore)

	data := nt.IM{}
	err := ut.ConvertFromReader(r.Body, &data)
	if err != nil {
		srv.respondMessage(w, 0, nil, http.StatusBadRequest, err)
		return
	}
	if _, found := data["username"]; found {
		if nstore.User.Scope != "admin" {
			srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
			return
		}
	}
	if _, found := data["custnumber"]; found {
		if nstore.User.Scope != "admin" {
			srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
			return
		}
	}
	if _, found := data["username"]; !found {
		if _, found := data["custnumber"]; !found {
			if nstore.Customer != nil {
				data["custnumber"] = nstore.Customer["custnumber"]
			} else {
				data["username"] = nstore.User.Username
			}
		}
	}
	err = (&nt.API{NStore: nstore}).UserPassword(data)
	srv.respondMessage(w, http.StatusNoContent, nil, http.StatusBadRequest, err)
}

func (srv *HTTPService) TokenRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}
	tokenStr, err := (&nt.API{NStore: r.Context().Value(NstoreCtxKey).(*nt.NervaStore)}).TokenRefresh()
	srv.respondMessage(w, http.StatusOK, nt.SM{"token": tokenStr}, http.StatusBadRequest, err)
}

func (srv *HTTPService) GetFilter(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}

	params := nt.IM{"nervatype": srv.GetParam(r, "nervatype"),
		"metadata": r.URL.Query().Get("metadata")}
	query := strings.Split(r.URL.RawQuery, "&")
	for index := 0; index < len(query); index++ {
		if strings.HasPrefix(query[index], "filter=") {
			params["filter"] = query[index][7:]
		}
	}
	results, err := (&nt.API{NStore: r.Context().Value(NstoreCtxKey).(*nt.NervaStore)}).Get(params)
	srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
}

func (srv *HTTPService) GetIds(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}

	params := nt.IM{"nervatype": srv.GetParam(r, "nervatype"),
		"metadata": r.URL.Query().Get("metadata"), "ids": srv.GetParam(r, "IDs")}
	results, err := (&nt.API{NStore: r.Context().Value(NstoreCtxKey).(*nt.NervaStore)}).Get(params)
	srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
}

func (srv *HTTPService) View(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}

	data := make([]nt.IM, 0)
	err := ut.ConvertFromReader(r.Body, &data)
	if err != nil {
		srv.respondMessage(w, 0, nil, http.StatusBadRequest, err)
		return
	}
	results, err := (&nt.API{NStore: r.Context().Value(NstoreCtxKey).(*nt.NervaStore)}).View(data)
	srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
}

func (srv *HTTPService) Function(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}

	data := nt.IM{}
	err := ut.ConvertFromReader(r.Body, &data)
	if err != nil {
		srv.respondMessage(w, 0, nil, http.StatusBadRequest, err)
		return
	}
	results, err := (&nt.API{NStore: r.Context().Value(NstoreCtxKey).(*nt.NervaStore)}).Function(data)
	srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
}

func (srv *HTTPService) Update(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}

	data := make([]nt.IM, 0)
	err := ut.ConvertFromReader(r.Body, &data)
	if err != nil {
		srv.respondMessage(w, 0, nil, http.StatusBadRequest, err)
		return
	}
	results, err := (&nt.API{NStore: r.Context().Value(NstoreCtxKey).(*nt.NervaStore)}).Update(srv.GetParam(r, "nervatype"), data)
	srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
}

func (srv *HTTPService) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}

	data := nt.IM{"nervatype": srv.GetParam(r, "nervatype"),
		"id": r.URL.Query().Get("id"), "key": r.URL.Query().Get("key")}
	err := (&nt.API{NStore: r.Context().Value(NstoreCtxKey).(*nt.NervaStore)}).Delete(data)
	srv.respondMessage(w, http.StatusNoContent, nil, http.StatusBadRequest, err)
}

func (srv *HTTPService) DatabaseCreate(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-Api-Key")
	if srv.Config["NT_API_KEY"] != apiKey {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}
	data := nt.IM{"database": r.URL.Query().Get("alias"), "demo": r.URL.Query().Get("demo")}
	log, err := (&nt.API{NStore: nt.New(&db.SQLDriver{Config: srv.Config}, srv.Config)}).DatabaseCreate(data)
	srv.respondMessage(w, http.StatusOK, log, http.StatusBadRequest, err)
}

func (srv *HTTPService) ReportList(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}

	params := nt.IM{"label": r.URL.Query().Get("label")}
	api := &nt.API{NStore: r.Context().Value(NstoreCtxKey).(*nt.NervaStore)}
	if api.NStore.User.Scope != "admin" {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}
	results, err := api.ReportList(params)
	srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
}

func (srv *HTTPService) ReportInstall(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}

	params := nt.IM{"reportkey": r.URL.Query().Get("reportkey")}
	api := &nt.API{NStore: r.Context().Value(NstoreCtxKey).(*nt.NervaStore)}
	if api.NStore.User.Scope != "admin" {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}
	results, err := api.ReportInstall(params)
	srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
}

func (srv *HTTPService) ReportDelete(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}

	params := nt.IM{"reportkey": r.URL.Query().Get("reportkey")}
	api := &nt.API{NStore: r.Context().Value(NstoreCtxKey).(*nt.NervaStore)}
	if api.NStore.User.Scope != "admin" {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}
	err := api.ReportDelete(params)
	srv.respondMessage(w, http.StatusNoContent, nil, http.StatusBadRequest, err)
}

func reportQueryFilters(values url.Values) nt.IM {
	options := nt.IM{"filters": nt.IM{}}
	for key, value := range values {
		if strings.HasPrefix(key, "filters[") {
			fkey := key[8 : len(key)-1]
			options["filters"].(nt.IM)[fkey] = value[0]
		} else {
			switch key {
			case "report_id":
				reportID, err := strconv.ParseInt(value[0], 10, 64)
				if err == nil {
					options["report_id"] = reportID
				}
			case "output":
				options["output"] = value[0]
				if value[0] == "data" {
					options["output"] = "tmp"
				}
			default:
				options[key] = value[0]
			}
		}
	}
	return options
}

func reportFilters(values url.Values, body io.ReadCloser) nt.IM {
	if len(values) > 0 {
		return reportQueryFilters(values)
	}
	data := nt.IM{}
	_ = ut.ConvertFromReader(body, &data)
	return data
}

func (srv *HTTPService) Report(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New(ut.GetMessage("error_unauthorized")))
		return
	}

	options := reportFilters(r.URL.Query(), r.Body)

	results, err := (&nt.API{NStore: r.Context().Value(NstoreCtxKey).(*nt.NervaStore)}).Report(options)
	if err != nil {
		srv.respondMessage(w, 0, nil, http.StatusBadRequest, err)
		return
	}
	if options["output"] == "tmp" {
		srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
		return
	}
	if results["filetype"] == "csv" {
		w.Header().Set(contentKey, "text/csv")
		w.Write([]byte(results["template"].(string)))
		return
	}
	if results["filetype"] == "xml" {
		w.Header().Set(contentKey, "application/xml")
		w.Write([]byte(results["template"].(string)))
		return
	}
	w.Header().Set(contentKey, "application/pdf")
	w.Write(results["template"].([]uint8))
}
