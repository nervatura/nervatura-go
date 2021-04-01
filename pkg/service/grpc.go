//+build grpc all

package service

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"

	ntura "github.com/nervatura/nervatura-go/pkg/nervatura"
	pb "github.com/nervatura/nervatura-go/pkg/proto"
)

// RPCService implements the Nervatura API service
type RPCService struct {
	GetNervaStore func(database string) *ntura.NervaStore
	pb.UnimplementedAPIServer
}

func (srv *RPCService) itemMap(key string, data ntura.IM) *pb.ResponseGet_Value {
	metaMap := func(data interface{}) []*pb.MetaData {
		metadata := []*pb.MetaData{}
		if mdata, valid := data.([]ntura.IM); valid {
			for i := 0; i < len(mdata); i++ {
				metadata = append(metadata, &pb.MetaData{
					Id:        ntura.ToInteger(mdata[i]["id"]),
					Fieldname: ntura.ToString(mdata[i]["fieldname"], ""),
					Fieldtype: ntura.ToString(mdata[i]["fieldtype"], ""),
					Value:     ntura.ToString(mdata[i]["value"], ""),
					Notes:     ntura.ToString(mdata[i]["notes"], ""),
				})
			}
		}
		return metadata
	}

	itemMap := map[string]func(data ntura.IM) *pb.ResponseGet_Value{
		"address": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Address{Address: &pb.Address{
					Id:        ntura.ToInteger(data["id"]),
					Nervatype: ntura.ToInteger(data["nervatype"]),
					RefId:     ntura.ToInteger(data["ref_id"]),
					Country:   ntura.ToString(data["country"], ""),
					State:     ntura.ToString(data["state"], ""),
					Zipcode:   ntura.ToString(data["zipcode"], ""),
					City:      ntura.ToString(data["city"], ""),
					Street:    ntura.ToString(data["street"], ""),
					Notes:     ntura.ToString(data["notes"], ""),
					Metadata:  metaMap(data["metadata"]),
				}}}
		},
		"barcode": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Barcode{Barcode: &pb.Barcode{
					Id:          ntura.ToInteger(data["id"]),
					Code:        ntura.ToString(data["code"], ""),
					ProductId:   ntura.ToInteger(data["product_id"]),
					Description: ntura.ToString(data["description"], ""),
					Barcodetype: ntura.ToInteger(data["barcodetype"]),
					Qty:         ntura.ToFloat(data["qty"]),
					Defcode:     ntura.ToBoolean(data["defcode"], false),
				}}}
		},
		"contact": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Contact{Contact: &pb.Contact{
					Id:        ntura.ToInteger(data["id"]),
					Nervatype: ntura.ToInteger(data["nervatype"]),
					RefId:     ntura.ToInteger(data["ref_id"]),
					Firstname: ntura.ToString(data["firstname"], ""),
					Surname:   ntura.ToString(data["surname"], ""),
					Status:    ntura.ToString(data["status"], ""),
					Phone:     ntura.ToString(data["phone"], ""),
					Fax:       ntura.ToString(data["fax"], ""),
					Mobil:     ntura.ToString(data["mobil"], ""),
					Email:     ntura.ToString(data["email"], ""),
					Notes:     ntura.ToString(data["notes"], ""),
					Metadata:  metaMap(data["metadata"]),
				}}}
		},
		"currency": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Currency{Currency: &pb.Currency{
					Id:          ntura.ToInteger(data["id"]),
					Curr:        ntura.ToString(data["curr"], ""),
					Description: ntura.ToString(data["description"], ""),
					Digit:       ntura.ToInteger(data["digit"]),
					Defrate:     ntura.ToFloat(data["defrate"]),
					Cround:      ntura.ToInteger(data["cround"]),
					Metadata:    metaMap(data["metadata"]),
				}}}
		},
		"customer": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Customer{Customer: &pb.Customer{
					Id:          ntura.ToInteger(data["id"]),
					Custtype:    ntura.ToInteger(data["custtype"]),
					Custnumber:  ntura.ToString(data["custnumber"], ""),
					Custname:    ntura.ToString(data["custname"], ""),
					Taxnumber:   ntura.ToString(data["taxnumber"], ""),
					Account:     ntura.ToString(data["account"], ""),
					Notax:       ntura.ToBoolean(data["notax"], false),
					Terms:       ntura.ToInteger(data["terms"]),
					Creditlimit: ntura.ToFloat(data["creditlimit"]),
					Discount:    ntura.ToFloat(data["discount"]),
					Notes:       ntura.ToString(data["notes"], ""),
					Inactive:    ntura.ToBoolean(data["inactive"], false),
					Metadata:    metaMap(data["metadata"]),
				}}}
		},
		"deffield": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Deffield{Deffield: &pb.Deffield{
					Id:          ntura.ToInteger(data["id"]),
					Fieldname:   ntura.ToString(data["fieldname"], ""),
					Nervatype:   ntura.ToInteger(data["nervatype"]),
					Subtype:     ntura.ToInteger(data["subtype"]),
					Fieldtype:   ntura.ToInteger(data["fieldtype"]),
					Description: ntura.ToString(data["description"], ""),
					Valuelist:   ntura.ToString(data["valuelist"], ""),
					Addnew:      ntura.ToBoolean(data["addnew"], false),
					Visible:     ntura.ToBoolean(data["visible"], false),
					Readonly:    ntura.ToBoolean(data["readonly"], false),
				}}}
		},
		"employee": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Employee{Employee: &pb.Employee{
					Id:              ntura.ToInteger(data["id"]),
					Empnumber:       ntura.ToString(data["empnumber"], ""),
					Username:        ntura.ToString(data["username"], ""),
					Usergroup:       ntura.ToInteger(data["Usergroup"]),
					Startdate:       ntura.ToString(data["startdate"], ""),
					Enddate:         ntura.ToString(data["enddate"], ""),
					Department:      ntura.ToInteger(data["department"]),
					RegistrationKey: ntura.ToString(data["registration_key"], ""),
					Inactive:        ntura.ToBoolean(data["inactive"], false),
					Metadata:        metaMap(data["metadata"]),
				}}}
		},
		"event": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Event{Event: &pb.Event{
					Id:          ntura.ToInteger(data["id"]),
					Calnumber:   ntura.ToString(data["calnumber"], ""),
					Nervatype:   ntura.ToInteger(data["nervatype"]),
					RefId:       ntura.ToInteger(data["ref_id"]),
					Uid:         ntura.ToString(data["uid"], ""),
					Eventgroup:  ntura.ToInteger(data["eventgroup"]),
					Fromdate:    ntura.ToString(data["fromdate"], ""),
					Todate:      ntura.ToString(data["todate"], ""),
					Subject:     ntura.ToString(data["subject"], ""),
					Place:       ntura.ToString(data["place"], ""),
					Description: ntura.ToString(data["description"], ""),
					Metadata:    metaMap(data["metadata"]),
				}}}
		},
		"fieldvalue": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Fieldvalue{Fieldvalue: &pb.Fieldvalue{
					Id:        ntura.ToInteger(data["id"]),
					RefId:     ntura.ToInteger(data["ref_id"]),
					Fieldname: ntura.ToString(data["fieldname"], ""),
					Value:     ntura.ToString(data["value"], ""),
					Notes:     ntura.ToString(data["notes"], ""),
				}}}
		},
		"groups": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Groups{Groups: &pb.Groups{
					Id:          ntura.ToInteger(data["id"]),
					Groupname:   ntura.ToString(data["groupname"], ""),
					Groupvalue:  ntura.ToString(data["groupvalue"], ""),
					Description: ntura.ToString(data["description"], ""),
					Inactive:    ntura.ToBoolean(data["inactive"], false),
				}}}
		},
		"item": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Item{Item: &pb.Item{
					Id:          ntura.ToInteger(data["id"]),
					TransId:     ntura.ToInteger(data["trans_id"]),
					ProductId:   ntura.ToInteger(data["product_id"]),
					Unit:        ntura.ToString(data["unit"], ""),
					Qty:         ntura.ToFloat(data["qty"]),
					Fxprice:     ntura.ToFloat(data["fxprice"]),
					Netamount:   ntura.ToFloat(data["netamount"]),
					Discount:    ntura.ToFloat(data["discount"]),
					TaxId:       ntura.ToInteger(data["tax_id"]),
					Vatamount:   ntura.ToFloat(data["vatamount"]),
					Amount:      ntura.ToFloat(data["amount"]),
					Description: ntura.ToString(data["description"], ""),
					Deposit:     ntura.ToBoolean(data["deposit"], false),
					Ownstock:    ntura.ToFloat(data["ownstock"]),
					Actionprice: ntura.ToBoolean(data["actionprice"], false),
					Metadata:    metaMap(data["metadata"]),
				}}}
		},
		"link": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Link{Link: &pb.Link{
					Id:          ntura.ToInteger(data["id"]),
					Nervatype_1: ntura.ToInteger(data["nervatype_1"]),
					RefId_1:     ntura.ToInteger(data["ref_id_1"]),
					Nervatype_2: ntura.ToInteger(data["nervatype_2"]),
					RefId_2:     ntura.ToInteger(data["ref_id_2"]),
					Metadata:    metaMap(data["metadata"]),
				}}}
		},
		"log": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Log{Log: &pb.Log{
					Id:         ntura.ToInteger(data["id"]),
					Nervatype:  ntura.ToInteger(data["nervatype"]),
					RefId:      ntura.ToInteger(data["ref_id"]),
					EmployeeId: ntura.ToInteger(data["employee_id"]),
					Crdate:     ntura.ToString(data["crdate"], ""),
					Logstate:   ntura.ToInteger(data["logstate"]),
					Metadata:   metaMap(data["metadata"]),
				}}}
		},
		"movement": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Movement{Movement: &pb.Movement{
					Id:           ntura.ToInteger(data["id"]),
					TransId:      ntura.ToInteger(data["trans_id"]),
					Shippingdate: ntura.ToString(data["shippingdate"], ""),
					Movetype:     ntura.ToInteger(data["movetype"]),
					ProductId:    ntura.ToInteger(data["product_id"]),
					ToolId:       ntura.ToInteger(data["tool_id"]),
					PlaceId:      ntura.ToInteger(data["place_id"]),
					Qty:          ntura.ToFloat(data["qty"]),
					Description:  ntura.ToString(data["description"], ""),
					Shared:       ntura.ToBoolean(data["shared"], false),
					Metadata:     metaMap(data["metadata"]),
				}}}
		},
		"numberdef": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Numberdef{Numberdef: &pb.Numberdef{
					Id:          ntura.ToInteger(data["id"]),
					Numberkey:   ntura.ToString(data["numberkey"], ""),
					Prefix:      ntura.ToString(data["prefix"], ""),
					Curvalue:    ntura.ToInteger(data["curvalue"]),
					Isyear:      ntura.ToBoolean(data["isyear"], false),
					Sep:         ntura.ToString(data["sep"], ""),
					Len:         ntura.ToInteger(data["len"]),
					Description: ntura.ToString(data["description"], ""),
					Visible:     ntura.ToBoolean(data["visible"], false),
					Readonly:    ntura.ToBoolean(data["readonly"], false),
					Orderby:     ntura.ToInteger(data["orderby"]),
				}}}
		},
		"pattern": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Pattern{Pattern: &pb.Pattern{
					Id:          ntura.ToInteger(data["id"]),
					Description: ntura.ToString(data["description"], ""),
					Transtype:   ntura.ToInteger(data["transtype"]),
					Notes:       ntura.ToString(data["notes"], ""),
					Defpattern:  ntura.ToBoolean(data["defpattern"], false),
				}}}
		},
		"payment": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Payment{Payment: &pb.Payment{
					Id:       ntura.ToInteger(data["id"]),
					TransId:  ntura.ToInteger(data["trans_id"]),
					Paiddate: ntura.ToString(data["paiddate"], ""),
					Amount:   ntura.ToFloat(data["amount"]),
					Notes:    ntura.ToString(data["notes"], ""),
					Metadata: metaMap(data["metadata"]),
				}}}
		},
		"place": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Place{Place: &pb.Place{
					Id:          ntura.ToInteger(data["id"]),
					Planumber:   ntura.ToString(data["planumber"], ""),
					Placetype:   ntura.ToInteger(data["placetype"]),
					Description: ntura.ToString(data["description"], ""),
					Curr:        ntura.ToString(data["curr"], ""),
					Defplace:    ntura.ToBoolean(data["defplace"], false),
					Notes:       ntura.ToString(data["notes"], ""),
					Inactive:    ntura.ToBoolean(data["inactive"], false),
					Metadata:    metaMap(data["metadata"]),
				}}}
		},
		"price": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Price{Price: &pb.Price{
					Id:          ntura.ToInteger(data["id"]),
					ProductId:   ntura.ToInteger(data["product_id"]),
					Validfrom:   ntura.ToString(data["validfrom"], ""),
					Validto:     ntura.ToString(data["validto"], ""),
					Curr:        ntura.ToString(data["curr"], ""),
					Qty:         ntura.ToFloat(data["qty"]),
					Pricevalue:  ntura.ToFloat(data["pricevalue"]),
					Vendorprice: ntura.ToBoolean(data["vendorprice"], false),
					Metadata:    metaMap(data["metadata"]),
				}}}
		},
		"product": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Product{Product: &pb.Product{
					Id:          ntura.ToInteger(data["id"]),
					Partnumber:  ntura.ToString(data["partnumber"], ""),
					Protype:     ntura.ToInteger(data["protype"]),
					Description: ntura.ToString(data["description"], ""),
					Unit:        ntura.ToString(data["unit"], ""),
					TaxId:       ntura.ToInteger(data["tax_id"]),
					Notes:       ntura.ToString(data["notes"], ""),
					Webitem:     ntura.ToBoolean(data["webitem"], false),
					Inactive:    ntura.ToBoolean(data["inactive"], false),
					Metadata:    metaMap(data["metadata"]),
				}}}
		},
		"project": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Project{Project: &pb.Project{
					Id:          ntura.ToInteger(data["id"]),
					Pronumber:   ntura.ToString(data["pronumber"], ""),
					Description: ntura.ToString(data["description"], ""),
					CustomerId:  ntura.ToInteger(data["customer_id"]),
					Startdate:   ntura.ToString(data["startdate"], ""),
					Enddate:     ntura.ToString(data["enddate"], ""),
					Notes:       ntura.ToString(data["notes"], ""),
					Inactive:    ntura.ToBoolean(data["inactive"], false),
					Metadata:    metaMap(data["metadata"]),
				}}}
		},
		"rate": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Rate{Rate: &pb.Rate{
					Id:        ntura.ToInteger(data["id"]),
					Ratetype:  ntura.ToInteger(data["ratetype"]),
					Ratedate:  ntura.ToString(data["ratedate"], ""),
					Curr:      ntura.ToString(data["curr"], ""),
					PlaceId:   ntura.ToInteger(data["place_id"]),
					Rategroup: ntura.ToInteger(data["rategroup"]),
					Ratevalue: ntura.ToFloat(data["ratevalue"]),
					Metadata:  metaMap(data["metadata"]),
				}}}
		},
		"tax": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Tax{Tax: &pb.Tax{
					Id:          ntura.ToInteger(data["id"]),
					Taxcode:     ntura.ToString(data["taxcode"], ""),
					Description: ntura.ToString(data["description"], ""),
					Rate:        ntura.ToFloat(data["rate"]),
					Inactive:    ntura.ToBoolean(data["inactive"], false),
					Metadata:    metaMap(data["metadata"]),
				}}}
		},
		"tool": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Tool{Tool: &pb.Tool{
					Id:          ntura.ToInteger(data["id"]),
					Serial:      ntura.ToString(data["serial"], ""),
					Description: ntura.ToString(data["description"], ""),
					ProductId:   ntura.ToInteger(data["product_id"]),
					Toolgroup:   ntura.ToInteger(data["toolgroup"]),
					Notes:       ntura.ToString(data["notes"], ""),
					Inactive:    ntura.ToBoolean(data["inactive"], false),
					Metadata:    metaMap(data["metadata"]),
				}}}
		},
		"trans": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_Trans{Trans: &pb.Trans{
					Id:             ntura.ToInteger(data["id"]),
					Transnumber:    ntura.ToString(data["transnumber"], ""),
					Transtype:      ntura.ToInteger(data["transtype"]),
					Direction:      ntura.ToInteger(data["direction"]),
					RefTransnumber: ntura.ToString(data["ref_transnumber"], ""),
					Crdate:         ntura.ToString(data["crdate"], ""),
					Transdate:      ntura.ToString(data["transdate"], ""),
					Duedate:        ntura.ToString(data["duedate"], ""),
					CustomerId:     ntura.ToInteger(data["customer_id"]),
					EmployeeId:     ntura.ToInteger(data["employee_id"]),
					Department:     ntura.ToInteger(data["department"]),
					ProjectId:      ntura.ToInteger(data["project_id"]),
					PlaceId:        ntura.ToInteger(data["place_id"]),
					Paidtype:       ntura.ToInteger(data["paidtype"]),
					Curr:           ntura.ToString(data["curr"], ""),
					Notax:          ntura.ToBoolean(data["notax"], false),
					Paid:           ntura.ToBoolean(data["paid"], false),
					Acrate:         ntura.ToFloat(data["acrate"]),
					Notes:          ntura.ToString(data["notes"], ""),
					Intnotes:       ntura.ToString(data["intnotes"], ""),
					Fnote:          ntura.ToString(data["fnote"], ""),
					Transtate:      ntura.ToInteger(data["transtate"]),
					Closed:         ntura.ToBoolean(data["closed"], false),
					Metadata:       metaMap(data["metadata"]),
				}}}
		},
		"ui_audit": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_UiAudit{UiAudit: &pb.UiAudit{
					Id:          ntura.ToInteger(data["id"]),
					Usergroup:   ntura.ToInteger(data["usergroup"]),
					Nervatype:   ntura.ToInteger(data["nervatype"]),
					Subtype:     ntura.ToInteger(data["subtype"]),
					Inputfilter: ntura.ToInteger(data["inputfilter"]),
					Supervisor:  ntura.ToBoolean(data["supervisor"], false),
				}}}
		},
		"ui_language": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_UiLanguage{UiLanguage: &pb.UiLanguage{
					Id:          ntura.ToInteger(data["id"]),
					Lang:        ntura.ToString(data["lang"], ""),
					Description: ntura.ToString(data["description"], ""),
				}}}
		},
		"ui_menu": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_UiMenu{UiMenu: &pb.UiMenu{
					Id:          ntura.ToInteger(data["id"]),
					Menukey:     ntura.ToString(data["menukey"], ""),
					Description: ntura.ToString(data["description"], ""),
					Modul:       ntura.ToString(data["modul"], ""),
					Icon:        ntura.ToString(data["icon"], ""),
					Funcname:    ntura.ToString(data["funcname"], ""),
					Url:         ntura.ToBoolean(data["url"], false),
					Address:     ntura.ToString(data["address"], ""),
				}}}
		},
		"ui_menufields": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_UiMenufields{UiMenufields: &pb.UiMenufields{
					Id:          ntura.ToInteger(data["id"]),
					MenuId:      ntura.ToInteger(data["menu_id"]),
					Fieldname:   ntura.ToString(data["fieldname"], ""),
					Description: ntura.ToString(data["description"], ""),
					Fieldtype:   ntura.ToInteger(data["fieldtype"]),
					Orderby:     ntura.ToInteger(data["orderby"]),
				}}}
		},
		"ui_message": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_UiMessage{UiMessage: &pb.UiMessage{
					Id:        ntura.ToInteger(data["id"]),
					Secname:   ntura.ToString(data["secname"], ""),
					Fieldname: ntura.ToString(data["fieldname"], ""),
					Lang:      ntura.ToString(data["lang"], ""),
					Msg:       ntura.ToString(data["msg"], ""),
				}}}
		},
		"ui_printqueue": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_UiPrintqueue{UiPrintqueue: &pb.UiPrintqueue{
					Id:         ntura.ToInteger(data["id"]),
					Nervatype:  ntura.ToInteger(data["nervatype"]),
					RefId:      ntura.ToInteger(data["ref_id"]),
					Qty:        ntura.ToFloat(data["qty"]),
					EmployeeId: ntura.ToInteger(data["employee_id"]),
					ReportId:   ntura.ToInteger(data["report_id"]),
					Crdate:     ntura.ToString(data["crdate"], ""),
				}}}
		},
		"ui_report": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_UiReport{UiReport: &pb.UiReport{
					Id:          ntura.ToInteger(data["id"]),
					Reportkey:   ntura.ToString(data["reportkey"], ""),
					Nervatype:   ntura.ToInteger(data["nervatype"]),
					Transtype:   ntura.ToInteger(data["transtype"]),
					Direction:   ntura.ToInteger(data["direction"]),
					Repname:     ntura.ToString(data["repname"], ""),
					Description: ntura.ToString(data["description"], ""),
					Label:       ntura.ToString(data["label"], ""),
					Filetype:    ntura.ToInteger(data["filetype"]),
					Report:      ntura.ToString(data["report"], ""),
				}}}
		},
		"ui_reportfields": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_UiReportfields{UiReportfields: &pb.UiReportfields{
					Id:          ntura.ToInteger(data["id"]),
					ReportId:    ntura.ToInteger(data["report_id"]),
					Fieldname:   ntura.ToString(data["fieldname"], ""),
					Fieldtype:   ntura.ToInteger(data["fieldtype"]),
					Wheretype:   ntura.ToInteger(data["wheretype"]),
					Description: ntura.ToString(data["description"], ""),
					Orderby:     ntura.ToInteger(data["orderby"]),
					Sqlstr:      ntura.ToString(data["sqlstr"], ""),
					Parameter:   ntura.ToBoolean(data["parameter"], false),
					Dataset:     ntura.ToString(data["dataset"], ""),
					Defvalue:    ntura.ToString(data["defvalue"], ""),
					Valuelist:   ntura.ToString(data["valuelist"], ""),
				}}}
		},
		"ui_reportsources": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_UiReportsources{UiReportsources: &pb.UiReportsources{
					Id:       ntura.ToInteger(data["id"]),
					ReportId: ntura.ToInteger(data["report_id"]),
					Dataset:  ntura.ToString(data["dataset"], ""),
					Sqlstr:   ntura.ToString(data["sqlstr"], ""),
				}}}
		},
		"ui_userconfig": func(data ntura.IM) *pb.ResponseGet_Value {
			return &pb.ResponseGet_Value{
				Value: &pb.ResponseGet_Value_UiUserconfig{UiUserconfig: &pb.UiUserconfig{
					Id:         ntura.ToInteger(data["id"]),
					EmployeeId: ntura.ToInteger(data["employee_id"]),
					Section:    ntura.ToString(data["section"], ""),
					Cfgroup:    ntura.ToString(data["cfgroup"], ""),
					Cfname:     ntura.ToString(data["cfname"], ""),
					Cfvalue:    ntura.ToString(data["cfvalue"], ""),
					Orderby:    ntura.ToInteger(data["orderby"]),
				}}}
		},
	}
	return itemMap[key](data)
}

