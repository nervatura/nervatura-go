package nervatura

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/jordan-wright/email"
	"github.com/nervatura/go-report"
)

//GetService - call Nervatura server side functions and services
func (nstore *NervaStore) GetService(key string, options IM) (interface{}, error) {

	switch key {
	case "callMenuCmd":
		return nstore.callMenuCmd(options)

	case "nextNumber":
		return nstore.nextNumber(options)

	case "getPriceValue":
		return nstore.getPriceValue(options)

	case "getReport":
		return nstore.getReport(options)

	case "sendEmail":
		return nstore.sendEmail(options)

	}
	return nil, errors.New(GetMessage("unknown_method") + ": " + key)
}

//callMenuCmd - sample service function code
func (nstore *NervaStore) callMenuCmd(options IM) (IM, error) {

	var fnum1 float64
	if _, found := options["number_1"]; found && GetIType(options["number_1"]) == "float64" {
		fnum1 = options["number_1"].(float64)
	}
	var fnum2 float64
	if _, found := options["number_2"]; found && GetIType(options["number_2"]) == "float64" {
		fnum2 = options["number_2"].(float64)
	}
	return IM{"result": "Successfully processed: " + strconv.FormatFloat(fnum1+fnum2, 'f', -1, 64)}, nil
}

//nextNumber - get the next value from the numberdef table (transnumber, custnumber, partnumber etc.)
func (nstore *NervaStore) nextNumber(options IM) (retnumber string, err error) {

	if _, found := options["numberkey"]; !found || GetIType(options["numberkey"]) != "string" {
		return retnumber, errors.New(GetMessage("missing_required_field") + ": numberkey")
	}
	numberkey := options["numberkey"].(string)

	step := true
	if _, found := options["step"]; found && GetIType(options["step"]) == "bool" {
		step = options["step"].(bool)
	}

	insertKey := true
	if _, found := options["insert_key"]; found && GetIType(options["insert_key"]) == "bool" {
		insertKey = options["insert_key"].(bool)
	}

	if ok, err := nstore.connected(); ok == false || err != nil {
		if err != nil {
			return retnumber, err
		}
		return retnumber, errors.New(GetMessage("not_connect"))
	}

	var trans interface{}
	if _, found := options["trans"]; found {
		trans = options["trans"]
	} else if nstore.ds.Properties().Transaction {
		trans, err = nstore.ds.BeginTransaction()
		if err != nil {
			return retnumber, err
		}
	}
	defer func() {
		pe := recover()
		if trans != nil {
			if _, found := options["trans"]; !found {
				if err != nil || pe != nil {
					nstore.ds.RollbackTransaction(trans)
				} else {
					err = nstore.ds.CommitTransaction(trans)
				}
			}
		}
		if pe != nil {
			panic(pe)
		}
	}()

	query := []Query{{
		Fields: []string{"*"}, From: "numberdef", Filters: []Filter{
			{Field: "numberkey", Comp: "==", Value: numberkey}}}}
	result, err := nstore.ds.Query(query, trans)
	if err != nil {
		return retnumber, err
	}

	var values IM
	id, curvalue, length := 0, 0, 5
	if len(result) == 0 {
		if insertKey {
			values = IM{"numberkey": numberkey,
				"prefix": strings.ToUpper(numberkey[:3]), "curvalue": curvalue,
				"isyear": 1, "sep": "/", "len": length, "visible": 1, "readonly": 0}
			data := Update{Values: values, Model: "numberdef", Trans: trans}
			id, err = nstore.ds.Update(data)
			if err != nil {
				return retnumber, err
			}
			values["id"] = id
		} else {
			return retnumber, errors.New(GetMessage("invalid_value") + ": refnumber")
		}
	} else {
		id = result[0]["id"].(int)
		curvalue = result[0]["curvalue"].(int)
		length = result[0]["len"].(int)
		values = result[0]
	}

	if values["prefix"] != "" && values["prefix"] != nil {
		retnumber = values["prefix"].(string) + values["sep"].(string)
	}
	if values["isyear"] == 1 || values["isyear"] == "1" {
		transyear := time.Now().Format("2006")
		query := []Query{{
			Fields: []string{"value"}, From: "fieldvalue", Filters: []Filter{
				{Field: "fieldname", Comp: "==", Value: "transyear"},
				{Field: "ref_id", Comp: "is", Value: "null"}}}}
		result, err = nstore.ds.Query(query, trans)
		if err != nil {
			return retnumber, err
		}
		if len(result) > 0 {
			transyear = result[0]["value"].(string)
		}
		retnumber += transyear + values["sep"].(string)
	}
	value := strings.Repeat("0", length)
	value += strconv.Itoa(curvalue + 1)
	vlen := len(value)
	retnumber += value[vlen-length : vlen]
	if step {
		data := Update{Values: IM{"curvalue": curvalue + 1}, Model: "numberdef",
			IDKey: id, Trans: trans}
		_, err := nstore.ds.Update(data)
		if err != nil {
			return retnumber, err
		}
	}
	return retnumber, nil
}

