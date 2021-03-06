package utils

import (
	"crypto/md5"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image/color"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

//go:embed static/views static/message.json
var Static embed.FS

//go:embed static/client static/css static/templates static/fonts
var Public embed.FS

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

//ToString - safe string conversion
func ToString(value interface{}, defValue string) string {
	if stringValue, valid := value.(string); valid {
		if stringValue == "" {
			return defValue
		}
		return stringValue
	}
	if boolValue, valid := value.(bool); valid {
		return strconv.FormatBool(boolValue)
	}
	if intValue, valid := value.(int); valid {
		return strconv.Itoa(intValue)
	}
	if intValue, valid := value.(int32); valid {
		return strconv.Itoa(int(intValue))
	}
	if intValue, valid := value.(int64); valid {
		return strconv.FormatInt(intValue, 10)
	}
	if floatValue, valid := value.(float32); valid {
		return strconv.FormatFloat(float64(floatValue), 'f', -1, 64)
	}
	if floatValue, valid := value.(float64); valid {
		return strconv.FormatFloat(floatValue, 'f', -1, 64)
	}
	if timeValue, valid := value.(time.Time); valid {
		return timeValue.Format("2006-01-02T15:04:05-07:00")
	}
	return defValue
}

//ToFloat - safe float64 conversion
func ToFloat(value interface{}, defValue float64) float64 {
	if floatValue, valid := value.(float64); valid {
		if floatValue == 0 {
			return defValue
		}
		return floatValue
	}
	if boolValue, valid := value.(bool); valid {
		if boolValue {
			return 1
		}
	}
	if intValue, valid := value.(int); valid {
		return float64(intValue)
	}
	if intValue, valid := value.(int32); valid {
		return float64(intValue)
	}
	if intValue, valid := value.(int64); valid {
		return float64(intValue)
	}
	if floatValue, valid := value.(float32); valid {
		return float64(floatValue)
	}
	if stringValue, valid := value.(string); valid {
		floatValue, err := strconv.ParseFloat(stringValue, 64)
		if err == nil {
			return float64(floatValue)
		}
	}
	return defValue
}

//ToRGBA - safe RGBA conversion
func ToRGBA(value interface{}, defValue color.RGBA) color.RGBA {
	parseHexColor := func(v string) (out color.RGBA, err error) {
		if len(v) != 7 {
			return out, errors.New("hex color must be 7 characters")
		}
		if v[0] != '#' {
			return out, errors.New("hex color must start with '#'")
		}
		red, redError := strconv.ParseUint(v[1:3], 16, 8)
		if redError != nil {
			return out, errors.New("red component invalid")
		}
		out.R = uint8(red)
		green, greenError := strconv.ParseUint(v[3:5], 16, 8)
		if greenError != nil {
			return out, errors.New("green component invalid")
		}
		out.G = uint8(green)
		blue, blueError := strconv.ParseUint(v[5:7], 16, 8)
		if blueError != nil {
			return out, errors.New("blue component invalid")
		}
		out.B = uint8(blue)
		return
	}

	if rgbaValue, valid := value.(color.RGBA); valid {
		return rgbaValue
	}
	if stringValue, valid := value.(string); valid {
		if strings.HasPrefix(stringValue, "#") {
			pvalue, err := parseHexColor(value.(string))
			if err == nil {
				return pvalue
			}
		} else {
			ivalue := ToInteger(value, -1)
			if ivalue > -1 && ivalue < 255 {
				return color.RGBA{uint8(ivalue), uint8(ivalue), uint8(ivalue), 0}
			}
		}
	}
	if intValue, valid := value.(int); valid {
		if intValue < 255 {
			return color.RGBA{uint8(intValue), uint8(intValue), uint8(intValue), 0}
		}
	}
	if int32Value, valid := value.(int32); valid {
		if int32Value < 255 {
			return color.RGBA{uint8(int32Value), uint8(int32Value), uint8(int32Value), 0}
		}
	}
	if int64Value, valid := value.(int64); valid {
		if int64Value < 255 {
			return color.RGBA{uint8(int64Value), uint8(int64Value), uint8(int64Value), 0}
		}
	}
	if float32Value, valid := value.(float32); valid {
		if float32Value < 255 {
			return color.RGBA{uint8(float32Value), uint8(float32Value), uint8(float32Value), 0}
		}
	}
	if float64Value, valid := value.(float64); valid {
		if float64Value < 255 {
			return color.RGBA{uint8(float64Value), uint8(float64Value), uint8(float64Value), 0}
		}
	}
	return defValue
}

//ToInteger - safe int64 conversion
func ToInteger(value interface{}, defValue int64) int64 {
	if intValue, valid := value.(int64); valid {
		if intValue == 0 {
			return defValue
		}
		return intValue
	}
	if boolValue, valid := value.(bool); valid {
		if boolValue {
			return 1
		}
	}
	if intValue, valid := value.(int); valid {
		return int64(intValue)
	}
	if intValue, valid := value.(int32); valid {
		return int64(intValue)
	}
	if floatValue, valid := value.(float32); valid {
		return int64(floatValue)
	}
	if floatValue, valid := value.(float64); valid {
		return int64(floatValue)
	}
	if stringValue, valid := value.(string); valid {
		intValue, err := strconv.ParseInt(stringValue, 10, 64)
		if err == nil {
			return int64(intValue)
		}
	}
	return defValue
}

//ToIntPointer - safe *int64 conversion
func ToIntPointer(value interface{}, defValue int64) *int64 {
	if value == nil {
		return nil
	}
	v := ToInteger(value, defValue)
	return &v
}

//ToStringPointer - safe *string conversion
func ToStringPointer(value interface{}, defValue string) *string {
	if value == nil {
		return nil
	}
	v := ToString(value, defValue)
	return &v
}

//ToBoolean - safe bool conversion
func ToBoolean(value interface{}, defValue bool) bool {
	if boolValue, valid := value.(bool); valid {
		return boolValue
	}
	if intValue, valid := value.(int); valid {
		if intValue == 1 {
			return true
		}
	}
	if intValue, valid := value.(int32); valid {
		if intValue == 1 {
			return true
		}
	}
	if intValue, valid := value.(int64); valid {
		if intValue == 1 {
			return true
		}
	}
	if floatValue, valid := value.(float32); valid {
		if floatValue == 1 {
			return true
		}
	}
	if floatValue, valid := value.(float64); valid {
		if floatValue == 1 {
			return true
		}
	}
	if stringValue, valid := value.(string); valid {
		boolValue, err := strconv.ParseBool(stringValue)
		if err == nil {
			return boolValue
		}
	}
	return defValue
}

//StringToDateTime - parse string to datetime
func StringToDateTime(value string) (time.Time, error) {
	tm, err := time.Parse("2006-01-02T15:04:05-07:00", value)
	if err != nil {
		tm, err = time.Parse("2006-01-02T15:04:05-0700", value)
	}
	if err != nil {
		tm, err = time.Parse("2006-01-02T15:04:05", value)
	}
	if err != nil {
		tm, err = time.Parse("2006-01-02T15:04:05Z", value)
	}
	if err != nil {
		tm, err = time.Parse("2006-01-02 15:04:05", value)
	}
	if err != nil {
		tm, err = time.Parse("2006-01-02 15:04", value)
	}
	if err != nil {
		tm, err = time.Parse("2006-01-02", value)
	}
	return tm, err
}

// Find returns the smallest index i at which x == a[i],
// or len(a) if there is no such index.
func Find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return len(a)
}