func (srv *RPCService) rowMap(values interface{}) *pb.ResponseRows {
	encodeMap := func(values interface{}) *pb.ResponseRows_Item {
		row := &pb.ResponseRows_Item{
			Values: make(ntura.SM),
		}

		switch v := values.(type) {
		case ntura.SM:
			for fieldname, value := range v {
				row.Values[fieldname] = value
			}
		case ntura.IM:
			for fieldname, value := range v {
				row.Values[fieldname] = ntura.ToString(value, "")
			}
		}
		return row
	}

	rows := &pb.ResponseRows{}
	switch v := values.(type) {
	case []ntura.IM:
		for index := 0; index < len(v); index++ {
			rows.Items = append(rows.Items, encodeMap(v[index]))
		}
	case []ntura.SM:
		for index := 0; index < len(v); index++ {
			rows.Items = append(rows.Items, encodeMap(v[index]))
		}
	}
	return rows
}

func (srv *RPCService) fieldsToIMap(values map[string]*pb.RequestField) ntura.IM {
	iMap := make(ntura.IM)
	for fieldname, value := range values {
		switch v := value.Value.(type) {
		case *pb.RequestField_Boolean:
			iMap[fieldname] = v.Boolean
		case *pb.RequestField_Number:
			iMap[fieldname] = v.Number
		case *pb.RequestField_Text:
			iMap[fieldname] = v.Text
		}
	}
	return iMap
}

