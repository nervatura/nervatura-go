package test

import (
	"os"
	"testing"

	"github.com/nervatura/nervatura-go/app"
	ut "github.com/nervatura/nervatura-go/pkg/utils"
)

func encodeOptions(data interface{}) string {
	jdata, _ := ut.ConvertToByte(data)
	return string(jdata)
}

func checkResult(result string, code float64) bool {
	var data interface{}
	err := ut.ConvertFromByte([]byte(result), &data)
	if err != nil {
		return false
	}

	if jdata, valid := data.(map[string]interface{}); valid {
		if _, found := jdata["code"]; found {
			if jdata["code"].(float64) != code {
				return false
			}
		}
	}
	return true
}

func getToken() string {
	api := getAPI()
	options := map[string]interface{}{"database": "demo", "username": "admin"}
	token, _, _ := api.UserLogin(options)
	return token
}

func TestCliDatabaseCreate(t *testing.T) {
	options := map[string]interface{}{"database": "demo", "demo": true}
	os.Args = append(os.Args, "-c", "DatabaseCreate")
	os.Args = append(os.Args, "-o", encodeOptions(options))
	os.Args = append(os.Args, "-k", "TEST_API_KEY")
	app, err := app.New("test")
	if err != nil {
		t.Fatal(err)
	}
	if !checkResult(app.GetResults(), 0) {
		t.Fail()
	}
}

func TestCliUserLogin(t *testing.T) {
	options := map[string]interface{}{"database": "demo", "username": "demo"}
	os.Args = append(os.Args, "-c", "UserLogin")
	os.Args = append(os.Args, "-o", encodeOptions(options))

	app, err := app.New("test")
	if err != nil {
		t.Fatal(err)
	}
	if !checkResult(app.GetResults(), 0) {
		t.Fail()
	}
}

func TestCliTokenLogin(t *testing.T) {
	token := getToken()

	os.Args = append(os.Args, "-c", "TokenLogin")
	os.Args = append(os.Args, "-t", token)
	app, err := app.New("test")
	if err != nil {
		t.Fatal(err)
	}
	if !checkResult(app.GetResults(), 0) {
		t.Fail()
	}

}

func TestCliTokenRefresh(t *testing.T) {
	token := getToken()

	os.Args = append(os.Args, "-c", "TokenRefresh")
	os.Args = append(os.Args, "-t", token)
	app, err := app.New("test")
	if err != nil {
		t.Fatal(err)
	}
	if !checkResult(app.GetResults(), 0) {
		t.Fail()
	}

}

func TestCliTokenDecode(t *testing.T) {
	token := getToken()

	os.Args = append(os.Args, "-c", "TokenDecode")
	os.Args = append(os.Args, "-t", token)
	app, err := app.New("test")
	if err != nil {
		t.Fatal(err)
	}
	if !checkResult(app.GetResults(), 0) {
		t.Fail()
	}

}

func TestCliUserPassword(t *testing.T) {
	api := getAPI()
	options := map[string]interface{}{"database": "demo", "username": "admin"}
	token, _, _ := api.UserLogin(options)

	options = map[string]interface{}{"username": "guest", "password": "321", "confirm": "321"}
	os.Args = append(os.Args, "-c", "UserPassword")
	os.Args = append(os.Args, "-o", encodeOptions(options))
	os.Args = append(os.Args, "-t", token)
	app, err := app.New("test")
	if err != nil {
		t.Fatal(err)
	}
	if !checkResult(app.GetResults(), 204) {
		t.Fail()
	}
}

func TestCliDelete(t *testing.T) {
	token := getToken()
	options := map[string]interface{}{"nervatype": "address", "id": 2}
	os.Args = append(os.Args, "-c", "Delete")
	os.Args = append(os.Args, "-o", encodeOptions(options))
	os.Args = append(os.Args, "-t", token)
	app, err := app.New("test")
	if err != nil {
		t.Fatal(err)
	}
	if !checkResult(app.GetResults(), 204) {
		t.Fail()
	}
}

func TestCliGet(t *testing.T) {
	token := getToken()
	options := map[string]interface{}{"nervatype": "customer", "metadata": true,
		"filter": "custname;==;First Customer Co.|custnumber;in;DMCUST/00001,DMCUST/00002"}
	os.Args = append(os.Args, "-c", "Get")
	os.Args = append(os.Args, "-o", encodeOptions(options))
	os.Args = append(os.Args, "-t", token)
	app, err := app.New("test")
	if err != nil {
		t.Fatal(err)
	}
	if !checkResult(app.GetResults(), 0) {
		t.Fail()
	}
}

