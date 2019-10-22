package nervatura

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/beevik/etree"
	"github.com/dgrijalva/jwt-go"
)

/*
API - Application Programming Interface

See more docs and examples: https://nervatura.github.io/nervatura-docs/#/api
*/
type API struct {
	NStore *NervaStore
}

func (api *API) getHashvalue(refname string) (string, error) {
	hashtable := api.NStore.settings.Hashtable
	err := api.NStore.ds.CheckHashtable(hashtable)
	if err != nil {
		return "", err
	}
	query := []Query{Query{
		Fields: []string{"*"}, From: hashtable, Filters: []Filter{
			Filter{Field: "refname", Comp: "==", Value: "'" + refname + "'"},
		}}}
	rows, err := api.NStore.ds.Query(query, nil)
	if err != nil {
		return "", err
	}
	if len(rows) > 0 {
		return rows[0]["value"].(string), nil
	}
	return "", nil
}

func (api *API) authUser(options IM) error {

	if !api.NStore.ds.Connection().Connected {
		if _, found := options["database"]; !found {
			return errors.New(GetMessage("missing_database"))
		}
		database := options["database"].(string)
		if _, found := api.NStore.settings.Alias[database]; !found {
			return errors.New(GetMessage("missing_database"))
		}
		err := api.NStore.ds.CreateConnection(database, api.NStore.settings.Alias[database], api.NStore.settings)
		if err != nil {
			return err
		}
	}

	if _, found := options["username"]; !found {
		return errors.New(GetMessage("missing_user"))
	}

	rows, err := api.NStore.ds.QueryKey(SM{"qkey": "user", "username": options["username"].(string)}, nil)
	if err != nil {
		return err
	}
	if len(rows) > 0 {
		api.NStore.User = rows[0]
	} else {
		query := []Query{Query{
			Fields: []string{"*"}, From: "customer", Filters: []Filter{
				Filter{Field: "inactive", Comp: "==", Value: "0"},
				Filter{Field: "deleted", Comp: "==", Value: "0"},
				Filter{Field: "custnumber", Comp: "==", Value: "'" + options["username"].(string) + "'"}}}}
		rows, err := api.NStore.ds.Query(query, nil)
		if err != nil {
			return err
		}
		if len(rows) > 0 {
			api.NStore.Customer = rows[0]
			rows, err = api.NStore.ds.QueryKey(SM{"qkey": "user_guest"}, nil)
			if err != nil {
				return err
			}
			if len(rows) > 0 {
				api.NStore.User = rows[0]
			} else {
				return errors.New(GetMessage("unknown_user"))
			}
		} else {
			return errors.New(GetMessage("unknown_user"))
		}
	}
	return nil
}

/*
AuthToken - create/refresh a JWT token
*/
func (api *API) AuthToken() (string, error) {
	conn := api.NStore.ds.Connection()
	if !conn.Connected {
		return "", errors.New(GetMessage("not_connect"))
	}
	username := api.NStore.User["username"].(string)
	if api.NStore.Customer != nil {
		username = api.NStore.Customer["custnumber"].(string)
	}
	expirationTime := time.Now().Add(api.NStore.settings.TokenExp * time.Hour)
	claims := NTClaims{
		username,
		conn.Alias,
		jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			Issuer:    api.NStore.settings.TokenIss,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = api.NStore.settings.TokenKid
	return token.SignedString([]byte(api.NStore.settings.TokenKey))
}

/*
AuthTokenLogin - database JWT token auth.

Example:

  options := map[string]interface{}{"token": "JWT_token"}
  err := getAPI().AuthTokenLogin(options)

*/
func (api *API) AuthTokenLogin(options IM) error {
	if _, found := options["token"]; !found {
		return errors.New(GetMessage("missing_required_field") + ": token")
	}
	tokenString := options["token"].(string)
	key := api.NStore.settings.TokenKey
	if _, found := options["key"]; found {
		key = options["key"].(string)
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Unexpected signing method: " + token.Header["alg"].(string))
		}
		return []byte(key), nil
	})
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if _, found := claims["database"]; !found {
			if _, found := api.NStore.settings.Alias["default"]; !found {
				return errors.New(GetMessage("missing_database"))
			}
			options["database"] = "default"
		}
		options["database"] = claims["database"]
		options["username"] = ""
		if _, found := claims["username"]; found {
			options["username"] = claims["username"]
		} else if _, found := claims["custnumber"]; found {
			options["username"] = claims["custnumber"]
		} else if _, found := claims["email"]; found {
			options["username"] = claims["email"]
		} else if _, found := claims["phone_number"]; found {
			options["username"] = claims["phone_number"]
		}
		if options["username"] == "" {
			return errors.New(GetMessage("missing_user"))
		}
		return api.authUser(options)

	} else if ve, ok := err.(*jwt.ValidationError); ok {
		if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
			return errors.New("Token is either expired or not active yet")
		}
		return err
	}
	return err
}

