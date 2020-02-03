package nervatura

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" //sqlite3 driver
	ntura "github.com/nervatura/nervatura-go"

	_ "github.com/go-sql-driver/mysql" // mysql driver
	_ "github.com/lib/pq"              // postgres driver
)

//IM is a map[string]interface{} type short alias
type IM = map[string]interface{}

//SM is a map[string]string type short alias
type SM = map[string]string

//SL is a []string type short alias
type SL = []string

//SQLDriver a go database/sql DataDriver
type SQLDriver struct {
	alias   string
	connStr string
	engine  string
	db      *sql.DB
}

func (ds *SQLDriver) decodeEngine(sqlStr string) string {
	const (
		dcCasInt      = "{CAS_INT}"
		dcCaeInt      = "{CAE_INT}"
		dcCasFloat    = "{CAS_FLOAT}"
		dcCaeFloat    = "{CAE_FLOAT}"
		dcCasDate     = "{CAS_DATE}"
		dcCaeDate     = "{CAE_DATE}"
		dcFmsDate     = "{FMS_DATE}"
		dcFmeDate     = "{FME_DATE}"
		dcFmsDateTime = "{FMS_DATETIME}"
		dcFmeDateTime = "{FME_DATETIME}"
		asDate        = " as date)"
	)

	switch ds.engine {
	case "sqlite", "sqlite3", "postgres":
		sqlStr = strings.ReplaceAll(sqlStr, dcCasInt, "cast(")
		sqlStr = strings.ReplaceAll(sqlStr, dcCaeInt, " as integer)")
		sqlStr = strings.ReplaceAll(sqlStr, dcCasFloat, "cast(")
		sqlStr = strings.ReplaceAll(sqlStr, dcCaeFloat, " as float8)")
		sqlStr = strings.ReplaceAll(sqlStr, dcCasDate, "cast(")
		sqlStr = strings.ReplaceAll(sqlStr, dcCaeDate, asDate)
		sqlStr = strings.ReplaceAll(sqlStr, dcFmsDate, "to_char(")
		sqlStr = strings.ReplaceAll(sqlStr, dcFmeDate, ", 'YYYY-MM-DD')")
		sqlStr = strings.ReplaceAll(sqlStr, dcFmsDateTime, "to_char(")
		sqlStr = strings.ReplaceAll(sqlStr, dcFmeDateTime, ", 'YYYY-MM-DD HH24:MI')")
	case "mysql":
		sqlStr = strings.ReplaceAll(sqlStr, dcCasInt, "cast(")
		sqlStr = strings.ReplaceAll(sqlStr, dcCaeInt, " as signed)")
		sqlStr = strings.ReplaceAll(sqlStr, dcCasFloat, "cast(")
		sqlStr = strings.ReplaceAll(sqlStr, dcCaeFloat, " as decimal)")
		sqlStr = strings.ReplaceAll(sqlStr, dcCasDate, "cast(")
		sqlStr = strings.ReplaceAll(sqlStr, dcCaeDate, asDate)
		sqlStr = strings.ReplaceAll(sqlStr, dcFmsDate, "date_format(")
		sqlStr = strings.ReplaceAll(sqlStr, dcFmeDate, ", '%Y-%m-%d')")
		sqlStr = strings.ReplaceAll(sqlStr, dcFmsDateTime, "date_format(")
		sqlStr = strings.ReplaceAll(sqlStr, dcFmeDateTime, ", '%Y-%m-%d %H:%i')")
	case "mssql":
		sqlStr = strings.ReplaceAll(sqlStr, dcCasInt, "cast(")
		sqlStr = strings.ReplaceAll(sqlStr, dcCaeInt, " as int)")
		sqlStr = strings.ReplaceAll(sqlStr, dcCasFloat, "cast(")
		sqlStr = strings.ReplaceAll(sqlStr, dcCaeFloat, " as float)")
		sqlStr = strings.ReplaceAll(sqlStr, dcCasDate, "cast(")
		sqlStr = strings.ReplaceAll(sqlStr, dcCaeDate, asDate)
		sqlStr = strings.ReplaceAll(sqlStr, dcFmsDate, "convert(varchar(10),")
		sqlStr = strings.ReplaceAll(sqlStr, dcFmeDate, ", 120)")
		sqlStr = strings.ReplaceAll(sqlStr, dcFmsDateTime, "convert(varchar(16),")
		sqlStr = strings.ReplaceAll(sqlStr, dcFmeDateTime, ", 120)")
	}
	//print(sqlStr)
	return sqlStr
}

func (ds *SQLDriver) getDataType(dtype string) string {
	types := IM{
		"id": SM{
			"base":     "INTEGER PRIMARY KEY AUTOINCREMENT",
			"postgres": "SERIAL PRIMARY KEY",
			"mysql":    "INT AUTO_INCREMENT NOT NULL, PRIMARY KEY (id)",
			"mssql":    "INT IDENTITY PRIMARY KEY"},
		"integer": SM{
			"base":  "INTEGER",
			"mysql": "INT",
			"mssql": "INT"},
		"float": SM{
			"base":     "DOUBLE",
			"postgres": "FLOAT8",
			"mssql":    "DECIMAL(19,4)"},
		"string": SM{
			"base":     "CHAR",
			"postgres": "VARCHAR",
			"mysql":    "VARCHAR",
			"mssql":    "VARCHAR"},
		"password": SM{
			"base":     "CHAR",
			"postgres": "VARCHAR",
			"mysql":    "VARCHAR",
			"mssql":    "VARCHAR"},
		"text": SM{
			"base":  "TEXT",
			"mysql": "LONGTEXT",
			"mssql": "VARCHAR(max)"},
		"date": SM{
			"base": "DATE"},
		"datetime": SM{
			"base":  "TIMESTAMP",
			"mysql": "DATETIME",
			"mssql": "DATETIME2"},
		"reference": SM{
			"base":     "INTEGER REFERENCES foreign_key ON DELETE ",
			"postgres": "INTEGER REFERENCES foreign_key ON DELETE ",
			"mysql":    "INT, INDEX index_name (field_name), FOREIGN KEY (field_name) REFERENCES foreign_key ON DELETE ",
			"mssql":    "INT NULL, CONSTRAINT constraint_name FOREIGN KEY (field_name) REFERENCES foreign_key ON DELETE "}}
	if _, found := types[dtype]; found {
		if _, found := types[dtype].(SM)[ds.engine]; found {
			return types[dtype].(SM)[ds.engine]
		}
		return types[dtype].(SM)["base"]
	}
	return ""
}

func setRefID(refID IM, mname string, keyFields []string, values IM, id int) IM {
	if _, found := refID[mname]; !found {
		refID[mname] = IM{}
	}
	if ntura.GetIType(values[keyFields[0]]) != "string" {
		return refID
	}
	kf1 := values[keyFields[0]].(string)
	switch len(keyFields) {
	case 1:
		refID[mname].(IM)[kf1] = strconv.Itoa(id)

	case 2:
		if ntura.GetIType(values[keyFields[1]]) != "string" {
			return refID
		}
		kf2 := values[keyFields[1]].(string)
		if _, found := refID[mname].(IM)[kf1]; !found {
			refID[mname].(IM)[kf1] = IM{}
		}
		refID[mname].(IM)[kf1].(IM)[kf2] = strconv.Itoa(id)

	case 3:
		if ntura.GetIType(values[keyFields[1]]) != "string" || ntura.GetIType(values[keyFields[2]]) != "string" {
			return refID
		}
		kf2 := values[keyFields[1]].(string)
		kf3 := values[keyFields[2]].(string)
		if _, found := refID[mname].(IM)[kf1]; !found {
			refID[mname].(IM)[kf1] = IM{}
		}
		if _, found := refID[mname].(IM)[kf1].(IM)[kf2]; !found {
			refID[mname].(IM)[kf1].(IM)[kf2] = IM{}
		}
		refID[mname].(IM)[kf1].(IM)[kf2].(IM)[kf3] = strconv.Itoa(id)
	}
	return refID
}

func getRefID(refID IM, keyFields []string) string {
	switch len(keyFields) {
	case 2:
		return refID[keyFields[0]].(IM)[keyFields[1]].(string)
	case 3:
		return refID[keyFields[0]].(IM)[keyFields[1]].(IM)[keyFields[2]].(string)
	case 4:
		return refID[keyFields[0]].(IM)[keyFields[1]].(IM)[keyFields[2]].(IM)[keyFields[3]].(string)
	default:
		return "0"
	}
}

//Properties - DataDriver features
func (ds *SQLDriver) Properties() struct{ SQL, Transaction bool } {
	return struct{ SQL, Transaction bool }{SQL: true, Transaction: true}
}