func TestCliView(t *testing.T) {
	token := getToken()
	options := []map[string]interface{}{
		{
			"key":    "customers",
			"text":   "select c.id, ct.groupvalue as custtype, c.custnumber, c.custname from customer c inner join groups ct on c.custtype = ct.id where c.deleted = 0 and c.custnumber <> 'HOME'",
			"values": []interface{}{},
		},
		{
			"key":    "invoices",
			"text":   "select t.id, t.transnumber, tt.groupvalue as transtype, td.groupvalue as direction, t.transdate, c.custname, t.curr, items.amount from trans t inner join groups tt on t.transtype = tt.id inner join groups td on t.direction = td.id inner join customer c on t.customer_id = c.id inner join ( select trans_id, sum(amount) amount from item where deleted = 0 group by trans_id) items on t.id = items.trans_id where t.deleted = 0 and tt.groupvalue = 'invoice'",
			"values": []interface{}{},
		},
	}
	os.Args = append(os.Args, "-c", "View")
	os.Args = append(os.Args, "-d", encodeOptions(options))
	os.Args = append(os.Args, "-t", token)
	app, err := app.New("test")
	if err != nil {
		t.Fatal(err)
	}
	if !checkResult(app.GetResults(), 0) {
		t.Fail()
	}
}

func TestCliFunction(t *testing.T) {
	token := getToken()
	options := map[string]interface{}{"key": "nextNumber",
		"values": map[string]interface{}{
			"numberkey": "custnumber",
			"step":      false,
		}}
	os.Args = append(os.Args, "-c", "Function")
	os.Args = append(os.Args, "-o", encodeOptions(options))
	os.Args = append(os.Args, "-t", token)
	app, err := app.New("test")
	if err != nil {
		t.Fatal(err)
	}
	if !checkResult(app.GetResults(), 0) {
		t.Fail()
	}
}

func TestCliUpdate(t *testing.T) {
	token := getToken()
	options := []map[string]interface{}{{
		"nervatype":         10,
		"ref_id":            2,
		"zipcode":           "12345",
		"city":              "BigCity",
		"notes":             "Create a new item by IDs",
		"address_metadata1": "value1",
		"address_metadata2": "value2~note2"},
		{
			"id":      6,
			"zipcode": "54321",
			"city":    "BigCity",
			"notes":   "Update an item by IDs"},
		{
			"keys": map[string]interface{}{
				"nervatype": "customer",
				"ref_id":    "customer/DMCUST/00001"},
			"zipcode":           "12345",
			"city":              "BigCity",
			"notes":             "Create a new item by Keys",
			"address_metadata1": "value1",
			"address_metadata2": "value2~note2"},
		{
			"keys": map[string]interface{}{
				"id": "customer/DMCUST/00001~1"},
			"zipcode": "54321",
			"city":    "BigCity",
			"notes":   "Update an item by Keys"}}
	os.Args = append(os.Args, "-c", "Update")
	os.Args = append(os.Args, "-nt", "address")
	os.Args = append(os.Args, "-d", encodeOptions(options))
	os.Args = append(os.Args, "-t", token)
	app, err := app.New("test")
	if err != nil {
		t.Fatal(err)
	}
	if !checkResult(app.GetResults(), 0) {
		t.Fail()
	}
}

func TestCliReport(t *testing.T) {
	token := getToken()
	/*
		options := map[string]interface{}{
			"reportkey":   "ntr_invoice_en",
			"orientation": "portrait",
			"size":        "a4",
			"nervatype":   "trans",
			"refnumber":   "DMINV/00001"}
	*/
	options := map[string]interface{}{
		"filters": map[string]interface{}{
			"posdate": "2019-02-23"},
		"orientation": "portrait",
		"output":      "auto",
		"reportkey":   "csv_custpos_en",
		"size":        "a4"}
	os.Args = append(os.Args, "-c", "Report")
	os.Args = append(os.Args, "-o", encodeOptions(options))
	os.Args = append(os.Args, "-t", token)
	app, err := app.New("test")
	if err != nil {
		t.Fatal(err)
	}
	if !checkResult(app.GetResults(), 0) {
		t.Fail()
	}
}

func TestCliReportList(t *testing.T) {
	token := getToken()
	options := map[string]interface{}{
		"report_dir": ""}
	os.Args = append(os.Args, "-c", "ReportList")
	os.Args = append(os.Args, "-o", encodeOptions(options))
	os.Args = append(os.Args, "-t", token)
	app, err := app.New("test")
	if err != nil {
		t.Fatal(err)
	}
	if !checkResult(app.GetResults(), 0) {
		t.Fail()
	}
}

func TestCliReportDelete(t *testing.T) {
	token := getToken()
	options := map[string]interface{}{
		"reportkey": "ntr_cash_in_en"}
	os.Args = append(os.Args, "-c", "ReportDelete")
	os.Args = append(os.Args, "-o", encodeOptions(options))
	os.Args = append(os.Args, "-t", token)
	app, err := app.New("test")
	if err != nil {
		t.Fatal(err)
	}
	if !checkResult(app.GetResults(), 204) {
		t.Fail()
	}
}

func TestCliReportInstall(t *testing.T) {
	token := getToken()
	options := map[string]interface{}{
		"reportkey": "ntr_cash_in_en"}
	os.Args = append(os.Args, "-c", "ReportInstall")
	os.Args = append(os.Args, "-o", encodeOptions(options))
	os.Args = append(os.Args, "-t", token)
	app, err := app.New("test")
	if err != nil {
		t.Fatal(err)
	}
	if !checkResult(app.GetResults(), 0) {
		t.Fail()
	}
}
