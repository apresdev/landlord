package cmd

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
)

type En struct {
	Name string
	Type reflect.Type
}

const AWSPolicyDocumentVersion = "2012-10-17"

// PolicyDocument defines a policy document as a Go struct that can be serialized
// to JSON.
type PolicyDocument struct {
	Version   string
	Statement []PolicyStatement
}

func NewPolicyDocument() *PolicyDocument {
	return &PolicyDocument{
		Version: AWSPolicyDocumentVersion,
	}
}

// PolicyStatement defines a statement in a policy document.
type PolicyStatement struct {
	Effect    string
	Action    []string
	Principal map[string]string `json:",omitempty"`
	Resource  *string           `json:",omitempty"`
}

func methods[T any](prefix string) []string {
	ts := reflect.TypeFor[T]()
	result := make([]string, 0)
	for i := 0; i < ts.NumMethod(); i++ {
		result = append(result, fmt.Sprintf("%s:%s", prefix, ts.Method(i).Name))
	}
	return result
}

func wildcardResourceCalls() []string {
	result := make([]string, 0)
	result = append(result, methods[authx.AWSSTS]("sts")...)
	result = append(result, methods[authx.AWSSES]("ses")...)
	result = append(result, methods[authx.AWSCognitoIdentityProvider]("cognito-idp")...)
	return result
}

func policyDoc(cs Clients) (*PolicyDocument, error) {
	result := NewPolicyDocument()
	wildcardResourcePolicy := PolicyStatement{
		Effect:   "Allow",
		Resource: aws.String("*"),
	}
	wildcardResourcePolicy.Action = wildcardResourceCalls()
	result.Statement = append(result.Statement, wildcardResourcePolicy)
	tables := db.TableNames()
	dynamoMethods := methods[db.AWSDynamoDB]("dynamodb")
	o, err := cs.Sts().GetCallerIdentity(cs.Ctx(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}
	for _, table := range tables {
		result.Statement = append(result.Statement, PolicyStatement{
			Effect: "Allow",
			Action: dynamoMethods,
			Resource: aws.String(fmt.Sprintf(
				"arn:aws:dynamodb:%s:%s:table/%s",
				cs.Cfg().Region, aws.ToString(o.Account), table)),
		})
		result.Statement = append(result.Statement, PolicyStatement{
			Effect: "Allow",
			Action: []string{"dynamodb:Query"},
			Resource: aws.String(fmt.Sprintf(
				"arn:aws:dynamodb:%s:%s:table/%s/index/*",
				cs.Cfg().Region, aws.ToString(o.Account), table)),
		})
	}
	return result, nil
}

func iamCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "iam",
		Short: "Describe the IAM policy needed for the app",
		RunE: func(cmd *cobra.Command, args []string) error {
			cs := MustClients()
			p, err := policyDoc(cs)
			if err != nil {
				return err
			}
			b, err := json.MarshalIndent(p, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		},
	}
	return c
}
