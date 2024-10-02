package apptest

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/localstack"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func StructToDynamoMap(input any) (map[string]types.AttributeValue, error) {
	output := make(map[string]types.AttributeValue)

	v := reflect.ValueOf(input)
	t := reflect.TypeOf(input)

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

// DangerousDeleteAllItems deletes all items from the DynamoDB table -- only
// for testing
func DangerousDeleteAllItems(tableName string) {
	if os.Getenv("AWS_PROFILE") != "localstack" {
		zap.L().Fatal("AWS_PROFILE must be set to 'localstack'")
	}
	if !localstack.IsRunning() {
		zap.L().Fatal("localstack not running")
	}
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		zap.L().Fatal("unable to load SDK config", zap.Error(err))
	}
	ddb := dynamodb.NewFromConfig(cfg)
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}
	scanResult, err := ddb.Scan(ctx, scanInput)
	if err != nil {
		zap.L().Fatal("failed to scan table", zap.Error(err))
	}
	for _, item := range scanResult.Items {
		key := map[string]types.AttributeValue{}
		key["pk"] = item["pk"]
		key["sk"] = item["sk"]
		_, err := ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: aws.String(tableName),
			Key:       key,
		})
		if err != nil {
			zap.L().Fatal("failed to delete item", zap.Error(err))
		}
	}
}