//Connection - returns the database connection
func (ds *SQLDriver) Connection() struct {
	Alias     string
	Connected bool
	Engine    string
} {
	return struct {
		Alias     string
		Connected bool
		Engine    string
	}{
		Alias:     ds.alias,
		Connected: (ds.db != nil),
		Engine:    ds.engine,
	}
}

//CreateConnection create a new database connection
func (ds *SQLDriver) CreateConnection(alias, connStr string, settings ntura.Settings) error {
	if ds.db != nil {
		if err := ds.db.Close(); err != nil {
			return err
		}
	}
	engine := strings.Split(connStr, "://")[0]
	switch engine {
	case "sqlite3":
		connStr = strings.ReplaceAll(connStr, "sqlite3://", "")
	case "postgres", "mysql", "mssql":
	default:
		return errors.New("Valid database types: sqlite3, postgres, mysql, mssql")
	}
	db, err := sql.Open(engine, connStr)
	if err != nil {
		return err
	}
	err = db.Ping()
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(settings.SQLMaxOpenConns)
	db.SetMaxIdleConns(settings.SQLMaxIdleConns)
	db.SetConnMaxLifetime(time.Minute * time.Duration(settings.SQLConnMaxLifetime))
	ds.db = db
	ds.alias = alias
	ds.engine = engine
	ds.connStr = connStr
	return nil
}

// getPrmString - get database parameter string
func (ds *SQLDriver) getPrmString(index int) string {
	if ds.engine == "postgres" {
		return "$" + strconv.Itoa(index)
	}
	return "?"
}

// CheckHashtable - check/create a password ref. table
func (ds *SQLDriver) CheckHashtable(hashtable string) error {

	if ds.db == nil {
		return errors.New(ntura.GetMessage("missing_driver"))
	}
	var name string
	sqlString := ""
	if ds.engine == "sqlite3" {
		sqlString = fmt.Sprintf(
			"select name from sqlite_master where name = %s ", ds.getPrmString(1))
	} else {
		sqlString = fmt.Sprintf(
			"select table_name from information_schema.tables where table_name = %s ", ds.getPrmString(1))
	}
	err := ds.db.QueryRow(sqlString, hashtable).Scan(&name)
	if err != nil {
		textType := ds.getDataType("text")
		sqlString = fmt.Sprintf("CREATE TABLE %s ( refname %s, value %s);", hashtable, textType, textType)
		_, err = ds.db.Exec(sqlString)
		if err != nil {
			return err
		}
		sqlString = fmt.Sprintf("CREATE UNIQUE INDEX %s_refname_idx ON %s (refname);", hashtable, hashtable)
		_, err = ds.db.Exec(sqlString)
	}

	return err
}

// UpdateHashtable - set a password
func (ds *SQLDriver) UpdateHashtable(hashtable, refname, value string) error {
	err := ds.CheckHashtable(hashtable)
	if err != nil {
		return err
	}
	sqlString := fmt.Sprintf(
		"select value from %s where refname = %s", hashtable, ds.getPrmString(1))
	var hash string
	err = ds.db.QueryRow(sqlString, refname).Scan(&hash)
	if err != nil {
		sqlString = fmt.Sprintf(
			"insert into %s(refname, value) values(%s,%s)",
			hashtable, ds.getPrmString(1), ds.getPrmString(2))
	} else {
		sqlString = fmt.Sprintf(
			"update %s set value=%s where refname=%s",
			hashtable, ds.getPrmString(1), ds.getPrmString(2))
	}
	_, err = ds.db.Exec(sqlString, value, refname)
	return err
}

// CreateDatabase - create a Nervatura Database
func (ds *SQLDriver) CreateDatabase(logData []SM) ([]SM, error) {
	var err error
	if ds.db == nil {
		logData = append(logData, SM{
			"stamp":   time.Now().Format(ntura.TimeLayout),
			"state":   "err",
			"message": ntura.GetMessage("missing_driver")})
		return logData, errors.New(ntura.GetMessage("missing_driver"))
	}

	trans, err := ds.db.Begin()
	if err != nil {
		logData = append(logData, SM{
			"stamp":   time.Now().Format(ntura.TimeLayout),
			"state":   "err",
			"message": err.Error()})
		return logData, err
	}
	defer func() {
		pe := recover()
		if trans != nil {
			if err != nil || pe != nil {
				trans.Rollback()
			}
		}
		if pe != nil {
			panic(pe)
		}
	}()

	logData = append(logData, SM{
		"database": ds.alias,
		"stamp":    time.Now().Format(ntura.TimeLayout),
		"state":    "create",
		"message":  ntura.GetMessage("log_start_process")})

	//drop all tables if exist
	logData = append(logData, SM{
		"stamp":   time.Now().Format(ntura.TimeLayout),
		"state":   "create",
		"message": ntura.GetMessage("log_drop_table")})

	var dropList = []string{
		"pattern", "movement", "payment", "item", "trans", "barcode", "price", "tool", "product", "tax", "rate",
		"place", "currency", "project", "customer", "event", "contact", "address", "numberdef", "log", "fieldvalue",
		"deffield", "ui_audit", "link", "ui_userconfig", "ui_printqueue", "employee", "ui_reportsources",
		"ui_reportfields", "ui_report", "ui_message", "ui_menufields", "ui_menu", "ui_language", "groups"}
	for index := 0; index < len(dropList); index++ {
		sqlString := ""
		if ds.engine == "mysql" {
			sqlString = "SET FOREIGN_KEY_CHECKS=0; "
		}
		if ds.engine == "mssql" {
			sqlString += "DROP TABLE " + dropList[index] + ";"
		} else {
			sqlString += "DROP TABLE IF EXISTS " + dropList[index] + ";"
		}
		if ds.engine == "mysql" {
			sqlString = " SET FOREIGN_KEY_CHECKS=1;"
		}
		_, err = trans.Exec(sqlString)
		if err != nil {
			logData = append(logData, SM{
				"stamp":   time.Now().Format(ntura.TimeLayout),
				"state":   "err",
				"message": err.Error()})
			return logData, err
		}
	}
	err = trans.Commit()
	if err != nil {
		logData = append(logData, SM{
			"stamp":   time.Now().Format(ntura.TimeLayout),
			"state":   "err",
			"message": err.Error()})
		return logData, err
	}

	//create all tables
	trans, err = ds.db.Begin()
	logData = append(logData, SM{
		"stamp":   time.Now().Format(ntura.TimeLayout),
		"state":   "create",
		"message": ntura.GetMessage("log_create_table")})
	model := ntura.DataModel()["model"].(IM)
	var createList = []string{
		"groups", "ui_language", "ui_menu", "ui_menufields", "ui_message", "ui_report", "ui_reportfields", "ui_reportsources",
		"employee", "ui_printqueue", "ui_userconfig", "link", "ui_audit", "deffield", "fieldvalue", "log", "numberdef",
		"address", "contact", "event", "customer", "project", "currency", "place", "rate", "tax", "product", "tool", "price",
		"barcode", "trans", "item", "payment", "movement", "pattern"}
	for index := 0; index < len(createList); index++ {
		sqlString := "CREATE TABLE " + createList[index] + "("
		for fld := 0; fld < len(model[createList[index]].(IM)["_fields"].(SL)); fld++ {
			fieldname := model[createList[index]].(IM)["_fields"].(SL)[fld]
			field := model[createList[index]].(IM)[fieldname].(ntura.MF)
			sqlString += fieldname
			if field.References != nil {
				reference := ds.getDataType("reference")
				reference = strings.ReplaceAll(reference, "foreign_key", field.References[0]+"(id)")
				reference = strings.ReplaceAll(reference, "field_name", fieldname)
				reference = strings.ReplaceAll(reference, "index_name", fieldname+"__idx")
				reference = strings.ReplaceAll(reference, "constraint_name", createList[index]+"__"+fieldname+"__constraint")
				if (ds.engine == "mssql") && (len(field.References) == 3) {
					reference += field.References[2]
				} else {
					reference += field.References[1]
				}
				sqlString += " " + reference
			} else {
				sqlString += " " + ds.getDataType(field.Type)
				if field.Length > 0 {
					sqlString += "(" + strconv.Itoa(field.Length) + ")"
				}
			}
			if field.NotNull && field.References == nil {
				sqlString += " NOT NULL"
			}
			if field.Default != nil && field.Default != "nextnumber" {
				switch field.Default.(type) {
				case int:
					sqlString += " " + "DEFAULT" + " " + strconv.Itoa(field.Default.(int))
				case float64:
					sqlString += " " + "DEFAULT" + " " + strconv.FormatFloat(field.Default.(float64), 'f', -1, 64)
				case string:
					sqlString += " " + "DEFAULT" + " " + field.Default.(string)
				}
			}
			sqlString += ", "
		}
		sqlString += ");"
		sqlString = strings.ReplaceAll(sqlString, ", );", ");")
		//println(sqlString)
		_, err = trans.Exec(sqlString)
		if err != nil {
			logData = append(logData, SM{
				"stamp":   time.Now().Format(ntura.TimeLayout),
				"state":   "err",
				"message": err.Error()})
			return logData, err
		}
	}

	//create indexes
	logData = append(logData, SM{
		"stamp":   time.Now().Format(ntura.TimeLayout),
		"state":   "create",
		"message": ntura.GetMessage("log_create_index")})
	indexRows := ntura.DataModel()["index"].(map[string]ntura.MI)
	for ikey, ifield := range indexRows {
		sqlString := "CREATE INDEX "
		if ifield.Unique {
			sqlString = "CREATE UNIQUE INDEX "
		}
		sqlString += ikey + " ON " + ifield.Model + "("
		for index := 0; index < len(ifield.Fields); index++ {
			sqlString += ifield.Fields[index] + ", "
		}
		sqlString += ");"
		sqlString = strings.ReplaceAll(sqlString, ", );", ");")
		_, err = trans.Exec(sqlString)
		if err != nil {
			logData = append(logData, SM{
				"stamp":   time.Now().Format(ntura.TimeLayout),
				"state":   "err",
				"message": err.Error()})
			return logData, err
		}
	}

	//insert data
	logData = append(logData, SM{
		"stamp":   time.Now().Format(ntura.TimeLayout),
		"state":   "create",
		"message": ntura.GetMessage("log_init_data")})
	dataRows := ntura.DataModel()["data"].(map[string][]IM)
	dataList := []string{
		"groups", "currency", "customer", "employee", "address", "contact", "place", "tax", "product",
		"numberdef", "deffield", "fieldvalue"}
	refID := IM{}
	for index := 0; index < len(dataList); index++ {
		mname := dataList[index]
		keyFields := model[mname].(IM)["_key"].([]string)
		insertID := ""
		if ds.engine == "mssql" {
			insertID = "SET IDENTITY_INSERT " + mname + " ON; "
		}
		for index := 0; index < len(dataRows[mname]); index++ {
			refID = setRefID(refID, mname, keyFields, dataRows[mname][index], index+1)
			fields := "id, "
			values := strconv.Itoa(index+1) + ", "
			for field, value := range dataRows[mname][index] {
				fields += field + ", "
				switch value.(type) {
				case []string:
					values += getRefID(refID, value.([]string)) + ", "
				default:
					switch model[mname].(IM)[field].(ntura.ModelField).Type {
					case "string", "password", "text", "date", "datetime":
						values += "'" + value.(string) + "', "
					default:
						switch value.(type) {
						case int:
							values += strconv.Itoa(value.(int)) + ", "
						case float64:
							values += strconv.FormatFloat(value.(float64), 'f', -1, 64) + ", "
						default:
							values += value.(string) + ", "
						}
					}

				}
			}
			fields += ") "
			fields = strings.ReplaceAll(fields, ", ) ", ") ")
			values += ");"
			values = strings.ReplaceAll(values, ", );", ");")
			sqlString := insertID + "INSERT INTO " + mname + "(" + fields + " VALUES(" + values
			//println(sqlString)
			_, err = trans.Exec(sqlString)
			if err != nil {
				logData = append(logData, SM{
					"stamp":   time.Now().Format(ntura.TimeLayout),
					"state":   "err",
					"message": err.Error()})
				return logData, err
			}
		}
	}

	switch ds.engine {
	case "postgres":
		//update all sequences
		sqlString := ""
		for index := 0; index < len(createList); index++ {
			sqlString += "SELECT setval('" + createList[index] + "_id_seq', (SELECT max(id) FROM " + createList[index] + "));"

		}
		_, err = trans.Exec(sqlString)
		if err != nil {
			logData = append(logData, SM{
				"stamp":   time.Now().Format(ntura.TimeLayout),
				"state":   "err",
				"message": err.Error()})
			return logData, err
		}
	}

	err = trans.Commit()
	if err != nil {
		logData = append(logData, SM{
			"stamp":   time.Now().Format(ntura.TimeLayout),
			"state":   "err",
			"message": err.Error()})
		return logData, err
	}

	//compact
	logData = append(logData, SM{
		"stamp":   time.Now().Format(ntura.TimeLayout),
		"state":   "create",
		"message": ntura.GetMessage("log_rebuilding")})
	switch ds.engine {
	case "postgres", "sqlite", "sqlite3":
		sqlString := "vacuum"
		_, err = ds.db.Exec(sqlString)
		if err != nil {
			logData = append(logData, SM{
				"stamp":   time.Now().Format(ntura.TimeLayout),
				"state":   "err",
				"message": err.Error()})
			return logData, err
		}
	}

	return logData, nil
}