func defaultValue(key string, options IM, defValue interface{}) interface{} {
	if _, found := options[key]; found {
		return options[key]
	}
	return defValue
}

func getFloatValue(value interface{}) (float64, error) {

	switch value.(type) {
	case int:
		return float64(value.(int)), nil
	case int64:
		return float64(value.(int64)), nil
	case float64:
		return value.(float64), nil
	case string:
		return strconv.ParseFloat(value.(string), 64)
	}
	return 0, nil
}

//getPriceValue - get product price
func (nstore *NervaStore) getPriceValue(options IM) (results IM, err error) {
	results = IM{"price": float64(0), "discount": float64(0)}
	params := IM{
		"qkey":        "listprice",
		"curr":        options["curr"],
		"product_id":  options["product_id"],
		"vendorprice": options["vendorprice"],
		"posdate":     options["posdate"],
		"qty":         options["qty"],
		"customer_id": options["customer_id"],
	}
	if _, found := options["curr"]; !found {
		return results, errors.New(GetMessage("missing_required_field") + ": curr")
	}

	if _, found := options["product_id"]; !found {
		return results, errors.New(GetMessage("missing_required_field") + ": product_id")
	}

	params["vendorprice"] = defaultValue("vendorprice", options, 0)
	params["posdate"] = defaultValue("posdate", options, time.Now().Format("2006-01-02"))
	params["qty"] = defaultValue("qty", options, 0)

	//best_listprice
	pdata, err := nstore.ds.QueryKey(params, nil)
	if err != nil {
		return results, err
	}
	if len(pdata) > 0 {
		if pdata[0]["mp"] != nil {
			results["price"], err = getFloatValue(pdata[0]["mp"])
			if err != nil {
				return results, err
			}
		}
	}

	if _, found := options["customer_id"]; found {
		//customer discount
		query := []Query{{
			Fields: []string{"*"}, From: "customer", Filters: []Filter{
				{Field: "id", Comp: "==", Value: params["customer_id"]},
			}}}
		discount, err := nstore.ds.Query(query, nil)
		if err != nil {
			return results, err
		}
		if len(discount) > 0 {
			if discount[0]["discount"] != nil {
				results["discount"] = discount[0]["discount"]
			}
		}
	}

	if _, found := options["customer_id"]; found {
		//best_custprice
		params["qkey"] = "custprice"
		pdata, err := nstore.ds.QueryKey(params, nil)
		if err != nil {
			return results, err
		}
		if len(pdata) > 0 {
			if pdata[0]["mp"] != nil {
				price, err := getFloatValue(pdata[0]["mp"])
				if err != nil {
					return results, err
				}
				if results["price"].(float64) > price || results["price"] == 0 {
					results["price"] = price
					results["discount"] = 0
				}
			}
		}
	}

	if _, found := options["customer_id"]; found {
		//best_grouprice
		params["qkey"] = "grouprice"
		pdata, err := nstore.ds.QueryKey(params, nil)
		if err != nil {
			return results, err
		}
		if len(pdata) > 0 {
			if pdata[0]["mp"] != nil {
				price, err := getFloatValue(pdata[0]["mp"])
				if err != nil {
					return results, err
				}
				if results["price"].(float64) > price || results["price"] == 0 {
					results["price"] = price
					results["discount"] = 0
				}
			}
		}
	}

	return results, nil
}

