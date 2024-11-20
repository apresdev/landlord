package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apresdev/landlord/internal/perms"
	"github.com/golang-jwt/jwt/v5"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"

	"github.com/apresdev/landlord/internal/apptest"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/mock"

	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/appenv"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/jwx"
	"github.com/apresdev/landlord/internal/mocks"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gin-gonic/gin"
)

type MockApp struct {
	cfg          aws.Config
	cogIdp       mocks.AWSCognitoIdentityProvider
	validator    mocks.TokenValidator
	ddb          mocks.AWSDynamoDB
	settings     appenv.Settings
	rootTenantID uuid.UUID
}

func (m *MockApp) Cfg() aws.Config {
	return m.cfg
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

func (m *MockApp) Settings() appenv.Settings {
	return m.settings
}

func (m *MockApp) RootTenantID() uuid.UUID {
	return m.rootTenantID
}

func NewMockApp() *MockApp {
	settings := appenv.MustSettings()
	app := &MockApp{
		settings:     settings,
		rootTenantID: uuid.New(),
	}
	app.ddb.EXPECT().DescribeTable(mock.Anything, mock.Anything).Return(nil, nil)
	db.MustInit(context.Background(), settings, app.DDB())
	return app
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

func marshalTenant(t *multi.Tenant) map[string]ddbtypes.AttributeValue {
	result, _ := attributevalue.MarshalMap(t)
	result["pk"] = &ddbtypes.AttributeValueMemberS{Value: db.GlobalEntityID("*")}
	result["sk"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(t.ID.String())}
	result["altName"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(t.Name)}
	return result
}

func withSetupRootTenant(app *MockApp) *multi.Tenant {
	rt := multi.NewTenant(multi.RootTenantName)
	rt.ID = app.RootTenantID()
	o := dynamodb.GetItemOutput{}
	o.Item = marshalTenant(rt)
	kv := make(map[string]ddbtypes.AttributeValue)
	kv["pk"] = &ddbtypes.AttributeValueMemberS{Value: db.GlobalEntityID("*")}
	kv["sk"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(rt.ID.String())}
	input := &dynamodb.GetItemInput{
		TableName: aws.String(app.Settings().DBTableName()),
		Key:       kv,
	}
	app.ddb.EXPECT().GetItem(mock.Anything, input).Return(&o, nil)
	return rt
}

func withSetupQueryAllTenants(app *MockApp, additionalTenants ...*multi.Tenant) {
	rt := multi.NewTenant(multi.RootTenantName)
	rt.ID = app.RootTenantID()

	et := multi.NewTenant("example-tenant")
	et.ID = uuid.New()

	o := dynamodb.QueryOutput{}
	o.Items = append(o.Items, marshalTenant(rt))
	o.Items = append(o.Items, marshalTenant(et))
	for _, additionalTenant := range additionalTenants {
		o.Items = append(o.Items, marshalTenant(additionalTenant))
	}

	kv := make(map[string]ddbtypes.AttributeValue)
	kv[":pk"] = &ddbtypes.AttributeValueMemberS{Value: db.GlobalEntityID("*")}
	kv[":skPrefix"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID("")}
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(app.Settings().DBTableName()),
		KeyConditionExpression:    aws.String("pk = :pk AND begins_with(sk, :skPrefix)"),
		ExpressionAttributeValues: kv,
	}
	app.ddb.EXPECT().Query(mock.Anything, input).Return(&o, nil)
}

func withSetupQueryTenant(app *MockApp) *multi.Tenant {
	ten := multi.NewTenant("ten1")
	ten.Roles = append(ten.Roles, "guest")
	o := dynamodb.QueryOutput{}
	o.Items = append(o.Items, marshalTenant(ten))
	kv := make(map[string]ddbtypes.AttributeValue)
	kv[":altName"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(ten.Name)}
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(db.TableName()),
		IndexName:                 aws.String("altNameIndex"),
		KeyConditionExpression:    aws.String("altName = :altName"),
		ExpressionAttributeValues: kv,
	}
	app.ddb.EXPECT().Query(mock.Anything, input).Return(&o, nil)
	return ten
}

