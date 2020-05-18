package nervatura

import (
	"crypto/md5"
	"encoding/hex"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/spf13/viper"
)

func getMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

//GetIType - returns the input type name
func GetIType(value interface{}) string {
	switch value.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case int8:
		return "int8"
	case uint8:
		return "uint8"
	case int16:
		return "int16"
	case uint16:
		return "uint16"
	case int32:
		return "int32"
	case uint32:
		return "uint32"
	case int64:
		return "int64"
	case uint64:
		return "uint64"
	case int:
		return "int"
	case uint:
		return "uint"
	case uintptr:
		return "uintptr"
	case float32:
		return "float32"
	case float64:
		return "float64"
	case complex64:
		return "complex64"
	case complex128:
		return "complex128"
	case []string:
		return "[]string"
	case map[string]string:
		return "map[string]string"
	case map[string]interface{}:
		return "map[string]interface{}"
	case []map[string]interface{}:
		return "[]map[string]interface{}"
	case []interface{}:
		return IList
	case []Filter:
		return "[]Filter"
	}
	return ""
}

//ReadConfig - read all config options
//Environment Variables: e.q. NTURA_DATABASES_DEMO_CONNECT_PASSWORD
func ReadConfig(confpath string) (Settings, error) {
	settings := Settings{}
	v := viper.New()
	viper.AddConfigPath(".")
	v.AddConfigPath(confpath)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("NT")
	v.AutomaticEnv()
	v.SetConfigName("settings")
	err := v.ReadInConfig()
	if err != nil {
		return settings, err
	}
	v.Unmarshal(&settings)
	return settings, nil
}

/*
TokenDecode - decoded JWT token but doesn't validate the signature.
*/
func TokenDecode(tokenString string) (IM, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	return token.Claims.(jwt.MapClaims), err
}

//GetMessage - error messages
func GetMessage(key string) string {
	messages := messages()
	if value, found := messages[key]; found {
		return value
	}
	return ""
}