/*
AuthPassword - set/change a user password

Example:

  options = map[string]interface{}{
    "username": "demo",
    "password": "321",
    "confirm": "321"}
  err = api.AuthPassword(options)

*/
func (api *API) AuthPassword(options IM) error {
	if _, found := options["username"]; !found {
		if _, found := options["custnumber"]; !found {
			return errors.New(GetMessage("missing_required_field") + ": username or custnumber")
		}
	}
	if _, found := options["password"]; !found {
		return errors.New(GetMessage("missing_required_field") + ": password")
	}
	if _, found := options["confirm"]; !found {
		return errors.New(GetMessage("missing_required_field") + ": confirm")
	}
	if options["password"] == "" {
		return errors.New(GetMessage("empty_password"))
	}
	if options["password"] != options["confirm"] {
		return errors.New(GetMessage("verify_password"))
	}
	if !api.NStore.ds.Connection().Connected {
		return errors.New(GetMessage("not_connect"))
	}
	refname := ""
	if _, found := options["custnumber"]; found && api.NStore.Customer != nil {
		if options["custnumber"] == api.NStore.Customer["custnumber"] {
			refname = "customer" + strconv.Itoa(api.NStore.Customer["id"].(int))
		}
	} else if _, found := options["username"]; found && api.NStore.User != nil {
		if options["username"] == api.NStore.User["username"] {
			refname = "employee" + strconv.Itoa(api.NStore.User["id"].(int))
		}
	}
	if refname == "" {
		var query []Query
		if _, found := options["custnumber"]; found {
			query = []Query{Query{
				Fields: []string{"*"}, From: "customer", Filters: []Filter{
					Filter{Field: "inactive", Comp: "==", Value: "0"},
					Filter{Field: "deleted", Comp: "==", Value: "0"},
					Filter{Field: "custnumber", Comp: "==", Value: "'" + options["custnumber"].(string) + "'"},
				}}}
			refname = "customer"
		} else {
			query = []Query{Query{
				Fields: []string{"*"}, From: "employee", Filters: []Filter{
					Filter{Field: "deleted", Comp: "==", Value: "0"},
					Filter{Field: "username", Comp: "==", Value: "'" + options["username"].(string) + "'"},
				}}}
			refname = "employee"
		}
		rows, err := api.NStore.ds.Query(query, nil)
		if err != nil {
			return err
		}
		if len(rows) > 0 {
			refname += strconv.Itoa(rows[0]["id"].(int))
		} else {
			return errors.New(GetMessage("unknown_user"))
		}
	}
	refname = getMD5Hash(refname)
	hash, err := argon2id.CreateHash(options["password"].(string), argon2id.DefaultParams)
	if err != nil {
		return err
	}
	return api.NStore.ds.UpdateHashtable(api.NStore.settings.Hashtable, refname, hash)
}

/*
AuthUserLogin - database user login

Returns a access token.

  options := map[string]interface{}{
    "database": "alias_name",
    "username": "username",
    "password": "password"}
  token, err := getAPI().AuthUserLogin(options)

*/
func (api *API) AuthUserLogin(options IM) (string, error) {

	if _, found := options["database"]; !found {
		if _, found := api.NStore.settings.Alias["default"]; !found {
			return "", errors.New(GetMessage("missing_database"))
		}
		options["database"] = api.NStore.settings.Alias["default"]
	}
	password := ""
	if _, found := options["password"]; found && options["password"] != nil {
		password = options["password"].(string)
	}

	err := api.authUser(options)
	if err != nil {
		return "", err
	}

	refname := "employee" + strconv.Itoa(api.NStore.User["id"].(int))
	if api.NStore.Customer != nil {
		refname = "customer" + strconv.Itoa(api.NStore.Customer["id"].(int))
	}
	refname = getMD5Hash(refname)
	hash, err := api.getHashvalue(refname)

	if password != "" && hash != "" {
		match, err := argon2id.ComparePasswordAndHash(password, hash)
		if err != nil {
			return "", err
		}
		if match == false {
			return "", errors.New(GetMessage("wrong_password"))
		}
	} else if password != hash {
		return "", errors.New(GetMessage("wrong_password"))
	}

	return api.AuthToken()
}

/*
DatabaseCreate - create a Nervatura Database

All data will be destroyed!

Example:

  options := map[string]interface{}{
    "database": alias,
    "demo": "true",
    "report_dir": "../demo/templates"}
  _, err := getAPI().DatabaseCreate(options)

*/
func (api *API) DatabaseCreate(options IM) ([]SM, error) {
	logData := []SM{}

	if _, found := options["database"]; !found || GetIType(options["database"]) != "string" {
		return logData, errors.New(GetMessage("missing_required_field") + ": database")
	}

	//check connect
	database := options["database"].(string)
	if err := api.NStore.ds.CreateConnection(database, api.NStore.settings.Alias[database], api.NStore.settings); err != nil {
		logData = append(logData, SM{
			"stamp":   time.Now().Format("2006-01-02 15:04:05"),
			"state":   "err",
			"message": GetMessage("not_connect")})
		return logData, errors.New(GetMessage("not_connect"))
	}

	logData, err := api.NStore.ds.CreateDatabase(logData)
	if err != nil {
		logData = append(logData, SM{
			"stamp":   time.Now().Format("2006-01-02 15:04:05"),
			"state":   "err",
			"message": err.Error()})
		return logData, err
	}

	if _, found := options["demo"]; found {
		if options["demo"] == "true" {
			options["logData"] = logData
			logData, err = api.demoDatabase(options)
			if err != nil {
				logData = append(logData, SM{
					"stamp":   time.Now().Format("2006-01-02 15:04:05"),
					"state":   "err",
					"message": err.Error()})
				return logData, err
			}
		}
	}

	logData = append(logData, SM{
		"stamp":   time.Now().Format("2006-01-02 15:04:05"),
		"state":   "log",
		"message": GetMessage("info_create_ok")})

	return logData, nil
}