func (ds *SQLDriver) getFilterString(filter ntura.Filter, start bool, sqlString string, params []interface{}) (string, []interface{}) {
	if start {
		sqlString += "("
	} else if filter.Or == false {
		sqlString += " and ("
	} else {
		sqlString += " or "
	}
	sqlString += filter.Field
	switch filter.Comp {
	case "==":
		params = append(params, filter.Value)
		sqlString += "=" + ds.getPrmString(len(params))
	case "like", "!=", "<", "<=", ">", ">=", "is":
		params = append(params, filter.Value)
		sqlString += " " + filter.Comp + " " + ds.getPrmString(len(params))
	case "in":
		if ntura.GetIType(filter.Value) == "string" {
			values := strings.Split(filter.Value.(string), ",")
			prmStr := make([]string, 0)
			for _, value := range values {
				params = append(params, value)
				prmStr = append(prmStr, ds.getPrmString(len(params)))
			}
			sqlString += " in(" + strings.Join(prmStr, ",") + ")"
		}
	}
	if filter.Or == false {
		sqlString += ")"
	}
	return sqlString, params
}

func (ds *SQLDriver) decodeSQL(queries []ntura.Query) (string, []interface{}, error) {
	sqlString := ""
	params := make([]interface{}, 0)
	for qi := 0; qi < len(queries); qi++ {
		query := queries[qi]
		if qi > 0 {
			sqlString += " union select "
		} else {
			sqlString += "select "
		}
		sqlString += strings.Join(query.Fields, ",") + " from " + query.From
		if len(query.Filters) > 0 || query.Filter != "" {
			sqlString += " where "
		}
		for wi := 0; wi < len(query.Filters); wi++ {
			sqlString, params = ds.getFilterString(query.Filters[wi], (wi == 0), sqlString, params)
		}
		sqlString += query.Filter
		order := strings.Join(query.OrderBy, ",")
		if order != "" {
			sqlString += " order by " + order
		}
	}
	return strings.Trim(sqlString, " "), params, nil
}

//Query is a basic nosql friendly queries the database
func (ds *SQLDriver) Query(queries []ntura.Query, trans interface{}) ([]IM, error) {
	sqlString, params, err := ds.decodeSQL(queries)
	if err != nil {
		return nil, err
	}
	return ds.QuerySQL(sqlString, params, trans)
}

func getParamValue(prm ntura.IM) string {
	switch prm["type"] {
	case "integer", "number":
		if ntura.GetIType(prm["value"]) == "float64" {
			return strconv.FormatFloat(prm["value"].(float64), 'f', -1, 64)
		}
		if ntura.GetIType(prm["value"]) == "int" {
			return strconv.Itoa(prm["value"].(int))
		}
	case "boolean":
		if ntura.GetIType(prm["value"]) == "bool" {
			return strconv.FormatBool(prm["value"].(bool))
		}
	case "string", "date":
		value := prm["value"].(string)
		if strings.Index(value, "'") == -1 {
			value = "'" + value + "'"
		}
		return value
	}
	return prm["value"].(string)
}

func setParamList(paramList []interface{}, whereStr, havingStr, sqlString string) (string, string, string) {
	for index := 0; index < len(paramList); index++ {
		prm := paramList[index].(IM)
		name := prm["name"].(string)
		value := getParamValue(prm)
		switch prm["wheretype"] {
		case "where":
			whereStr = strings.ReplaceAll(whereStr, name, value)
		case "having":
			havingStr = strings.ReplaceAll(havingStr, name, value)
		case "in":
			sqlString = strings.ReplaceAll(sqlString, name, value)
		}
	}
	return whereStr, havingStr, sqlString
}

func setParamLimit(options IM, sqlString string) string {
	if _, found := options["rlimit"]; found {
		if options["rlimit"] == true {
			if _, found := options["rowlimit"]; found && ntura.GetIType(options["rowlimit"]) == "int" {
				sqlString = strings.ReplaceAll(sqlString, ";", "")
				sqlString += " limit " + strconv.Itoa(options["rowlimit"].(int))
			}
		}
	}
	return sqlString
}

