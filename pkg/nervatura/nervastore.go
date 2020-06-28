package nervatura

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

const (
	datetimeISOFmt string = "2006-01-02T15:04:05-0700"
	datetimeFmt    string = "2006-01-02 15:04:05"
	dateFmt        string = "2006-01-02"
)

// NervaStore is the core structure of the Nervatura
type NervaStore struct {
	ds       DataDriver
	settings Settings
	User     IM
	Customer IM
	models   IM
}

// New returns a pointer to a new NervaStore instance.
func New(options Settings, driver DataDriver) (nstore *NervaStore) {
	nstore = new(NervaStore)
	nstore.models = DataModel()["model"].(IM)
	nstore.settings = options
	nstore.ds = driver
	return
}

func (nstore *NervaStore) connected() (bool, error) {
	if nstore.ds == nil {
		return false, errors.New(GetMessage("missing_driver"))
	}
	return nstore.ds.Connection().Connected, nil
}

func validFieldValue(nervatype string) bool {
	if nervatype[0:2] == "ui" {
		return false
	}
	switch nervatype {
	case "groups", "numberdef", "deffield", "pattern", "fieldvalue":
		return false
	default:
		return true
	}
}

func (nstore *NervaStore) getTableKey(nervatype string) string {
	if _, found := nstore.models[nervatype]; found {
		if len(nstore.models[nervatype].(IM)["_key"].(SL)) == 1 {
			return nstore.models[nervatype].(IM)["_key"].(SL)[0]
		}
		return ""
	}
	return ""
}

func checkFieldvalueBool(value interface{}) (interface{}, error) {
	if value == nil {
		return "false", nil
	}
	switch value {
	case "true", "True", "TRUE", "t", "T", "y", "YES", "yes", float64(1), int(1), "1", true:
		return "true", nil
	default:
		return "false", nil
	}
}

func checkFieldvalueInteger(value interface{}, fieldname string) (interface{}, error) {
	if value == nil {
		return int(0), nil
	}
	switch value.(type) {
	case int32:
		return int(value.(int32)), nil
	case int64:
		return int(value.(int64)), nil
	case int:
		return value, nil
	case float64:
		return int(value.(float64)), nil
	case float32:
		return int(value.(float32)), nil
	case string:
		value, err := strconv.Atoi(value.(string))
		if err != nil {
			return value, err
		}
		return value, nil
	default:
		return value, errors.New(GetMessage("invalid_value") + ": " + fieldname + " (integer)")
	}
}

func checkFieldvalueFloat(value interface{}, fieldname string) (interface{}, error) {
	if value == nil {
		return float64(0), nil
	}
	switch value.(type) {
	case int32:
		return float64(value.(int32)), nil
	case int64:
		return float64(value.(int64)), nil
	case int:
		return float64(value.(int)), nil
	case float32:
		return float64(value.(float32)), nil
	case float64:
		return value, nil
	case string:
		value, err := strconv.ParseFloat(value.(string), 64)
		if err != nil {
			return value, err
		}
		return value, nil
	default:
		return value, errors.New(GetMessage("invalid_value") + ": " + fieldname + " (float)")
	}
}

func checkFieldvalueDate(value interface{}, fieldname, fieldtype string) (interface{}, error) {
	if value == nil {
		return value, errors.New(GetMessage("invalid_value") + ": " + fieldname + " (" + fieldtype + ")")
	}
	switch value.(type) {
	case time.Time:
		return value.(time.Time).Format(dateFmt), nil
	case string:
		tm, err := time.Parse(dateFmt, value.(string))
		if err != nil {
			tm, err = time.Parse(dateFmt, value.(string))
		}
		if err != nil {
			return value, errors.New(GetMessage("invalid_value") + ": " + fieldname + " (" + fieldtype + ")")
		}
		return tm.Format(dateFmt), nil
	default:
		return value, errors.New(GetMessage("invalid_value") + ": " + fieldname + " (" + fieldtype + ")")
	}
}

func checkFieldvalueTime(value interface{}, fieldname, fieldtype string) (interface{}, error) {
	if value == nil {
		return value, errors.New(GetMessage("invalid_value") + ": " + fieldname + " (" + fieldtype + ")")
	}
	switch value.(type) {
	case time.Time:
		return value.(time.Time).Format("15:04"), nil
	case string:
		tm, err := time.Parse("15:04:05", value.(string))
		if err != nil {
			tm, err = time.Parse("15:04", value.(string))
		}
		if err != nil {
			return value, errors.New(GetMessage("invalid_value") + ": " + fieldname + " (" + fieldtype + ")")
		}
		return tm.Format("15:04"), nil
	default:
		return value, errors.New(GetMessage("invalid_value") + ": " + fieldname + " (" + fieldtype + ")")
	}
}

func (nstore *NervaStore) checkFieldvalueNervatype(value interface{}, fieldname, fieldtype string, trans interface{}) (interface{}, error) {
	query := Query{Fields: []string{"id"}}
	if fieldtype == "transitem" || fieldtype == "transmovement" || fieldtype == "transpayment" {
		query.From = "trans"
	} else {
		query.From = fieldtype
	}
	switch value.(type) {
	case int, int32, int64, float64:
		query.Filters = []Filter{
			{Field: "id", Comp: "==", Value: value}}
	case string:
		_, err := strconv.Atoi(value.(string))
		if err == nil {
			query.Filters = []Filter{
				{Field: "id", Comp: "==", Value: value}}
		} else {
			query.Filters = []Filter{
				{Field: nstore.getTableKey(fieldtype), Comp: "like", Value: value}}
		}
	default:
		return value, errors.New(GetMessage("invalid_value") + ": " + fieldname)
	}
	rows, err := nstore.ds.Query([]Query{query}, trans)
	if err != nil {
		return value, err
	}
	if len(rows) == 0 {
		return value, errors.New(GetMessage("invalid_value") + ": " + fieldname)
	}
	return rows[0]["id"], nil
}

// checkFieldvalue
func (nstore *NervaStore) checkFieldvalue(fieldname string, value, trans interface{}) (interface{}, error) {
	rows, err := nstore.ds.QueryKey(IM{"qkey": "update_deffields", "fieldname": fieldname}, trans)
	if err != nil {
		return value, err
	}
	if len(rows) == 0 {
		return value, errors.New(GetMessage("missing_fieldname"))
	}
	fieldtype := rows[0]["fieldtype"].(string)

	switch fieldtype {
	case "bool":
		return checkFieldvalueBool(value)

	case "integer":
		return checkFieldvalueInteger(value, fieldname)

	case "float":
		return checkFieldvalueFloat(value, fieldname)

	case "date":
		return checkFieldvalueDate(value, fieldname, fieldtype)

	case "time":
		return checkFieldvalueTime(value, fieldname, fieldtype)

	case "string", "password", "valuelist", "notes", "urlink":
		return value, nil

	case "customer", "tool", "product", "project", "employee", "place", "transitem", "transmovement", "transpayment":
		return nstore.checkFieldvalueNervatype(value, fieldname, fieldtype, trans)

	default:
		return value, errors.New(GetMessage("invalid_value") + ": " + fieldname)
	}
}

func stringValue(options IM, key string) (string, error) {
	if _, found := options[key]; found && GetIType(options[key]) == "string" {
		return options[key].(string), nil
	}
	return "", errors.New(GetMessage("invalid_value"))
}