func (api *API) demoDatabase(options IM) ([]SM, error) {
	var err error
	logData := options["logData"].([]SM)
	data := demoData()

	//create 3 departments and 3 eventgroups
	result, err := api.APIPost("groups", data["groups"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr := ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"datatype": "groups",
		"result":   resultStr[1:],
	})

	//customer
	//-> def. 4 customer additional data (float,date,valuelist,customer types),
	//-> create 3 customers,
	//-> and more create and link to contacts, addresses and events
	result, err = api.APIPost("deffield", data["customer"].(IM)["deffield"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "customer",
		"datatype": "deffield",
		"result":   resultStr[1:],
	})

	result, err = api.APIPost("customer", data["customer"].(IM)["customer"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "customer",
		"datatype": "customer",
		"result":   resultStr[1:],
	})

	result, err = api.APIPost("address", data["customer"].(IM)["address"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "customer",
		"datatype": "address",
		"result":   resultStr[1:],
	})

	result, err = api.APIPost("contact", data["customer"].(IM)["contact"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "customer",
		"datatype": "contact",
		"result":   resultStr[1:],
	})

	result, err = api.APIPost("event", data["customer"].(IM)["event"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "customer",
		"datatype": "event",
		"result":   resultStr[1:],
	})

	//employee
	//-> def. 1 employee additional data (integer type),
	//->create 1 employee,
	//->and more create and link to contact, address and event
	result, err = api.APIPost("deffield", data["employee"].(IM)["deffield"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "employee",
		"datatype": "deffield",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("employee", data["employee"].(IM)["employee"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "employee",
		"datatype": "employee",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("address", data["employee"].(IM)["address"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "employee",
		"datatype": "address",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("contact", data["employee"].(IM)["contact"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "employee",
		"datatype": "contact",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("event", data["employee"].(IM)["event"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "employee",
		"datatype": "event",
		"result":   resultStr[1:],
	})

	//product
	//-> def. 3 product additional data (product,integer and valulist types),
	//->create 13 products,
	//->and more create and link to barcodes, events, prices, additional data
	result, err = api.APIPost("deffield", data["product"].(IM)["deffield"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "product",
		"datatype": "deffield",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("product", data["product"].(IM)["product"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "product",
		"datatype": "product",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("barcode", data["product"].(IM)["barcode"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "product",
		"datatype": "barcode",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("price", data["product"].(IM)["price"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "product",
		"datatype": "price",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("event", data["product"].(IM)["event"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "product",
		"datatype": "event",
		"result":   resultStr[1:],
	})

	//project
	//-> def. 2 project additional data,
	//->create 1 project,
	//->and more create and link to contact, address and event
	result, err = api.APIPost("deffield", data["project"].(IM)["deffield"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "project",
		"datatype": "deffield",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("project", data["project"].(IM)["project"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "project",
		"datatype": "project",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("address", data["project"].(IM)["address"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "project",
		"datatype": "address",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("contact", data["project"].(IM)["contact"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "project",
		"datatype": "contact",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("event", data["project"].(IM)["event"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "project",
		"datatype": "event",
		"result":   resultStr[1:],
	})

	//tool
	//-> def. 2 tool additional data,
	//->create 3 tools,
	//->and more create and link to event and additional data
	result, err = api.APIPost("deffield", data["tool"].(IM)["deffield"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "tool",
		"datatype": "deffield",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("tool", data["tool"].(IM)["tool"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "tool",
		"datatype": "tool",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("event", data["tool"].(IM)["event"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "tool",
		"datatype": "event",
		"result":   resultStr[1:],
	})

	//create +1 warehouse
	result, err = api.APIPost("place", data["place"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"datatype": "place",
		"result":   resultStr[1:],
	})

	//documents
	//offer, order, invoice, worksheet, rent
	result, err = api.APIPost("trans", data["trans_item"].(IM)["trans"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "document(offer,order,invoice,rent,worksheet)",
		"datatype": "trans",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("item", data["trans_item"].(IM)["item"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"datatype": "item",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("link", data["trans_item"].(IM)["link"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"datatype": "link",
		"result":   resultStr[1:],
	})

	//payments
	//bank and petty cash
	result, err = api.APIPost("trans", data["trans_payment"].(IM)["trans"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "payment(bank,petty cash)",
		"datatype": "trans",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("payment", data["trans_payment"].(IM)["payment"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"datatype": "payment",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("link", data["trans_payment"].(IM)["link"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"datatype": "link",
		"result":   resultStr[1:],
	})

	//stock control
	//tool movement (for employee)
	//create delivery,stock transfer,correction
	//formula and production
	result, err = api.APIPost("trans", data["trans_movement"].(IM)["trans"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "stock control(tool movement,delivery,stock transfer,correction,formula,production)",
		"datatype": "trans",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("movement", data["trans_movement"].(IM)["movement"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"datatype": "movement",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("link", data["trans_movement"].(IM)["link"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"datatype": "link",
		"result":   resultStr[1:],
	})

	//sample menus and menufields
	result, err = api.APIPost("ui_menu", data["menu"].(IM)["ui_menu"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"section":  "sample menus",
		"datatype": "ui_menu",
		"result":   resultStr[1:],
	})
	result, err = api.APIPost("ui_menufields", data["menu"].(IM)["ui_menufields"].([]IM))
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(result); index++ {
		resultStr += "," + strconv.Itoa(result[index])
	}
	logData = append(logData, SM{
		"stamp":    time.Now().Format("2006-01-02 15:04:05"),
		"state":    "demo",
		"datatype": "ui_menufields",
		"result":   resultStr[1:],
	})

	//load general reports and other templates
	reports, err := api.ReportList(options)
	if err != nil {
		return logData, err
	}
	resultStr = ""
	for index := 0; index < len(reports); index++ {
		params := IM{"reportkey": reports[index]["reportkey"]}
		if _, found := options["report_dir"]; found {
			params["report_dir"] = options["report_dir"]
		}
		_, err = api.ReportInstall(params)
		if err != nil {
			return logData, err
		}
		resultStr += "," + reports[index]["reportkey"].(string)
	}
	logData = append(logData, SM{
		"stamp":   time.Now().Format("2006-01-02 15:04:05"),
		"state":   "demo",
		"section": "report templates",
		"result":  resultStr[1:],
	})

	return logData, err
}

/*
APIDelete - delete a record

Examples:

  Delete data by ID:

  options = map[string]interface{}{"nervatype": "address", "id": 2}
  err = api.APIDelete(options)

  Delete data by Key:

  options = map[string]interface{}{"nervatype": "address", "key": "customer/DMCUST/00001~1"}
  err = api.APIDelete(options)
*/
func (api *API) APIDelete(options IM) error {
	if _, found := options["id"]; found {
		if GetIType(options["id"]) == "string" {
			refID, err := strconv.Atoi(options["id"].(string))
			if err == nil {
				options["ref_id"] = refID
			}
		} else if GetIType(options["id"]) == "int" {
			options["ref_id"] = options["id"].(int)
		}
	}
	return api.NStore.DeleteData(IM{
		"nervatype": options["nervatype"],
		"ref_id":    options["ref_id"],
		"refnumber": options["key"],
	})
}

/*
APIGet - returns one or more records

Examples:

  Find data by Filter:

  options = map[string]interface{}{"nervatype": "customer", "metadata": true,
    "filter": "custname;==;First Customer Co.|custnumber;in;DMCUST/00001,DMCUST/00002"}
  _, err = api.APIGet(options)

  Find data by IDs:

  options = map[string]interface{}{"nervatype": "customer", "metadata": true, "ids": "2,4"}
  _, err = api.APIGet(options)

*/
func (api *API) APIGet(options IM) (results []IM, err error) {
	if _, found := options["nervatype"]; !found || GetIType(options["nervatype"]) != "string" {
		return results, errors.New(GetMessage("missing_required_field") + ": nervatype")
	}
	nervatype := options["nervatype"].(string)
	if _, found := api.NStore.models[nervatype]; !found {
		return results, errors.New(GetMessage("invalid_nervatype") + " " + nervatype)
	}
	metadata := false
	if _, found := options["metadata"]; found {
		if GetIType(options["metadata"]) == "string" {
			bvalue, err := strconv.ParseBool(options["metadata"].(string))
			if err == nil {
				metadata = bvalue
			}
		} else if GetIType(options["metadata"]) == "bool" {
			metadata = options["metadata"].(bool)
		}
	}

	query := []Query{Query{
		Fields: []string{"*"}, From: nervatype, Filters: []Filter{
			Filter{Field: "deleted", Comp: "==", Value: "0"},
		}}}
	if _, found := options["ids"]; found && GetIType(options["ids"]) == "string" {
		query[0].Filters = append(query[0].Filters, Filter{Field: "id", Comp: "in", Value: options["ids"].(string)})
	} else if _, found := options["filter"]; found && GetIType(options["filter"]) == "string" {
		filters := strings.Split(options["filter"].(string), "|")
		for index := 0; index < len(filters); index++ {
			fields := strings.Split(filters[index], ";")
			if len(fields) != 3 {
				return results, errors.New(GetMessage("invalid_value") + "- filter: " + filters[index])
			}
			if _, found := api.NStore.models[nervatype].(IM)[fields[0]]; !found {
				return results, errors.New(GetMessage("invalid_value") + "- fieldname: " + fields[0])
			}
			switch fields[1] {
			case "==", "!=", "<", "<=", ">", ">=", "in":
			default:
				return results, errors.New(GetMessage("invalid_value") + "- comparison: " + fields[1])
			}
			value := fields[2]
			switch api.NStore.models[nervatype].(IM)[fields[0]].(MF).Type {
			case "string", "text", "date":
				fields[2] = strings.ReplaceAll(fields[2], "'", "")
				values := strings.Split(fields[2], ",")
				value = ""
				for fld := 0; fld < len(values); fld++ {
					value += ",'" + values[fld] + "'"
				}
				value = value[1:]
			}
			query[0].Filters = append(query[0].Filters, Filter{Field: fields[0], Comp: fields[1], Value: value})
		}
	} else {
		return results, errors.New(GetMessage("missing_required_field") + ": filter or ids")
	}

	results, err = api.NStore.ds.Query(query, nil)
	if err != nil {
		return results, err
	}
	if len(results) > 0 && metadata {
		switch nervatype {
		case "address", "barcode", "contact", "currency", "customer", "employee", "event", "groups",
			"item", "link", "log", "movement", "price", "place", "product", "project", "rate",
			"tax", "tool", "trans":
			ids := ""
			for index := 0; index < len(results); index++ {
				ids += "," + strconv.Itoa(results[index]["id"].(int))
			}
			ids = ids[1:]
			metadata, err := api.NStore.ds.QueryKey(SM{"qkey": "metadata", "nervatype": nervatype, "ids": ids}, nil)
			if err != nil {
				return results, err
			}
			if len(metadata) > 0 {
				for index := 0; index < len(results); index++ {
					results[index]["metadata"] = []IM{}
					for mi := 0; mi < len(metadata); mi++ {
						if metadata[mi]["ref_id"] == results[index]["id"] {
							results[index]["metadata"] = append(results[index]["metadata"].([]IM), IM{
								"id":        metadata[mi]["id"],
								"fieldname": metadata[mi]["fieldname"],
								"fieldtype": metadata[mi]["fieldtype"],
								"value":     metadata[mi]["value"],
								"notes":     metadata[mi]["notes"],
							})
						}
					}
				}
			}
		}
	}
	return results, err
}

/*
APIView - run raw SQL queries in safe mode

Only "select" queries and functions can be executed. Changes to the data are not saved in the database.

Examples:

  options := []map[string]interface{}{
    map[string]interface{}{
      "key":    "customers",
      "text":   "select c.id, ct.groupvalue as custtype, c.custnumber, c.custname from customer c inner join groups ct on c.custtype = ct.id where c.deleted = 0 and c.custnumber <> 'HOME'",
    },
    map[string]interface{}{
      "key":    "invoices",
      "text":   "select t.id, t.transnumber, tt.groupvalue as transtype, td.groupvalue as direction, t.transdate, c.custname, t.curr, items.amount from trans t inner join groups tt on t.transtype = tt.id inner join groups td on t.direction = td.id inner join customer c on t.customer_id = c.id inner join ( select trans_id, sum(amount) amount from item where deleted = 0 group by trans_id) items on t.id = items.trans_id where t.deleted = 0 and tt.groupvalue = 'invoice'",
    },
  }
  _, err = api.APIView(options)
*/
func (api *API) APIView(options []IM) (results IM, err error) {
	results = IM{}
	var trans interface{}
	if api.NStore.ds.Properties().Transaction {
		trans, err = api.NStore.ds.BeginTransaction()
		if err != nil {
			return results, err
		}
	}

	defer func() {
		pe := recover()
		if trans != nil {
			api.NStore.ds.RollbackTransaction(trans)
		}
		if pe != nil {
			panic(pe)
		}
	}()

	for index := 0; index < len(options); index++ {
		if _, found := options[index]["key"]; !found || GetIType(options[index]["key"]) != "string" {
			return results, errors.New(GetMessage("missing_required_field") + ": key")
		}
		if _, found := options[index]["text"]; !found || GetIType(options[index]["text"]) != "string" {
			return results, errors.New(GetMessage("missing_required_field") + ": text")
		}
		result, err := api.NStore.ds.QuerySQL(options[index]["text"].(string), trans)
		if err != nil {
			return results, err
		}
		results[options[index]["key"].(string)] = result
	}
	return results, err
}

/*
APIFunction - call a server-side function

Examples:

  The next value from the numberdef table (customer, product, invoice etc.):

  options := map[string]interface{}{
    "key": "nextNumber",
    "values": map[string]interface{}{
      "numberkey": "custnumber",
      "step":      false,
    },
  }
  _, err = api.APIFunction(options)

  Product price (current date, all customer, all qty):

  options = map[string]interface{}{
    "key": "getPriceValue",
    "values": map[string]interface{}{
      "curr":        "EUR",
      "product_id":  2,
      "customer_id": 2,
    },
  }
  _, err = api.APIFunction(options)

  Email sending with attached report:

  options = map[string]interface{}{
    "key": "sendEmail",
    "values": map[string]interface{}{
      "provider": "smtp",
      "email": map[string]interface{}{
        "from": "info@nervatura.com", "name": "Nervatura" },
      "recipients": []interface{}{
        map[string]interface{}{ "email": "sample@company.com" }},
      "subject": "Demo Invoice",
      "text": "Email sending with attached invoice",
      "attachments" : []interface{}{
        map[string]interface{}{
          "reportkey":  "ntr_invoice_en",
          "nervatype": "trans",
          "refnumber": "DMINV/00001" }},
    },
  }


*/
func (api *API) APIFunction(options IM) (results interface{}, err error) {
	if _, found := options["key"]; !found || GetIType(options["key"]) != "string" {
		return results, errors.New(GetMessage("missing_required_field") + ": key")
	}
	if _, found := options["values"]; !found || GetIType(options["values"]) != "map[string]interface{}" {
		return results, errors.New(GetMessage("missing_required_field") + ": values")
	}
	return api.NStore.GetService(options["key"].(string), options["values"].(IM))
}

/*
APIPost - Add or update one or more items

If the ID (or Key) value is missing, it creates a new item. Returns the all new/updated IDs values.

Examples:

  addressData := []map[string]interface{}{
    map[string]interface{}{
      "nervatype":         10,
      "ref_id":            2,
      "zipcode":           "12345",
      "city":              "BigCity",
      "notes":             "Create a new item by IDs",
      "address_metadata1": "value1",
      "address_metadata2": "value2~note2"},
    map[string]interface{}{
      "id":                6,
      "zipcode":           "54321",
      "city":              "BigCity",
      "notes":             "Update an item by IDs",
      "address_metadata1": "value1",
      "address_metadata2": "value2~note2"},
    map[string]interface{}{
      "keys": map[string]interface{}{
        "nervatype": "customer",
        "ref_id":    "customer/DMCUST/00001"},
      "zipcode":           "12345",
      "city":              "BigCity",
      "notes":             "Create a new item by Keys",
      "address_metadata1": "value1",
      "address_metadata2": "value2~note2"},
    map[string]interface{}{
      "keys": map[string]interface{}{
        "id": "customer/DMCUST/00001~1"},
      "zipcode":           "54321",
      "city":              "BigCity",
      "notes":             "Update an item by Keys",
      "address_metadata1": "value1",
      "address_metadata2": "value2~note2"}}

  _, err = api.APIPost("address", addressData)

*/
func (api *API) APIPost(nervatype string, data []IM) (results []int, err error) {
	if _, found := api.NStore.models[nervatype]; !found {
		return results, errors.New(GetMessage("invalid_nervatype") + " " + nervatype)
	}

	if nervatype == "trans" {
		for index := 0; index < len(data); index++ {
			_, fkeys := data[index]["keys"]
			ftranstype := false
			fcustomer := false
			if fkeys {
				_, ftranstype = data[index]["keys"].(IM)["transtype"]
				_, fcustomer = data[index]["keys"].(IM)["customer_id"]
			}
			if !(fkeys && ftranstype && fcustomer) {
				options := SM{"qkey": "post_transtype"}
				if _, found := data[index]["transtype"]; found && GetIType(data[index]["transtype"]) == "int" {
					options["transtype_id"] = strconv.Itoa(data[index]["transtype"].(int))
				} else {
					options["transtype_id"] = "null"
				}
				if fkeys && ftranstype {
					if GetIType(data[index]["keys"].(IM)["transtype"]) == "string" {
						options["transtype_key"] = data[index]["keys"].(IM)["transtype"].(string)
					} else {
						options["transtype_key"] = "null"
					}
				} else {
					options["transtype_key"] = "null"
				}
				if _, found := data[index]["customer_id"]; found {
					if GetIType(data[index]["customer_id"]) == "int" {
						options["customer_id"] = strconv.Itoa(data[index]["customer_id"].(int))
					} else {
						options["customer_id"] = "null"
					}
				} else {
					options["customer_id"] = "null"
				}
				if fkeys && fcustomer {
					if GetIType(data[index]["keys"].(IM)["customer_id"]) == "string" {
						options["custnumber"] = data[index]["keys"].(IM)["customer_id"].(string)
					} else {
						options["custnumber"] = "null"
					}
				} else {
					options["custnumber"] = "null"
				}
				if _, found := data[index]["id"]; found {
					if GetIType(data[index]["id"]) == "int" {
						options["trans_id"] = strconv.Itoa(data[index]["id"].(int))
					} else {
						options["trans_id"] = "null"
					}
				} else {
					options["trans_id"] = "null"
				}
				info, err := api.NStore.ds.QueryKey(options, nil)
				if err != nil {
					return results, err
				}
				if len(info) > 0 {
					if !fkeys {
						data[index]["keys"] = IM{}
					}
					keys := map[string][]interface{}{}
					for index := 0; index < len(info); index++ {
						keys[info[index]["rtype"].(string)] = IL{info[index]["transtype"], info[index]["custnumber"]}
					}
					if _, found := keys["groups"]; found {
						if !ftranstype {
							data[index]["keys"].(IM)["transtype"] = keys["groups"][0]
						}
					} else if _, found := keys["trans"]; found {
						if !ftranstype {
							data[index]["keys"].(IM)["transtype"] = keys["trans"][0]
						}
					}
					if _, found := keys["customer"]; found {
						if !fcustomer {
							data[index]["keys"].(IM)["customer_id"] = keys["customer"][1]
						}
					} else if _, found := keys["trans"]; found {
						if !fcustomer && keys["trans"][1] != nil {
							data[index]["keys"].(IM)["customer_id"] = keys["trans"][1]
						}
					}
				}
			}
		}
	}

	for index := 0; index < len(data); index++ {
		if _, found := data[index]["keys"]; found {
			for key, value := range data[index]["keys"].(IM) {
				info := IM{"fieldname": key, "reftype": "id"}
				switch key {
				case "id":
					info["nervatype"] = nervatype
					info["refnumber"] = value

				case "ref_id", "ref_id_1", "ref_id_2":
					info["nervatype"] = strings.Split(value.(string), "/")[0]
					info["refnumber"] = strings.ReplaceAll(value.(string), strings.Split(value.(string), "/")[0]+"/", "")

				default:
					if _, found := api.NStore.models[nervatype].(IM)[key]; found {
						if api.NStore.models[nervatype].(IM)[key].(MF).References != nil {
							info["nervatype"] = api.NStore.models[nervatype].(IM)[key].(MF).References[0]
							if info["nervatype"] == "groups" {
								switch key {
								case "nervatype_1", "nervatype_2":
									info["refnumber"] = "nervatype~" + value.(string)
								default:
									info["refnumber"] = key + "~" + value.(string)
								}
							} else {
								info["refnumber"] = value
								if key == "customer_id" && data[index]["keys"].(IM)["transtype"] == "invoice" {
									info["extra_info"] = true
								}
							}
						} else if api.NStore.models[nervatype].(IM)["_key"].(SL)[0] == key {
							if GetIType(value) == "string" && value == "numberdef" {
								info["reftype"] = "numberdef"
								info["numberkey"] = key
								info["step"] = true
								info["insert_key"] = false
							} else if GetIType(value) == "[]interface{}" {
								info["reftype"] = "numberdef"
								if len(value.(IL)) > 1 {
									info["numberkey"] = value.(IL)[1]
								}
								info["step"] = true
								info["insert_key"] = false
							} else {
								info["nervatype"] = nervatype
								info["refnumber"] = value
								info["fieldname"] = "id"
							}
						}
					}
					if _, found := info["nervatype"]; !found {
						if info["reftype"] == "id" {
							info["nervatype"] = "invalid"
							info["refnumber"] = value
						}
					}
				}
				if info["reftype"] == "numberdef" {
					retnumber, err := api.NStore.nextNumber(info)
					if err != nil {
						return results, err
					}
					data[index][info["fieldname"].(string)] = retnumber
				} else {
					refValues, err := api.NStore.GetInfofromRefnumber(info)
					if err != nil {
						return results, err
					}
					data[index][info["fieldname"].(string)] = refValues["id"]
					if _, found := info["extra_info"]; found {
						if info["extra_info"].(bool) {
							data[index]["trans_custinvoice_compname"] = refValues["compname"]
							data[index]["trans_custinvoice_comptax"] = refValues["comptax"]
							data[index]["trans_custinvoice_compaddress"] = refValues["compaddress"]
							data[index]["trans_custinvoice_custname"] = refValues["custname"]
							data[index]["trans_custinvoice_custtax"] = refValues["custtax"]
							data[index]["trans_custinvoice_custaddress"] = refValues["custaddress"]
						}
					}
				}
			}
		}
	}

	model := api.NStore.models[nervatype].(IM)
	for index := 0; index < len(data); index++ {
		if _, found := data[index]["keys"]; found {
			delete(data[index], "keys")
		}
		if _, found := data[index]["id"]; !found {
			for ikey, ifield := range model {
				switch ikey {
				case "_access", "_key", "_fields", "id":

				case "crdate":
					if ifield.(MF).Type == "datetime" {
						data[index]["crdate"] = time.Now().Format("2006-01-02T15:04:05-0700")
					} else if ifield.(MF).Type == "date" {
						if _, found := data[index]["crdate"]; !found {
							data[index]["crdate"] = time.Now().Format("2006-01-02")
						}
					}

				case "cruser_id":
					if api.NStore.User != nil {
						data[index]["cruser_id"] = api.NStore.User["id"]
					} else {
						data[index]["cruser_id"] = 1
					}

				default:
					if ifield.(MF).NotNull && ifield.(MF).Default == nil {
						if _, found := data[index][ikey]; !found {
							return results, errors.New(GetMessage("missing_required_field") + " " + ikey)
						}
					}
				}
			}
			if nervatype == "trans" {
				if _, found := data[index]["trans_transcast"]; !found {
					data[index]["trans_transcast"] = "normal"
				}
			}
		} else {
			if _, found := data[index]["crdate"]; found {
				if _, found := model["crdate"]; found {
					delete(data[index], "crdate")
				}
			}
		}
	}

	var trans interface{}
	if api.NStore.ds.Properties().Transaction {
		trans, err = api.NStore.ds.BeginTransaction()
		if err != nil {
			return results, err
		}
	}

	defer func() {
		pe := recover()
		if trans != nil {
			if err != nil || pe != nil {
				api.NStore.ds.RollbackTransaction(trans)
			} else {
				err = api.NStore.ds.CommitTransaction(trans)
			}
		}
		if pe != nil {
			panic(pe)
		}
	}()

	for index := 0; index < len(data); index++ {
		id, err := api.NStore.UpdateData(IM{
			"nervatype":    nervatype,
			"values":       data[index],
			"validate":     true,
			"insert_row":   true,
			"insert_field": true,
			"trans":        trans,
		})
		if err != nil {
			return results, err
		}
		results = append(results, id)
	}

	return results, err
}

/*
Report - server-side PDF and Excel report generation

Examples:

  Customer PDF invoice:

  options := map[string]interface{}{
    "reportkey":   "ntr_invoice_en",
    "orientation": "portrait",
    "size":        "a4",
    "nervatype":   "trans",
    "refnumber":   "DMINV/00001",
  }
  _, err = api.Report(options)

  Customer invoice XML data:

  options = map[string]interface{}{
    "reportkey": "ntr_invoice_en",
    "output":    "xml",
    "nervatype": "trans",
    "refnumber": "DMINV/00001",
  }
  _, err = api.Report(options)

  Excel report:

  options = map[string]interface{}{
    "reportkey": "xls_vat_en",
    "filters": map[string]interface{}{
      "date_from": "2014-01-01",
      "date_to":   "2019-01-01",
      "curr":      "EUR",
    },
  }
  _, err = api.Report(options)

*/
func (api *API) Report(options IM) (results IM, err error) {
	return api.NStore.getReport(options)
}

/*
ReportList - returns all installable files from the NT_REPORT_DIR (environment variable) or report_dir (options) directory

Example:

  options := map[string]interface{}{
    "report_dir": "../demo/templates",
  }
  _, err = api.ReportList(options)

*/
func (api *API) ReportList(options IM) (results []IM, err error) {
	query := []Query{Query{
		Fields: []string{"id", "reportkey"}, From: "ui_report"}}
	reports, err := api.NStore.ds.Query(query, nil)
	if err != nil {
		return results, err
	}
	dbReports := IM{}
	for index := 0; index < len(reports); index++ {
		dbReports[reports[index]["reportkey"].(string)] = reports[index]["id"]
	}
	reportDir := api.NStore.settings.ReportDir
	if _, found := options["report_dir"]; found && GetIType(options["report_dir"]) == "string" {
		reportDir = options["report_dir"].(string)
	}
	filter := ""
	if _, found := options["label"]; found && GetIType(options["label"]) == "string" {
		filter = options["label"].(string)
	}
	results = []IM{}
	err = filepath.Walk(reportDir, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".xml" {
			xdoc := etree.NewDocument()
			if err := xdoc.ReadFromFile(path); err != nil {
				return err
			}
			if xreport := xdoc.SelectElement("report"); xreport != nil {
				report := IM{"installed": false, "label": ""}
				if attr := xreport.SelectAttr("reportkey"); attr != nil {
					report["reportkey"] = attr.Value
					if _, found := dbReports[attr.Value]; found {
						report["installed"] = true
					}
				}
				if attr := xreport.SelectAttr("repname"); attr != nil {
					report["repname"] = attr.Value
				}
				if attr := xreport.SelectAttr("description"); attr != nil {
					report["description"] = attr.Value
				}
				if attr := xreport.SelectAttr("filetype"); attr != nil {
					report["reptype"] = attr.Value
				}
				if attr := xreport.SelectAttr("nervatype"); attr != nil {
					report["label"] = attr.Value
				}
				if attr := xreport.SelectAttr("transtype"); attr != nil {
					report["label"] = attr.Value
				}
				report["filename"] = info.Name()
				if (filter == "") || (filter == report["label"]) {
					if (report["reptype"] == "xls") || (report["reptype"] == "ntr") {
						results = append(results, report)
					}
				}
			}
		}
		return err
	})
	if err != nil {
		return results, err
	}
	return results, err
}

/*
ReportDelete - delete a report from the database

Example:

  options := nt.IM{
    "reportkey": "ntr_cash_in_en",
  }
  err = api.ReportDelete(options)

*/
func (api *API) ReportDelete(options IM) (err error) {
	if _, found := options["reportkey"]; !found || GetIType(options["reportkey"]) != "string" {
		return errors.New(GetMessage("missing_required_field") + ": reportkey")
	}

	query := []Query{Query{
		Fields: []string{"*"}, From: "ui_report", Filters: []Filter{
			Filter{Field: "reportkey", Comp: "==", Value: "'" + options["reportkey"].(string) + "'"},
		}}}
	rows, err := api.NStore.ds.Query(query, nil)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return errors.New(GetMessage("missing_reportkey") + ": " + options["reportkey"].(string))
	}
	refID := rows[0]["id"].(int)

	var trans interface{}
	if api.NStore.ds.Properties().Transaction {
		trans, err = api.NStore.ds.BeginTransaction()
		if err != nil {
			return err
		}
	}

	defer func() {
		pe := recover()
		if trans != nil {
			if err != nil || pe != nil {
				api.NStore.ds.RollbackTransaction(trans)
			} else {
				err = api.NStore.ds.CommitTransaction(trans)
			}
		}
		if pe != nil {
			panic(pe)
		}
	}()

	query = []Query{Query{
		Fields: []string{"*"}, From: "ui_reportfields", Filters: []Filter{
			Filter{Field: "report_id", Comp: "==", Value: strconv.Itoa(refID)},
		}}}
	rows, err = api.NStore.ds.Query(query, nil)
	if err != nil {
		return err
	}
	for index := 0; index < len(rows); index++ {
		_, err = api.NStore.ds.Update(Update{IDKey: rows[index]["id"].(int), Model: "ui_reportfields", Trans: trans})
		if err != nil {
			return err
		}
	}

	query = []Query{Query{
		Fields: []string{"*"}, From: "ui_reportsources", Filters: []Filter{
			Filter{Field: "report_id", Comp: "==", Value: strconv.Itoa(refID)},
		}}}
	rows, err = api.NStore.ds.Query(query, nil)
	if err != nil {
		return err
	}
	for index := 0; index < len(rows); index++ {
		_, err = api.NStore.ds.Update(Update{IDKey: rows[index]["id"].(int), Model: "ui_reportsources", Trans: trans})
		if err != nil {
			return err
		}
	}

	query = []Query{Query{
		Fields: []string{"*"}, From: "ui_message", Filters: []Filter{
			Filter{Field: "secname", Comp: "==", Value: "'" + options["reportkey"].(string) + "'"},
		}}}
	rows, err = api.NStore.ds.Query(query, nil)
	if err != nil {
		return err
	}
	for index := 0; index < len(rows); index++ {
		_, err = api.NStore.ds.Update(Update{IDKey: rows[index]["id"].(int), Model: "ui_message", Trans: trans})
		if err != nil {
			return err
		}
	}

	_, err = api.NStore.ds.Update(Update{IDKey: refID, Model: "ui_report", Trans: trans})
	if err != nil {
		return err
	}

	return nil
}

/*
ReportInstall - install a report to the database

Example:

  options := nt.IM{
    "report_dir": "../demo/templates",
    "reportkey":  "ntr_cash_in_en",
  }
  _, err = api.ReportInstall(options)

*/
func (api *API) ReportInstall(options IM) (result int, err error) {
	if _, found := options["reportkey"]; !found || GetIType(options["reportkey"]) != "string" {
		return result, errors.New(GetMessage("missing_required_field") + ": reportkey")
	}
	reportDir := api.NStore.settings.ReportDir
	if _, found := options["report_dir"]; found && GetIType(options["report_dir"]) == "string" {
		reportDir = options["report_dir"].(string)
	}
	xdoc := etree.NewDocument()
	err = xdoc.ReadFromFile(filepath.Join(reportDir, options["reportkey"].(string)+".xml"))
	if err != nil {
		return result, err
	}
	report := IM{}
	xreport := xdoc.SelectElement("report")
	if xreport != nil {
		if attr := xreport.SelectAttr("reportkey"); attr != nil {
			report["reportkey"] = attr.Value
			query := []Query{Query{
				Fields: []string{"*"}, From: "ui_report", Filters: []Filter{
					Filter{Field: "reportkey", Comp: "==", Value: "'" + attr.Value + "'"},
				}}}
			rows, err := api.NStore.ds.Query(query, nil)
			if err != nil {
				return result, err
			}
			if len(rows) > 0 {
				return result, errors.New(GetMessage("exists_template"))
			}
		} else {
			return result, errors.New(GetMessage("invalid_template"))
		}
	} else {
		return result, errors.New(GetMessage("invalid_template"))
	}

	groups := IM{}
	query := []Query{Query{
		Fields: []string{"*"}, From: "groups", Filters: []Filter{
			Filter{Field: "groupname", Comp: "in",
				Value: "'nervatype','transtype','direction','filetype','fieldtype','wheretype'"},
		}}}
	rows, err := api.NStore.ds.Query(query, nil)
	if err != nil {
		return result, err
	}
	for index := 0; index < len(rows); index++ {
		if _, found := groups[rows[index]["groupname"].(string)]; !found {
			groups[rows[index]["groupname"].(string)] = IM{}
		}
		groups[rows[index]["groupname"].(string)].(IM)[rows[index]["groupvalue"].(string)] = rows[index]["id"]
	}

	if attr := xreport.SelectAttr("repname"); attr != nil {
		report["repname"] = attr.Value
	}
	if attr := xreport.SelectAttr("description"); attr != nil {
		report["description"] = attr.Value
	}
	if attr := xreport.SelectAttr("nervatype"); attr != nil {
		report["nervatype"] = groups["nervatype"].(IM)[attr.Value]
	}
	if attr := xreport.SelectAttr("filetype"); attr != nil {
		report["filetype"] = groups["filetype"].(IM)[attr.Value]
	}
	if attr := xreport.SelectAttr("transtype"); attr != nil {
		report["transtype"] = groups["transtype"].(IM)[attr.Value]
	}
	if attr := xreport.SelectAttr("direction"); attr != nil {
		report["direction"] = groups["direction"].(IM)[attr.Value]
	}
	if attr := xreport.SelectAttr("label"); attr != nil {
		report["label"] = attr.Value
	}
	if el := xreport.SelectElement("template"); el != nil {
		report["report"] = el.Text()
	}

	var trans interface{}
	if api.NStore.ds.Properties().Transaction {
		trans, err = api.NStore.ds.BeginTransaction()
		if err != nil {
			return 0, err
		}
	}

	defer func() {
		pe := recover()
		if trans != nil {
			if err != nil || pe != nil {
				api.NStore.ds.RollbackTransaction(trans)
			} else {
				err = api.NStore.ds.CommitTransaction(trans)
			}
		}
		if pe != nil {
			panic(pe)
		}
	}()

	result, err = api.NStore.ds.Update(Update{Model: "ui_report", Values: report, Trans: trans})
	if err != nil {
		return result, err
	}

	dsValues := IM{}
	cengine := strings.ReplaceAll(api.NStore.ds.Connection().Engine, "3", "")
	if ds := xreport.SelectElements("dataset"); ds != nil {
		for index := 0; index < len(ds); index++ {
			engine := ""
			if attr := ds[index].SelectAttr("engine"); attr != nil {
				engine = attr.Value
			}
			if engine == "" || engine == cengine {
				if attr := ds[index].SelectAttr("name"); attr != nil {
					if _, found := dsValues[attr.Value]; !found || engine == cengine {
						dsValues[attr.Value] = IM{"report_id": result, "dataset": attr.Value,
							"sqlstr": ds[index].Text()}
					}
				}
			}
		}
	}
	for _, values := range dsValues {
		_, err = api.NStore.ds.Update(Update{Model: "ui_reportsources", Values: values.(IM), Trans: trans})
		if err != nil {
			return result, err
		}
	}

	if rf := xreport.SelectElements("field"); rf != nil {
		for index := 0; index < len(rf); index++ {
			values := IM{"report_id": result}
			if attr := rf[index].SelectAttr("fieldname"); attr != nil {
				values["fieldname"] = attr.Value
			}
			if attr := rf[index].SelectAttr("description"); attr != nil {
				values["description"] = attr.Value
			}
			if attr := rf[index].SelectAttr("orderby"); attr != nil {
				values["orderby"] = attr.Value
			}
			if attr := rf[index].SelectAttr("dataset"); attr != nil {
				values["dataset"] = attr.Value
			}
			if attr := rf[index].SelectAttr("defvalue"); attr != nil {
				values["defvalue"] = attr.Value
			}
			if attr := rf[index].SelectAttr("valuelist"); attr != nil {
				values["valuelist"] = attr.Value
			}
			if attr := rf[index].SelectAttr("fieldtype"); attr != nil {
				values["fieldtype"] = groups["fieldtype"].(IM)[attr.Value]
			}
			if attr := rf[index].SelectAttr("wheretype"); attr != nil {
				values["wheretype"] = groups["wheretype"].(IM)[attr.Value]
			}
			values["sqlstr"] = rf[index].Text()
			_, err = api.NStore.ds.Update(Update{Model: "ui_reportfields", Values: values, Trans: trans})
			if err != nil {
				return result, err
			}
		}
	}

	if ms := xreport.SelectElements("message"); ms != nil {
		for index := 0; index < len(ms); index++ {
			values := IM{}
			if attr := ms[index].SelectAttr("secname"); attr != nil {
				values["secname"] = report["reportkey"].(string) + "_" + attr.Value
			}
			if attr := ms[index].SelectAttr("fieldname"); attr != nil {
				values["fieldname"] = attr.Value
			}
			values["msg"] = ms[index].Text()
			_, err = api.NStore.ds.Update(Update{Model: "ui_message", Values: values, Trans: trans})
			if err != nil {
				return result, err
			}
		}
	}

	return result, err
}