//getReport - server-side PDF and Excel report generation
func (nstore *NervaStore) getReport(options IM) (results IM, err error) {

	orientation := "p"
	if _, found := options["orientation"]; found && GetIType(options["orientation"]) == "string" {
		orientation = options["orientation"].(string)
	}
	size := "a4"
	if _, found := options["size"]; found && GetIType(options["size"]) == "string" {
		size = options["size"].(string)
	}

	results = IM{}
	filters := IM{}
	if _, found := options["filters"]; found && GetIType(options["filters"]) == "map[string]interface{}" {
		filters = options["filters"].(IM)
	}

	if _, found := options["nervatype"]; found && GetIType(options["nervatype"]) == "string" {
		if _, found := options["refnumber"]; found && GetIType(options["refnumber"]) == "string" {
			if _, found := filters["@id"]; !found {
				refValues, err := nstore.GetInfofromRefnumber(options)
				if err != nil {
					return results, err
				}
				filters["@id"] = strconv.Itoa(int(refValues["id"].(int)))

				if _, found := options["reportkey"]; !found {
					if _, found := options["report_id"]; !found {
						params := IM{
							"qkey":      "default_report",
							"nervatype": options["nervatype"].(string)}
						if _, found := refValues["transtype"]; found && GetIType(refValues["transtype"]) == "string" {
							params["transtype"] = refValues["transtype"].(string)
						}
						if _, found := refValues["direction"]; found && GetIType(refValues["direction"]) == "string" {
							params["direction"] = refValues["direction"].(string)
						}
						rdata, err := nstore.ds.QueryKey(params, nil)
						if err != nil {
							return results, err
						}
						if len(rdata) == 0 {
							return results, errors.New(GetMessage("not_exist"))
						}
						results["report"] = rdata[0]
					}
				}
			}
		}
	}

	if _, found := results["report"]; !found {
		if _, found := options["report_id"]; !found {
			if _, found := options["reportkey"]; !found {
				return results, errors.New(GetMessage("missing_required_field") + ": report_id or reportkey")
			}
		}
		params := IM{"qkey": "default_report"}
		if _, found := options["report_id"]; !found {
			params["reportkey"] = options["reportkey"]
		}
		rdata, err := nstore.ds.QueryKey(params, nil)
		if err != nil {
			return results, err
		}
		if len(rdata) == 0 {
			return results, errors.New(GetMessage("not_exist"))
		}
		results["report"] = rdata[0]
	}
	reportkey := results["report"].(IM)["reportkey"].(string)

	query := []Query{{
		Fields: []string{"*"}, From: "ui_reportsources", Filters: []Filter{
			{Field: "report_id", Comp: "==", Value: results["report"].(IM)["id"]},
		}}}
	results["sources"], err = nstore.ds.Query(query, nil)
	if err != nil {
		return results, err
	}

	params := IM{"qkey": "reportfields", "report_id": results["report"].(IM)["id"]}
	fields, err := nstore.ds.QueryKey(params, nil)
	if err != nil {
		return results, err
	}
	results["fields"] = IM{}
	for index := 0; index < len(fields); index++ {
		results["fields"].(IM)[fields[index]["fieldname"].(string)] = IM{
			"fieldtype": fields[index]["fieldtype"],
			"wheretype": fields[index]["wheretype"],
			"dataset":   fields[index]["dataset"],
			"sql":       fields[index]["sqlstr"],
		}
	}

	results["datarows"] = IM{}
	secname := make([]string, 0)
	secname = append(secname, reportkey+"_report")
	for index := 0; index < len(results["sources"].([]IM)); index++ {
		secname = append(secname, reportkey+"_"+results["sources"].([]IM)[index]["dataset"].(string))
	}
	query = []Query{{
		Fields: []string{"*"}, From: "ui_message", Filters: []Filter{
			{Field: "secname", Comp: "in", Value: strings.Join(secname, ",")},
		}}}
	labels, err := nstore.ds.Query(query, nil)
	if err != nil {
		return results, err
	}
	results["datarows"].(IM)["labels"] = IM{}
	for index := 0; index < len(labels); index++ {
		if labels[index]["secname"] == reportkey+"_report" {
			results["datarows"].(IM)["labels"].(IM)[labels[index]["fieldname"].(string)] = labels[index]["msg"]
		} else {
			for si := 0; si < len(results["sources"].([]IM)); si++ {
				ds := results["sources"].([]IM)[si]
				if labels[index]["secname"] == reportkey+"_"+ds["dataset"].(string) {
					ds["sqlstr"] = strings.ReplaceAll(ds["sqlstr"].(string), "={{"+labels[index]["fieldname"].(string)+"}}", labels[index]["msg"].(string))
				}
			}
		}
	}

	results["where_str"] = IM{}
	for fieldname, value := range filters {
		if _, found := results["fields"].(IM)[fieldname]; !found {
			if fieldname == "@id" {
				for index := 0; index < len(results["sources"].([]IM)); index++ {
					ds := results["sources"].([]IM)[index]
					ds["sqlstr"] = strings.ReplaceAll(ds["sqlstr"].(string), "@id", value.(string))
				}
			} else {
				return results, errors.New(GetMessage("invalid_fieldname") + ": " + fieldname)
			}
		} else {
			rel := " = "
			if results["fields"].(IM)[fieldname].(IM)["fieldtype"] == "date" && GetIType(filters[fieldname]) == "string" {
				filters[fieldname] = "'" + filters[fieldname].(string) + "'"
			}
			if results["fields"].(IM)[fieldname].(IM)["fieldtype"] == "string" && GetIType(filters[fieldname]) == "string" {
				if !strings.HasPrefix(filters[fieldname].(string), "'") {
					filters[fieldname] = "'" + filters[fieldname].(string) + "'"
				}
				rel = " like "
			}
			for index := 0; index < len(results["sources"].([]IM)); index++ {
				ds := results["sources"].([]IM)[index]
				if results["fields"].(IM)[fieldname].(IM)["dataset"] == ds["dataset"] || results["fields"].(IM)[fieldname].(IM)["dataset"] == nil {
					if results["fields"].(IM)[fieldname].(IM)["wheretype"] == "where" {
						wkey := results["fields"].(IM)[fieldname].(IM)["dataset"]
						if wkey == nil {
							wkey = "nods"
						}
						fstr := ""
						if results["fields"].(IM)[fieldname].(IM)["sql"] == nil || results["fields"].(IM)[fieldname].(IM)["sql"] == "" {
							fstr = fieldname + rel + filters[fieldname].(string)
						} else {
							fstr = strings.ReplaceAll(fstr, "@"+fieldname, filters[fieldname].(string))
						}
						if _, found := results["where_str"].(IM)[wkey.(string)]; !found {
							results["where_str"].(IM)[wkey.(string)] = " and " + fstr
						} else {
							results["where_str"].(IM)[wkey.(string)] = results["where_str"].(IM)[wkey.(string)].(string) + " and " + fstr
						}
					} else {
						if results["fields"].(IM)[fieldname].(IM)["sql"] == nil || results["fields"].(IM)[fieldname].(IM)["sql"] == "" {
							ds["sqlstr"] = strings.ReplaceAll(ds["sqlstr"].(string), "@"+fieldname, filters[fieldname].(string))
						} else {
							fstr := strings.ReplaceAll(results["fields"].(IM)[fieldname].(IM)["sql"].(string), "@"+fieldname, filters[fieldname].(string))
							ds["sqlstr"] = strings.ReplaceAll(ds["sqlstr"].(string), "@"+fieldname, fstr)
						}
					}
				}
			}
		}
	}

	trows := 0
	const whereKey = "@where_str"
	for index := 0; index < len(results["sources"].([]IM)); index++ {
		ds := results["sources"].([]IM)[index]
		if _, found := results["where_str"].(IM)[ds["dataset"].(string)]; found {
			ds["sqlstr"] = strings.ReplaceAll(ds["sqlstr"].(string), whereKey, results["where_str"].(IM)[ds["dataset"].(string)].(string))
		}
		if _, found := results["where_str"].(IM)["nods"]; found {
			ds["sqlstr"] = strings.ReplaceAll(ds["sqlstr"].(string), whereKey, results["where_str"].(IM)["nods"].(string))
		}
		ds["sqlstr"] = strings.ReplaceAll(ds["sqlstr"].(string), whereKey, "")
		params := make([]interface{}, 0)
		results["datarows"].(IM)[ds["dataset"].(string)], err = nstore.ds.QuerySQL(ds["sqlstr"].(string), params, nil)
		if err != nil {
			return results, err
		}
		trows += len(results["datarows"].(IM)[ds["dataset"].(string)].([]IM))
	}
	results["datarows"].(IM)["title"] = results["report"].(IM)["repname"]
	results["datarows"].(IM)["crtime"] = time.Now().Format(TimeLayout)
	if trows == 0 {
		return results, errors.New(GetMessage("nodata"))
	}
	if _, found := results["datarows"].(IM)["ds"]; found {
		if len(results["datarows"].(IM)["ds"].([]IM)) == 0 {
			return results, errors.New(GetMessage("nodata"))
		}
	}

	if options["output"] == "tmp" {
		return IM{
			"filetype": results["report"].(IM)["reptype"],
			"template": results["report"].(IM)["report"],
			"data":     results["datarows"]}, nil
	}
	switch results["report"].(IM)["reptype"] {
	case "xls":
		template := IM{}
		if err := json.Unmarshal([]byte(results["report"].(IM)["report"].(string)), &template); err != nil {
			return results, err
		}
		xlsx := excelize.NewFile()
		for key, value := range results["datarows"].(IM) {
			switch value.(type) {
			case []IM:
				if len(value.([]IM)) > 0 {
					sname := key
					columns := IL{}
					if _, found := template[key]; found {
						if _, found := template[key].(IM)["sheetName"]; found && GetIType(template[key].(IM)["sheetName"]) == "string" {
							sname = template[key].(IM)["sheetName"].(string)
						}
						if _, found := template[key].(IM)["columns"]; found && GetIType(template[key].(IM)["columns"]) == IList {
							columns = template[key].(IM)["columns"].(IL)
						}
					} else {
						for colname := range value.([]IM)[0] {
							columns = append(columns, IM{"name": colname})
						}
					}
					xlsx.NewSheet(sname)
					if _, found := results["datarows"].(IM)["labels"]; found {
						for index := 0; index < len(columns); index++ {
							if index <= 25 {
								colname := columns[index].(IM)["name"].(string)
								cell, err := excelize.CoordinatesToCellName(index+1, 1)
								if err != nil {
									return results, err
								}
								if _, found := results["datarows"].(IM)["labels"].(IM)[colname]; found {
									xlsx.SetCellValue(sname, cell, results["datarows"].(IM)["labels"].(IM)[colname])
								} else {
									xlsx.SetCellValue(sname, cell, colname)
								}
							}
						}
					}
					if _, found := results["datarows"].(IM)[key]; found && GetIType(results["datarows"].(IM)[key]) == "[]map[string]interface{}" {
						for index := 0; index < len(results["datarows"].(IM)[key].([]IM)); index++ {
							drow := results["datarows"].(IM)[key].([]IM)[index]
							for c := 0; c < len(columns); c++ {
								colname := columns[c].(IM)["name"].(string)
								cell, err := excelize.CoordinatesToCellName(c+1, index+2)
								if err != nil {
									return results, err
								}
								if _, found := drow[colname]; found {
									xlsx.SetCellValue(sname, cell, drow[colname])
								}
							}
						}
					}
				}
			}
		}
		xlsx.DeleteSheet("Sheet1")
		var b bytes.Buffer
		writr := bufio.NewWriter(&b)
		xlsx.Write(writr)
		writr.Flush()
		return IM{"filetype": "xlsx", "template": b.Bytes(), "data": nil}, nil

	case "ntr":
		rpt := report.New(orientation, size)
		rpt.ImagePath = nstore.settings.ReportDir
		_, err := rpt.LoadDefinition(results["report"].(IM)["report"].(string))
		if err != nil {
			return results, err
		}
		for key, value := range results["datarows"].(IM) {
			switch value.(type) {
			case string, map[string]string, []map[string]string:
				_, err := rpt.SetData(key, value)
				if err != nil {
					return results, err
				}
			case map[string]interface{}:
				values := SM{}
				for skey, ivalue := range value.(IM) {
					if ivalue == nil {
						values[skey] = ""
					} else {
						switch ivalue.(type) {
						case bool:
							values[skey] = strconv.FormatBool(ivalue.(bool))
						case int:
							values[skey] = strconv.Itoa(ivalue.(int))
						case float64:
							values[skey] = strconv.FormatFloat(ivalue.(float64), 'f', -1, 64)
						case time.Time:
							values[skey] = ivalue.(time.Time).Format("2006-01-02 15:04")
						default:
							values[skey] = ivalue.(string)
						}
					}
				}
				_, err := rpt.SetData(key, values)
				if err != nil {
					return results, err
				}
			case []map[string]interface{}:
				ivalues := []SM{}
				for index := 0; index < len(value.([]IM)); index++ {
					values := SM{}
					for skey, ivalue := range value.([]IM)[index] {
						if ivalue == nil {
							values[skey] = ""
						} else {
							switch ivalue.(type) {
							case bool:
								values[skey] = strconv.FormatBool(ivalue.(bool))
							case int:
								values[skey] = strconv.Itoa(ivalue.(int))
							case float64:
								values[skey] = strconv.FormatFloat(ivalue.(float64), 'f', -1, 64)
							case time.Time:
								values[skey] = ivalue.(time.Time).Format("2006-01-02 15:04")
							default:
								values[skey] = ivalue.(string)
							}
						}
					}
					ivalues = append(ivalues, values)
				}
				_, err := rpt.SetData(key, ivalues)
				if err != nil {
					return results, err
				}
			}
		}
		rpt.CreateReport()
		switch options["output"] {
		case "xml":
			xml := rpt.Save2Xml()
			return IM{"filetype": "xml", "template": xml, "data": nil}, nil

		case "base64":
			pdf, err := rpt.Save2DataURLString("Report.pdf")
			if err != nil {
				return results, err
			}
			return IM{"filetype": "base64", "template": pdf, "data": nil}, nil

		default:
			pdf, err := rpt.Save2Pdf()
			if err != nil {
				return results, err
			}
			return IM{"filetype": "ntr", "template": pdf, "data": nil}, nil
		}

	default:
		return results, errors.New(GetMessage("invalid_fieldname") + ": " + results["report"].(IM)["reptype"].(string))
	}
}

