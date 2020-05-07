package app

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/labstack/echo"
	ntura "github.com/nervatura/nervatura-go"
	driver "github.com/nervatura/nervatura-go/driver"
)

func (app *App) apiAuthLogin(c echo.Context) error {
	data := ntura.IM{}
	if err := c.Bind(&data); err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	nstore := app.getNStore(data)
	if nstore == nil {
		return echo.ErrUnauthorized
	}
	tokenStr, engine, err := (&ntura.API{NStore: nstore}).AuthUserLogin(data)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, ntura.SM{"token": tokenStr, "engine": engine})
}

func (app *App) apiAuthPassword(c echo.Context) error {
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

func (app *App) apiAuthRefresh(c echo.Context) error {
	tokenStr, err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).AuthToken()
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, ntura.SM{"token": tokenStr})
}

func (app *App) apiGetFilter(c echo.Context) error {
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

func (app *App) apiGetIds(c echo.Context) error {
	params := ntura.IM{"nervatype": c.ParamValues()[0],
		"metadata": c.QueryParam("metadata"), "ids": c.ParamValues()[1]}
	results, err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).APIGet(params)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, results)
}

func (app *App) apiView(c echo.Context) error {
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

func (app *App) apiFunction(c echo.Context) error {
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

func (app *App) apiPost(c echo.Context) error {
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

func (app *App) apiDelete(c echo.Context) error {
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

func (app *App) apiDatabase(c echo.Context) error {
	apiKey := c.Request().Header.Get("X-Api-Key")
	if app.config.APIKey != apiKey {
		return echo.ErrUnauthorized
	}
	data := ntura.IM{"database": c.QueryParam("alias"), "demo": c.QueryParam("demo")}
	log, err := (&ntura.API{NStore: ntura.New(app.config, &driver.SQLDriver{})}).DatabaseCreate(data)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, log)
}