func (srv *RPCService) TokenAuth(authorization []string, parent context.Context) (ctx context.Context, err error) {
	if len(authorization) < 1 {
		return ctx, errors.New("Unauthorized")
	}
	tokenStr := strings.TrimPrefix(authorization[0], "Bearer ")
	if tokenStr == "" {
		return ctx, errors.New("Unauthorized")
	}
	claim, err := ntura.TokenDecode(tokenStr)
	if err != nil {
		return ctx, err
	}
	tokenCtx := context.WithValue(parent, TokenCtxKey, tokenStr)

	database := ""
	if _, found := claim["database"]; found {
		database = claim["database"].(string)
	}
	nstore := srv.GetNervaStore(database)
	if nstore == nil {
		return ctx, errors.New("Unauthorized")
	}
	err = (&ntura.API{NStore: nstore}).TokenLogin(ntura.IM{"token": tokenStr})
	if err != nil {
		return ctx, err
	}
	ctx = context.WithValue(tokenCtx, NstoreCtxKey, &ntura.API{NStore: nstore})
	return ctx, nil
}

func (srv *RPCService) ApiKeyAuth(authorization []string, parent context.Context) (ctx context.Context, err error) {
	if len(authorization) < 1 {
		return ctx, errors.New("Unauthorized")
	}
	apiKey := strings.Trim(authorization[0], " ")
	if apiKey == "" {
		return ctx, errors.New("Unauthorized")
	}
	if os.Getenv("NT_API_KEY") != apiKey {
		return ctx, errors.New("Unauthorized")
	}
	nstore := srv.GetNervaStore("")
	ctx = context.WithValue(parent, NstoreCtxKey, &ntura.API{NStore: nstore})
	return ctx, nil
}

