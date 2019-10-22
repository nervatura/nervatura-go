package nervatura

import (
	"errors"
	"strconv"
	"strings"
)

/*
Npi - Nervatura Programming Interface

Deprecated.

*/
type Npi struct {
	NStore *NervaStore
}

/*
GetLogin - extended NervaStore database login. Successful login additional information is available.
 • database: an NAS alias set for a database
 • username, password: valid username and password for the database

 The result value is a (map[string]interface{}):
 • valid: true or false
 • message: '' or error message
 If valid == true:
 • token: JSON Web Token
 • engine: Database type (sqlite,mysql,postgres,mssql,...)
 • employee: Login user data
 • audit: Login user audit data
 • transfilter: Login user transfilter data or null
 • groups: All groups values: usergroup, nervatype, transtype, inputfilter, transfilter, department, logstate, fieldtype
 • menucmd: All ui_menu data
 • menufields: All ui_menufields
 • userlogin: 'true' or 'false' from database settings
*/
func (npi *Npi) GetLogin(options IM) (IM, error) {

	login := options
	if _, found := options["login"]; found && GetIType(options["login"]) == "map[string]interface{}" {
		login = options["login"].(IM)
	}

	tokenString, err := (&API{NStore: npi.NStore}).AuthUserLogin(login)
	if err != nil {
		return IM{"valid": false, "message": err.Error()}, err
	}
	validator := IM{
		"valid":    true,
		"employee": npi.NStore.User,
		"token":    SM{"token": tokenString},
		"engine":   npi.NStore.ds.Connection().Engine,
	}

	groups, err := npi.NStore.GetGroups(IM{
		"groupname": []string{"nervatype", "usergroup", "nervatype", "transtype", "inputfilter",
			"transfilter", "department", "logstate", "fieldtype"},
	})
	if err != nil {
		validator["valid"] = false
		return validator, err
	}
	validator["groups"] = groups["all"]

	if validator["employee"] != nil {
		query := []Query{Query{
			Fields: []string{"*"}, From: "ui_audit",
			Filters: []Filter{
				Filter{Field: "usergroup", Comp: "==",
					Value: strconv.Itoa(validator["employee"].(IM)["usergroup"].(int))},
			}}}
		results, err := npi.NStore.ds.Query(query, nil)
		if err != nil {
			validator["valid"] = false
			return validator, err
		}
		validator["audit"] = results

		query = []Query{Query{
			Fields: []string{"ref_id_2"}, From: "link",
			Filters: []Filter{
				Filter{Field: "deleted", Comp: "==", Value: "0"},
				Filter{Field: "nervatype_1", Comp: "==",
					Value: strconv.Itoa(groups["nervatype"].(IM)["groups"].(int))},
				Filter{Field: "nervatype_2", Comp: "==",
					Value: strconv.Itoa(groups["nervatype"].(IM)["groups"].(int))},
			}}}
		results, err = npi.NStore.ds.Query(query, nil)
		if err != nil {
			validator["valid"] = false
			return validator, err
		}
		if len(results) > 0 {
			validator["transfilter"] = results[0]["ref_id_2"]
		} else {
			validator["transfilter"] = nil
		}
	}

	query := []Query{Query{
		Fields: []string{"*"}, From: "ui_menu"}}
	results, err := npi.NStore.ds.Query(query, nil)
	if err != nil {
		validator["valid"] = false
		return validator, err
	}
	validator["menucmd"] = results

	query = []Query{Query{
		Fields: []string{"*"}, From: "ui_menufields"}}
	results, err = npi.NStore.ds.Query(query, nil)
	if err != nil {
		validator["valid"] = false
		return validator, err
	}
	validator["menufields"] = results

	query = []Query{Query{
		Fields: []string{"value"}, From: "fieldvalue",
		Filters: []Filter{
			Filter{Field: "ref_id", Comp: "is", Value: "null"},
			Filter{Field: "fieldname", Comp: "==", Value: "'log_login'"},
		}}}
	results, err = npi.NStore.ds.Query(query, nil)
	if err != nil {
		validator["valid"] = false
		return validator, err
	}
	if len(results) > 0 {
		validator["userlogin"] = results[0]["value"]
	} else {
		validator["userlogin"] = "false"
	}

	return validator, nil
}

