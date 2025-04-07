package cmd

import (
	"encoding/json"
	"fmt"
	"reflect"

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
	result = append(result, methods[authx.AWSCognitoIdentityProvider]("cognito-idp")...)
	return result
}

func policyDoc() (*PolicyDocument, error) {
	result := NewPolicyDocument()
	wildcardResourcePolicy := PolicyStatement{
		Effect:   "Allow",
		Resource: aws.String("*"),
	}
	wildcardResourcePolicy.Action = wildcardResourceCalls()
	result.Statement = append(result.Statement, wildcardResourcePolicy)
	dynamoMethods := methods[db.AWSDynamoDB]("dynamodb")
	result.Statement = append(result.Statement, PolicyStatement{
		Effect:   "Allow",
		Action:   dynamoMethods,
		Resource: aws.String("${module.landlord_dynamo.table_arn}"),
	})
	result.Statement = append(result.Statement, PolicyStatement{
		Effect:   "Allow",
		Action:   []string{"dynamodb:Query"},
		Resource: aws.String("${module.landlord_dynamo.table_arn}" + "/index*"),
	})
	return result, nil
}

func iamCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "iam",
		Short: "Describe the IAM policy needed for the app",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := policyDoc()
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