func (nstore *NervaStore) sendEmail(options IM) (results IM, err error) {
	results = IM{}
	if _, found := options["email"]; !found || GetIType(options["email"]) != "map[string]interface{}" {
		return results, errors.New(GetMessage("missing_required_field") + ": email")
	}
	if _, found := options["provider"]; !found || GetIType(options["provider"]) != "string" {
		return results, errors.New(GetMessage("missing_required_field") + ": provider")
	} else if options["provider"] != "smtp" {
		return results, errors.New(GetMessage("invalid_provider"))
	}

	e := email.NewEmail()
	if _, found := options["email"].(IM)["from"]; found && GetIType(options["email"].(IM)["from"]) == "string" {
		if _, found := options["email"].(IM)["name"]; found && GetIType(options["email"].(IM)["name"]) == "string" {
			e.From = options["email"].(IM)["name"].(string) + " <" + options["email"].(IM)["from"].(string) + ">"
		} else {
			e.From = options["email"].(IM)["from"].(string)
		}
	}
	if _, found := options["email"].(IM)["recipients"]; found && GetIType(options["email"].(IM)["recipients"]) == IList {
		recipients := options["email"].(IM)["recipients"].([]interface{})
		for index := 0; index < len(recipients); index++ {
			if _, found := recipients[index].(IM)["email"]; found && GetIType(recipients[index].(IM)["email"]) == "string" {
				e.To = append(e.To, recipients[index].(IM)["email"].(string))
			}
		}
	}
	if _, found := options["email"].(IM)["subject"]; found && GetIType(options["email"].(IM)["subject"]) == "string" {
		e.Subject = options["email"].(IM)["subject"].(string)
	}
	if _, found := options["email"].(IM)["text"]; found && GetIType(options["email"].(IM)["text"]) == "string" {
		e.Text = []byte(options["email"].(IM)["text"].(string))
	}
	if _, found := options["email"].(IM)["html"]; found && GetIType(options["email"].(IM)["html"]) == "string" {
		e.HTML = []byte(options["email"].(IM)["html"].(string))
	}

	if _, found := options["email"].(IM)["attachments"]; found && GetIType(options["email"].(IM)["attachments"]) == IList {
		attachments := options["email"].(IM)["attachments"].([]interface{})
		for index := 0; index < len(attachments); index++ {
			attachment := attachments[index].(IM)
			params := IM{"output": "pdf"}
			if _, found := attachment["reportkey"]; found {
				params["reportkey"] = attachment["reportkey"]
			}
			if _, found := attachment["report_id"]; found {
				params["report_id"] = attachment["report_id"]
			}
			if _, found := attachment["nervatype"]; found {
				params["nervatype"] = attachment["nervatype"]
			}
			if _, found := attachment["refnumber"]; found {
				params["refnumber"] = attachment["refnumber"]
			}
			if _, found := attachment["ref_id"]; found {
				params["filters"] = IM{"@id": attachment["ref_id"]}
			}
			filename := "docs_" + strconv.Itoa(index+1) + ".pdf"
			if _, found := attachment["filename"]; found && GetIType(attachment["filename"]) == "string" {
				filename = attachment["filename"].(string)
			}
			report, err := nstore.getReport(params)
			if err != nil {
				return results, err
			}
			_, err = e.Attach(bytes.NewReader(report["template"].([]uint8)), filename, "application/pdf")
			if err != nil {
				return results, err
			}
		}
	}
	username := ""
	if _, found := nstore.settings.SMTP["user"]; found && GetIType(nstore.settings.SMTP["user"]) == "string" {
		username = nstore.settings.SMTP["user"].(string)
	}
	var password string
	if _, found := nstore.settings.SMTP["password"]; found && GetIType(nstore.settings.SMTP["password"]) == "string" {
		password = nstore.settings.SMTP["password"].(string)
	}
	host := ""
	if _, found := nstore.settings.SMTP["host"]; found && GetIType(nstore.settings.SMTP["host"]) == "string" {
		host = nstore.settings.SMTP["host"].(string)
	}
	port := 465
	if _, found := nstore.settings.SMTP["port"]; found && GetIType(nstore.settings.SMTP["port"]) == "int" {
		port = nstore.settings.SMTP["port"].(int)
	}
	if nstore.settings.SMTP["secure"] == true {
		err = e.SendWithTLS(host+":"+strconv.Itoa(port), smtp.PlainAuth("", username, password, host),
			&tls.Config{ServerName: host, InsecureSkipVerify: true})
	} else {
		err = e.Send(host+":"+strconv.Itoa(port), smtp.PlainAuth("", username, password, host))
	}
	if err == nil {
		results["result"] = "OK"
	}

	return results, err
}