//QueryParams - custom sql queries with parameters
func (ds *SQLDriver) QueryParams(options IM, trans interface{}) ([]IM, error) {
	sqlString, whereStr, havingStr, orderStr := "", "", "", ""
	params := make([]interface{}, 0)
	if _, found := options["sqlStr"]; found && ntura.GetIType(options["sqlStr"]) == "string" {
		sqlString = options["sqlStr"].(string)
	}
	if _, found := options["whereStr"]; found && ntura.GetIType(options["whereStr"]) == "string" {
		whereStr = options["whereStr"].(string)
	}
	if _, found := options["havingStr"]; found && ntura.GetIType(options["havingStr"]) == "string" {
		havingStr = options["havingStr"].(string)
	}
	if _, found := options["orderStr"]; found && ntura.GetIType(options["orderStr"]) == "string" {
		orderStr = options["orderStr"].(string)
	}
	if _, found := options["paramList"]; found && ntura.GetIType(options["paramList"]) == "[]interface{}" {
		whereStr, havingStr, sqlString = setParamList(options["paramList"].([]interface{}), whereStr, havingStr, sqlString)
	}

	sqlString = strings.ReplaceAll(sqlString, "@where_str", whereStr)
	sqlString = strings.ReplaceAll(sqlString, "@having_str", havingStr)
	sqlString = strings.ReplaceAll(sqlString, "@orderby_str", orderStr)
	sqlString = setParamLimit(options, sqlString)

	//println(sqlString)
	return ds.QuerySQL(sqlString, params, trans)
}

func initQueryCols(cols []*sql.ColumnType) ([]interface{}, []string, []string) {
	values := make([]interface{}, len(cols))
	fields := make([]string, len(cols))
	dbtypes := make([]string, len(cols))
	for i := range cols {
		fields[i] = cols[i].Name()
		dbtypes[i] = cols[i].DatabaseTypeName()
		//println(cols[i].DatabaseTypeName())
		switch cols[i].DatabaseTypeName() {
		case "BOOL", "BOOLEAN", "BIT":
			values[i] = new(sql.NullBool)
		case "INTEGER", "SERIAL", "INT", "INT4", "INT8":
			values[i] = new(sql.NullInt64)
		case "DOUBLE", "FLOAT8", "DECIMAL(19,4)", "DECIMAL":
			values[i] = new(sql.NullFloat64)
		case "DATETIME", "TIMESTAMP", "DATE":
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.NullString)
		}
	}
	return values, fields, dbtypes
}

func getQueryRowValue(value interface{}, dbtype string) interface{} {
	switch value.(type) {
	case *sql.NullBool:
		if value.(*sql.NullBool).Valid {
			return value.(*sql.NullBool).Bool
		}
		return nil

	case *sql.NullInt32:
		if value.(*sql.NullInt32).Valid {
			return int(value.(*sql.NullInt32).Int32)
		}
		return nil

	case *sql.NullInt64:
		if value.(*sql.NullInt64).Valid {
			return int(value.(*sql.NullInt64).Int64)
		}
		return nil

	case *sql.NullFloat64:
		if value.(*sql.NullFloat64).Valid {
			return value.(*sql.NullFloat64).Float64
		}
		return nil

	case *sql.NullTime:
		if value.(*sql.NullTime).Valid {
			if dbtype == "DATE" {
				return value.(*sql.NullTime).Time.Format("2006-01-02")
			}
			return value.(*sql.NullTime).Time
		}
		return nil

	case *sql.NullString:
		if value.(*sql.NullString).Valid {
			return value.(*sql.NullString).String
		}
		return nil

	}
	return value
}

//QuerySQL executes a SQL query
func (ds *SQLDriver) QuerySQL(sqlString string, params []interface{}, trans interface{}) ([]IM, error) {
	result := make([]IM, 0)
	var rows *sql.Rows
	var err error
	if trans != nil {
		switch trans.(type) {
		case *sql.Tx:
		default:
			return result, errors.New(ntura.GetMessage("invalid_trans"))
		}
	}

	//println(ds.decodeEngine(sqlString))
	if trans != nil {
		rows, err = trans.(*sql.Tx).Query(ds.decodeEngine(sqlString), params...)
	} else {
		rows, err = ds.db.Query(ds.decodeEngine(sqlString), params...)
	}
	if err != nil {
		return result, err
	}
	defer rows.Close()

	cols, err := rows.ColumnTypes()
	if err != nil {
		return result, err
	}
	values, fields, dbtypes := initQueryCols(cols)

	for rows.Next() {
		err = rows.Scan(values...)
		if err != nil {
			return result, err
		}
		row := make(IM)
		for index, value := range values {
			row[fields[index]] = getQueryRowValue(value, dbtypes[index])
		}
		result = append(result, row)
	}
	return result, nil
}

func (ds *SQLDriver) lastInsertID(model string, result sql.Result, trans interface{}) (int, error) {
	var sqlString string
	resid, err := result.LastInsertId()
	if err != nil {
		switch ds.engine {
		case "postgres":
			sqlString = fmt.Sprintf("select currval('%s_id_seq') as id", model)
		case "mssql":
			sqlString = fmt.Sprintf("select ident_current('%s') as id", model)
		default:
			return -1, err
		}
		if trans != nil {
			err = trans.(*sql.Tx).QueryRow(sqlString).Scan(&resid)
		} else {
			err = ds.db.QueryRow(sqlString).Scan(&resid)
		}
		if err != nil {
			return -1, err
		}
	}
	return int(resid), nil
}

//Update is a basic nosql friendly update/insert/delete and returns the update/insert id
func (ds *SQLDriver) Update(options ntura.Update) (int, error) {
	sqlString := ""
	id := options.IDKey
	params := make([]interface{}, 0)
	fields := make([]string, 0)
	values := make([]string, 0)
	sets := make([]string, 0)
	for fieldname, value := range options.Values {
		if value == nil {
			params = append(params, "null")
		} else {
			params = append(params, value)
		}
		fields = append(fields, fieldname)
		values = append(values, ds.getPrmString(len(params)))
		sets = append(sets, fmt.Sprintf("%s=%s", fieldname, ds.getPrmString(len(params))))
	}
	if id <= 0 {
		sqlString += fmt.Sprintf(
			"insert into %s (%s) values (%s)",
			options.Model, strings.Join(fields, ","), strings.Join(values, ","))
	} else if len(options.Values) == 0 {
		params = append(params, id)
		sqlString += fmt.Sprintf(
			"delete from %s where id=%s", options.Model, ds.getPrmString(1))
	} else {
		params = append(params, id)
		sqlString += fmt.Sprintf(
			"update %s set %s where id=%s", options.Model, strings.Join(sets, ","), ds.getPrmString(1))
	}
	if options.Trans != nil {
		switch options.Trans.(type) {
		case *sql.Tx:
		default:
			return id, errors.New(ntura.GetMessage("invalid_trans"))
		}
	}
	//println(sqlString)
	var result sql.Result
	var err error
	if options.Trans != nil {
		result, err = options.Trans.(*sql.Tx).Exec(ds.decodeEngine(sqlString), params...)
	} else {
		result, err = ds.db.Exec(ds.decodeEngine(sqlString), params...)
	}
	if err != nil {
		return id, err
	}
	if id <= 0 {
		return ds.lastInsertID(options.Model, result, options.Trans)
	}
	return id, nil
}

//BeginTransaction begins a transaction and returns an *sql.Tx
func (ds *SQLDriver) BeginTransaction() (interface{}, error) {
	return ds.db.Begin()
}

//CommitTransaction commit a *sql.Tx transaction
func (ds *SQLDriver) CommitTransaction(trans interface{}) error {
	switch trans.(type) {
	case *sql.Tx:
	default:
		return errors.New(ntura.GetMessage("invalid_trans"))
	}
	return trans.(*sql.Tx).Commit()
}

//RollbackTransaction rollback a *sql.Tx transaction
func (ds *SQLDriver) RollbackTransaction(trans interface{}) error {
	switch trans.(type) {
	case *sql.Tx:
	default:
		return errors.New(ntura.GetMessage("invalid_trans"))
	}
	return trans.(*sql.Tx).Rollback()
}

