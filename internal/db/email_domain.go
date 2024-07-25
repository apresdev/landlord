package db

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

const (
	emailDomainsTable = "landlord-email-domains"
	domainAttr        = "domain"
)

func initEmailDomains(ctx context.Context, dynamo AWSDynamoDB) error {
	attrs := []types.AttributeDefinition{
		{
			AttributeName: aws.String(domainAttr),
			AttributeType: types.ScalarAttributeTypeS,
		},
	}
	keySchema := []types.KeySchemaElement{
		{
			AttributeName: aws.String(domainAttr),
			KeyType:       types.KeyTypeHash,
		},
	}
	return ensureTable(ctx, dynamo, emailDomainsTable, attrs, keySchema, nil)
}

type EmailDomain struct {
	Domain    string
	Exclusive bool
	TenantIDs []uuid.UUID
}

func StoreEmailDomain(ctx context.Context, ddb AWSDynamoDB, domain *EmailDomain) error {
	av, err := attributevalue.MarshalMap(domain)
	if err != nil {
		return err
	}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(emailDomainsTable),
	}
	_, err = ddb.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put item to %s: %w", emailDomainsTable, err)
	}
	return err
}

func DeleteEmailDomain(
	ctx context.Context, ddb AWSDynamoDB, dom string) error {

	input := &dynamodb.DeleteItemInput{
		Key: map[string]types.AttributeValue{
			domainAttr: &types.AttributeValueMemberS{Value: dom},
		},
		TableName: aws.String(emailDomainsTable),
	}
	_, err := ddb.DeleteItem(ctx, input)
	return err
}

func GetEmailDomain(
	ctx context.Context, ddb AWSDynamoDB, dom string) (*EmailDomain, error) {

	input := &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			domainAttr: &types.AttributeValueMemberS{Value: dom},
		},
		TableName: aws.String(emailDomainsTable),
	}
	i, err := ddb.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get item from DynamoDB: %w", err)
	}
	if i.Item == nil {
		return nil, fmt.Errorf("item not found")
	}
	var result EmailDomain
	err = attributevalue.UnmarshalMap(i.Item, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal DynamoDB item to email domain: %w", err)
	}
	return &result, nil
}