/*
SetData - general low level functions. The method parameters:
 • function - It calls for the functionName server-side function.
   The paramList (map[string]interface{}) name/value parameters are passed.
   The result map["data"] for the called function result.
 • update - Insert or update a data row. A new record, set the returned id.
   The result map["data"] value of the input record map.
 • delete - Delete a data row.
   The result map["data"] value: input values (if the deletion is successful).
 • table - Loads data from a database table.
   - tableName: a table name
   - filters: []Filter fields
   - filterStr: the valid sql having or where string.
   - order([]string) or orderStr(string): The valid sql orderby string (eg. field1,field2)
   The result map["data"]: table rows.
 • view - Loads the sqlStr query from the database.
   - whereStr: the valid sql where filter string
   - havingStr: the valid sql having filter string
   - paramList: the []interface{} of
   •• name: the replacement name
   •• value: replacement value of the variable
   •• wheretype: specifies which part replace the variable. Valid values: where, having, in.
   •• type: set up the correct sql syntax. Valid values: string, date, integer, number, boolean.
   - orderStr: The valid sql orderby string.
   The result map["data"] for the execute query result.
 • execute - Execute the
   - sqlStr statement on the database.
   - paramList: the []interface{} of
   •• name: the replacement name
   •• value: replacement value of the variable
   •• wheretype: specifies which part replace the variable. Valid values: where, having, in.
   •• type: set up the correct sql syntax. Valid values: string, date, integer, number, boolean.
   The result map["data"] of the SQL statement.

Example:

 options = map[string]interface{}{
   "method":   "view",
   "sqlStr":   "select * from customer where deleted=0 @where_str order by @orderby_str",
   "whereStr": "and creditlimit>=@limit and custnumber like @custnumber",
   "orderStr": "custtype,id",
   "paramList": []interface{}{
	   map[string]interface{}{"name": "@limit", "value": 0, "wheretype": "where", "type": "number"},
	   map[string]interface{}{"name": "@custnumber", "value": "HOME", "wheretype": "where", "type": "string"},
 }}
 result, err = npi.SetData(options)
*/
func (npi *Npi) SetData(options IM) (IM, error) {
	data := IM{}

	if _, found := options["method"]; !found || GetIType(options["method"]) != "string" {
		return data, errors.New(GetMessage("missing_required_field") + ": method")
	}

	if !npi.NStore.ds.Connection().Connected {
		return data, errors.New(GetMessage("not_connect"))
	}

	switch options["method"] {
	case "function":
		if _, found := options["functionName"]; !found || GetIType(options["functionName"]) != "string" {
			return nil, errors.New(GetMessage("missing_required_field") + ": functionName")
		}
		if _, found := options["paramList"]; !found || GetIType(options["paramList"]) != "map[string]interface{}" {
			return nil, errors.New(GetMessage("missing_required_field") + ": paramList")
		}
		results, err := npi.NStore.GetService(options["functionName"].(string), options["paramList"].(IM))
		if err != nil {
			return data, err
		}
		data["data"] = results

	case "update":
		if _, found := options["record"]; !found {
			return data, errors.New(GetMessage("missing_required_field") + ": record")
		}
		validate := true
		if _, found := options["validate"]; found && GetIType(options["validate"]) == "bool" {
			validate = options["validate"].(bool)
		}
		id, err := npi.NStore.UpdateData(IM{
			"nervatype": options["record"].(IM)["__tablename__"],
			"values":    options["record"],
			"validate":  validate, "insert_row": true, "trans": data["trans"],
		})
		if err != nil {
			return data, err
		}
		data["data"] = options["record"]
		if data["data"].(IM)["id"] == nil {
			data["data"].(IM)["id"] = id
		}

	case "delete":
		if _, found := options["record"]; !found {
			return data, errors.New(GetMessage("missing_required_field") + ": record")
		}
		err := npi.NStore.DeleteData(IM{
			"nervatype": options["record"].(IM)["__tablename__"],
			"ref_id":    options["record"].(IM)["id"],
			"refnumber": options["record"].(IM)["refnumber"],
			"trans":     data["trans"],
		})
		if err != nil {
			return data, err
		}
		data["data"] = options["record"]

	case "table":
		query := Query{}
		if _, found := options["classAlias"]; found {
			query.From = options["classAlias"].(string)
		} else if _, found := options["tableName"]; found {
			query.From = options["tableName"].(string)
		} else {
			return data, errors.New(GetMessage("missing_required_field") + ": tableName")
		}
		if _, found := options["fields"]; found && GetIType(options["fields"]) == "[]string" {
			query.Fields = options["fields"].([]string)
		} else {
			query.Fields = []string{"*"}
		}
		if _, found := options["filterStr"]; found {
			query.Filter = options["filterStr"].(string)
		}
		if _, found := options["filters"]; found && GetIType(options["filters"]) == "[]Filter" {
			query.Filters = options["filters"].([]Filter)
		}
		if _, found := options["order"]; found && GetIType(options["order"]) == "[]string" {
			query.OrderBy = options["order"].([]string)
		} else if _, found := options["orderStr"]; found && GetIType(options["orderStr"]) == "string" {
			query.OrderBy = strings.Split(options["orderStr"].(string), ",")
		}
		results, err := npi.NStore.ds.Query([]Query{query}, data["trans"])
		if err != nil {
			return data, err
		}
		data["data"] = results

	case "view", "execute":
		results, err := npi.NStore.ds.QueryParams(options, data["trans"])
		if err != nil {
			return data, err
		}
		data["data"] = results

	}

	return data, nil
}