func withSetupGetTenantByID(app *MockApp) *multi.Tenant {
	ten := multi.NewTenant("ten1")
	withSetupGetTenantByID2(app, ten)
	return ten
}

func withSetupGetTenantByID2(app *MockApp, ten *multi.Tenant) {
	o := dynamodb.GetItemOutput{}
	o.Item = marshalTenant(ten)
	kv := make(map[string]ddbtypes.AttributeValue)
	kv["pk"] = &ddbtypes.AttributeValueMemberS{Value: db.GlobalEntityID("*")}
	kv["sk"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(ten.ID.String())}
	input := &dynamodb.GetItemInput{
		TableName: aws.String(db.TableName()),
		Key:       kv,
	}
	app.ddb.EXPECT().GetItem(mock.Anything, input).Return(&o, nil)
}

func marshalUser(u *multi.User) map[string]ddbtypes.AttributeValue {
	result := apptest.MustStructToDynamoMap(*u)
	result["pk"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(u.TenantID.String())}
	result["sk"] = &ddbtypes.AttributeValueMemberS{Value: db.UserEntityID(u.ID.String())}
	result["altName"] = &ddbtypes.AttributeValueMemberS{Value: db.UserEntityID(u.Username)}
	return result
}

func newTestUser(t *multi.Tenant) *multi.User {
	u := multi.NewUser(t.ID, uuid.New(), "user1@example.com", "guest", multi.MFAPolicyNone)
	u.Status = multi.UserStatusActive
	return u
}

func withSetupQueryUser(app *MockApp, t *multi.Tenant) {
	withSetupQueryUser2(app, newTestUser(t))
}

func withSetupQueryUser2(app *MockApp, u *multi.User) {
	o := dynamodb.QueryOutput{}
	o.Items = append(o.Items, marshalUser(u))
	kv := make(map[string]ddbtypes.AttributeValue)
	kv[":altName"] = &ddbtypes.AttributeValueMemberS{Value: db.UserEntityID(u.Username)}
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(db.TableName()),
		IndexName:                 aws.String("altNameIndex"),
		KeyConditionExpression:    aws.String("altName = :altName"),
		ExpressionAttributeValues: kv,
	}
	app.ddb.EXPECT().Query(mock.Anything, input).Return(&o, nil)
}

func withSetupQueryUsers(app *MockApp, ten *multi.Tenant, users []*multi.User) {
	o := dynamodb.QueryOutput{}
	for _, u := range users {
		o.Items = append(o.Items, marshalUser(u))
	}
	kv := make(map[string]ddbtypes.AttributeValue)
	kv[":pk"] = &ddbtypes.AttributeValueMemberS{Value: db.TenantEntityID(ten.ID.String())}
	kv[":skPrefix"] = &ddbtypes.AttributeValueMemberS{Value: db.UserEntityID("")}
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(db.TableName()),
		KeyConditionExpression:    aws.String("pk = :pk AND begins_with(sk, :skPrefix)"),
		ExpressionAttributeValues: kv,
	}
	app.ddb.EXPECT().Query(mock.Anything, input).Return(&o, nil)
}

func withGetUserPoolClient(app *MockApp, ctx context.Context) {
	app.cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	app.cogIdp.EXPECT().DescribeUserPoolClient(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolClientOutput{
			UserPoolClient: &types.UserPoolClientType{
				ClientId:     aws.String("test-client-id1"),
				ClientSecret: aws.String("test-client-secret")}}, nil).Once()
}

func withSetupJWT(c *gin.Context, tenant *multi.Tenant, user *multi.User) {
	claims := make(jwt.MapClaims)
	claims[authx.CustomTenantClaim] = tenant.ID.String()
	claims[authx.CognitoGroupsClaim] = []string{tenant.GroupName(user.Role)}
	claims[authx.EmailClaim] = user.Username
	claims[authx.SubClaim] = user.ID.String()
	tok := &jwt.Token{Claims: claims}
	c.Set(perms.TokenKey, tok)
}