// UserLogin - Logs in user by username and password
func (srv *RPCService) UserLogin(ctx context.Context, req *pb.RequestUserLogin) (res *pb.ResponseUserLogin, err error) {
	if req.Database == "" {
		return res, errors.New(ntura.GetMessage("missing_database"))
	}
	nstore := srv.GetNervaStore(req.Database)
	login := ntura.IM{"username": req.Username, "password": req.Password, "database": req.Database}
	token, engine, err := (&ntura.API{NStore: nstore}).UserLogin(login)
	return &pb.ResponseUserLogin{Token: token, Engine: engine}, err
}

// User (employee or customer) password change.
func (srv *RPCService) UserPassword(ctx context.Context, req *pb.RequestUserPassword) (res *pb.ResponseEmpty, err error) {
	api := ctx.Value(NstoreCtxKey).(*ntura.API)
	options := ntura.IM{"username": req.Username, "custnumber": req.Custnumber,
		"password": req.Password, "confirm": req.Confirm}
	if req.Username != "" {
		if api.NStore.User.Scope != "admin" {
			return res, errors.New("Unauthorized")
		}
	}
	if req.Custnumber != "" {
		if api.NStore.User.Scope != "admin" {
			return res, errors.New("Unauthorized")
		}
	}
	if req.Username == "" {
		if req.Custnumber == "" {
			if api.NStore.Customer != nil {
				options["custnumber"] = api.NStore.Customer["custnumber"]
			} else {
				options["username"] = api.NStore.User.Username
			}
		}
	}
	err = api.UserPassword(options)
	return &pb.ResponseEmpty{}, err
}

