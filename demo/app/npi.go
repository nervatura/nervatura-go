package app

import (
	"net/http"

	"github.com/labstack/echo"
	ntura "github.com/nervatura/nervatura-go"
)

func (app *App) npiTokenLogin(c echo.Context) error {
	data := ntura.JSONData{}
	if err := c.Bind(&data); err != nil {
		data.Error = ntura.IM{"code": "json_decode", "message": err.Error(), "data": ""}
		return c.JSON(http.StatusBadRequest, data)
	}
	nstore := app.getNStore(data.Params)
	if nstore == nil {
		return echo.ErrUnauthorized
	}
	result, _ := (&ntura.Npi{NStore: nstore}).GetLogin(data.Params)
	data.Result = result
	return c.JSON(http.StatusOK, data)
}

func (app *App) npi(c echo.Context) error {
	data := ntura.JSONData{}
	if err := c.Bind(&data); err != nil {
		data.Error = ntura.IM{"code": "json_decode", "message": err.Error(), "data": ""}
		return c.JSON(http.StatusBadRequest, data)
	}
	result := (&ntura.Npi{NStore: c.Get("nstore").(*ntura.NervaStore)}).GetAPI(data)
	return c.JSON(http.StatusOK, result)
}
