package app

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo"
	ntura "github.com/nervatura/nervatura-go"
)

func (app *App) report(c echo.Context) error {
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

func (app *App) reportList(c echo.Context) error {
	params := ntura.IM{"label": c.QueryParam("label")}
	results, err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).ReportList(params)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, results)
}

func (app *App) reportInstall(c echo.Context) error {
	params := ntura.IM{"reportkey": c.QueryParam("reportkey")}
	results, err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).ReportInstall(params)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, results)
}

func (app *App) reportDelete(c echo.Context) error {
	params := ntura.IM{"reportkey": c.QueryParam("reportkey")}
	err := (&ntura.API{NStore: c.Get("nstore").(*ntura.NervaStore)}).ReportDelete(params)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ntura.SM{"code": "400", "message": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}