// TokenDecode - decoded JWT token but doesn't validate the signature.
func (srv *RPCService) TokenDecode(ctx context.Context, req *pb.RequestTokenDecode) (*pb.ResponseTokenDecode, error) {
	mClaims, err := ntura.TokenDecode(req.Value)
	if err != nil {
		return nil, err
	}
	claims := &pb.ResponseTokenDecode{
		Username: mClaims["username"].(string), Database: mClaims["database"].(string),
		Exp: mClaims["exp"].(float64), Iss: mClaims["iss"].(string),
	}
	return claims, err
}

// TokenLogin - JWT token auth.
func (srv *RPCService) TokenLogin(ctx context.Context, req *pb.RequestEmpty) (*pb.ResponseTokenLogin, error) {
	api := ctx.Value(NstoreCtxKey).(*ntura.API)
	user := &pb.ResponseTokenLogin{
		Id: api.NStore.User.Id, Username: api.NStore.User.Username, Empnumber: api.NStore.User.Empnumber,
		Usergroup: api.NStore.User.Usergroup, Scope: api.NStore.User.Scope, Department: api.NStore.User.Department,
	}
	return user, nil
}

// TokenRefresh - Refreshes JWT token by checking at database whether refresh token exists.
func (srv *RPCService) TokenRefresh(ctx context.Context, req *pb.RequestEmpty) (res *pb.ResponseTokenRefresh, err error) {
	if ctx.Value(NstoreCtxKey) == nil {
		return res, errors.New("Unauthorized")
	}
	api := ctx.Value(NstoreCtxKey).(*ntura.API)
	token, err := api.TokenRefresh()
	return &pb.ResponseTokenRefresh{Value: token}, err
}