//QueryKey - complex data queries
func (ds *SQLDriver) QueryKey(options SM, trans interface{}) ([]IM, error) {
	result := []IM{}
	sqlString := ""
	params := make([]interface{}, 0)
	switch options["qkey"] {
	case "user":
		sqlString = `select e.id, e.username, e.empnumber, e.usergroup, ug.groupvalue as scope, dg.groupvalue as department
		from employee e
		inner join groups ug on e.usergroup = ug.id
		left join groups dg on e.department = dg.id
		where e.deleted = 0 and e.inactive = 0 and e.username = '` + options["username"] + `'`

	case "user_guest":
		sqlString = `select e.id, e.username, e.empnumber, ug.id as usergroup, ug.groupvalue as scope, dg.groupvalue as department
		from employee e, groups ug
		left join groups dg on e.department = dg.id
		where e.deleted = 0 and e.inactive = 0 and ug.groupname = 'usergroup' and ug.groupvalue = 'guest' and e.username = 'admin'`

	case "metadata":
		sqlString = `select fv.*, ft.groupvalue as fieldtype from fieldvalue fv 
		inner join deffield df on fv.fieldname = df.fieldname
    inner join groups nt on df.nervatype = nt.id 
    inner join groups ft on df.fieldtype = ft.id 
		where fv.deleted = 0 and df.deleted = 0 and nt.groupvalue = '` + options["nervatype"] + `'
      and fv.ref_id in (` + options["ids"] + `) 
		order by fv.fieldname, fv.id `

	case "post_transtype":
		sqlString = `select 'trans' as rtype, tt.groupvalue as transtype, c.custnumber
		from trans t
		inner join groups tt on t.transtype=tt.id
		left join customer c on t.customer_id=c.id
		where t.id = ` + options["trans_id"] + `
		union select 'groups' as rtype, groupvalue as transtype, null 
		from groups
		where groupname='transtype' 
		  and (groupvalue='` + options["transtype_key"] + `' or id=` + options["transtype_id"] + `)
		union select 'customer' as rtype, null as transtype, custnumber
		from customer
		where id=` + options["customer_id"] + ` or custnumber='` + options["custnumber"] + `'`

	case "default_report":
		sqlString = `select r.*, ft.groupvalue as reptype
		from ui_report r
		inner join groups ft on r.filetype=ft.id`
		if _, found := options["nervatype"]; found {
			sqlString += ` inner join groups nt on r.nervatype=nt.id and nt.groupvalue='` + options["nervatype"] + `'`
			if options["nervatype"] == "trans" {
				sqlString += ` inner join groups tt on r.transtype=tt.id and tt.groupvalue='` + options["transtype"] + `'
        inner join groups dir on r.direction=dir.id and dir.groupvalue='` + options["direction"] + `'`
			}
		}
		if _, found := options["reportkey"]; found {
			sqlString += ` where r.reportkey='` + options["reportkey"] + `'`
		}
		if _, found := options["report_id"]; found {
			sqlString += ` where r.id=` + options["report_id"]
		}

	case "reportfields":
		sqlString = `select rf.fieldname as fieldname, ft.groupvalue as fieldtype, rf.dataset as dataset, wt.groupvalue as wheretype, rf.sqlstr as sqlstr
		from ui_reportfields rf
		inner join groups ft on rf.fieldtype=ft.id
		inner join groups wt on rf.wheretype=wt.id
		where rf.report_id=` + options["report_id"]

	case "listprice":
		sqlString = `select min(p.pricevalue) as mp 
		from price p 
		left join link l on l.ref_id_1 = p.id and l.nervatype_1 = ( 
			select id from groups 
			where groupname = 'nervatype' and groupvalue = 'price') and l.deleted = 0 
		where p.deleted = 0 and p.discount is null and p.pricevalue <> 0 
			and l.ref_id_2 is null and p.vendorprice = ` + options["vendorprice"] + ` and p.product_id = ` + options["product_id"] + ` 
			and p.validfrom <= '` + options["posdate"] + `' and ( p.validto >= '` + options["posdate"] + `' or 
			p.validto is null) and p.curr = '` + options["curr"] + `' and p.qty <= ` + options["qty"]

	case "custprice":
		sqlString = `select min(p.pricevalue) as mp 
		from price p 
		inner join link l on l.ref_id_1 = p.id 
			and l.nervatype_1 = (select id from groups where groupname = 'nervatype' and groupvalue = 'price') 
			and l.nervatype_2 = (select id from groups where groupname = 'nervatype' and groupvalue = 'customer') 
			and l.deleted = 0 
		where p.deleted = 0 and p.discount is null and p.pricevalue <> 0 
			and p.vendorprice = ` + options["vendorprice"] + ` and p.product_id = ` + options["product_id"] + ` and p.validfrom <= '` + options["posdate"] + `' 
			and ( p.validto >= '` + options["posdate"] + `' or p.validto is null) and p.curr = '` + options["curr"] + `' 
			and p.qty <= ` + options["qty"] + ` and l.ref_id_2 = ` + options["customer_id"]

	case "grouprice":
		sqlString = `select min(p.pricevalue) as mp 
		from price p 
		inner join link l on l.ref_id_1 = p.id and l.deleted = 0 
			and l.nervatype_1 = (select id from groups where groupname = 'nervatype' and groupvalue = 'price') 
			and l.nervatype_2 = (select id from groups where groupname = 'nervatype' and groupvalue = 'groups') 
		inner join groups g on g.id = l.ref_id_2 
			and g.id in (select l.ref_id_2 from link l where l.deleted = 0 
			and l.nervatype_1 = (select id from groups where groupname = 'nervatype' and groupvalue = 'customer') 
			and l.nervatype_2 = (select id from groups where groupname = 'nervatype' and groupvalue = 'groups') 
			and l.ref_id_1 = ` + options["customer_id"] + `) 
		where p.deleted = 0 and p.discount is null and p.pricevalue <> 0 and p.vendorprice = ` + options["vendorprice"] + ` 
			and p.product_id = ` + options["product_id"] + ` and p.validfrom <= '` + options["posdate"] + `' 
			and ( p.validto >= '` + options["posdate"] + `' or p.validto is null) 
			and p.curr = '` + options["curr"] + `' and p.qty <= ` + options["qty"]

	case "data_audit":
		sqlString = `select tt.groupvalue as transfilter 
		  from employee e inner join link l on l.ref_id_1 = e.usergroup and l.deleted = 0 
			inner join groups nt1 on l.nervatype_1 = nt1.id and nt1.groupname = 'nervatype' and nt1.groupvalue = 'groups' 
			inner join groups nt2 on l.nervatype_2 = nt2.id and nt2.groupname = 'nervatype' and nt2.groupvalue = 'groups' 
			inner join groups tt on l.ref_id_2 = tt.id where e.id = ` + options["id"]

	case "object_audit":
		sqlString = `select inf.groupvalue as inputfilter from ui_audit a 
		  inner join groups inf on a.inputfilter = inf.id 
			inner join groups nt on a.nervatype = nt.id 
			left join groups st on a.subtype = st.id 
			where (a.usergroup = ` + options["usergroup"] + `) `
		if _, found := options["subtype"]; found {
			sqlString += ` and a.subtype = ` + options["subtype"]
		}
		if _, found := options["subtypeIn"]; found {
			sqlString += ` and a.subtype in (` + options["subtypeIn"] + `) `
		}
		if _, found := options["transtype"]; found {
			sqlString += ` and st.groupvalue = '` + options["transtype"] + `' `
		}
		if _, found := options["transtypeIn"]; found {
			sqlString += ` and st.groupvalue in (` + options["transtypeIn"] + `) `
		}
		if _, found := options["nervatype"]; found {
			sqlString += ` and a.nervatype = ` + options["nervatype"]
		}
		if _, found := options["nervatypeIn"]; found {
			sqlString += ` and a.nervatype in (` + options["nervatypeIn"] + `) `
		}
		if _, found := options["groupvalue"]; found {
			sqlString += ` and nt.groupvalue = '` + options["groupvalue"] + `' `
		}
		if _, found := options["groupvalueIn"]; found {
			sqlString += ` and nt.groupvalue in (` + options["groupvalueIn"] + `) `
		}

	case "dbs_settings":
		sqlString = `select 'setting' as stype, fieldname, value, notes as data, id as info 
		  from fieldvalue where deleted = 0 and ref_id is null  
			union select 'pattern' as stype, p.description as fieldname, p.notes as value, 
			  tt.groupvalue as data, p.defpattern as info 
			from pattern p inner join groups tt on p.transtype = tt.id where p.deleted = 0`

	case "id->refnumber":
		switch options["nervatype"] {
		case "address", "contact":
			if _, found := options["refId"]; found {
				sqlString = `select count(*) as count from ` + options["nervatype"] + ` 
				  where nervatype = ` + options["refTypeId"] + ` and ref_id = ` + options["refId"] + ` and id <= ` + options["id"] + ` `
				if options["useDeleted"] == "false" {
					sqlString += ` and deleted = 0`
				}
			} else {
				sqlString = `select nt.groupvalue as head_nervatype, t.* 
			  from ` + options["nervatype"] + ` t inner join groups nt on t.nervatype = nt.id 
				where t.id = ` + options["id"] + ` `
				if options["useDeleted"] == "false" {
					sqlString += ` and t.deleted = 0`
				}
			}

		case "fieldvalue", "setting":
			if _, found := options["refId"]; found {
				sqlString = `select count(*) as count from fieldvalue 
				  where fieldname = '` + options["fieldname"] + `' and ref_id = ` + options["refId"] + ` and id <= ` + options["id"] + ` `
				if options["useDeleted"] == "false" {
					sqlString += ` and deleted = 0`
				}
			} else {
				sqlString = `select fv.*, nt.groupvalue as head_nervatype 
			  from fieldvalue fv inner join deffield df on fv.fieldname = df.fieldname 
				inner join groups nt on df.nervatype = nt.id 
				where fv.id = ` + options["id"] + ` `
				if options["useDeleted"] == "false" {
					sqlString += ` and fv.deleted = 0`
				}
			}

		case "item", "payment", "movement":
			if _, found := options["refId"]; found {
				sqlString = `select count(*) as count from ` + options["nervatype"] + ` 
				  where trans_id = ` + options["refId"] + ` and id <= ` + options["id"] + ` `
				if options["useDeleted"] == "false" {
					sqlString += ` and deleted = 0`
				}
			} else {
				sqlString = `select ti.*, t.transnumber, tt.groupvalue as transtype 
				  from ` + options["nervatype"] + ` ti inner join trans t on ti.trans_id = t.id 
					inner join groups tt on t.transtype = tt.id 
					where ti.id = ` + options["id"] + ` `
				if options["useDeleted"] == "false" {
					sqlString += ` and ( t.deleted = 0 or tt.groupvalue in ('cash', 'invoice', 'receipt'))`
				}
			}

		case "price":
			sqlString = `select pr.*, p.partnumber from price pr 
			  inner join product p on pr.product_id = p.id where pr.id = ` + options["id"] + `  `
			if options["useDeleted"] == "false" {
				sqlString += ` and p.deleted = 0`
			}

		case "link":
			sqlString = `select l.*, nt1.groupvalue as nervatype1, nt2.groupvalue as nervatype2 
			  from link l inner join groups nt1 on l.nervatype_1 = nt1.id 
				inner join groups nt2 on l.nervatype_2 = nt2.id where l.id = ` + options["id"] + `  `
			if options["useDeleted"] == "false" {
				sqlString += ` and l.deleted = 0`
			}

		case "rate":
			sqlString = `select r.*, rt.groupvalue as rate_type, p.planumber 
			  from rate r inner join groups rt on r.ratetype = rt.id 
				left join place p on r.place_id = p.id where r.id = ` + options["id"] + ` `
			if options["useDeleted"] == "false" {
				sqlString += ` and r.deleted = 0`
			}

		case "log":
			sqlString = `select l.*, e.empnumber from log l 
			  inner join employee e on l.employee_id = e.id where l.id = ` + options["id"]

		default:
			sqlString = `select * from ` + options["nervatype"] + ` 
			  where  id = ` + options["id"] + ` `
			if options["useDeleted"] == "false" {
				sqlString += " and deleted = 0 "
			}
		}

	case "refnumber->id":
		switch options["nervatype"] {

		case "employee", "pattern", "project", "tool", "currency", "numberdef",
			"ui_language", "ui_report", "ui_menu":
			sqlString = "select id from " + options["nervatype"] + " where " +
				options["refField"] + " = '" + options["refnumber"] + "' "
			if options["useDeleted"] == "false" {
				sqlString += " and deleted = 0 "
			}

		case "address", "contact":
			sqlString = `select ` + options["nervatype"] + `.id as id 
				from ` + options["nervatype"] + ` 
				inner join groups nt on ` + options["nervatype"] + `.nervatype = nt.id 
				  and nt.groupname = 'nervatype' and nt.groupvalue = '` + options["refType"] + `' 
				inner join ` + options["refType"] + ` on ` + options["nervatype"] + `.ref_id = ` + options["refType"] + `.id 
				  and ` + options["refType"] + `.` + options["refField"] + ` = '` + options["refnumber"] + `' `
			if options["useDeleted"] == "false" {
				sqlString += `where ` + options["refType"] + `.deleted = 0 and ` + options["nervatype"] + `.deleted = 0`
			}

		case "barcode":
			if options["useDeleted"] == "false" {
				sqlString = `select barcode.id from barcode 
					inner join product on barcode.product_id = product.id
					where product.deleted = 0 and code='` + options["refnumber"] + `'`
			} else {
				sqlString = `select id from barcode where code='` + options["refnumber"] + `'`
			}

		case "customer":
			if options["extraInfo"] == "true" {
				sqlString = `select c.id as id, ct.groupvalue as custtype, c.terms as terms, c.custname as custname, 
						c.taxnumber as taxnumber, addr.zipcode as zipcode, addr.city as city, addr.street as street 
					from customer c 
					inner join groups ct on c.custtype = ct.id 
					left join ( 
						select * from address where id = ( select min(id) from address where deleted = 0 
							and nervatype = ( select id from groups where groupname = 'nervatype' and groupvalue = 'customer') 
							and ref_id = ( select min(c.id) from customer c inner join groups ct on c.custtype = ct.id and groupvalue = 'own' where c.deleted = 0))
					) addr on c.id = addr.ref_id 
					where c.id = ( select min(c.id) from customer c inner join groups ct on c.custtype = ct.id and groupvalue = 'own' where c.deleted = 0)  
					union select c.id as id, ct.groupvalue as custtype, c.terms as terms, c.custname as custname, 
						c.taxnumber as taxnumber, addr.zipcode as zipcode, addr.city as city, addr.street as street 
					from customer c 
					inner join groups ct on c.custtype = ct.id 
					left join ( 
						select * from address where id = ( select min(id) from address where deleted = 0 
							and nervatype = ( select id from groups where groupname = 'nervatype' and groupvalue = 'customer') 
							and ref_id = ( select id from customer where custnumber = '` + options["refnumber"] + `'))
					) addr on c.id = addr.ref_id 
					where c.custnumber = '` + options["refnumber"] + `'`
			} else {
				sqlString = `select c.id as id, ct.groupvalue as custtype 
					from customer c inner join groups ct on c.custtype = ct.id 
					where c.custnumber = '` + options["refnumber"] + `'`
			}
			if options["useDeleted"] == "false" {
				sqlString += ` and c.deleted = 0`
			}

		case "event":
			sqlString = `select e.id as id, ntype.groupvalue as ref_nervatype 
				from event e inner join groups ntype on e.nervatype = ntype.id 
				where e.calnumber = '` + options["refnumber"] + `'`
			if options["useDeleted"] == "false" {
				sqlString += ` and e.deleted = 0`
			}

		case "groups":
			sqlString = `select id from groups 
				where groupname = '` + options["refType"] + `' and groupvalue = '` + options["refnumber"] + `'`
			if options["useDeleted"] == "false" {
				sqlString += ` and deleted = 0`
			}

		case "deffield":
			sqlString = `select df.id, nt.groupvalue as ref_nervatype from deffield df 
			  inner join groups nt on df.nervatype = nt.id 
				where df.fieldname = '` + options["refnumber"] + `' `
			if options["useDeleted"] == "false" {
				sqlString += ` and df.deleted = 0`
			}

		case "fieldvalue":
			sqlString = `select id from fieldvalue  
				where ref_id = ` + options["refID"] + ` and fieldname = '` + options["refnumber"] + `' `
			if options["useDeleted"] == "false" {
				sqlString += ` and deleted = 0`
			}

		case "item":
			sqlString = `select it.id as id, ttype.groupvalue as transtype, dir.groupvalue as direction, cu.digit as digit, 
					it.qty as qty, it.discount as discount, it.tax_id as tax_id, ta.rate as rate 
				from item it 
				inner join trans t on it.trans_id = t.id and t.transnumber = '` + options["refnumber"] + `' 
				inner join tax ta on it.tax_id = ta.id 
				inner join groups ttype on t.transtype = ttype.id 
				inner join groups dir on t.direction = dir.id 
				left join currency cu on t.curr = cu.curr `
			if options["useDeleted"] == "false" {
				sqlString += ` where t.deleted = 0 and it.deleted = 0`
			}

		case "payment":
			sqlString = `select it.id as id, ttype.groupvalue as transtype, dir.groupvalue as direction 
				from payment it 
				inner join trans t on it.trans_id = t.id and t.transnumber = '` + options["refnumber"] + `' 
				inner join groups ttype on t.transtype = ttype.id 
				inner join groups dir on t.direction = dir.id `
			if options["useDeleted"] == "false" {
				sqlString += ` where t.deleted = 0 and it.deleted = 0`
			}

		case "movement":
			sqlString = `select it.id as id, ttype.groupvalue as transtype, dir.groupvalue as direction, mt.groupvalue as movetype 
				from movement it 
				inner join groups mt on it.movetype = mt.id 
				inner join trans t on it.trans_id = t.id and t.transnumber = '` + options["refnumber"] + `' 
				inner join groups ttype on t.transtype = ttype.id 
				inner join groups dir on t.direction = dir.id `
			if options["useDeleted"] == "false" {
				sqlString += ` where t.deleted = 0 and it.deleted = 0`
			}

		case "price":
			sqlString = `select pr.id as id from price pr 
				inner join product p on pr.product_id = p.id 
				where p.partnumber = '` + options["refnumber"] + `' and pr.curr = '` + options["curr"] + `' and pr.validfrom = '` + options["validfrom"] + `' and pr.qty = ` + options["qty"] + `  `
			if options["pricetype"] == "price" {
				sqlString += ` and pr.discount is null`
			} else {
				sqlString += ` and pr.discount is not null`
			}
			if options["useDeleted"] == "false" {
				sqlString += ` and p.deleted = 0 and pr.deleted = 0`
			}

		case "product":
			sqlString = `select p.id as id, p.description as description, p.unit as unit, p.tax_id as tax_id, t.rate as rate 
			  from product p left join tax t on p.tax_id = t.id 
				where p.partnumber = '` + options["refnumber"] + `' `
			if options["useDeleted"] == "false" {
				sqlString += ` and p.deleted = 0`
			}

		case "place":
			sqlString = `select p.id as id, pt.groupvalue as placetype 
			  from place p inner join groups pt on p.placetype = pt.id 
				where p.planumber = '` + options["refnumber"] + `' `
			if options["useDeleted"] == "false" {
				sqlString += ` and p.deleted = 0`
			}

		case "tax":
			sqlString = `select id, rate from tax where taxcode = '` + options["refnumber"] + `' `

		case "trans":
			if options["useDeleted"] == "true" {
				sqlString = `select t.id as id, ttype.groupvalue as transtype, dir.groupvalue as direction, cu.digit as digit 
					from trans t 
					inner join groups ttype on t.transtype = ttype.id 
					inner join groups dir on t.direction = dir.id 
					left join currency cu on t.curr = cu.curr where t.transnumber = '` + options["refnumber"] + `'`
			} else {
				sqlString = `select t.id as id, ttype.groupvalue as transtype, dir.groupvalue as direction, cu.digit as digit 
					from trans t 
					inner join groups ttype on t.transtype = ttype.id 
					inner join groups dir on t.direction = dir.id 
					left join currency cu on t.curr = cu.curr 
					where t.transnumber = '` + options["refnumber"] + `' and ( t.deleted = 0 or ( ttype.groupvalue = 'invoice' and dir.groupvalue = 'out') 
						or ( ttype.groupvalue = 'receipt' and dir.groupvalue = 'out') or ( ttype.groupvalue = 'cash'))`
			}

		case "setting":
			sqlString = `select id from fieldvalue  
				where ref_id is null and fieldname = '` + options["refnumber"] + `' `
			if options["useDeleted"] == "false" {
				sqlString += ` and deleted = 0`
			}

		case "link":
			sqlString = `select l.id as id from link l 
				inner join groups nt1 on l.nervatype_1 = nt1.id and nt1.groupname = 'nervatype' and nt1.groupvalue = '` + options["refType1"] + `' 
				inner join groups nt2 on l.nervatype_2 = nt2.id and nt2.groupname = 'nervatype' and nt2.groupvalue = '` + options["refType2"] + `' 
				where l.ref_id_1 = ` + options["refID1"] + ` and l.ref_id_2 = ` + options["refID2"] + ` `
			if options["useDeleted"] == "false" {
				sqlString += ` and l.deleted = 0`
			}

		case "rate":
			sqlString = `select r.id as id from rate r 
			  inner join groups rt on r.ratetype = rt.id and rt.groupvalue = '` + options["ratetype"] + `' 
				left join place p on r.place_id = p.id 
				where r.ratedate = '` + options["ratedate"] + `' and r.curr = '` + options["curr"] + `' `
			if _, found := options["planumber"]; found {
				sqlString += ` and p.planumber = '` + options["planumber"] + `' `
			} else {
				sqlString += ` and r.place_id is null `
			}
			if options["useDeleted"] == "false" {
				sqlString += ` and r.deleted = 0`
			}

		case "log":
			sqlString = `select l.id as id from log l inner join employee e on l.employee_id = e.id 
			  where e.empnumber = '` + options["refnumber"] + `' and l.crdate = '` + options["crdate"] + `' `

		case "ui_audit":
			sqlString = `select au.id as id from ui_audit au 
			  inner join groups ug on au.usergroup = ug.id and ug.groupvalue = '` + options["usergroup"] + `' 
				inner join groups nt on au.nervatype = nt.id `
			if _, found := options["transType"]; found {
				if options["refType"] == "trans" {
					sqlString += ` inner join groups st on au.subtype = st.id and st.groupvalue = '` + options["transType"] + `' where `
				} else {
					sqlString += ` inner join ui_report rp on au.subtype = rp.id and rp.reportkey = '` + options["transType"] + `' where `
				}
			} else {
				sqlString += ` where subtype is null and `
			}
			sqlString += ` nt.groupvalue = '` + options["refType"] + `' `

		case "ui_menufields":
			sqlString = `select mf.id as id from ui_menufields mf 
			  inner join ui_menu m on mf.menu_id = m.id and m.menukey = '` + options["refnumber"] + `' 
				where mf.fieldname = '` + options["fieldname"] + `' `

		case "ui_reportfields":
			sqlString = `select rf.id as id from ui_reportfields rf 
			  inner join ui_report r on rf.report_id = r.id and r.reportkey = '` + options["refnumber"] + `' 
				where rf.fieldname = '` + options["fieldname"] + `' `

		case "ui_reportsources":
			sqlString = `select rs.id as id from ui_reportsources rs 
			  inner join ui_report r on rs.report_id = r.id and r.reportkey = '` + options["refnumber"] + `' 
				where rs.dataset = '` + options["dataset"] + `' `

		default:
			return nil, errors.New(ntura.GetMessage("invalid_refnumber"))
		}

	case "integrity":
		switch options["nervatype"] {
		case "currency":
			//(link), place,price,rate,trans
			if _, found := options["ref_id"]; found {
				sqlString = `select sum(co) as sco from (
				select count(place.id) as co from place
				inner join currency on (place.curr=currency.curr)
				where ((place.deleted=0) and (currency.id=` + options["ref_id"] + `))
				union select count(price.id) as co from price
				inner join currency on (price.curr=currency.curr) 
				where ((price.deleted=0) and (currency.id=` + options["ref_id"] + `)) 
				union select count(rate.id) as co from rate 
				inner join currency on (rate.curr=currency.curr)
				where ((rate.deleted=0) and (currency.id=` + options["ref_id"] + `))
				union select count(trans.id) as co from trans
				inner join currency on (trans.curr=currency.curr)
				where ((trans.deleted=0) and (currency.id=` + options["ref_id"] + `))) foo`
			}

		case "customer":
			//(address,contact), event,project,trans,link
			if _, found := options["ref_id"]; found {
				sqlString = `select sum(co) as sco from (
				select count(*) as co  from trans where (customer_id=` + options["ref_id"] + `)
				union select count(*) as co  from project where (customer_id=` + options["ref_id"] + `)
				union select count(*) as co  from event
				inner join groups nt on ((event.nervatype=nt.id) and (nt.groupvalue='customer'))
				where (event.deleted=0) and (event.ref_id=` + options["ref_id"] + `)
				union select count(*) as co  from link
				where nervatype_2=(
					select id  from groups 
					where (groupname='nervatype') and (groupvalue='customer')
						and (ref_id_2=` + options["ref_id"] + `))
				) foo`
			}

		case "deffield":
			//fieldvalue
			if _, found := options["ref_id"]; found {
				sqlString = `select sum(co) as sco from (
			  select count(fieldvalue.id) as co from fieldvalue
				inner join deffield on (deffield.fieldname=fieldvalue.fieldname)
				where (fieldvalue.deleted=0) and (deffield.id=` + options["ref_id"] + `)
				) foo`
			}

		case "employee":
			//(address,contact), event,trans,log,link,ui_printqueue,ui_userconfig
			if _, found := options["ref_id"]; found {
				sqlString = `select sum(co) as sco from (
				select count(*) as co from trans where (employee_id=` + options["ref_id"] + `)
				union select count(*) as co from trans where (cruser_id=` + options["ref_id"] + `)
				union select count(*) as co from log where (employee_id=` + options["ref_id"] + `)
				union select count(*) as co from ui_printqueue where (employee_id=` + options["ref_id"] + `)
				union select count(*) as co from ui_userconfig where (employee_id=` + options["ref_id"] + `)
				union select count(*) as co from event
				  inner join groups nt on ((event.nervatype=nt.id) and (nt.groupvalue='employee'))
				  where (event.deleted=0) and (event.ref_id=` + options["ref_id"] + `)
				union select count(*) as co from link 
					where (nervatype_2=(select id from groups 
						where (groupname='nervatype') and (groupvalue='employee') 
						  and (ref_id_2=` + options["ref_id"] + `)))
				) foo`
			}

		case "groups":
			//barcode,deffield,employee,event,rate,tool,trans,link
			if _, found := options["ref_id"]; found {
				sqlString = `select sum(co) as sco from (
				select count(*) as co from groups 
				where groupname in ('nervatype', 'custtype', 'fieldtype', 'logstate', 'movetype', 'transtype', 
					'placetype', 'calcmode', 'protype', 'ratetype', 'direction', 'transtate', 
					'inputfilter', 'filetype', 'wheretype', 'aggretype') and id = ` + options["ref_id"] + `  
				union select count(*) as co from link 
				where nervatype_2 = ( select id from groups where groupname = 'nervatype' and groupvalue = 'groups') and ref_id_2 = ` + options["ref_id"] + `  
				union select count(*) from barcode where barcode.barcodetype = ` + options["ref_id"] + `  
				union select count(*) from deffield where deffield.deleted = 0 and deffield.subtype = ` + options["ref_id"] + `  
				union select count(*) from employee where employee.deleted = 0 and employee.usergroup = ` + options["ref_id"] + `  
				union select count(*) from employee where employee.deleted = 0 and employee.department = ` + options["ref_id"] + `  
				union select count(*) from event where event.deleted = 0 and event.eventgroup = ` + options["ref_id"] + `  
				union select count(*) from rate where rate.deleted = 0 and rate.rategroup = ` + options["ref_id"] + `  
				union select count(*) from tool where tool.deleted = 0 and tool.toolgroup = ` + options["ref_id"] + `  
				union select count(*) from trans where trans.deleted = 0 and trans.department = ` + options["ref_id"] + `
			) foo`
			}

		case "place":
			//(address,contact), event,movement,place,rate,trans,link
			if _, found := options["ref_id"]; found {
				sqlString = `select sum(co) as sco from (
				select count(*) as co from event inner join groups nt on event.nervatype = nt.id and nt.groupvalue = 'place' 
				where event.deleted = 0 and event.ref_id = ` + options["ref_id"] + `  
				union select count(*) as co from link where nervatype_2 = ( 
					select id from groups where groupname = 'nervatype' and groupvalue = 'place') and ref_id_2 = ` + options["ref_id"] + `  
				union select count(*) as co from movement where movement.deleted = 0 and movement.place_id = ` + options["ref_id"] + `    
				union select count(*) as co from rate where rate.deleted = 0 and rate.place_id = ` + options["ref_id"] + `  
				union select count(*) as co from trans where trans.deleted = 0 and trans.place_id = ` + options["ref_id"] + `
			) foo`
			}

		case "product":
			//address,barcode,contact,event,item,movement,price,tool,link
			if _, found := options["ref_id"]; found {
				sqlString = `select sum(co) as sco from (
				select count(*) as co from event inner join groups nt on event.nervatype = nt.id and nt.groupvalue = 'product' 
				where event.deleted = 0 and event.ref_id = ` + options["ref_id"] + `  
				union select count(*) as co from address inner join groups nt on address.nervatype = nt.id and nt.groupvalue = 'product' 
				where address.deleted = 0 and address.ref_id = ` + options["ref_id"] + `  
				union select count(*) as co from contact inner join groups nt on contact.nervatype = nt.id and nt.groupvalue = 'product' 
				where contact.deleted = 0 and contact.ref_id = ` + options["ref_id"] + `  
				union select count(*) as co from link where nervatype_2 = ( 
					select id from groups where groupname = 'nervatype' and groupvalue = 'product') and ref_id_2 = ` + options["ref_id"] + `  
				union select count(*) as co from barcode where barcode.product_id = ` + options["ref_id"] + `  
				union select count(*) as co from movement where movement.deleted = 0 and movement.product_id = ` + options["ref_id"] + `  
				union select count(*) as co from item where item.deleted = 0 and item.product_id = ` + options["ref_id"] + `  
				union select count(*) as co from price where price.deleted = 0 and price.product_id = ` + options["ref_id"] + `  
				union select count(*) as co from tool where tool.deleted = 0 and tool.product_id = ` + options["ref_id"] + `
			) foo`
			}

		case "project":
			//(address,contact), event,trans,link
			if _, found := options["ref_id"]; found {
				sqlString = `select sum(co) as sco from (
				select count(*) as co from trans where project_id = ` + options["ref_id"] + `  
				union select count(*) as co from event inner join groups nt on event.nervatype = nt.id and nt.groupvalue = 'project' 
				where event.deleted = 0 and event.ref_id = ` + options["ref_id"] + `  
				union select count(*) as co from link where nervatype_2 = ( 
					select id from groups where groupname = 'nervatype' and groupvalue = 'project') and ref_id_2 = ` + options["ref_id"] + `
			) foo`
			}

		case "tax":
			//item,product
			if _, found := options["ref_id"]; found {
				sqlString = `select sum(co) as sco from (
				select count(*) as co from item where item.deleted = 0 and item.tax_id = ` + options["ref_id"] + `  
				union select count(*) as co from product where product.deleted = 0 and product.tax_id = ` + options["ref_id"] + `
			) foo`
			}

		case "tool":
			//(address,contact), event,movement,link
			if _, found := options["ref_id"]; found {
				sqlString = `select sum(co) as sco from (
				select count(*) as co from movement where tool_id = ` + options["ref_id"] + `  
				union select count(*) as co from event inner join groups nt on event.nervatype = nt.id and nt.groupvalue = 'tool' 
				where event.deleted = 0 and event.ref_id = ` + options["ref_id"] + `  
				union select count(*) as co from link where nervatype_2 = ( 
					select id from groups where groupname = 'nervatype' and groupvalue = 'tool') and ref_id_2 = ` + options["ref_id"] + `
			) foo`
			}

		case "trans":
			//(address,contact), event,link
			if _, found := options["ref_id"]; found {
				sqlString = `select sum(co) as sco from (
				select count(*) as co from event inner join groups nt on event.nervatype = nt.id and nt.groupvalue = 'trans' 
				where event.deleted = 0 and event.ref_id = ` + options["ref_id"] + `  
				union select count(*) as co from link where nervatype_2 = ( 
					select id from groups where groupname = 'nervatype' and groupvalue = 'trans') and ref_id_2 = ` + options["ref_id"] + `
			) foo`
			}

		default:
			return nil, errors.New(ntura.GetMessage("integrity_error"))

		}

	case "delete_deffields":
		if _, found := options["nervatype"]; found {
			if _, found := options["ref_id"]; found {
				sqlString = `select fv.id as id from deffield df 
          inner join groups nt on ((df.nervatype=nt.id) 
					  and (nt.groupvalue='` + options["nervatype"] + `'))
          inner join fieldvalue fv on ((df.fieldname=fv.fieldname) 
					  and (fv.deleted=0) and (fv.ref_id=` + options["ref_id"] + `))`
			}
		}

	case "update_deffields":
		if _, found := options["fieldname"]; found {
			sqlString = `select ft.groupvalue as fieldtype from deffield df
        inner join groups ft on (df.fieldtype=ft.id)
        where df.fieldname='` + options["fieldname"] + `'`
		} else {
			if _, found := options["nervatype"]; found {
				if _, found := options["ref_id"]; found {
					if options["ref_id"] == "" {
						options["ref_id"] = "null"
					}
					sqlString = `select df.fieldname as fieldname, fv.id as fieldvalue_id
						from deffield df
						inner join groups nt on ((df.nervatype=nt.id) and (nt.groupvalue='` + options["nervatype"] + `'))
						left join fieldvalue fv on ((df.fieldname=fv.fieldname) and (fv.deleted=0) and (fv.ref_id=` + options["ref_id"] + `))
					union select 'fieldtype_string' as fieldname, id as fieldvalue_id 
						from groups where (groupname='fieldtype') and (groupvalue='string')
					union select 'nervatype_id' as fieldname, id as fieldvalue_id 
						from groups where (groupname='nervatype') and (groupvalue='` + options["nervatype"] + `')`
				} else {
					sqlString = `select df.fieldname, ft.groupvalue as fieldtype, df.addnew, df.visible
						from deffield df
						inner join groups nt on (df.nervatype=nt.id)
							and (nt.groupvalue='` + options["nervatype"] + `')
						inner join groups ft on (df.fieldtype=ft.id)
						where df.deleted=0`
				}
			}
		}
	default:
		return result, errors.New(ntura.GetMessage("missing_fieldname"))
	}
	//print(sqlString)
	return ds.QuerySQL(sqlString, params, trans)
}

//GetIDfromRefnumber - returns dbs id from public key
func (ds *SQLDriver) GetIDfromRefnumber(options IM) (id int, info IM, err error) {
	return
}
