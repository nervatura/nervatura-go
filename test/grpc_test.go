package test

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/nervatura/nervatura-go/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func getClient(t *testing.T) (*grpc.ClientConn, pb.APIClient) {
	conn, err := grpc.Dial("localhost:9200", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("fail to dial: %v", err)
	}
	return conn, pb.NewAPIClient(conn)
}

func getRPCToken(t *testing.T) string {
	conn, client := getClient(t)
	defer conn.Close()

	resp, err := client.UserLogin(context.Background(), &pb.RequestUserLogin{
		Username: "admin", Database: "demo",
	})
	if err != nil {
		t.Fatalf("UserLogin failed: %v", err)
	}
	return resp.Token
}

func getAuth(token string) string {
	return "Bearer " + token
}

func TestRpcUserLogin(t *testing.T) {
	token := getRPCToken(t)
	fmt.Printf("UserLogin: %+v\n", token)
}

func TestRpcDatabaseCreate(t *testing.T) {
	conn, client := getClient(t)
	defer conn.Close()

	md := metadata.Pairs("X-Api-Key", "TEST_API_KEY")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := client.DatabaseCreate(ctx, &pb.RequestDatabaseCreate{Alias: "demo", Demo: true})
	if err != nil {
		t.Fatalf("DatabaseCreate failed: %v", err)
	}
	fmt.Printf("DatabaseCreate %+v\n", resp.Details)
}

func TestRpcTokenLogin(t *testing.T) {
	conn, client := getClient(t)
	defer conn.Close()

	token := getRPCToken(t)
	md := metadata.Pairs("Authorization", getAuth(token))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := client.TokenLogin(ctx, &pb.RequestEmpty{})
	if err != nil {
		t.Fatalf("TokenLogin failed: %v", err)
	}
	fmt.Printf("TokenLogin: %+v\n", resp)
}

func TestRpcTokenRefresh(t *testing.T) {
	conn, client := getClient(t)
	defer conn.Close()

	token := getRPCToken(t)
	md := metadata.Pairs("Authorization", getAuth(token))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := client.TokenRefresh(ctx, &pb.RequestEmpty{})
	if err != nil {
		t.Fatalf("TokenRefresh failed: %v", err)
	}
	fmt.Printf("TokenRefresh %+v\n", resp.Value)
}

func TestRpcTokenDecode(t *testing.T) {
	conn, client := getClient(t)
	defer conn.Close()

	token := getRPCToken(t)
	resp, err := client.TokenDecode(context.Background(), &pb.RequestTokenDecode{Value: token})
	if err != nil {
		t.Fatalf("TokenDecode failed: %v", err)
	}
	fmt.Printf("TokenDecode %+v\n", resp)
}

func TestRpcUserPassword(t *testing.T) {
	conn, client := getClient(t)
	defer conn.Close()

	login, _ := client.UserLogin(context.Background(), &pb.RequestUserLogin{
		Username: "admin", Database: "demo",
	})

	md := metadata.Pairs("Authorization", "Bearer "+login.Token)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	_, err := client.UserPassword(ctx,
		&pb.RequestUserPassword{Username: "guest", Password: "321", Confirm: "321"})
	if err != nil {
		t.Fatalf("UserPassword failed: %v", err)
	}
	fmt.Println("UserPassword OK")
}

func TestRpcGet(t *testing.T) {
	conn, client := getClient(t)
	defer conn.Close()

	token := getRPCToken(t)
	md := metadata.Pairs("Authorization", getAuth(token))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := client.Get(ctx, &pb.RequestGet{
		Nervatype: pb.DataType_trans, Metadata: true, Ids: []int64{4},
	})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	fmt.Printf("Get %+v\n", resp.Values)
}

func TestRpcUpdate(t *testing.T) {
	conn, client := getClient(t)
	defer conn.Close()

	token := getRPCToken(t)
	md := metadata.Pairs("Authorization", getAuth(token))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := client.Update(ctx,
		&pb.RequestUpdate{
			Nervatype: pb.DataType_address,
			Items: []*pb.RequestUpdate_Item{
				{Values: map[string]*pb.Value{
					"nervatype":         {Value: &pb.Value_Number{Number: 10}},
					"ref_id":            {Value: &pb.Value_Number{Number: 2}},
					"zipcode":           {Value: &pb.Value_Text{Text: "12345"}},
					"city":              {Value: &pb.Value_Text{Text: "BigCity"}},
					"notes":             {Value: &pb.Value_Text{Text: "Create a new item by IDs"}},
					"address_metadata1": {Value: &pb.Value_Text{Text: "value1"}},
					"address_metadata2": {Value: &pb.Value_Text{Text: "value2~note2"}},
				}},
				{Values: map[string]*pb.Value{
					"id":      {Value: &pb.Value_Number{Number: 6}},
					"zipcode": {Value: &pb.Value_Text{Text: "54321"}},
					"city":    {Value: &pb.Value_Text{Text: "BigCity"}},
					"notes":   {Value: &pb.Value_Text{Text: "Update an item by IDs"}},
				}},
				{Values: map[string]*pb.Value{
					"zipcode":           {Value: &pb.Value_Text{Text: "12345"}},
					"city":              {Value: &pb.Value_Text{Text: "BigCity"}},
					"notes":             {Value: &pb.Value_Text{Text: "Create a new item by IDs"}},
					"address_metadata1": {Value: &pb.Value_Text{Text: "value1"}},
					"address_metadata2": {Value: &pb.Value_Text{Text: "value2~note2"}},
				},
					Keys: map[string]*pb.Value{
						"nervatype": {Value: &pb.Value_Text{Text: "customer"}},
						"ref_id":    {Value: &pb.Value_Text{Text: "customer/DMCUST/00001"}},
					}},
				{Values: map[string]*pb.Value{
					"zipcode": {Value: &pb.Value_Text{Text: "12345"}},
					"city":    {Value: &pb.Value_Text{Text: "BigCity"}},
					"notes":   {Value: &pb.Value_Text{Text: "Update an item by Keys"}},
				},
					Keys: map[string]*pb.Value{
						"id": {Value: &pb.Value_Text{Text: "customer/DMCUST/00001~1"}},
					}},
			},
		},
	)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	fmt.Printf("Update %+v\n", resp.Values)
}

