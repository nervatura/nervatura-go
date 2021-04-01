package nervatura

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

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

	fnum1 := ToFloat(options["number_1"])
	fnum2 := ToFloat(options["number_2"])
	result := ToString(fnum1+fnum2, "")
	return IM{"result": "Successfully processed: " + result}, nil
}

//nextNumber - get the next value from the numberdef table (transnumber, custnumber, partnumber etc.)
func (nstore *NervaStore) nextNumber(options IM) (retnumber string, err error) {

	numberkey := ToString(options["numberkey"], "")
	if numberkey == "" {
		return retnumber, errors.New(GetMessage("missing_required_field") + ": numberkey")
	}
	step := ToBoolean(options["step"], true)
	insertKey := ToBoolean(options["insert_key"], true)

	if ok, err := nstore.connected(); !ok || err != nil {
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
	id, curvalue, length := int64(0), int64(0), 5
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
		id = result[0]["id"].(int64)
		curvalue = result[0]["curvalue"].(int64)
		length = int(result[0]["len"].(int64))
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
	value += strconv.FormatInt((curvalue + 1), 10)
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

	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
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

//getReport - server-side PDF and CSV report generation
func (nstore *NervaStore) getReport(options IM) (results IM, err error) {

	orientation := ToString(options["orientation"], "p")
	size := ToString(options["size"], "a4")

	results = IM{}
	filters := IM{}
	if ofilters, valid := options["filters"].(IM); valid {
		filters = ofilters
	}

	if nervatype, valid := options["nervatype"].(string); valid {
		if _, found := options["refnumber"]; found {
			if _, found := filters["@id"]; !found {
				refValues, err := nstore.GetInfofromRefnumber(options)
				if err != nil {
					return results, err
				}
				filters["@id"] = strconv.FormatInt(refValues["id"].(int64), 10)

				if _, found := options["reportkey"]; !found {
					if _, found := options["report_id"]; !found {
						params := IM{
							"qkey":      "default_report",
							"nervatype": nervatype}
						if _, found := refValues["transtype"]; found {
							params["transtype"] = ToString(refValues["transtype"], "")
						}
						if _, found := refValues["direction"]; found {
							params["direction"] = ToString(refValues["direction"], "")
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
			if results["fields"].(IM)[fieldname].(IM)["fieldtype"] == "date" {
				filters[fieldname] = "'" + ToString(filters[fieldname], "") + "'"
			}
			if results["fields"].(IM)[fieldname].(IM)["fieldtype"] == "string" {
				fieldtype := ToString(filters[fieldname], "")
				if !strings.HasPrefix(fieldtype, "'") {
					filters[fieldname] = "'" + fieldtype + "'"
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
	case "xls", "csv":
		template := IM{}
		if err := ConvertFromByte([]byte(results["report"].(IM)["report"].(string)), &template); err != nil {
			return results, err
		}
		rows := make([][]string, 0)
		for key, value := range results["datarows"].(IM) {
			switch value.(type) {
			case []IM:
				if len(value.([]IM)) > 0 {
					sname := key
					columns := IL{}
					if _, found := template[key]; found {
						if _, found := template[key].(IM)["sheetName"]; found {
							sname = ToString(template[key].(IM)["sheetName"], sname)
						}
						if icolumns, valid := template[key].(IM)["columns"].(IL); valid {
							columns = icolumns
						}
					} else {
						for colname := range value.([]IM)[0] {
							columns = append(columns, IM{"name": colname})
						}
					}
					rows = append(rows, []string{sname})
					if _, found := results["datarows"].(IM)["labels"]; found {
						row := make([]string, 0)
						for index := 0; index < len(columns); index++ {
							colname := columns[index].(IM)["name"].(string)
							if _, found := results["datarows"].(IM)["labels"].(IM)[colname]; found {
								row = append(row, results["datarows"].(IM)["labels"].(IM)[colname].(string))
							} else {
								row = append(row, colname)
							}
						}
						rows = append(rows, row)
					}
					if datarows, valid := results["datarows"].(IM)[key].([]IM); valid {
						for index := 0; index < len(datarows); index++ {
							row := make([]string, 0)
							drow := datarows[index]
							for c := 0; c < len(columns); c++ {
								colname := columns[c].(IM)["name"].(string)
								if _, found := drow[colname]; found {
									switch v := drow[colname].(type) {
									case bool:
										row = append(row, strconv.FormatBool(v))
									case int64:
										row = append(row, strconv.FormatInt(v, 10))
									case float64:
										row = append(row, strconv.FormatFloat(drow[colname].(float64), 'f', -1, 64))
									case time.Time:
										row = append(row, drow[colname].(time.Time).String())
									default:
										row = append(row, drow[colname].(string))
									}
								}
							}
							rows = append(rows, row)
						}
					}
				}
			}
		}
		var b bytes.Buffer
		writr := csv.NewWriter(&b)
		writr.WriteAll(rows)
		if err := writr.Error(); err != nil {
			return results, err
		}
		if options["output"] == "base64" {
			return IM{"filetype": "csv",
				"template": base64.URLEncoding.EncodeToString(b.Bytes()), "data": nil}, nil
		}
		return IM{"filetype": "csv", "template": b.String(), "data": nil}, nil

	case "ntr":
		rpt := report.New(orientation, size)
		rpt.ImagePath = os.Getenv("NT_REPORT_DIR")
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
						switch v := ivalue.(type) {
						case bool:
							values[skey] = strconv.FormatBool(v)
						case int64:
							values[skey] = strconv.FormatInt(v, 10)
						case float64:
							values[skey] = strconv.FormatFloat(v, 'f', -1, 64)
						case time.Time:
							values[skey] = v.Format("2006-01-02 15:04")
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
							switch v := ivalue.(type) {
							case bool:
								values[skey] = strconv.FormatBool(v)
							case int64:
								values[skey] = strconv.FormatInt(v, 10)
							case float64:
								values[skey] = strconv.FormatFloat(v, 'f', -1, 64)
							case time.Time:
								values[skey] = v.Format("2006-01-02 15:04")
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
	results = IM{"result": "OK"}

	emailOpt, valid := options["email"].(IM)
	if !valid {
		return results, errors.New(GetMessage("missing_required_field") + ": email")
	}
	if ToString(options["provider"], "smtp") != "smtp" {
		return results, errors.New(GetMessage("invalid_provider"))
	}

	delimeter := "**=myohmy689407924327"
	username := os.Getenv("NT_SMTP_USER")
	password := os.Getenv("NT_SMTP_PASSWORD")
	host := os.Getenv("NT_SMTP_HOST")
	port := os.Getenv("NT_SMTP_PORT")
	if port == "" {
		port = "465"
	}

	tlsConfig := tls.Config{ServerName: host, InsecureSkipVerify: true}
	conn, connErr := tls.Dial("tcp", fmt.Sprintf("%s:%s", host, port), &tlsConfig)
	if connErr != nil {
		return results, connErr
	}
	defer conn.Close()

	client, clientErr := smtp.NewClient(conn, host)
	if clientErr != nil {
		return results, clientErr
	}
	defer client.Close()

	auth := smtp.PlainAuth("", username, password, host)
	if err := client.Auth(auth); err != nil {
		return results, err
	}

	from := ToString(emailOpt["from"], username)
	if err := client.Mail(from); err != nil {
		return results, err
	}
	emailTo := []string{}
	if recipients, valid := emailOpt["recipients"].([]interface{}); valid {
		for index := 0; index < len(recipients); index++ {
			if email, valid := recipients[index].(IM)["email"].(string); valid {
				emailTo = append(emailTo, email)
				if err := client.Rcpt(email); err != nil {
					return results, err
				}
			}
		}
	} else {
		return results, errors.New(GetMessage("missing_required_field") + ": recipients")
	}

	writer, writerErr := client.Data()
	if writerErr != nil {
		return results, writerErr
	}

	emailMsg := fmt.Sprintf("From: %s\r\n", from)
	emailMsg += fmt.Sprintf("To: %s\r\n", strings.Join(emailTo, ";"))
	emailMsg += fmt.Sprintf("Subject: %s\r\n", ToString(emailOpt["subject"], ""))

	emailMsg += "MIME-Version: 1.0\r\n"
	emailMsg += fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", delimeter)

	emailMsg += fmt.Sprintf("\r\n--%s\r\n", delimeter)
	emailMsg += "Content-Type: text/html; charset=\"utf-8\"\r\n"
	emailMsg += "Content-Transfer-Encoding: 7bit\r\n"
	if _, found := emailOpt["html"]; found {
		emailMsg += fmt.Sprintf("\r\n%s\r\n", ToString(emailOpt["html"], ""))
	} else {
		emailMsg += fmt.Sprintf("\r\n%s\r\n", ToString(emailOpt["text"], ""))
	}

	if attachments, withAttachments := emailOpt["attachments"].([]interface{}); withAttachments {
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
			if _, found := attachment["filename"]; found {
				filename = ToString(attachment["filename"], "")
			}
			report, err := nstore.getReport(params)
			if err != nil {
				return results, err
			}

			emailMsg += fmt.Sprintf("\r\n--%s\r\n", delimeter)
			emailMsg += "Content-Type: application/pdf; charset=\"utf-8\"\r\n"
			emailMsg += "Content-Transfer-Encoding: base64\r\n"
			emailMsg += "Content-Disposition: attachment;filename=\"" + filename + "\"\r\n"
			emailMsg += "\r\n" + base64.StdEncoding.EncodeToString(report["template"].([]uint8))
		}
	}

	if _, err := writer.Write([]byte(emailMsg)); err != nil {
		return results, err
	}

	if closeErr := writer.Close(); closeErr != nil {
		return results, closeErr
	}

	client.Quit()

	return results, nil
}
