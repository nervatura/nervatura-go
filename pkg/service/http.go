//+build http all

package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	db "github.com/nervatura/nervatura-go/pkg/database"
	ntura "github.com/nervatura/nervatura-go/pkg/nervatura"
)

// HTTPService implements the Nervatura API service
type HTTPService struct {
	GetNervaStore func(database string) *ntura.NervaStore
	GetParam      func(req *http.Request, name string) string
}

// respondMessage write json response format
func (srv *HTTPService) respondMessage(w http.ResponseWriter, code int, payload interface{}, errCode int, err error) {
	var response []byte
	var jerr error
	if err != nil || payload != nil {
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(errCode)
			response, jerr = ntura.ConvertToByte(ntura.SM{"code": strconv.Itoa(errCode), "message": err.Error()})
		} else {
			w.WriteHeader(code)
			response, jerr = ntura.ConvertToByte(payload)
		}
		if jerr == nil {
			w.Write(response)
		}
	} else {
		w.WriteHeader(code)
	}
}

func (srv *HTTPService) TokenLogin(w http.ResponseWriter, r *http.Request) (ctx context.Context, err error) {
	tokenStr := ""
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER" {
		tokenStr = bearer[7:]
	}
	if tokenStr == "" {
		return ctx, errors.New("Unauthorized")
	}
	claim, err := ntura.TokenDecode(tokenStr)
	if err != nil {
		return ctx, err
	}
	database := ntura.ToString(claim["database"], "")
	nstore := srv.GetNervaStore(database)
	if nstore == nil {
		return ctx, errors.New("Unauthorized")
	}
	err = (&ntura.API{NStore: nstore}).TokenLogin(ntura.IM{"token": tokenStr})
	if err != nil {
		return ctx, err
	}
	ctx = context.WithValue(r.Context(), NstoreCtxKey, nstore)
	return ctx, nil
}

func (srv *HTTPService) UserLogin(w http.ResponseWriter, r *http.Request) {
	data := ntura.IM{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		srv.respondMessage(w, 0, nil, http.StatusBadRequest, err)
	}
	if _, found := data["database"]; !found {
		srv.respondMessage(w, 0, nil, http.StatusBadRequest, errors.New(ntura.GetMessage("missing_database")))
	}
	nstore := srv.GetNervaStore(data["database"].(string))
	if nstore == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}
	token, engine, err := (&ntura.API{NStore: nstore}).UserLogin(data)
	srv.respondMessage(w, http.StatusOK, ntura.SM{"token": token, "engine": engine}, http.StatusBadRequest, err)
}

func (srv *HTTPService) UserPassword(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}
	nstore := r.Context().Value(NstoreCtxKey).(*ntura.NervaStore)

	data := ntura.IM{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		srv.respondMessage(w, 0, nil, http.StatusBadRequest, err)
		return
	}
	if _, found := data["username"]; found {
		if nstore.User.Scope != "admin" {
			srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
			return
		}
	}
	if _, found := data["custnumber"]; found {
		if nstore.User.Scope != "admin" {
			srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
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
	err = (&ntura.API{NStore: nstore}).UserPassword(data)
	srv.respondMessage(w, http.StatusNoContent, nil, http.StatusBadRequest, err)
}

func (srv *HTTPService) TokenRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}
	tokenStr, err := (&ntura.API{NStore: r.Context().Value(NstoreCtxKey).(*ntura.NervaStore)}).TokenRefresh()
	srv.respondMessage(w, http.StatusOK, ntura.SM{"token": tokenStr}, http.StatusBadRequest, err)
}

func (srv *HTTPService) GetFilter(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}

	params := ntura.IM{"nervatype": srv.GetParam(r, "nervatype"),
		"metadata": r.URL.Query().Get("metadata")}
	query := strings.Split(r.URL.RawQuery, "&")
	for index := 0; index < len(query); index++ {
		if strings.HasPrefix(query[index], "filter=") {
			params["filter"] = query[index][7:]
		}
	}
	results, err := (&ntura.API{NStore: r.Context().Value(NstoreCtxKey).(*ntura.NervaStore)}).Get(params)
	srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
}

func (srv *HTTPService) GetIds(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}

	params := ntura.IM{"nervatype": srv.GetParam(r, "nervatype"),
		"metadata": r.URL.Query().Get("metadata"), "ids": srv.GetParam(r, "IDs")}
	results, err := (&ntura.API{NStore: r.Context().Value(NstoreCtxKey).(*ntura.NervaStore)}).Get(params)
	srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
}

