package test

import (
	"testing"

	driver "github.com/nervatura/nervatura-go/pkg/driver"
	nt "github.com/nervatura/nervatura-go/pkg/nervatura"
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
		{"tool", "ABC-123"},
		{"address", "customer/DMCUST/00001~1"},
		{"contact", "customer/DMCUST/00001~1"},
		{"barcode", "BC0123456789"},
		{"customer", "DMCUST/00001"},
		{"event", "DMEVT/00001"},
		{"groups", "barcodetype~AZTEC"},
		{"fieldvalue", "DMCUST/00001~~sample_customer_date~1"},
		{"setting", "default_unit"},
		{"item", "DMINV/00001~1"},
		{"payment", "DMPMT/00001~1"},
		{"movement", "DMCORR/00001~1"},
		{"price", "DMPROD/00001~price~2019-04-05~EUR~0"},
		{"product", "DMPROD/00001"},
		{"place", "bank"},
		{"tax", "15%"},
		{"trans", "DMINV/00001"},
		{"link", "movement~DMDEL/00001~2~~item~DMORD/00001~2"},
		{"ui_menufields", "mnu_exp_1~number_1"},
		{"ui_reportfields", "ntr_custpos_en~transdate_from"},
		{"ui_reportsources", "ntr_invoice_en~head"},
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
		{"address", 6, "city"},
		{"contact", 6, "surname"},
		{"fieldvalue", 32, "value"},
		{"setting", 6, "value"},
		{"groups", 6, ""},
		{"item", 6, ""},
		{"payment", 4, ""},
		{"movement", 4, ""},
		{"price", 2, ""},
		{"link", 12, ""},
		//{"rate", 1, ""},
		//{"log", 1, ""},
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