/*
LoadDataset - In a group, any number can be sent table, view, execute and function requests.
 • dataSetInfo: the list of ([]interface{}) name/value parameters:
   - infoName: required. unique request identifier name
   - infoType: required. Valid values: table, view, execute, function
   - functionName: only if infoType = function
   - tableName: only if infoType = table
   - filterStr: only if infoType = table
   - orderStr: only if infoType = table
   - sqlStr: only if infoType = view or execute
   - whereStr: only if infoType = view
   - havingStr: only if infoType = view
   - paramList: []interface{} (view, execute) or map[string]interface{} (function)
 The result map["data"] value a list of ([]map[string]interface{}) name/value results:
  - infoName: dataSetInfo.infoName
  - recordSet: result value

Example:
 options := IM{
	 "dataSetInfo": []interface{}{
	   IM{"infoName": "customer", "infoType": "table", "classAlias": "customer", "filterStr": ""},
   },
 }
 result, err := npi.LoadDataset(options)
*/
func (npi *Npi) LoadDataset(options IM) (IM, error) {
	results := IM{"data": []IM{}}

	if _, found := options["dataSetInfo"]; !found || GetIType(options["dataSetInfo"]) != "[]interface{}" {
		return results, errors.New(GetMessage("missing_required_field") + ": dataSetInfo")
	}

	if !npi.NStore.ds.Connection().Connected {
		return results, errors.New(GetMessage("not_connect"))
	}

	for index := 0; index < len(options["dataSetInfo"].([]interface{})); index++ {
		info := options["dataSetInfo"].([]interface{})[index].(IM)
		if _, found := info["infoType"]; found && GetIType(info["infoType"]) == "string" {
			if _, found := info["infoName"]; found && GetIType(info["infoName"]) == "string" {
				infoName := info["infoName"].(string)
				info["method"] = info["infoType"]
				result, err := npi.SetData(info)
				if err != nil {
					return results, err
				}
				results["data"] = append(results["data"].([]IM), IM{"infoName": infoName, "recordSet": result["data"]})
			}
		}
	}

	return results, nil
}