// TokenLogin - JWT token auth.
func (srv *RPCService) DatabaseCreate(ctx context.Context, req *pb.RequestDatabaseCreate) (log *pb.ResponseDatabaseCreate, err error) {
	options := ntura.IM{
		"database": req.Alias, "demo": strconv.FormatBool(req.Demo),
	}
	api := ctx.Value(NstoreCtxKey).(*ntura.API)
	results, err := api.DatabaseCreate(options)
	if err != nil {
		return log, err
	}
	log = &pb.ResponseDatabaseCreate{Details: srv.rowMap(results)}
	return log, err
}

// Get - returns one or more records
func (srv *RPCService) Get(ctx context.Context, req *pb.RequestGet) (res *pb.ResponseGet, err error) {
	res = &pb.ResponseGet{
		Values: []*pb.ResponseGet_Value{},
	}
	api := ctx.Value(NstoreCtxKey).(*ntura.API)
	options := ntura.IM{
		"nervatype": pb.DataType_name[int32(req.Nervatype)], "metadata": req.Metadata,
	}
	if len(req.Ids) > 0 {
		ids := []string{}
		for i := 0; i < len(req.Ids); i++ {
			ids = append(ids, strconv.FormatInt(req.Ids[i], 10))
		}
		options["ids"] = strings.Join(ids, ",")
	} else if len(req.Filters) > 0 {
		options["filters"] = strings.Join(req.Filters, ",")
	}
	results, err := api.Get(options)

	for i := 0; i < len(results); i++ {
		res.Values = append(res.Values, srv.itemMap(pb.DataType_name[int32(req.Nervatype)], results[i]))
	}

	return res, err
}