func (srv *HTTPService) View(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}

	data := make([]ntura.IM, 0)
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		srv.respondMessage(w, 0, nil, http.StatusBadRequest, err)
		return
	}
	results, err := (&ntura.API{NStore: r.Context().Value(NstoreCtxKey).(*ntura.NervaStore)}).View(data)
	srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
}

func (srv *HTTPService) Function(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}

	data := ntura.IM{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		srv.respondMessage(w, 0, nil, http.StatusBadRequest, err)
		return
	}
	results, err := (&ntura.API{NStore: r.Context().Value(NstoreCtxKey).(*ntura.NervaStore)}).Function(data)
	srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
}

func (srv *HTTPService) Update(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}

	data := make([]ntura.IM, 0)
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		srv.respondMessage(w, 0, nil, http.StatusBadRequest, err)
		return
	}
	results, err := (&ntura.API{NStore: r.Context().Value(NstoreCtxKey).(*ntura.NervaStore)}).Update(srv.GetParam(r, "nervatype"), data)
	srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
}

func (srv *HTTPService) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}

	data := ntura.IM{"nervatype": srv.GetParam(r, "nervatype"),
		"id": r.URL.Query().Get("id"), "key": r.URL.Query().Get("key")}
	err := (&ntura.API{NStore: r.Context().Value(NstoreCtxKey).(*ntura.NervaStore)}).Delete(data)
	srv.respondMessage(w, http.StatusNoContent, nil, http.StatusBadRequest, err)
}

func (srv *HTTPService) DatabaseCreate(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-Api-Key")
	if os.Getenv("NT_API_KEY") != apiKey {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}
	data := ntura.IM{"database": r.URL.Query().Get("alias"), "demo": r.URL.Query().Get("demo")}
	log, err := (&ntura.API{NStore: ntura.New(&db.SQLDriver{})}).DatabaseCreate(data)
	srv.respondMessage(w, http.StatusOK, log, http.StatusBadRequest, err)
}

func (srv *HTTPService) ReportList(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}

	params := ntura.IM{"label": r.URL.Query().Get("label")}
	api := &ntura.API{NStore: r.Context().Value(NstoreCtxKey).(*ntura.NervaStore)}
	if api.NStore.User.Scope != "admin" {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}
	results, err := api.ReportList(params)
	srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
}

func (srv *HTTPService) ReportInstall(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}

	params := ntura.IM{"reportkey": r.URL.Query().Get("reportkey")}
	api := &ntura.API{NStore: r.Context().Value(NstoreCtxKey).(*ntura.NervaStore)}
	if api.NStore.User.Scope != "admin" {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}
	results, err := api.ReportInstall(params)
	srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
}

func (srv *HTTPService) ReportDelete(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}

	params := ntura.IM{"reportkey": r.URL.Query().Get("reportkey")}
	api := &ntura.API{NStore: r.Context().Value(NstoreCtxKey).(*ntura.NervaStore)}
	if api.NStore.User.Scope != "admin" {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}
	err := api.ReportDelete(params)
	srv.respondMessage(w, http.StatusNoContent, nil, http.StatusBadRequest, err)
}

func (srv *HTTPService) Report(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(NstoreCtxKey) == nil {
		srv.respondMessage(w, 0, nil, http.StatusUnauthorized, errors.New("Unauthorized"))
		return
	}

	options := ntura.IM{"filters": ntura.IM{}}
	for key, value := range r.URL.Query() {
		if strings.HasPrefix(key, "filters[") {
			fkey := key[8 : len(key)-1]
			options["filters"].(ntura.IM)[fkey] = value[0]
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
	results, err := (&ntura.API{NStore: r.Context().Value(NstoreCtxKey).(*ntura.NervaStore)}).Report(options)
	if err != nil {
		srv.respondMessage(w, 0, nil, http.StatusBadRequest, err)
		return
	}
	if options["output"] == "tmp" {
		srv.respondMessage(w, http.StatusOK, results, http.StatusBadRequest, err)
		return
	}
	if results["filetype"] == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte(results["template"].(string)))
		return
	}
	if results["filetype"] == "xml" {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(results["template"].(string)))
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Write(results["template"].([]uint8))
}
