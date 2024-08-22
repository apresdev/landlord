package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/jwx"
	"github.com/apresdev/landlord/internal/mocks"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gin-gonic/gin"
)

type MockApp struct {
	cfg       aws.Config
	sts       mocks.AWSSTS
	cogIdp    mocks.AWSCognitoIdentityProvider
	validator mocks.TokenValidator
	ddb       mocks.AWSDynamoDB
}

func (m *MockApp) Cfg() aws.Config {
	return m.cfg
}

func (m *MockApp) Sts() authx.AWSSTS {
	return &m.sts
}

func (m *MockApp) CogIdp() authx.AWSCognitoIdentityProvider {
	return &m.cogIdp
}

func (m *MockApp) Validator() jwx.TokenValidator {
	return &m.validator
}

func (m *MockApp) DDB() db.AWSDynamoDB {
	return &m.ddb
}

func NewMockApp() *MockApp {
	return &MockApp{}
}

type ginTest struct {
	t *testing.T
	w *httptest.ResponseRecorder
	c *gin.Context
}

func newGinTest(t *testing.T) *ginTest {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return &ginTest{t: t, w: w, c: c}
}

func (gt *ginTest) withRequest(r *http.Request) *ginTest {
	gt.c.Request = r
	gt.c.Request.Header.Set("Content-Type", "application/json")
	return gt
}

func MustJSONMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func MustJSONUnmarshal(data []byte, v any) {
	if err := json.Unmarshal(data, v); err != nil {
		panic(err)
	}
}

func StructToDynamoMap(input any) (map[string]types.AttributeValue, error) {
	output := make(map[string]types.AttributeValue)

	v := reflect.ValueOf(input)
	t := reflect.TypeOf(input)

	// Ensure the input is a struct
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input is not a struct")
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		fieldName := fieldType.Name

		// Use the json struct tag as the DynamoDB key, if present
		if tag := fieldType.Tag.Get("json"); tag != "" && tag != "-" {
			fieldName = tag
		}

		var attrValue types.AttributeValue

		switch field.Kind() {
		case reflect.String:
			attrValue = &types.AttributeValueMemberS{Value: field.String()}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			attrValue = &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", field.Int())}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			attrValue = &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", field.Uint())}
		case reflect.Float32, reflect.Float64:
			attrValue = &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", field.Float())}
		case reflect.Bool:
			attrValue = &types.AttributeValueMemberBOOL{Value: field.Bool()}
		case reflect.Slice, reflect.Array:
			switch field.Type().Elem().Kind() {
			case reflect.String:
				var ss []string
				for j := 0; j < field.Len(); j++ {
					ss = append(ss, field.Index(j).String())
				}
				attrValue = &types.AttributeValueMemberSS{Value: ss}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				var ns []string
				for j := 0; j < field.Len(); j++ {
					ns = append(ns, fmt.Sprintf("%d", field.Index(j).Int()))
				}
				attrValue = &types.AttributeValueMemberNS{Value: ns}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				var ns []string
				for j := 0; j < field.Len(); j++ {
					ns = append(ns, fmt.Sprintf("%d", field.Index(j).Uint()))
				}
				attrValue = &types.AttributeValueMemberNS{Value: ns}
			case reflect.Float32, reflect.Float64:
				var ns []string
				for j := 0; j < field.Len(); j++ {
					ns = append(ns, fmt.Sprintf("%f", field.Index(j).Float()))
				}
				attrValue = &types.AttributeValueMemberNS{Value: ns}
			default:
				return nil, fmt.Errorf("unsupported slice/array element type: %s", field.Type().Elem().Kind())
			}
		default:
			return nil, fmt.Errorf("unsupported type: %s", field.Kind())
		}

		output[fieldName] = attrValue
	}

	return output, nil
}

func MustStructToDynamoMap(input any) map[string]types.AttributeValue {
	output, err := StructToDynamoMap(input)
	if err != nil {
		panic(err)
	}
	return output
}