// Add/update one or more items
func (srv *RPCService) Update(ctx context.Context, req *pb.RequestUpdate) (res *pb.ResponseUpdate, err error) {
	res = &pb.ResponseUpdate{}
	api := ctx.Value(NstoreCtxKey).(*ntura.API)
	options := []ntura.IM{}
	for i := 0; i < len(req.Items); i++ {
		item := srv.fieldsToIMap(req.Items[i].Values)
		item["keys"] = srv.fieldsToIMap(req.Items[i].Keys)
		options = append(options, item)
	}
	res.Values, err = api.Update(pb.DataType_name[int32(req.Nervatype)], options)
	return res, nil
}

// Delete - delete a record
func (srv *RPCService) Delete(ctx context.Context, req *pb.RequestDelete) (res *pb.ResponseEmpty, err error) {
	api := ctx.Value(NstoreCtxKey).(*ntura.API)
	options := ntura.IM{"nervatype": pb.DataType_name[int32(req.Nervatype)], "id": int(req.Id), "key": req.Key}
	err = api.Delete(options)
	return &pb.ResponseEmpty{}, err
}

// Run raw SQL queries in safe mode
func (srv *RPCService) View(ctx context.Context, req *pb.RequestView) (res *pb.ResponseView, err error) {
	api := ctx.Value(NstoreCtxKey).(*ntura.API)
	res = &pb.ResponseView{
		Values: make(map[string]*pb.ResponseRows),
	}
	options := []ntura.IM{}
	for i := 0; i < len(req.Options); i++ {
		values := []interface{}{}
		for vi := 0; vi < len(req.Options[i].Values); vi++ {
			values = append(values, req.Options[i].Values[vi])
		}
		prm := ntura.IM{
			"key":    req.Options[i].Key,
			"text":   req.Options[i].Text,
			"values": values,
		}
		options = append(options, prm)
	}
	results, err := api.View(options)
	if err == nil {
		for fieldname, values := range results {
			res.Values[fieldname] = srv.rowMap(values)
		}
	}
	return res, err
}