func TestRpcDelete(t *testing.T) {
	conn, client := getClient(t)
	defer conn.Close()

	token := getRPCToken(t)
	md := metadata.Pairs("Authorization", getAuth(token))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	_, err := client.Delete(ctx,
		&pb.RequestDelete{Nervatype: pb.DataType_address, Id: 2})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	fmt.Println("Delete OK")
}

func TestRpcView(t *testing.T) {
	conn, client := getClient(t)
	defer conn.Close()

	token := getRPCToken(t)
	md := metadata.Pairs("Authorization", getAuth(token))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := client.View(ctx,
		&pb.RequestView{
			Options: []*pb.RequestView_Query{
				{
					Key:    "customers",
					Text:   "select c.id, ct.groupvalue as custtype, c.custnumber, c.custname from customer c inner join groups ct on c.custtype = ct.id where c.deleted = 0 and c.custnumber <> 'HOME'",
					Values: []*pb.Value{},
				},
				{
					Key: "invoices",
					Text: `select t.id, t.transnumber, tt.groupvalue as transtype, td.groupvalue as direction, t.transdate, c.custname, t.curr, items.amount
							  from trans t inner join groups tt on t.transtype = tt.id
								inner join groups td on t.direction = td.id
								inner join customer c on t.customer_id = c.id
								inner join (
									select trans_id, sum(amount) amount
								  from item where deleted = 0 group by trans_id) items on t.id = items.trans_id
								where t.deleted = 0 and tt.groupvalue = 'invoice'`,
					Values: []*pb.Value{},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("View failed: %v", err)
	}
	fmt.Printf("View %+v\n", resp.Values)
}

func TestRpcFunction(t *testing.T) {
	conn, client := getClient(t)
	defer conn.Close()

	token := getRPCToken(t)
	md := metadata.Pairs("Authorization", getAuth(token))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := client.Function(ctx,
		&pb.RequestFunction{
			Key: "nextNumber",
			Values: map[string]*pb.Value{
				"numberkey": {Value: &pb.Value_Text{Text: "custnumber"}},
				"step":      {Value: &pb.Value_Boolean{Boolean: false}},
			},
		},
	)
	if err != nil {
		t.Fatalf("Function failed: %v", err)
	}
	fmt.Printf("Function %+v\n", string(resp.Value))
}

func TestRpcReportList(t *testing.T) {
	conn, client := getClient(t)
	defer conn.Close()

	token := getRPCToken(t)
	md := metadata.Pairs("Authorization", getAuth(token))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := client.ReportList(ctx,
		&pb.RequestReportList{
			Label: "",
		},
	)
	if err != nil {
		t.Fatalf("ReportList failed: %v", err)
	}
	fmt.Printf("ReportList %+v\n", resp.Items)
}

func TestRpcReportDelete(t *testing.T) {
	conn, client := getClient(t)
	defer conn.Close()

	token := getRPCToken(t)
	md := metadata.Pairs("Authorization", getAuth(token))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	_, err := client.ReportDelete(ctx,
		&pb.RequestReportDelete{
			Reportkey: "ntr_cash_in_en",
		},
	)
	if err != nil {
		t.Fatalf("ReportDelete failed: %v", err)
	}
	fmt.Println("ReportDelete OK")
}

func TestRpcReportInstall(t *testing.T) {
	conn, client := getClient(t)
	defer conn.Close()

	token := getRPCToken(t)
	md := metadata.Pairs("Authorization", getAuth(token))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	id, err := client.ReportInstall(ctx,
		&pb.RequestReportInstall{
			Reportkey: "ntr_cash_in_en",
		},
	)
	if err != nil {
		t.Fatalf("ReportInstall failed: %v", err)
	}
	fmt.Printf("ReportInstall %+v\n", id)
}

func TestRpcReport(t *testing.T) {
	conn, client := getClient(t)
	defer conn.Close()

	token := getRPCToken(t)
	md := metadata.Pairs("Authorization", getAuth(token))
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	result, err := client.Report(ctx,
		&pb.RequestReport{
			Reportkey:   "ntr_invoice_en",
			Orientation: pb.ReportOrientation_portrait,
			Size:        pb.ReportSize_a4,
			Type:        pb.ReportType_report_trans,
			Refnumber:   "DMINV/00001",
		},
	)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}
	fmt.Printf("Report %+v\n", result.Value)
}