/*
UpdateRecordset - Insert, update or delete multiple rows of data.
 • method: update or delete
 • recordSet: the list of ([]interface{}) data rows. A new record, set the returned id.
 The result value of the input data.

Example:
 options := IM{
	 "method": "update",
   "recordSet": []interface{}{
     IM{"__tablename__": "customer", "id": 1, "account": "12345678"},
   },
 }
 _, err := npi.UpdateRecordset(options)
*/
func (npi *Npi) UpdateRecordset(options IM) (results IM, err error) {
	results = IM{"data": []IM{}}

	if _, found := options["recordSet"]; !found || GetIType(options["recordSet"]) != "[]interface{}" {
		return results, errors.New(GetMessage("missing_required_field") + ": recordSet")
	}
	if _, found := options["method"]; !found || GetIType(options["method"]) != "string" {
		return results, errors.New(GetMessage("missing_required_field") + ": method")
	}

	if !npi.NStore.ds.Connection().Connected {
		return results, errors.New(GetMessage("not_connect"))
	}

	var trans interface{}
	if npi.NStore.ds.Properties().Transaction {
		trans, err = npi.NStore.ds.BeginTransaction()
		if err != nil {
			return results, err
		}
	}

	defer func() {
		pe := recover()
		if trans != nil {
			if err != nil || pe != nil {
				npi.NStore.ds.RollbackTransaction(trans)
			} else {
				err = npi.NStore.ds.CommitTransaction(trans)
			}
		}
		if pe != nil {
			panic(pe)
		}
	}()

	for index := 0; index < len(options["recordSet"].([]interface{})); index++ {
		params := IM{
			"trans":  trans,
			"method": options["method"],
			"record": options["recordSet"].([]interface{})[index]}
		result, err := npi.SetData(params)
		if err != nil {
			return results, err
		}
		results["data"] = append(results["data"].([]IM), result["data"].(IM))
	}

	return results, nil
}

/*
SaveDataset - In a group, any number can be sent update, delete and function requests.
 • dataSetInfo: the list of ([]interface{}) name/value parameters:
   - updateType: required. Valid values: update, delete, function
   - tableName: only if updateType = update or delete
   - recordSet: only if updateType = update or delete
   - functionName: only if infoType = function
   - paramList: only if infoType = function
   - value: result value if infoType = function
 The result value of the input data (with returned new id).

Example:
 options := IM{
   "dataSetInfo": []interface{}{
     IM{"updateType": "update",
       "recordSet": []interface{}{
         IM{"__tablename__": "customer", "id": 1, "account": "87654321"},
       },
     },
   },
 }
	_, err := npi.SaveDataset(options)
*/
func (npi *Npi) SaveDataset(options IM) (results IM, err error) {
	results = IM{"data": []IM{}}

	if _, found := options["dataSetInfo"]; !found || GetIType(options["dataSetInfo"]) != "[]interface{}" {
		return results, errors.New(GetMessage("missing_required_field") + ": dataSetInfo")
	}

	if !npi.NStore.ds.Connection().Connected {
		return results, errors.New(GetMessage("not_connect"))
	}

	var trans interface{}
	if npi.NStore.ds.Properties().Transaction {
		trans, err = npi.NStore.ds.BeginTransaction()
		if err != nil {
			return results, err
		}
	}

	defer func() {
		pe := recover()
		if trans != nil {
			if err != nil || pe != nil {
				npi.NStore.ds.RollbackTransaction(trans)
			} else {
				err = npi.NStore.ds.CommitTransaction(trans)
			}
		}
		if pe != nil {
			panic(pe)
		}
	}()

	for index := 0; index < len(options["dataSetInfo"].([]interface{})); index++ {
		info := options["dataSetInfo"].([]interface{})[index].(IM)
		switch info["updateType"] {
		case "update":
			if _, found := info["recordSet"]; found && GetIType(info["recordSet"]) != "[]interface{}" {
				for index := 0; index < len(options["recordSet"].([]interface{})); index++ {
					params := IM{
						"trans":  trans,
						"method": "update",
						"record": info["recordSet"].([]interface{})[index]}
					result, err := npi.SetData(params)
					if err != nil {
						return results, err
					}
					info["recordSet"].([]interface{})[index] = result["data"].(IM)
				}
			}

		case "delete":
			if _, found := info["recordSet"]; found && GetIType(info["recordSet"]) != "[]interface{}" {
				for index := 0; index < len(options["recordSet"].([]interface{})); index++ {
					params := IM{
						"trans":  trans,
						"method": "delete",
						"record": info["recordSet"].([]interface{})[index]}
					result, err := npi.SetData(params)
					if err != nil {
						return results, err
					}
					info["recordSet"].([]interface{})[index] = result["data"].(IM)
				}
			}

		case "function":
			params := IM{
				"trans":        trans,
				"method":       "function",
				"functionName": info["functionName"],
				"paramList":    info["paramList"],
			}
			result, err := npi.SetData(params)
			if err != nil {
				return results, err
			}
			info["value"] = result["data"]

		}
		results["data"] = append(results["data"].([]IM), info)
	}

	return results, nil
}