func (nstore *NervaStore) insertLog(options IM) error {

	nervatype, err := stringValue(options, "nervatype")
	if err != nil {
		return errors.New("missing_nervatype")
	}
	//if _, found := options["nervatype"]; found && GetIType(options["nervatype"]) == "string" {
	//	nervatype = options["nervatype"].(string)
	//} else {
	//	return errors.New("missing_nervatype")
	//}
	logstate, _ := stringValue(options, "logstate")
	//if _, found := options["logstate"]; found && GetIType(options["logstate"]) == "string" {
	//	logstate = options["logstate"].(string)
	//}
	if ok, err := nstore.connected(); ok == false || err != nil {
		//if err != nil {
		//	return err
		//}
		return errors.New(GetMessage("not_connect"))
	}

	if nstore.User != nil && nervatype != "log" && logstate != "" {
		query := []Query{{
			Fields: []string{"id", "groupname as fieldname", "groupvalue as value"},
			From:   "groups", Filters: []Filter{
				{Field: "groupname", Comp: "==", Value: "logstate"},
				{Field: "groupvalue", Comp: "==", Value: logstate}}},
			{
				Fields: []string{"id", "fieldname", "value"},
				From:   "fieldvalue", Filters: []Filter{
					{Field: "fieldname", Comp: "==", Value: "log_" + logstate},
					{Or: true, Field: "fieldname", Comp: "==", Value: "log_" + nervatype + "_" + logstate}}},
			{
				Fields: []string{"id", "groupname as fieldname", "groupvalue as value"},
				From:   "groups", Filters: []Filter{
					{Field: "groupname", Comp: "==", Value: "nervatype"},
					{Field: "groupvalue", Comp: "==", Value: nervatype}}}}
		logdata, err := nstore.ds.Query(query, options["trans"])
		if err != nil {
			return err
		}
		var logEnabled bool
		var logstateID, nervatypeID int
		for index := 0; index < len(logdata); index++ {
			row := logdata[index]
			if row["fieldname"] == "logstate" {
				logstateID = row["id"].(int)
			}
			if row["fieldname"] == "nervatype" {
				nervatypeID = row["id"].(int)
			}
			if row["fieldname"] == "log_"+nervatype+"_"+logstate {
				if row["value"] == "true" {
					logEnabled = true
				}
			}
			if (row["fieldname"] == "log_"+logstate) &&
				((len(logdata) == 2) && (nervatype == "") ||
					(len(logdata) == 3) && (nervatype != "")) {
				if row["value"] == "true" {
					logEnabled = true
				}
			}
		}
		if logEnabled == true && logstateID > 0 {
			values := IM{"logstate": logstateID, "employee_id": nstore.User["id"],
				"crdate": time.Now().Format(datetimeISOFmt)}
			if nervatypeID > 0 {
				values["nervatype"] = nervatypeID
			}
			if _, found := options["ref_id"]; found && GetIType(options["ref_id"]) == "int" {
				values["ref_id"] = options["ref_id"].(int)
			}
			data := Update{Values: values, Model: "log", Trans: options["trans"]}
			_, err := nstore.ds.Update(data)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateData - update a record data
func (nstore *NervaStore) UpdateData(options IM) (id int, err error) {

	var nervatype string
	if _, found := options["nervatype"]; !found || GetIType(options["nervatype"]) != "string" {
		return id, errors.New(GetMessage("missing_nervatype"))
	}
	nervatype = options["nervatype"].(string)
	if _, found := nstore.models[nervatype]; !found {
		return id, errors.New(GetMessage("invalid_nervatype") + " " + nervatype)
	}

	var values IM
	if _, found := options["values"]; !found || GetIType(options["values"]) != "map[string]interface{}" {
		return id, errors.New(GetMessage("invalid_value"))
	}
	values = options["values"].(IM)

	logEnabled := true
	if _, found := options["log_enabled"]; found && GetIType(options["log_enabled"]) == "bool" {
		logEnabled = options["log_enabled"].(bool)
	}

	validate := true
	if _, found := options["validate"]; found && GetIType(options["validate"]) == "bool" {
		validate = options["validate"].(bool)
	}

	insertField := false
	if _, found := options["insert_field"]; found && GetIType(options["insert_field"]) == "bool" {
		insertField = options["insert_field"].(bool)
	}

	insertRow := false
	if _, found := options["insert_row"]; found && GetIType(options["insert_row"]) == "bool" {
		insertRow = options["insert_row"].(bool)
	}

	updateRow := true
	if _, found := options["update_row"]; found && GetIType(options["update_row"]) == "bool" {
		updateRow = options["update_row"].(bool)
	}

	if ok, err := nstore.connected(); ok == false || err != nil {
		if err != nil {
			return id, err
		}
		return id, errors.New(GetMessage("not_connect"))
	}

	var trans interface{}
	var result []IM
	if _, found := options["trans"]; found {
		trans = options["trans"]
	} else if nstore.ds.Properties().Transaction {
		trans, err = nstore.ds.BeginTransaction()
		if err != nil {
			return id, err
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

	if _, found := values["id"]; found && values["id"] != nil {
		//id check
		switch values["id"].(type) {
		case int:
			id = values["id"].(int)
		case int64:
			id = int(values["id"].(int64))
		case float64:
			id = int(values["id"].(float64))
		case string:
			id, err = strconv.Atoi(values["id"].(string))
			if err == nil {
				return id, err
			}
		}
		query := []Query{{
			Fields: []string{"*"}, From: nervatype, Filters: []Filter{
				{Field: "id", Comp: "==", Value: id}}}}
		result, err = nstore.ds.Query(query, trans)
		if err != nil {
			return id, err
		}
		if len(result) == 0 {
			return id, errors.New(GetMessage("invalid_id"))
		}
		if updateRow == false {
			//readonly record
			return id, errors.New(GetMessage("disabled_update"))
		}

	}

	if id <= 0 && insertRow == false {
		return id, errors.New(GetMessage("disabled_insert"))
	}

	//check fieldnames
	checkValues := IM{"values": IM{}, "fvalues": IM{}, "dvalues": IM{}, "deffield": make([]IM, 0), "fieldvalue": make([]IM, 0)}
	for fieldname, value := range values {
		switch fieldname {
		case "id", "__tablename__":
		default:
			if _, found := nstore.models[nervatype].(IM)[fieldname]; found {
				checkValues["values"].(IM)[fieldname] = value
			} else {
				if validFieldValue(nervatype) {
					checkValues["fvalues"].(IM)[fieldname] = value
				} else {
					return id, errors.New(GetMessage("unknown_fieldname") + " " + fieldname)
				}
			}
		}
	}

	if validFieldValue(nervatype) && id <= 0 {
		//add auto deffields
		result, err = nstore.ds.QueryKey(IM{"qkey": "update_deffields", "nervatype": nervatype}, trans)
		if err != nil {
			return id, err
		}
		for index := 0; index < len(result); index++ {
			row := result[index]
			if row["addnew"] == 1 && row["visible"] == 1 {
				if _, found := checkValues["fvalues"].(IM)[row["fieldname"].(string)]; !found {
					switch row["fieldtype"] {
					case "bool":
						checkValues["fvalues"].(IM)[row["fieldname"].(string)] = "false"
					case "integer", "float":
						checkValues["fvalues"].(IM)[row["fieldname"].(string)] = 0
					default:
						checkValues["fvalues"].(IM)[row["fieldname"].(string)] = ""
					}
				}
			}
		}
	}

	if validFieldValue(nervatype) {
		//check deffields
		result, err = nstore.ds.QueryKey(IM{"qkey": "update_deffields", "nervatype": nervatype, "ref_id": id}, trans)
		if err != nil {
			return id, err
		}
		for index := 0; index < len(result); index++ {
			row := result[index]
			if _, found := checkValues["dvalues"].(IM)[row["fieldname"].(string)]; !found {
				checkValues["dvalues"].(IM)[row["fieldname"].(string)] = make(IL, 0)
			}
			if row["fieldvalue_id"] != nil {
				checkValues["dvalues"].(IM)[row["fieldname"].(string)] = append(checkValues["dvalues"].(IM)[row["fieldname"].(string)].(IL), row["fieldvalue_id"])
			}
		}
	}

	if len(checkValues["values"].(IM)) > 0 && validate {
		//validate
		query := []Query{{
			Fields: []string{"g.id as id", "g.groupname as groupname", "g.groupvalue as groupvalue"},
			From:   "groups g", Filters: []Filter{
				{Field: "g.deleted", Comp: "==", Value: 0}}},
			{
				Fields: []string{"c.id as id", "'curr' as groupname", "c.curr as groupvalue"},
				From:   "currency c"}}
		result, err = nstore.ds.Query(query, trans)
		if err != nil {
			return id, err
		}
		groups := make(map[int]interface{})
		curr := make(IM)
		for index := 0; index < len(result); index++ {
			row := result[index]
			if row["groupname"] == "curr" {
				curr[row["groupvalue"].(string)] = row["id"]
			} else {
				switch row["id"].(type) {
				case int:
					groups[row["id"].(int)] = row
				case int64:
					groups[int(row["id"].(int64))] = row
				case float64:
					groups[int(row["id"].(float64))] = row
				case string:
					rid, err := strconv.Atoi(row["id"].(string))
					if err == nil {
						groups[rid] = row
					}
				}
			}
		}
		for fieldname, value := range checkValues["values"].(IM) {
			var field = nstore.models[nervatype].(IM)[fieldname].(MF)
			switch field.Type {
			case "integer":
				if field.Requires == nil {
					if value == nil {
						if field.NotNull {
							checkValues["values"].(IM)[fieldname] = int(0)
						}
					} else {
						switch value.(type) {
						case int32:
							checkValues["values"].(IM)[fieldname] = int(value.(int32))
						case int64:
							checkValues["values"].(IM)[fieldname] = int(value.(int64))
						case float64:
							checkValues["values"].(IM)[fieldname] = int(value.(float64))
						case float32:
							checkValues["values"].(IM)[fieldname] = int(value.(float32))
						case int:
						default:
							return id, errors.New(GetMessage("invalid_value") + ": " + fieldname + " (integer)")
						}
					}
				}

			case "float":
				if value == nil {
					if field.NotNull {
						checkValues["values"].(IM)[fieldname] = float64(0)
					}
				} else {
					switch value.(type) {
					case float32:
						checkValues["values"].(IM)[fieldname] = float64(value.(float32))
					case float64:
						checkValues["values"].(IM)[fieldname] = value.(float64)
					default:
						return id, errors.New(GetMessage("invalid_value") + ": " + fieldname + " (float)")
					}
				}

			case "date":
				if ((value == nil) || (value == "")) && field.NotNull == false {
					checkValues["values"].(IM)[fieldname] = nil
				} else {
					switch value.(type) {
					case time.Time:
						checkValues["values"].(IM)[fieldname] = value.(time.Time).Format(dateFmt)
					case string:
						tm, err := time.Parse(dateFmt, value.(string))
						if err != nil {
							tm, err = time.Parse(dateFmt, value.(string))
						}
						if err != nil {
							return id, errors.New(GetMessage("invalid_value") + ": " + fieldname + " (" + field.Type + ")")
						}
						checkValues["values"].(IM)[fieldname] = tm.Format(dateFmt)
					default:
						return id, errors.New(GetMessage("invalid_value") + ": " + fieldname + " (" + field.Type + ")")
					}
				}

			case "datetime":
				if ((value == nil) || (value == "")) && field.NotNull == false {
					checkValues["values"].(IM)[fieldname] = nil
				} else {
					switch value.(type) {
					case time.Time:
						checkValues["values"].(IM)[fieldname] = value.(time.Time).Format(dateFmt)
					case string:
						tm, err := time.Parse(datetimeISOFmt, value.(string))
						if err != nil {
							tm, err = time.Parse("2006-01-02T15:04:05", value.(string))
						}
						if err != nil {
							tm, err = time.Parse(TimeLayout, value.(string))
						}
						if err != nil {
							tm, err = time.Parse("2006-01-02 15:04", value.(string))
						}
						if err != nil {
							tm, err = time.Parse(dateFmt, value.(string))
						}
						if err != nil {
							tm, err = time.Parse(datetimeFmt, value.(string))
						}
						if err != nil {
							tm, err = time.Parse(dateFmt, value.(string))
						}
						if err != nil {
							return id, errors.New(GetMessage("invalid_value") + ": " + fieldname + " (" + field.Type + ")")
						}
						checkValues["values"].(IM)[fieldname] = tm.Format(datetimeISOFmt)
					default:
						return id, errors.New(GetMessage("invalid_value") + ": " + fieldname + " (" + field.Type + ")")
					}
				}

			case "password", "string", "text":
				if ((value == nil) || (value == "")) && field.NotNull && field.Default != nil {
					checkValues["values"].(IM)[fieldname] = strings.ReplaceAll(field.Default.(string), "'", "")
				} else if ((value == nil) || (value == "")) && field.NotNull == false {
					checkValues["values"].(IM)[fieldname] = nil
				} else {
					switch value.(type) {
					case int:
						checkValues["values"].(IM)[fieldname] = strconv.Itoa(value.(int))
					case float64:
						checkValues["values"].(IM)[fieldname] = strconv.FormatFloat(value.(float64), 'f', -1, 64)
					case bool:
						checkValues["values"].(IM)[fieldname] = strconv.FormatBool(value.(bool))
					case string:
						if field.Length > 0 && len(value.(string)) > field.Length {
							checkValues["values"].(IM)[fieldname] = value.(string)[:field.Length]
						}
					default:
						return id, errors.New(GetMessage("invalid_value") + ": " + fieldname + " (string)")
					}
				}
			}
			value = checkValues["values"].(IM)[fieldname]
			if value == nil && field.NotNull {
				return id, errors.New(GetMessage("missing_required_field") + ": " + fieldname)
			}
			if field.Requires != nil {
				_, minFound := field.Requires["min"]
				_, maxFound := field.Requires["max"]
				if minFound || maxFound {
					if _, found := field.Requires["min"]; found {
						switch value.(type) {
						case int:
							if value.(int) < field.Requires["min"].(int) {
								return id, errors.New(GetMessage("invalid_value") + " (min. value): " + fieldname)
							}
						case float64:
							if value.(float64) < field.Requires["min"].(float64) {
								return id, errors.New(GetMessage("invalid_value") + " (min. value): " + fieldname)
							}
						}
					}
					if _, found := field.Requires["max"]; found {
						switch value.(type) {
						case int:
							if value.(int) > field.Requires["max"].(int) {
								return id, errors.New(GetMessage("invalid_value") + " (max. value): " + fieldname)
							}
						case float64:
							if value.(float64) > field.Requires["max"].(float64) {
								return id, errors.New(GetMessage("invalid_value") + " (max. value): " + fieldname)
							}
						}
					}
				} else if _, found := field.Requires["bool"]; found {
					switch value {
					case "true", "True", "TRUE", "t", "T", "y", "YES", "yes", float64(1), int(1), "1":
						checkValues["values"].(IM)[fieldname] = 1
					default:
						checkValues["values"].(IM)[fieldname] = 0
					}
				} else if _, found := field.Requires["curr"]; found {
					if value == "" || value == nil {
						checkValues["values"].(IM)[fieldname] = nil
					} else {
						if GetIType(value) != "string" {
							return id, errors.New(GetMessage("invalid_value") + ": " + fieldname)
						} else if _, found := curr[value.(string)]; !found {
							return id, errors.New(GetMessage("invalid_value") + ": " + value.(string))
						}
					}
				} else {
					if GetIType(value) == "float64" {
						value = int(value.(float64))
					} else if GetIType(value) == "float32" {
						value = int(value.(float32))
					} else if GetIType(value) == "int64" {
						value = int(value.(int64))
					} else if GetIType(value) == "int32" {
						value = int(value.(int32))
					}
					if value == "" || value == nil {
						checkValues["values"].(IM)[fieldname] = nil
					} else if GetIType(value) != "int" {
						return id, errors.New(GetMessage("invalid_value") + ": " + fieldname)
					} else if _, found := groups[value.(int)]; !found {
						return id, errors.New(GetMessage("invalid_value") + ": " + fieldname)
					} else {
						var gvalid bool
						for req, rvalue := range field.Requires {
							if groups[value.(int)].(IM)["groupname"].(string) == req {
								gvalid = true
							} else if len(rvalue.(SL)) > 0 {
								for index := 0; index < len(rvalue.(SL)); index++ {
									if rvalue.(SL)[index] == groups[value.(int)].(IM)["groupvalue"].(string) {
										gvalid = true
									}
								}
							}
						}
						if gvalid == false {
							return id, errors.New(GetMessage("invalid_value") + ": " + fieldname)
						}
					}
				}
			}
		}
	}

	if len(checkValues["values"].(IM)) > 0 && nervatype == "fieldvalue" {
		//check_fieldvalue (fieldvalue table)
		if _, found := checkValues["values"].(IM)["fieldname"]; !found {
			return id, errors.New(GetMessage("missing_fieldname"))
		}
		if _, found := checkValues["values"].(IM)["value"]; !found {
			checkValues["values"].(IM)["value"] = ""
		}
		var value interface{}
		value, err = nstore.checkFieldvalue(checkValues["values"].(IM)["fieldname"].(string), checkValues["values"].(IM)["value"], trans)
		if err != nil {
			return id, err
		}
		checkValues["values"].(IM)["value"] = value
	}

	if len(checkValues["values"].(IM)) > 0 {
		//update data
		values := Update{Values: checkValues["values"].(IM), Model: nervatype, IDKey: id, Trans: trans}
		id, err = nstore.ds.Update(values)
		if err != nil {
			return id, err
		}
	}

	if len(checkValues["fvalues"].(IM)) > 0 {
		//update additional data
		//check_fieldvalue
		for fieldname, value := range checkValues["fvalues"].(IM) {
			var fieldID interface{}
			fieldIndex := int(1)
			fieldNotes := ""
			if strings.Index(fieldname, "~") > 1 {
				fieldIndex, err = strconv.Atoi(strings.Split(fieldname, "~")[1])
				if err != nil || fieldIndex < 1 {
					fieldIndex = 1
				}
				fieldname = strings.Split(fieldname, "~")[0]
			}
			if _, found := checkValues["dvalues"].(IM)[fieldname]; !found {
				if insertField == false {
					return id, errors.New(GetMessage("missing_insert_field"))
				}
				checkValues["deffield"] = append(checkValues["deffield"].([]IM), IM{
					"fieldname":   fieldname,
					"nervatype":   checkValues["dvalues"].(IM)["nervatype_id"].(IL)[0],
					"fieldtype":   checkValues["dvalues"].(IM)["fieldtype_string"].(IL)[0],
					"description": fieldname, "addnew": 0, "visible": 1, "readonly": 0})
			} else if len(checkValues["dvalues"].(IM)[fieldname].(IL)) >= fieldIndex {
				fieldID = checkValues["dvalues"].(IM)[fieldname].(IL)[fieldIndex-1].(int)
			}
			if GetIType(value) == "string" {
				if len(strings.Split(value.(string), "~")) > 1 {
					fieldNotes = strings.Split(value.(string), "~")[1]
					value = strings.Split(value.(string), "~")[0]
				}
			}
			fieldvalue := IM{"fieldname": fieldname, "ref_id": id, "value": value, "notes": fieldNotes}
			if fieldID != nil {
				fieldvalue["id"] = fieldID
			}
			checkValues["fieldvalue"] = append(checkValues["fieldvalue"].([]IM), fieldvalue)
		}
	}

	if len(checkValues["deffield"].([]IM)) > 0 {
		//create new deffields
		for index := 0; index < len(checkValues["deffield"].([]IM)); index++ {
			values := Update{Values: checkValues["deffield"].([]IM)[index], Model: "deffield", Trans: trans}
			id, err = nstore.ds.Update(values)
			if err != nil {
				return id, err
			}
		}
	}

	if len(checkValues["fieldvalue"].([]IM)) > 0 {
		//check_fieldvalue (!== fielvalue table)
		for index := 0; index < len(checkValues["fieldvalue"].([]IM)); index++ {
			fieldvalue := checkValues["fieldvalue"].([]IM)[index]
			var value interface{}
			value, err = nstore.checkFieldvalue(fieldvalue["fieldname"].(string), fieldvalue["value"], trans)
			if err != nil {
				return id, err
			}
			fieldvalue["value"] = value
			values := Update{Values: fieldvalue, Model: "fieldvalue", Trans: trans}
			_, err = nstore.ds.Update(values)
			if err != nil {
				return id, err
			}
		}
	}

	if logEnabled == true {
		err = nstore.insertLog(IM{"trans": trans, "logstate": "update", "nervatype": nervatype, "ref_id": id})
		if err != nil {
			return id, err
		}
	}

	return
}

// GetInfofromRefnumber - returns id and other info from public key
func (nstore *NervaStore) GetInfofromRefnumber(options IM) (IM, error) {

	var md1 = SM{"deffield": "fieldname", "employee": "empnumber",
		"pattern": "description", "project": "pronumber", "tool": "serial"}
	var md2 = SM{"currency": "curr", "numberdef": "numberkey", "ui_language": "lang",
		"ui_report": "reportkey", "ui_menu": "menukey"}
	var infoData = IM{"qkey": "refnumber->id", "nervatype": "", "refnumber": "",
		"useDeleted": "false", "extraInfo": "false"}
	var err error
	refIndex := 0

	if _, found := options["nervatype"]; !found || GetIType(options["nervatype"]) != "string" {
		return nil, errors.New(GetMessage("missing_nervatype"))
	}
	infoData["nervatype"] = options["nervatype"].(string)
	if _, found := nstore.models[infoData["nervatype"].(string)]; !found && infoData["nervatype"] != "setting" {
		return nil, errors.New(GetMessage("invalid_nervatype") + " " + infoData["nervatype"].(string))
	}

	if _, found := options["refnumber"]; !found || GetIType(options["refnumber"]) != "string" {
		return nil, errors.New(GetMessage("missing_fieldname") + ": refnumber")
	}
	infoData["refnumber"] = options["refnumber"].(string)

	if _, found := options["use_deleted"]; found && GetIType(options["use_deleted"]) == "bool" {
		if options["use_deleted"].(bool) {
			infoData["useDeleted"] = "true"
		}
	}

	if _, found := options["extra_info"]; found && GetIType(options["extra_info"]) == "bool" {
		if options["extra_info"].(bool) {
			infoData["extraInfo"] = "true"
		}
	}

	if ok, err := nstore.connected(); ok == false || err != nil {
		if err != nil {
			return nil, err
		}
		return nil, errors.New(GetMessage("not_connect"))
	}

	if _, found := md1[infoData["nervatype"].(string)]; found {
		infoData["refField"] = md1[infoData["nervatype"].(string)]
	} else if _, found := md2[infoData["nervatype"].(string)]; found {
		infoData["refField"] = md2[infoData["nervatype"].(string)]
		infoData["useDeleted"] = "true"
	} else {
		switch infoData["nervatype"] {
		case "barcode", "customer", "event", "product", "place", "tax", "trans", "setting":
			//custom query
		case "address", "contact":
			//ref_nervatype/refnumber~rownumber
			if strings.Index(infoData["refnumber"].(string), "/") > 1 {
				infoData["refType"] = strings.Split(infoData["refnumber"].(string), "/")[0]
				infoData["refnumber"] = infoData["refnumber"].(string)[len(infoData["refType"].(string))+1:]
				if len(strings.Split(infoData["refnumber"].(string), "~")) > 1 {
					refIndex, err = strconv.Atoi(strings.Split(infoData["refnumber"].(string), "~")[1])
					if err != nil {
						return nil, errors.New(GetMessage("invalid_refnumber"))
					}
					if refIndex < 1 {
						return nil, errors.New(GetMessage("invalid_refnumber"))
					}
					refIndex--
					infoData["refnumber"] = strings.Split(infoData["refnumber"].(string), "~")[0]
				}
				refTypes := []string{"customer", "employee", "event", "place", "product", "project", "tool", "trans"}
				for index := 0; index < len(refTypes); index++ {
					if refTypes[index] == infoData["refType"] {
						infoData["refField"] = nstore.getTableKey(infoData["refType"].(string))
						break
					}
				}
				if infoData["refField"] == "" {
					return nil, errors.New(GetMessage("invalid_refnumber"))
				}
			} else {
				return nil, errors.New(GetMessage("invalid_refnumber"))
			}

		case "groups":
			//groupname~groupvalue
			if strings.Index(infoData["refnumber"].(string), "~") > 1 {
				infoData["refType"] = strings.Split(infoData["refnumber"].(string), "~")[0]
				infoData["refnumber"] = strings.Split(infoData["refnumber"].(string), "~")[1]
			} else {
				return nil, errors.New(GetMessage("invalid_refnumber"))
			}

		case "fieldvalue":
			//refnumber~~fieldname~rownumber
			snum := strings.Split(infoData["refnumber"].(string), "~~")
			if len(snum) > 1 {
				fieldname := strings.Split(infoData["refnumber"].(string), "~~")[len(snum)-1]
				refnumber := strings.Replace(infoData["refnumber"].(string), "~~"+fieldname, "", 1)
				if len(strings.Split(fieldname, "~")) > 1 {
					refIndex, err = strconv.Atoi(strings.Split(fieldname, "~")[1])
					if err != nil {
						return nil, errors.New(GetMessage("invalid_refnumber"))
					}
					if refIndex < 1 {
						return nil, errors.New(GetMessage("invalid_refnumber"))
					}
					refIndex--
					fieldname = strings.Split(fieldname, "~")[0]
				}
				//get refNervatype
				info1, err := nstore.GetInfofromRefnumber(IM{"nervatype": "deffield", "refnumber": fieldname})
				if err != nil {
					return nil, errors.New(GetMessage("invalid_refnumber"))
				}
				//get refId
				info2, err := nstore.GetInfofromRefnumber(IM{"nervatype": info1["ref_nervatype"].(string),
					"refnumber": refnumber, "extra_info": false})
				if err != nil {
					return nil, errors.New(GetMessage("invalid_refnumber"))
				}
				infoData["refID"] = strconv.Itoa(info2["id"].(int))
				infoData["refnumber"] = fieldname
			} else {
				//setting
				infoData["refnumber"] = "setting"
			}

		case "item", "payment", "movement":
			//refnumber~rownumber
			if len(strings.Split(infoData["refnumber"].(string), "~")) > 1 {
				refIndex, err = strconv.Atoi(strings.Split(infoData["refnumber"].(string), "~")[1])
				if err != nil {
					return nil, errors.New(GetMessage("invalid_refnumber"))
				}
				if refIndex < 1 {
					return nil, errors.New(GetMessage("invalid_refnumber"))
				}
				refIndex--
				infoData["refnumber"] = strings.Split(infoData["refnumber"].(string), "~")[0]
			}

		case "price":
			//partnumber~validfrom~curr~qty -> def. price
			//partnumber~pricetype~validfrom~curr~qty
			if len(strings.Split(infoData["refnumber"].(string), "~")) == 5 || len(strings.Split(infoData["refnumber"].(string), "~")) == 4 {
				var qtyStr string
				if len(strings.Split(infoData["refnumber"].(string), "~")) == 4 {
					infoData["pricetype"] = "price"
					infoData["validfrom"] = strings.Split(infoData["refnumber"].(string), "~")[1]
					infoData["curr"] = strings.Split(infoData["refnumber"].(string), "~")[2]
					qtyStr = strings.Split(infoData["refnumber"].(string), "~")[3]
				} else {
					infoData["pricetype"] = strings.Split(infoData["refnumber"].(string), "~")[1]
					infoData["validfrom"] = strings.Split(infoData["refnumber"].(string), "~")[2]
					infoData["curr"] = strings.Split(infoData["refnumber"].(string), "~")[3]
					qtyStr = strings.Split(infoData["refnumber"].(string), "~")[4]
				}
				qty, err := strconv.Atoi(qtyStr)
				if err != nil {
					return nil, errors.New(GetMessage("invalid_refnumber"))
				}
				infoData["qty"] = strconv.Itoa(qty)
				infoData["refnumber"] = strings.Split(infoData["refnumber"].(string), "~")[0]
			} else {
				return nil, errors.New(GetMessage("invalid_refnumber"))
			}

		case "link":
			//nervatype_1~refnumber_1~~nervatype_2~refnumber_2
			if len(strings.Split(infoData["refnumber"].(string), "~~")) > 1 {
				infoData["refType1"] = strings.Split(strings.Split(infoData["refnumber"].(string), "~~")[0], "~")[0]
				infoData["refValue1"] = strings.Replace(strings.Split(infoData["refnumber"].(string), "~~")[0], infoData["refType1"].(string)+"~", "", 1)
				info1, err := nstore.GetInfofromRefnumber(IM{"nervatype": infoData["refType1"], "refnumber": infoData["refValue1"]})
				if err != nil {
					return nil, errors.New(GetMessage("invalid_refnumber"))
				}
				infoData["refID1"] = strconv.Itoa(info1["id"].(int))
				infoData["refType2"] = strings.Split(strings.Split(infoData["refnumber"].(string), "~~")[1], "~")[0]
				infoData["refValue2"] = strings.Replace(strings.Split(infoData["refnumber"].(string), "~~")[1], infoData["refType2"].(string)+"~", "", 1)
				info2, err := nstore.GetInfofromRefnumber(IM{"nervatype": infoData["refType2"], "refnumber": infoData["refValue2"]})
				if err != nil {
					return nil, errors.New(GetMessage("invalid_refnumber"))
				}
				infoData["refID2"] = strconv.Itoa(info2["id"].(int))
			} else {
				return nil, errors.New(GetMessage("invalid_refnumber"))
			}

		case "rate":
			//ratetype~ratedate~curr~planumber
			if len(strings.Split(infoData["refnumber"].(string), "~")) >= 3 {
				infoData["ratetype"] = strings.Split(infoData["refnumber"].(string), "~")[0]
				infoData["ratedate"] = strings.Split(infoData["refnumber"].(string), "~")[1]
				infoData["curr"] = strings.Split(infoData["refnumber"].(string), "~")[2]
				if len(strings.Split(infoData["refnumber"].(string), "~")) > 3 {
					infoData["planumber"] = strings.Split(infoData["refnumber"].(string), "~")[3]
				}
			} else {
				return nil, errors.New(GetMessage("invalid_refnumber"))
			}

		case "log":
			//empnumber~crdate
			if len(strings.Split(infoData["refnumber"].(string), "~")) > 1 {
				infoData["crdate"] = strings.Split(infoData["refnumber"].(string), "~")[1]
				infoData["refnumber"] = strings.Split(infoData["refnumber"].(string), "~")[0]
			} else {
				return nil, errors.New(GetMessage("invalid_refnumber"))
			}

		case "ui_audit":
			//usergroup~nervatype~transtype
			if len(strings.Split(infoData["refnumber"].(string), "~")) > 1 {
				infoData["refType"] = strings.Split(infoData["refnumber"].(string), "~")[1]
				if len(strings.Split(infoData["refnumber"].(string), "~")) > 2 {
					infoData["transType"] = strings.Split(infoData["refnumber"].(string), "~")[2]
					if infoData["transType"] != "trans" && infoData["transType"] != "report" {
						return nil, errors.New(GetMessage("invalid_refnumber"))
					}
				}
				infoData["usergroup"] = strings.Split(infoData["refnumber"].(string), "~")[0]
			} else {
				return nil, errors.New(GetMessage("invalid_refnumber"))
			}

		case "ui_menufields", "ui_reportfields":
			//menukey~fieldname
			//reportkey~fieldname
			if len(strings.Split(infoData["refnumber"].(string), "~")) > 1 {
				infoData["fieldname"] = strings.Split(infoData["refnumber"].(string), "~")[1]
				infoData["refnumber"] = strings.Split(infoData["refnumber"].(string), "~")[0]
			} else {
				return nil, errors.New(GetMessage("invalid_refnumber"))
			}

		case "ui_reportsources":
			//reportkey~dataset
			if len(strings.Split(infoData["refnumber"].(string), "~")) > 1 {
				infoData["dataset"] = strings.Split(infoData["refnumber"].(string), "~")[1]
				infoData["refnumber"] = strings.Split(infoData["refnumber"].(string), "~")[0]
			} else {
				return nil, errors.New(GetMessage("invalid_refnumber"))
			}

		default:
			return nil, errors.New(GetMessage("invalid_refnumber"))
		}
	}

	rows, err := nstore.ds.QueryKey(infoData, nil)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 || refIndex > len(rows)-1 {
		return nil, errors.New(GetMessage("invalid_refnumber"))
	}

	info := IM{"id": rows[refIndex]["id"]}

	switch infoData["nervatype"] {

	case "customer":
		if infoData["extraInfo"] == "true" && len(rows) < 2 {
			return nil, errors.New(GetMessage("invalid_refnumber"))
		}
		if infoData["extraInfo"] == "true" {
			for index := 0; index < len(rows); index++ {
				zipcode := ""
				if rows[index]["zipcode"] != nil {
					zipcode = rows[index]["zipcode"].(string)
				}
				city := ""
				if rows[index]["city"] != nil {
					city = rows[index]["city"].(string)
				}
				street := ""
				if rows[index]["street"] != nil {
					street = rows[index]["street"].(string)
				}
				if rows[index]["custtype"] == "own" {
					info["compname"] = rows[index]["custname"]
					info["comptax"] = rows[index]["taxnumber"]
					info["compaddress"] = zipcode + city + street
				} else {
					info["id"] = rows[index]["id"]
					info["custtype"] = rows[index]["custtype"]
					info["terms"] = rows[index]["terms"]
					info["custname"] = rows[index]["custname"]
					info["custtax"] = rows[index]["taxnumber"]
					info["custaddress"] = zipcode + city + street
				}
			}
		} else {
			info["custtype"] = rows[refIndex]["custtype"]
		}

	case "deffield":
		info["ref_nervatype"] = rows[refIndex]["ref_nervatype"]

	case "product":
		info["description"] = rows[refIndex]["description"]
		info["unit"] = rows[refIndex]["unit"]
		info["tax_id"] = rows[refIndex]["tax_id"]
		info["rate"] = rows[refIndex]["rate"]

	case "place":
		info["placetype"] = rows[refIndex]["placetype"]

	case "event":
		info["ref_nervatype"] = rows[refIndex]["ref_nervatype"]

	case "tax":
		info["rate"] = rows[refIndex]["rate"]

	case "item":
		info["transtype"] = rows[refIndex]["transtype"]
		info["direction"] = rows[refIndex]["direction"]
		info["digit"] = rows[refIndex]["digit"]
		info["qty"] = rows[refIndex]["qty"]
		info["discount"] = rows[refIndex]["discount"]
		info["tax_id"] = rows[refIndex]["tax_id"]
		info["rate"] = rows[refIndex]["rate"]

	case "movement":
		info["movetype"] = rows[refIndex]["movetype"]
		info["transtype"] = rows[refIndex]["transtype"]
		info["direction"] = rows[refIndex]["direction"]

	case "trans":
		info["transtype"] = rows[refIndex]["transtype"]
		info["direction"] = rows[refIndex]["direction"]
		info["digit"] = rows[refIndex]["digit"]

	case "payment":
		info["transtype"] = rows[refIndex]["transtype"]
		info["direction"] = rows[refIndex]["direction"]

	}

	return info, nil
}

// DeleteData - delete a record data
func (nstore *NervaStore) DeleteData(options IM) (err error) {

	var nervatype string
	if _, found := options["nervatype"]; !found || GetIType(options["nervatype"]) != "string" {
		return errors.New(GetMessage("missing_nervatype"))
	}
	nervatype = options["nervatype"].(string)
	if _, found := nstore.models[nervatype]; !found {
		return errors.New(GetMessage("invalid_nervatype") + " " + nervatype)
	}

	var refID int
	if _, found := options["ref_id"]; found && GetIType(options["ref_id"]) == "int" {
		refID = options["ref_id"].(int)
	}
	var refnumber string
	if _, found := options["refnumber"]; found && GetIType(options["refnumber"]) == "string" {
		refnumber = options["refnumber"].(string)
	}
	if refID == 0 && refnumber == "" {
		return errors.New(GetMessage("missing_fieldname") + ": ref_id or refnumber")
	}
	if refID == 0 {
		info, err := nstore.GetInfofromRefnumber(IM{"nervatype": nervatype, "refnumber": refnumber})
		if err != nil {
			return err
		}
		refID = info["id"].(int)
	}
	logEnabled := true
	if _, found := options["log_enabled"]; found && GetIType(options["log_enabled"]) == "bool" {
		logEnabled = options["log_enabled"].(bool)
	}

	if ok, err := nstore.connected(); ok == false || err != nil {
		if err != nil {
			return err
		}
		return errors.New(GetMessage("not_connect"))
	}

	//check integrity
	switch nervatype {
	case "address", "barcode", "contact", "event", "fieldvalue", "item", "link", "log",
		"movement", "pattern", "payment", "price", "rate":
	case "numberdef":
		//protected, always false
		return errors.New(GetMessage("integrity_error"))
	case "currency", "customer", "deffield", "employee", "groups", "place", "product", "project", "tax", "tool", "trans":
		rows, err := nstore.ds.QueryKey(IM{"qkey": "integrity", "nervatype": nervatype, "ref_id": refID}, nil)
		if err != nil {
			return err
		}
		if len(rows) > 0 {
			if rows[0]["count"].(int) > 0 {
				return errors.New(GetMessage("integrity_error"))
			}
		}
	default:
		if nervatype[:3] != "ui_" {
			return errors.New(GetMessage("integrity_error"))
		}
	}

	//check logical delete
	logicalDelete := true
	if _, found := nstore.models[nervatype].(IM)["deleted"]; !found {
		logicalDelete = false
	}
	if logicalDelete {
		query := []Query{{
			Fields: []string{"value"},
			From:   "fieldvalue", Filters: []Filter{
				{Field: "ref_id", Comp: "is", Value: "null"},
				{Field: "fieldname", Comp: "==", Value: "not_logical_delete"}}}}
		result, err := nstore.ds.Query(query, nil)
		if err != nil {
			return err
		}
		if len(result) > 0 {
			if result[0]["value"] == "true" {
				logicalDelete = false
			}
		}
	}

	var trans interface{}
	if _, found := options["trans"]; found {
		trans = options["trans"]
	} else if nstore.ds.Properties().Transaction {
		trans, err = nstore.ds.BeginTransaction()
		if err != nil {
			return err
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

	var data Update
	if logicalDelete {
		data = Update{Values: IM{"deleted": 1}, IDKey: refID, Model: nervatype, Trans: trans}
	} else {
		data = Update{IDKey: refID, Model: nervatype, Trans: trans}
	}
	_, err = nstore.ds.Update(data)
	if err != nil {
		return err
	}

	if logicalDelete == false {
		//delete all fieldvalue records
		result, err := nstore.ds.QueryKey(IM{"qkey": "delete_deffields", "nervatype": nervatype, "ref_id": refID}, trans)
		if err != nil {
			return err
		}
		for index := 0; index < len(result); index++ {
			data = Update{Model: "fieldvalue", Trans: trans}
			switch result[index]["id"].(type) {
			case int:
				data.IDKey = result[index]["id"].(int)
			case float64:
				data.IDKey = int(result[index]["id"].(float64))
			case string:
				IDKey, err := strconv.Atoi(result[index]["id"].(string))
				if err == nil {
					data.IDKey = IDKey
				}
			}
			if data.IDKey > 0 {
				_, err = nstore.ds.Update(data)
				if err != nil {
					return err
				}
			}
		}
	}

	if logEnabled == true {
		//insert log
		err := nstore.insertLog(IM{"trans": trans, "logstate": "delete", "nervatype": nervatype, "ref_id": refID})
		if err != nil {
			return err
		}
	}

	return nil
}

//GetRefnumber - returns public key from id
func (nstore *NervaStore) GetRefnumber(options IM) (IM, error) {

	info := IM{"index": 1}
	var infoData = IM{"qkey": "id->refnumber", "nervatype": "", "id": "",
		"useDeleted": "false", "retfield": ""}
	if _, found := options["nervatype"]; !found || GetIType(options["nervatype"]) != "string" {
		return nil, errors.New(GetMessage("missing_nervatype"))
	}
	infoData["nervatype"] = options["nervatype"].(string)
	if _, found := nstore.models[infoData["nervatype"].(string)]; !found && infoData["nervatype"] != "setting" {
		return nil, errors.New(GetMessage("invalid_nervatype") + " " + infoData["nervatype"].(string))
	}

	if _, found := options["ref_id"]; !found || GetIType(options["ref_id"]) != "int" {
		return nil, errors.New(GetMessage("missing_fieldname") + ": ref_id")
	}
	infoData["id"] = strconv.Itoa(options["ref_id"].(int))

	if _, found := options["retfield"]; found && GetIType(options["retfield"]) == "string" {
		infoData["retfield"] = options["retfield"].(string)
	}

	if _, found := options["use_deleted"]; found && GetIType(options["use_deleted"]) == "bool" {
		if options["use_deleted"].(bool) {
			infoData["useDeleted"] = "true"
		}
	}
	if _, found := nstore.models[infoData["nervatype"].(string)]; found {
		if _, found := nstore.models[infoData["nervatype"].(string)].(IM)["deleted"]; !found {
			infoData["useDeleted"] = "true"
		}
	}

	if ok, err := nstore.connected(); ok == false || err != nil {
		if err != nil {
			return nil, err
		}
		return nil, errors.New(GetMessage("not_connect"))
	}

	rows, err := nstore.ds.QueryKey(infoData, nil)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New(GetMessage("invalid_refnumber"))
	}
	if _, found := rows[0][infoData["retfield"].(string)]; found {
		info[infoData["retfield"].(string)] = rows[0][infoData["retfield"].(string)]
	}

	switch infoData["nervatype"] {
	case "address", "contact":
		//ref_nervatype/refnumber~rownumber
		info["headNervatype"] = rows[0]["head_nervatype"]
		info["refId"] = rows[0]["ref_id"].(int)
		infoData["refTypeId"] = strconv.Itoa(rows[0]["nervatype"].(int))
		infoData["refId"] = strconv.Itoa(rows[0]["ref_id"].(int))
		rows, err := nstore.ds.QueryKey(infoData, nil)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return nil, errors.New(GetMessage("invalid_refnumber"))
		}
		info["index"] = rows[0]["count"]
		head, err := nstore.GetRefnumber(IM{
			"nervatype": info["headNervatype"], "ref_id": info["refId"],
			"use_deleted": (infoData["useDeleted"] == "true")})
		if err != nil {
			return nil, err
		}
		info["refnumber"] = info["headNervatype"].(string) + "/" + head["refnumber"].(string) + "~" + strconv.Itoa(info["index"].(int))

	case "fieldvalue", "setting":
		//refnumber~~fieldname~rownumber
		info["headNervatype"] = rows[0]["head_nervatype"]
		if rows[0]["ref_id"] == nil && info["headNervatype"].(string) == "setting" {
			info["refnumber"] = rows[0]["fieldname"]
		} else {
			info["refId"] = rows[0]["ref_id"].(int)
			infoData["fieldname"] = rows[0]["fieldname"].(string)
			infoData["refId"] = strconv.Itoa(rows[0]["ref_id"].(int))
			rows, err := nstore.ds.QueryKey(infoData, nil)
			if err != nil {
				return nil, err
			}
			if len(rows) == 0 {
				return nil, errors.New(GetMessage("invalid_refnumber"))
			}
			info["index"] = rows[0]["count"]
			if infoData["retfield"] == "fieldname" {
				info[infoData["retfield"].(string)] = infoData["fieldname"].(string) + "~" + strconv.Itoa(info["index"].(int))
			}
			head, err := nstore.GetRefnumber(IM{
				"nervatype": info["headNervatype"], "ref_id": info["refId"],
				"use_deleted": (infoData["useDeleted"] == "true")})
			if err != nil {
				return nil, err
			}
			info["refnumber"] = head["refnumber"].(string) + "~" + infoData["fieldname"].(string) + "~" + strconv.Itoa(info["index"].(int))
		}

	case "groups":
		//groupname~groupvalue
		info["refnumber"] = rows[0]["groupname"].(string) + "~" + rows[0]["groupvalue"].(string)

	case "item", "payment", "movement":
		//refnumber~rownumber
		info["transnumber"] = rows[0]["transnumber"]
		infoData["refId"] = strconv.Itoa(rows[0]["trans_id"].(int))
		rows, err := nstore.ds.QueryKey(infoData, nil)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return nil, errors.New(GetMessage("invalid_refnumber"))
		}
		info["index"] = rows[0]["count"]
		info["refnumber"] = info["transnumber"].(string) + "~" + strconv.Itoa(info["index"].(int))

	case "price":
		//partnumber~pricetype~validfrom~curr~qty
		pricetype := "price"
		if rows[0]["discount"] != nil {
			pricetype = "discount"
		}
		validfrom := rows[0]["validfrom"].(string)[:10]
		info["refnumber"] = rows[0]["partnumber"].(string) + "~" + pricetype + "~" + validfrom + "~" + rows[0]["curr"].(string) + "~" + strconv.FormatFloat(rows[0]["qty"].(float64), 'f', -1, 64)

	case "link":
		//nervatype_1~refnumber_1~~nervatype_2~refnumber_2
		refnumber1, err := nstore.GetRefnumber(IM{
			"nervatype": rows[0]["nervatype1"].(string), "ref_id": rows[0]["ref_id_1"].(int),
			"use_deleted": (infoData["useDeleted"] == "true")})
		if err != nil {
			return nil, err
		}
		refnumber2, err := nstore.GetRefnumber(IM{
			"nervatype": rows[0]["nervatype2"].(string), "ref_id": rows[0]["ref_id_2"].(int),
			"use_deleted": (infoData["useDeleted"] == "true")})
		if err != nil {
			return nil, err
		}
		info["refnumber"] = rows[0]["nervatype1"].(string) + "~" + refnumber1["refnumber"].(string) + "~~" + rows[0]["nervatype2"].(string) + "~" + refnumber2["refnumber"].(string)

	case "rate":
		//ratetype~ratedate~curr~planumber
		ratedate := rows[0]["ratedate"].(string)[:10]
		info["refnumber"] = rows[0]["rate_type"].(string) + "~" + ratedate + "~" + rows[0]["rate_type"].(string)
		if rows[0]["planumber"] != nil {
			info["refnumber"] = info["refnumber"].(string) + "~" + rows[0]["planumber"].(string)
		}

	case "log":
		//empnumber~crdate
		info["refnumber"] = rows[0]["empnumber"].(string) + "~" + rows[0]["crdate"].(string)

	default:
		//table_key
		info["refnumber"] = rows[0][nstore.getTableKey(infoData["nervatype"].(string))]
	}

	return info, err
}

//GetDataAudit - Nervatura data access rights: own,usergroup,all (transfilter)
//see more: employee.usergroup+link+transfilter
func (nstore *NervaStore) GetDataAudit() (string, error) {
	result := "own"
	if nstore.User == nil {
		return result, errors.New(GetMessage("invalid_login"))
	}
	if nstore.User["usergroup"] == nil {
		return result, errors.New(GetMessage("missing_usergroup"))
	}

	if ok, err := nstore.connected(); ok == false || err != nil {
		if err != nil {
			return result, err
		}
		return result, errors.New(GetMessage("not_connect"))
	}

	rows, err := nstore.ds.QueryKey(IM{"qkey": "data_audit",
		"id": strconv.Itoa(nstore.User["id"].(int))}, nil)
	if err != nil {
		return result, err
	}
	if len(rows) == 0 {
		result = "all"
	} else {
		result = rows[0]["transfilter"].(string)
	}

	return result, nil
}

//GetObjectAudit - Nervatura objects access rights: disabled,readonly,update,all (inputfilter)
//see more: audit
func (nstore *NervaStore) GetObjectAudit(options IM) ([]string, error) {
	result := []string{"disabled", ""}
	if nstore.User == nil {
		return result, errors.New(GetMessage("invalid_login"))
	}
	if nstore.User["usergroup"] == nil {
		return result, errors.New(GetMessage("missing_usergroup"))
	}
	if _, found := options["nervatype"]; !found {
		if _, found := options["transtype"]; !found {
			return result, errors.New(GetMessage("missing_nervatype"))
		}
	}

	if options["nervatype"] == "sql" || options["nervatype"] == "fieldvalue" {
		result[0] = "all"
		return result, nil
	}

	if ok, err := nstore.connected(); ok == false || err != nil {
		if err != nil {
			return result, err
		}
		return result, errors.New(GetMessage("not_connect"))
	}

	var infoData = IM{"qkey": "object_audit", "usergroup": nstore.User["usergroup"]}
	if _, found := options["transtype_id"]; found {
		switch options["transtype_id"].(type) {
		case int:
			infoData["subtype"] = strconv.Itoa(options["transtype_id"].(int))
		case []int:
			infoData["subtypeIn"] = ""
			for index := 0; index < len(options["transtype_id"].([]int)); index++ {
				infoData["subtypeIn"] = infoData["subtypeIn"].(string) + "," + strconv.Itoa(options["transtype_id"].([]int)[index])
			}
			if infoData["subtypeIn"] != "" {
				infoData["subtypeIn"] = infoData["subtypeIn"].(string)[1:]
			}
		}
	}
	if _, found := options["transtype"]; found {
		switch options["transtype"].(type) {
		case string:
			infoData["transtype"] = options["transtype"].(string)
			result[1] = infoData["transtype"].(string)
		case []string:
			infoData["transtypeIn"] = ""
			for index := 0; index < len(options["transtype"].([]string)); index++ {
				infoData["transtypeIn"] = infoData["transtypeIn"].(string) + ",'" + options["transtype"].([]string)[index] + "'"
			}
			if infoData["transtypeIn"] != "" {
				infoData["transtypeIn"] = infoData["transtypeIn"].(string)[1:]
			}
			result[1] = strings.ReplaceAll(infoData["transtypeIn"].(string), "'", "")
		}
	}
	if _, found := options["nervatype_id"]; found {
		switch options["nervatype_id"].(type) {
		case int:
			infoData["nervatype"] = strconv.Itoa(options["nervatype_id"].(int))
		case []int:
			infoData["nervatypeIn"] = ""
			for index := 0; index < len(options["nervatype_id"].([]int)); index++ {
				infoData["nervatypeIn"] = infoData["nervatypeIn"].(string) + "," + strconv.Itoa(options["nervatype_id"].([]int)[index])
			}
			if infoData["nervatypeIn"] != "" {
				infoData["nervatypeIn"] = infoData["nervatypeIn"].(string)[1:]
			}
		}
	}
	if _, found := options["nervatype"]; found {
		switch options["nervatype"].(type) {
		case string:
			infoData["groupvalue"] = options["nervatype"].(string)
			result[1] = infoData["groupvalue"].(string)
		case []string:
			infoData["groupvalueIn"] = ""
			for index := 0; index < len(options["nervatype"].([]string)); index++ {
				infoData["groupvalueIn"] = infoData["groupvalueIn"].(string) + ",'" + options["nervatype"].([]string)[index] + "'"
			}
			if infoData["groupvalueIn"] != "" {
				infoData["groupvalueIn"] = infoData["groupvalueIn"].(string)[1:]
			}
			result[1] = strings.ReplaceAll(infoData["groupvalueIn"].(string), "'", "")
		}
	}

	rows, err := nstore.ds.QueryKey(infoData, nil)
	if err != nil {
		return result, err
	}
	if len(rows) == 0 {
		result[0] = "all"
		return result, nil
	}

	result[0] = "all"
	for index := 0; index < len(rows); index++ {
		switch rows[index]["inputfilter"] {
		case "disabled":
			result[0] = "disabled"
		case "readonly":
			if result[0] != "disabled" {
				result[0] = "readonly"
			}
		case "update":
			if result[0] == "all" {
				result[0] = "update"
			}
		}
	}
	return result, nil
}

//GetGroups - returns Nervatura groups map
func (nstore *NervaStore) GetGroups(options IM) (IM, error) {
	groups := IM{"all": []IM{}}

	if ok, err := nstore.connected(); ok == false || err != nil {
		if err != nil {
			return groups, err
		}
		return groups, errors.New(GetMessage("not_connect"))
	}

	filters :=
		[]Filter{
			{Field: "deleted", Comp: "==", Value: 0},
		}
	if _, found := options["groupname"]; found {
		switch options["groupname"].(type) {
		case string:
			filters = append(filters, Filter{Field: "groupname", Comp: "==", Value: options["groupname"]})
		case []string:
			filters = append(filters, Filter{Field: "groupname", Comp: "in",
				Value: strings.Join(options["groupname"].([]string), ",")})
		}
	}
	query := []Query{{
		Fields: []string{"*"}, From: "groups",
		Filters: filters}}
	results, err := nstore.ds.Query(query, nil)
	if err != nil {
		return groups, err
	}
	groups["all"] = results
	for index := 0; index < len(results); index++ {
		groupname := results[index]["groupname"].(string)
		if _, found := groups[groupname]; !found {
			groups[groupname] = IM{}
		}
		groupvalue := results[index]["groupvalue"].(string)
		groups[groupname].(IM)[groupvalue] = results[index]["id"].(int)
	}

	return groups, nil
}

//GetDatabaseSettings - returns general database settings and document patterns
func (nstore *NervaStore) GetDatabaseSettings() (IM, error) {
	settings := IM{"setting": IM{}, "pattern": IM{}}

	if ok, err := nstore.connected(); ok == false || err != nil {
		if err != nil {
			return settings, err
		}
		return settings, errors.New(GetMessage("not_connect"))
	}

	rows, err := nstore.ds.QueryKey(IM{"qkey": "dbs_settings"}, nil)
	if err != nil {
		return settings, err
	}
	for index := 0; index < len(rows); index++ {
		if rows[index]["stype"] == "setting" {
			settings["setting"].(IM)[rows[index]["fieldname"].(string)] = IM{"value": rows[index]["value"], "data": rows[index]["data"]}
		} else {
			data := rows[index]["data"].(string)
			if _, found := settings["pattern"].(IM)[data]; !found {
				settings["pattern"].(IM)[data] = []IM{}
			}
			if rows[index]["info"] == 1 {
				settings["pattern"].(IM)[data] = append([]IM{{
					"description": rows[index]["fieldname"], "notes": rows[index]["value"], "defpattern": false}},
					settings["pattern"].(IM)[data].([]IM)...)
			} else {
				settings["pattern"].(IM)[data] = append(settings["pattern"].(IM)[data].([]IM), IM{
					"description": rows[index]["fieldname"], "notes": rows[index]["value"], "defpattern": false})
			}
		}
	}

	return settings, nil
}
