package test

import (
	"testing"

	nt "github.com/nervatura/nervatura-go"
	driver "github.com/nervatura/nervatura-go/driver"
)

func getAPI() *nt.API {
	config, _ := nt.ReadConfig(confPath)
	return &nt.API{NStore: nt.New(config, &driver.SQLDriver{})}
}

func getLogin() (string, *nt.API, error) {
	api := getAPI()
	options := nt.IM{"database": alias, "username": username, "password": password}
	token, err := api.AuthUserLogin(options)
	return token, api, err
}

func TestDatabaseCreate(t *testing.T) {
	options := nt.IM{"database": alias, "demo": "true", "report_dir": reportDir}
	_, err := getAPI().DatabaseCreate(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestApiAuthUserLogin(t *testing.T) {
	options := nt.IM{"database": alias, "username": username, "password": password}
	_, err := getAPI().AuthUserLogin(options)
	if err != nil {
		t.Fatal(err)
	}
	//print(token)
}

func TestApiAuthTokenLogin(t *testing.T) {
	token, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}
	options := nt.IM{"token": token}
	err = api.AuthTokenLogin(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestApiAuthPassword(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}
	options := nt.IM{"username": "demo", "password": "321", "confirm": "321"}
	err = api.AuthPassword(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIDelete(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}
	options := nt.IM{"nervatype": "address", "id": 2}
	err = api.APIDelete(options)
	if err != nil {
		t.Fatal(err)
	}
	options = nt.IM{"nervatype": "address", "key": "customer/DMCUST/00001~1"}
	err = api.APIDelete(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIGet(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}
	options := nt.IM{"nervatype": "customer", "metadata": true,
		"filter": "custname;==;First Customer Co.|custnumber;in;DMCUST/00001,DMCUST/00002"}
	_, err = api.APIGet(options)
	if err != nil {
		t.Fatal(err)
	}
	options = nt.IM{"nervatype": "customer", "metadata": true, "ids": "2,4"}
	_, err = api.APIGet(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIView(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}
	options := []nt.IM{
		nt.IM{
			"key":  "customers",
			"text": "select c.id, ct.groupvalue as custtype, c.custnumber, c.custname from customer c inner join groups ct on c.custtype = ct.id where c.deleted = 0 and c.custnumber <> 'HOME'",
		},
		nt.IM{
			"key":  "invoices",
			"text": "select t.id, t.transnumber, tt.groupvalue as transtype, td.groupvalue as direction, t.transdate, c.custname, t.curr, items.amount from trans t inner join groups tt on t.transtype = tt.id inner join groups td on t.direction = td.id inner join customer c on t.customer_id = c.id inner join ( select trans_id, sum(amount) amount from item where deleted = 0 group by trans_id) items on t.id = items.trans_id where t.deleted = 0 and tt.groupvalue = 'invoice'",
		},
	}
	_, err = api.APIView(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIFunction(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}
	options := nt.IM{
		"key": "nextNumber",
		"values": nt.IM{
			"numberkey": "custnumber",
			"step":      false,
		},
	}
	_, err = api.APIFunction(options)
	if err != nil {
		t.Fatal(err)
	}

	options = nt.IM{
		"key": "getPriceValue",
		"values": nt.IM{
			"curr":        "EUR",
			"product_id":  2,
			"customer_id": 2,
		},
	}
	_, err = api.APIFunction(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIReport(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}
	options := nt.IM{
		"reportkey":   "ntr_invoice_en",
		"orientation": "portrait",
		"size":        "a4",
		"nervatype":   "trans",
		"refnumber":   "DMINV/00001",
	}
	_, err = api.Report(options)
	if err != nil {
		t.Fatal(err)
	}

	options = nt.IM{
		"reportkey": "ntr_invoice_en",
		"output":    "xml",
		"nervatype": "trans",
		"refnumber": "DMINV/00002",
	}
	_, err = api.Report(options)
	if err != nil {
		t.Fatal(err)
	}

	options = nt.IM{
		"reportkey": "xls_vat_en",
		"filters": nt.IM{
			"date_from": "2014-01-01",
			"date_to":   "2022-01-01",
			"curr":      "EUR",
		},
	}
	_, err = api.Report(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIReportList(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}

	options := nt.IM{
		"report_dir": reportDir,
	}
	_, err = api.ReportList(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIReportDelete(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}

	options := nt.IM{
		"reportkey": "ntr_cash_in_en",
	}
	err = api.ReportDelete(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIReportInstall(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}

	options := nt.IM{
		"report_dir": reportDir,
		"reportkey":  "ntr_cash_in_en",
	}
	_, err = api.ReportInstall(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIPost(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}
	addressData := []nt.IM{
		nt.IM{
			"nervatype":         10,
			"ref_id":            2,
			"zipcode":           "12345",
			"city":              "BigCity",
			"notes":             "Create a new item by IDs",
			"address_metadata1": "value1",
			"address_metadata2": "value2~note2"},
		nt.IM{
			"id":      6,
			"zipcode": "54321",
			"city":    "BigCity",
			"notes":   "Update an item by IDs"},
		nt.IM{
			"keys": nt.IM{
				"nervatype": "customer",
				"ref_id":    "customer/DMCUST/00001"},
			"zipcode":           "12345",
			"city":              "BigCity",
			"notes":             "Create a new item by Keys",
			"address_metadata1": "value1",
			"address_metadata2": "value2~note2"},
		nt.IM{
			"keys": nt.IM{
				"id": "customer/DMCUST/00001~1"},
			"zipcode": "54321",
			"city":    "BigCity",
			"notes":   "Update an item by Keys"}}
	transData := []nt.IM{
		nt.IM{
			"transtype":   57,
			"direction":   70,
			"crdate":      "2019-09-01",
			"transdate":   "2019-09-02",
			"duedate":     "2019-09-08T00:00:00",
			"customer_id": 2,
			"department":  149,
			"paidtype":    135,
			"curr":        "EUR",
			"notes":       "Create a new item by IDs",
			"fnote":       "A long and <b><i>rich text</b></i> at the bottom of the invoice...<br><br>Can be multiple lines ...",
			"transtate":   105,
			"keys": nt.IM{
				"transnumber": nt.IL{"numberdef", "invoice_out"}}},
		nt.IM{
			"crdate":    "2019-09-03",
			"transdate": "2019-09-04",
			"duedate":   "2019-09-08T00:00:00",
			"curr":      "EUR",
			"notes":     "Create a new item by Keys",
			"keys": nt.IM{
				"transnumber": nt.IL{"numberdef", "invoice_out"},
				"transtype":   "invoice",
				"direction":   "out",
				"customer_id": "DMCUST/00001",
				"department":  "sales",
				"paidtype":    "transfer",
				"transtate":   "ok"}},
		nt.IM{
			"paid":   1,
			"closed": 1,
			"notes":  "Update an item by Keys",
			"keys": nt.IM{
				"id": "DMINV/00003"}}}
	_, err = api.APIPost("address", addressData)
	if err != nil {
		t.Fatal(err)
	}
	_, err = api.APIPost("trans", transData)
	if err != nil {
		t.Fatal(err)
	}
}
