package test

import (
	"testing"

	db "github.com/nervatura/nervatura-go/pkg/database"
	nt "github.com/nervatura/nervatura-go/pkg/nervatura"
)

func getAPI() *nt.API {
	return &nt.API{NStore: nt.New(&db.SQLDriver{})}
}

func getLogin() (string, *nt.API, error) {
	api := getAPI()
	options := nt.IM{"database": alias, "username": username, "password": password}
	token, _, err := api.UserLogin(options)
	return token, api, err
}

func TestDatabaseCreate(t *testing.T) {
	options := nt.IM{"database": alias, "demo": true}
	_, err := getAPI().DatabaseCreate(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestApiUserLogin(t *testing.T) {
	options := nt.IM{"database": alias, "username": username, "password": password}
	_, _, err := getAPI().UserLogin(options)
	if err != nil {
		t.Fatal(err)
	}
	//print(token)
}

func TestApiTokenLogin(t *testing.T) {
	token, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}
	options := nt.IM{"token": token}
	err = api.TokenLogin(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestApiUserPassword(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}
	options := nt.IM{"username": "demo", "password": "321", "confirm": "321"}
	err = api.UserPassword(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDelete(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}
	options := nt.IM{"nervatype": "address", "id": 2}
	err = api.Delete(options)
	if err != nil {
		t.Fatal(err)
	}
	options = nt.IM{"nervatype": "address", "key": "customer/DMCUST/00001~1"}
	err = api.Delete(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGet(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}
	options := nt.IM{"nervatype": "customer", "metadata": true,
		"filter": "custname;==;First Customer Co.|custnumber;in;DMCUST/00001,DMCUST/00002"}
	_, err = api.Get(options)
	if err != nil {
		t.Fatal(err)
	}
	options = nt.IM{"nervatype": "customer", "metadata": true, "ids": "2,4"}
	_, err = api.Get(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestView(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}
	options := []nt.IM{
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
	_, err = api.View(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFunction(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}

	/*
		options := map[string]interface{}{
			"key": "sendEmail",
			"values": map[string]interface{}{
				"provider": "smtp",
				"email": map[string]interface{}{
					"from": "info@nervatura.com", "name": "Nervatura",
					"recipients": []interface{}{
						map[string]interface{}{"email": "sample@company.com"}},
					"subject": "Demo Invoice",
					"text":    "Email sending with attached invoice",
					"attachments": []interface{}{
						map[string]interface{}{
							"reportkey": "ntr_invoice_en",
							"nervatype": "trans",
							"refnumber": "DMINV/00001"}},
				},
			},
		}
	*/

	options := nt.IM{
		"key": "nextNumber",
		"values": nt.IM{
			"numberkey": "custnumber",
			"step":      false,
		},
	}
	_, err = api.Function(options)
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
	_, err = api.Function(options)
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
		"reportkey": "csv_vat_en",
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
		"report_dir": "",
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
		"reportkey": "ntr_cash_in_en",
	}
	_, err = api.ReportInstall(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdate(t *testing.T) {
	_, api, err := getLogin()
	if err != nil {
		t.Fatal(err)
	}
	addressData := []nt.IM{
		{
			"nervatype":         int64(10),
			"ref_id":            int64(2),
			"zipcode":           "12345",
			"city":              "BigCity",
			"notes":             "Create a new item by IDs",
			"address_metadata1": "value1",
			"address_metadata2": "value2~note2"},
		{
			"id":      int64(6),
			"zipcode": "54321",
			"city":    "BigCity",
			"notes":   "Update an item by IDs"},
		{
			"keys": nt.IM{
				"nervatype": "customer",
				"ref_id":    "customer/DMCUST/00001"},
			"zipcode":           "12345",
			"city":              "BigCity",
			"notes":             "Create a new item by Keys",
			"address_metadata1": "value1",
			"address_metadata2": "value2~note2"},
		{
			"keys": nt.IM{
				"id": "customer/DMCUST/00001~1"},
			"zipcode": "54321",
			"city":    "BigCity",
			"notes":   "Update an item by Keys"}}
	transData := []nt.IM{
		{
			"transtype":   int64(57),
			"direction":   int64(70),
			"crdate":      "2019-09-01",
			"transdate":   "2019-09-02",
			"duedate":     "2019-09-08T00:00:00",
			"customer_id": int64(2),
			"department":  int64(149),
			"paidtype":    int64(135),
			"curr":        "EUR",
			"notes":       "Create a new item by IDs",
			"fnote":       "A long and <b><i>rich text</b></i> at the bottom of the invoice...<br><br>Can be multiple lines ...",
			"transtate":   int64(105),
			"keys": nt.IM{
				"transnumber": nt.IL{"numberdef", "invoice_out"}}},
		{
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
		{
			"paid":   1,
			"closed": 1,
			"notes":  "Update an item by Keys",
			"keys": nt.IM{
				"id": "DMINV/00003"}},
	}

	_, err = api.Update("address", addressData)
	if err != nil {
		t.Fatal(err)
	}
	_, err = api.Update("trans", transData)
	if err != nil {
		t.Fatal(err)
	}

}