/*
GetAPI - JSON-RPC interface
 • method values:
   - getLogin, getLogin_json -> GetLogin
   - loadView, loadView_json -> SetData/view
   - loadTable, loadTable_json -> SetData/table
   - loadDataSet, loadDataSet_json -> LoadDataset
   - executeSql, executeSql_json -> SetData/execute
   - saveRecord, saveRecord_json -> SetData/update
   - saveRecordSet, saveRecordSet_json -> UpdateRecordset/update
   - saveDataSet, saveDataSet_json -> SaveDataset
   - deleteRecord, deleteRecord_json -> SetData/delete
   - deleteRecordSet, deleteRecordSet_json -> UpdateRecordset/delete
   - callFunction, callFunction_json -> SetData/function

*/
func (npi *Npi) GetAPI(data JSONData) JSONData {

	var result IM
	var err error
	switch data.Method {
	case "getLogin", "getLogin_json":
		result, _ = npi.GetLogin(data.Params)
		data.Result = result
		return data

	case "loadView", "loadView_json":
		data.Params["method"] = "view"
		result, err = npi.SetData(data.Params)

	case "loadTable", "loadTable_json":
		data.Params["method"] = "table"
		result, err = npi.SetData(data.Params)

	case "loadDataSet", "loadDataSet_json":
		result, err = npi.LoadDataset(data.Params)

	case "executeSql", "executeSql_json":
		data.Params["method"] = "execute"
		result, err = npi.SetData(data.Params)

	case "saveRecord", "saveRecord_json":
		data.Params["method"] = "update"
		result, err = npi.SetData(data.Params)

	case "saveRecordSet", "saveRecordSet_json":
		data.Params["method"] = "update"
		result, err = npi.UpdateRecordset(data.Params)

	case "saveDataSet", "saveDataSet_json":
		result, err = npi.SaveDataset(data.Params)

	case "deleteRecord", "deleteRecord_json":
		data.Params["method"] = "delete"
		result, err = npi.SetData(data.Params)

	case "deleteRecordSet", "deleteRecordSet_json":
		data.Params["method"] = "delete"
		result, err = npi.UpdateRecordset(data.Params)

	case "callFunction", "callFunction_json":
		data.Params["method"] = "function"
		result, err = npi.SetData(data.Params)

	default:
		data.Error = IM{"code": "invalid", "message": GetMessage("unknown_method"), "data": ""}
		return data
	}
	if err != nil {
		data.Error = IM{"code": "load_view", "message": err.Error(), "data": ""}
	} else {
		data.Result = result["data"]
	}
	return data
}