// Call a server-side function
func (srv *RPCService) Function(ctx context.Context, req *pb.RequestFunction) (res *pb.ResponseFunction, err error) {
	api := ctx.Value(NstoreCtxKey).(*ntura.API)
	res = &pb.ResponseFunction{}
	options := ntura.IM{
		"key":    req.Key,
		"values": srv.fieldsToIMap(req.Values),
	}
	result, err := api.Function(options)
	if err != nil {
		return res, err
	}
	res.Value, err = ntura.ConvertToByte(result)
	return res, err
}

// List all available Nervatura Report.
func (srv *RPCService) ReportList(ctx context.Context, req *pb.RequestReportList) (res *pb.ResponseReportList, err error) {
	api := ctx.Value(NstoreCtxKey).(*ntura.API)
	res = &pb.ResponseReportList{
		Items: []*pb.ResponseReportList_Info{},
	}
	if api.NStore.User.Scope != "admin" {
		return res, errors.New("Unauthorized")
	}
	options := ntura.IM{
		"label": req.Label,
	}
	results, err := api.ReportList(options)
	if err != nil {
		return res, err
	}
	for i := 0; i < len(results); i++ {
		res.Items = append(res.Items, &pb.ResponseReportList_Info{
			Reportkey:   results[i]["reportkey"].(string),
			Repname:     results[i]["repname"].(string),
			Description: results[i]["description"].(string),
			Label:       results[i]["label"].(string),
			Reptype:     results[i]["reptype"].(string),
			Filename:    results[i]["filename"].(string),
			Installed:   results[i]["installed"].(bool),
		})
	}
	return res, err
}

// Install a report to the database.
func (srv *RPCService) ReportInstall(ctx context.Context, req *pb.RequestReportInstall) (res *pb.ResponseReportInstall, err error) {
	api := ctx.Value(NstoreCtxKey).(*ntura.API)
	res = &pb.ResponseReportInstall{}
	if api.NStore.User.Scope != "admin" {
		return res, errors.New("Unauthorized")
	}
	options := ntura.IM{
		"reportkey": req.Reportkey,
	}
	res.Id, err = api.ReportInstall(options)
	if err != nil {
		return res, err
	}
	return res, err
}

// Delete a report from the database.
func (srv *RPCService) ReportDelete(ctx context.Context, req *pb.RequestReportDelete) (res *pb.ResponseEmpty, err error) {
	api := ctx.Value(NstoreCtxKey).(*ntura.API)
	res = &pb.ResponseEmpty{}
	if api.NStore.User.Scope != "admin" {
		return res, errors.New("Unauthorized")
	}
	options := ntura.IM{
		"reportkey": req.Reportkey,
	}
	err = api.ReportDelete(options)
	return res, err
}

func (srv *RPCService) Report(ctx context.Context, req *pb.RequestReport) (res *pb.ResponseReport, err error) {
	orientation := []string{"portrait", "landscape"}
	size := []string{"a3", "a4", "a5", "letter", "legal"}
	output := []string{"auto", "xml", "tmp"}
	nervatype := []string{"none", "customer", "employee", "event", "place", "product", "project", "tool", "trans"}

	api := ctx.Value(NstoreCtxKey).(*ntura.API)
	res = &pb.ResponseReport{}
	options := ntura.IM{
		"reportkey":   req.Reportkey,
		"orientation": orientation[req.Orientation],
		"size":        size[req.Size],
		"output":      output[req.Output],
		"nervatype":   nervatype[req.Type],
		"refnumber":   req.Refnumber,
		"filters":     srv.fieldsToIMap(req.Filters),
	}
	results, err := api.Report(options)
	if err == nil {
		res.Value, err = ntura.ConvertToByte(results)
	}
	return res, err
}
