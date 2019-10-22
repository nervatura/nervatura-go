package test

import (
	"testing"

	nt "github.com/nervatura/nervatura-go"
	driver "github.com/nervatura/nervatura-go/driver"
)

func getNpi() *nt.Npi {
	config, _ := nt.ReadConfig(confPath)
	return &nt.Npi{NStore: nt.New(config, &driver.SQLDriver{})}
}

func TestNpiGetLogin(t *testing.T) {

	options := nt.IM{"database": alias, "username": username, "password": password}
	_, err := getNpi().GetLogin(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNpiSetData(t *testing.T) {

	npi := getNpi()
	options := nt.IM{"database": alias, "username": username, "password": password}
	_, err := npi.GetLogin(options)
	if err != nil {
		t.Fatal(err)
	}

	options = nt.IM{
		"method": "update",
		"record": nt.IM{"__tablename__": "currency", "curr": "hel", "description": "hello"}}
	_, err = npi.SetData(options)
	if err != nil {
		t.Fatal(err)
	}

	options = nt.IM{
		"method": "delete",
		"record": nt.IM{"__tablename__": "currency", "refnumber": "hel"}}
	_, err = npi.SetData(options)
	if err != nil {
		t.Fatal(err)
	}
	options = nt.IM{
		"method":    "table",
		"tableName": "currency",
		"filterStr": "curr='EUR'"}
	_, err = npi.SetData(options)
	if err != nil {
		t.Fatal(err)
	}

	options = nt.IM{
		"method":   "view",
		"sqlStr":   "select * from customer where deleted=0 @where_str order by @orderby_str",
		"whereStr": "and creditlimit>=@limit and custnumber like @custnumber",
		"orderStr": "custtype,id",
		"paramList": []interface{}{
			nt.IM{"name": "@limit", "value": 0, "wheretype": "where", "type": "number"},
			nt.IM{"name": "@custnumber", "value": "HOME", "wheretype": "where", "type": "string"},
		}}
	_, err = npi.SetData(options)
	if err != nil {
		t.Fatal(err)
	}

	options = nt.IM{
		"method":   "execute",
		"sqlStr":   "update customer set @where_str",
		"whereStr": "account=@account where custnumber=@custnumber",
		"paramList": []interface{}{
			nt.IM{"name": "@account", "value": "12345678", "wheretype": "where", "type": "string"},
			nt.IM{"name": "@custnumber", "value": "HOME", "wheretype": "where", "type": "string"},
		}}
	_, err = npi.SetData(options)
	if err != nil {
		t.Fatal(err)
	}

	options = nt.IM{
		"method":       "function",
		"functionName": "nextNumber",
		"paramList": nt.IM{
			"numberkey": "custnumber",
			"step":      false,
		}}
	_, err = npi.SetData(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNpiLoadDataset(t *testing.T) {

	npi := getNpi()
	options := nt.IM{"database": alias, "username": username, "password": password}
	_, err := npi.GetLogin(options)
	if err != nil {
		t.Fatal(err)
	}

	options = nt.IM{
		"dataSetInfo": []interface{}{
			nt.IM{"infoName": "customer", "infoType": "table", "classAlias": "customer", "filterStr": ""},
		},
	}
	_, err = npi.LoadDataset(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNpiUpdateRecordset(t *testing.T) {

	npi := getNpi()
	options := nt.IM{"database": alias, "username": username, "password": password}
	_, err := npi.GetLogin(options)
	if err != nil {
		t.Fatal(err)
	}

	options = nt.IM{
		"method": "update",
		"recordSet": []interface{}{
			nt.IM{"__tablename__": "customer", "id": 1, "account": "12345678"},
		},
	}
	_, err = npi.UpdateRecordset(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSaveDataset(t *testing.T) {

	npi := getNpi()
	options := nt.IM{"database": alias, "username": username, "password": password}
	_, err := npi.GetLogin(options)
	if err != nil {
		t.Fatal(err)
	}

	options = nt.IM{
		"dataSetInfo": []interface{}{
			nt.IM{"updateType": "update",
				"recordSet": []interface{}{
					nt.IM{"__tablename__": "customer", "id": 1, "account": "87654321"},
				},
			},
		},
	}
	_, err = npi.SaveDataset(options)
	if err != nil {
		t.Fatal(err)
	}
}
