package test

import (
	"testing"

	nt "github.com/nervatura/nervatura-go"
	driver "github.com/nervatura/nervatura-go/driver"
)

func getNstore() *nt.NervaStore {
	config, _ := nt.ReadConfig(confPath)
	return nt.New(config, &driver.SQLDriver{})
}

func TestUpdateData(t *testing.T) {

	nstore := getNstore()
	options := nt.IM{"database": alias, "username": username, "password": password}
	_, _, err := (&nt.API{NStore: nstore}).AuthUserLogin(options)
	if err != nil {
		t.Fatal(err)
	}

	options = nt.IM{
		"nervatype":  "currency",
		"insert_row": true,
		"values":     nt.IM{"curr": "hel", "description": "hello"}}
	_, err = nstore.UpdateData(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteData(t *testing.T) {

	nstore := getNstore()
	options := nt.IM{"database": alias, "username": username, "password": password}
	_, _, err := (&nt.API{NStore: nstore}).AuthUserLogin(options)
	if err != nil {
		t.Fatal(err)
	}

	options = nt.IM{
		"nervatype": "currency",
		"refnumber": "hel"}
	err = nstore.DeleteData(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetInfofromRefnumber(t *testing.T) {
	nstore := getNstore()

	options := nt.IM{"database": alias, "username": username, "password": password}
	_, _, err := (&nt.API{NStore: nstore}).AuthUserLogin(options)
	if err != nil {
		t.Fatal(err)
	}

	params := [][]string{
		[]string{"tool", "ABC-123"},
		[]string{"address", "customer/DMCUST/00001~1"},
		[]string{"contact", "customer/DMCUST/00001~1"},
		[]string{"barcode", "BC0123456789"},
		[]string{"customer", "DMCUST/00001"},
		[]string{"event", "DMEVT/00001"},
		[]string{"groups", "barcodetype~AZTEC"},
		[]string{"fieldvalue", "DMCUST/00001~~sample_customer_date~1"},
		[]string{"setting", "default_unit"},
		[]string{"item", "DMINV/00001~1"},
		[]string{"payment", "DMPMT/00001~1"},
		[]string{"movement", "DMCORR/00001~1"},
		[]string{"price", "DMPROD/00001~price~2019-04-05~EUR~0"},
		[]string{"product", "DMPROD/00001"},
		[]string{"place", "bank"},
		[]string{"tax", "15%"},
		[]string{"trans", "DMINV/00001"},
		[]string{"link", "movement~DMDEL/00001~2~~item~DMORD/00001~2"},
		[]string{"ui_menufields", "mnu_exp_1~number_1"},
		[]string{"ui_reportfields", "ntr_custpos_en~transdate_from"},
		[]string{"ui_reportsources", "ntr_invoice_en~head"},
	}
	for index := 0; index < len(params); index++ {
		options := nt.IM{
			"nervatype": params[index][0], "refnumber": params[index][1],
			"use_deleted": false, "extra_info": true}
		_, err := nstore.GetInfofromRefnumber(options)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetRefnumber(t *testing.T) {
	nstore := getNstore()

	options := nt.IM{"database": alias, "username": username, "password": password}
	_, _, err := (&nt.API{NStore: nstore}).AuthUserLogin(options)
	if err != nil {
		t.Fatal(err)
	}

	params := [][]interface{}{
		[]interface{}{"address", 6, "city"},
		[]interface{}{"contact", 6, "surname"},
		[]interface{}{"fieldvalue", 32, "value"},
		[]interface{}{"setting", 6, "value"},
		[]interface{}{"groups", 6, ""},
		[]interface{}{"item", 6, ""},
		[]interface{}{"payment", 4, ""},
		[]interface{}{"movement", 4, ""},
		[]interface{}{"price", 2, ""},
		[]interface{}{"link", 12, ""},
		//[]interface{}{"rate", 1, ""},
		//[]interface{}{"log", 1, ""},
	}
	for index := 0; index < len(params); index++ {
		options := nt.IM{
			"nervatype": params[index][0], "ref_id": params[index][1], "retfield": params[index][2],
			"use_deleted": false}
		_, err := nstore.GetRefnumber(options)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetDataAudit(t *testing.T) {
	nstore := getNstore()

	options := nt.IM{"database": alias, "username": username, "password": password}
	_, _, err := (&nt.API{NStore: nstore}).AuthUserLogin(options)
	if err != nil {
		t.Fatal(err)
	}

	_, err = nstore.GetDataAudit()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetObjectAudit(t *testing.T) {
	nstore := getNstore()

	options := nt.IM{"database": alias, "username": username, "password": password}
	_, _, err := (&nt.API{NStore: nstore}).AuthUserLogin(options)
	if err != nil {
		t.Fatal(err)
	}

	options = nt.IM{
		"nervatype": []string{"customer", "product"},
	}
	_, err = nstore.GetObjectAudit(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetGroups(t *testing.T) {
	nstore := getNstore()

	options := nt.IM{"database": alias, "username": username, "password": password}
	_, _, err := (&nt.API{NStore: nstore}).AuthUserLogin(options)
	if err != nil {
		t.Fatal(err)
	}

	options = nt.IM{
		//"groupname": "transtype",
		"groupname": []string{"transtype", "usergroup"},
	}
	_, err = nstore.GetGroups(options)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetDatabaseSettings(t *testing.T) {
	nstore := getNstore()

	options := nt.IM{"database": alias, "username": username, "password": password}
	_, _, err := (&nt.API{NStore: nstore}).AuthUserLogin(options)
	if err != nil {
		t.Fatal(err)
	}

	_, err = nstore.GetDatabaseSettings()
	if err != nil {
		t.Fatal(err)
	}
}
