/*
Package nervatura -

Open Source Business Management Framework

It can handle any type of business related information, starting from customer details, up to shipping, stock or payment information.

The framework is based on Nervatura Object Model (https://nervatura.github.io/nervatura-docs/#/model) specification. The
Nervatura is a set of open source applications and documentations. It enables to easily create a basic open-data business management
system. Its key elements are:

• Nervatura DOCS (https://nervatura.github.io/nervatura-docs) for a quick start

• Nervatura API (https://nervatura.github.io/nervatura/api) for applications that would use the data

• Nervatura Client (https://github.com/nervatura/nervatura-client) is a free PWA Client

• Client- and server-side data reporting (https://nervatura.github.io/nervatura-demo/)

• and other documents, resources, sample code and demo programs

Installation

To install the package on your system, run
 go get github.com/nervatura/nervatura-go

Later, to receive updates, run
 go get -u -v github.com/nervatura/nervatura-go/...

Quick Start

Compile and run the cmd/main.go file and open https://localhost:8080/

More golang examples: test/api_test.go, test/nervastore_test.go, test/npi_test.go

Homepage: http://www.nervatura.com

*/
package nervatura

//TimeLayout DateTime format
const TimeLayout = "2006-01-02 15:04:05"

//IM is a map[string]interface{} type short alias
type IM = map[string]interface{}

//SM is a map[string]string type short alias
type SM = map[string]string

//IL is a []interface{} type short alias
type IL = []interface{}

//IList is a []interface{} string const
const IList = "[]interface{}"

//SL is a []string type short alias
type SL = []string

//DataDriver a general database interface
type DataDriver interface {
	Properties() struct {
		SQL, Transaction bool
	} //DataDriver features
	Connection() struct {
		Alias     string
		Connected bool
		Engine    string
	} //Returns the database connection
	CreateConnection(string, string) error                                                  //Create a new database connection
	CreateDatabase(logData []SM) ([]SM, error)                                              //Create a Nervatura Database
	CheckHashtable(hashtable string) error                                                  //Check/create a password ref. table
	UpdateHashtable(hashtable, refname, value string) error                                 //Set a password
	Query(queries []Query, transaction interface{}) ([]IM, error)                           //Query is a basic nosql friendly queries the database
	QuerySQL(sqlString string, params []interface{}, transaction interface{}) ([]IM, error) //Executes a SQL query
	QueryKey(options IM, transaction interface{}) ([]IM, error)                             //Complex data queries
	Update(options Update) (int64, error)                                                   //Update is a basic nosql friendly update/insert/delete and returns the update/insert id
	BeginTransaction() (interface{}, error)                                                 //Begins a transaction and returns an it
	CommitTransaction(trans interface{}) error                                              //Commit a transaction
	RollbackTransaction(trans interface{}) error                                            //Rollback a transaction
}

//Filter query filter type
type Filter struct {
	Or    bool   // and (False) or (True)
	Field string //Fieldname and alias
	Comp  string //==,!=,<,<=,>,>=,in,is
	Value interface{}
}

//Query data desc. type
type Query struct {
	Fields  []string //Returns fields
	From    string   //Table or doc. name and alias
	Filters []Filter
	Filter  string //filter string (eg. "id=1 and field='value'")
	OrderBy []string
}

//Update data desc. type
type Update struct {
	Values IM
	Model  string
	IDKey  int64       //Update, delete or insert exec
	Trans  interface{} //Transaction
}

// User - Nervatura user properties
type User struct {
	Id         int64  `json:"id"`
	Username   string `json:"username"`
	Empnumber  string `json:"empnumber"`
	Usergroup  int64  `json:"usergroup"`
	Scope      string `json:"scope"`
	Department string `json:"department,omitempty"`
}