// Contains tells whether a contains x.
func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func ConvertToByte(data interface{}) ([]byte, error) {
	//var json = jsoniter.ConfigCompatibleWithStandardLibrary
	return json.Marshal(data)
}

func ConvertFromByte(data []byte, result interface{}) error {
	//var json = jsoniter.ConfigCompatibleWithStandardLibrary
	return json.Unmarshal(data, result)
}

func ConvertFromReader(data io.Reader, result interface{}) error {
	return json.NewDecoder(data).Decode(&result)
}

//GetMessage - error messages
func GetMessage(key string) string {
	var messages map[string]string
	var jsonMessages, _ = Static.ReadFile("static/message.json")
	ConvertFromByte(jsonMessages, &messages)
	if value, found := messages[key]; found {
		return value
	}
	return ""
}

/*
CreateToken - create/refresh a Nervatura JWT token
*/
func CreateToken(username, database string, config map[string]interface{}) (string, error) {
	// ntClaims is a custom Nervatura claims type
	type ntClaims struct {
		Username string `json:"username"`
		Database string `json:"database"`
		jwt.StandardClaims
	}

	expirationTime := time.Now().Add(time.Duration(ToFloat(config["NT_TOKEN_EXP"], 1)) * time.Hour)
	claims := ntClaims{
		username,
		database,
		jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			Issuer:    ToString(config["NT_TOKEN_ISS"], "nervatura"),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = ToString(config["NT_TOKEN_KID"], GetMD5Hash("nervatura"))
	return token.SignedString([]byte(ToString(config["NT_TOKEN_PRIVATE_KEY"], GetMD5Hash(time.Now().Format("20060102")))))
}

/*
TokenDecode - decoded JWT token but doesn't validate the signature.
*/
func TokenDecode(tokenString string) (map[string]interface{}, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err == nil {
		return token.Claims.(jwt.MapClaims), err
	}
	return nil, err
}

func parsePEM(key map[string]string) (interface{}, error) {
	if key["ktype"] == "RSA" && key["type"] == "private" {
		return jwt.ParseRSAPrivateKeyFromPEM([]byte(key["value"]))
	}
	if key["ktype"] == "ECP" && key["type"] == "private" {
		return jwt.ParseECPrivateKeyFromPEM([]byte(key["value"]))
	}
	if key["ktype"] == "RSA" && key["type"] == "public" {
		return jwt.ParseRSAPublicKeyFromPEM([]byte(key["value"]))
	}
	if key["ktype"] == "ECP" && key["type"] == "public" {
		return jwt.ParseECPublicKeyFromPEM([]byte(key["value"]))
	}
	return []byte(key["value"]), nil
}

/*
ParseToken - Parse, validate, and return a token data.
*/
func ParseToken(tokenString string, keyMap map[string]map[string]string, config map[string]interface{}) (map[string]interface{}, error) {
	data := make(map[string]interface{})
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Unexpected signing method: " + token.Header["alg"].(string))
		}
		kid := ToString(token.Header["kid"], ToString(config["NT_TOKEN_KID"], GetMD5Hash("nervatura")))
		if keyMap, found := keyMap[kid]; found {
			return parsePEM(keyMap)
		}
		return []byte(ToString(config["NT_TOKEN_PRIVATE_KEY"], GetMD5Hash(time.Now().Format("20060102")))), nil
	})
	if err != nil {
		return data, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
				return data, errors.New("token is either expired or not active yet")
			}
		}
		return data, err
	}

	if _, found := claims["database"]; !found {
		if ToString(config["NT_ALIAS_DEFAULT"], "") == "" {
			return data, errors.New(GetMessage("missing_database"))
		}
		data["database"] = ToString(config["NT_ALIAS_DEFAULT"], "")
	}
	data["database"] = claims["database"]
	data["username"] = ""
	if _, found := claims["username"]; found {
		data["username"] = claims["username"]
	} else if _, found := claims["custnumber"]; found {
		data["username"] = claims["custnumber"]
	} else if _, found := claims["email"]; found {
		data["username"] = claims["email"]
	} else if _, found := claims["phone_number"]; found {
		data["username"] = claims["phone_number"]
	}
	if data["username"] == "" {
		return data, errors.New(GetMessage("missing_user"))
	}
	return data, nil

}

func RandString(length int) string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}
