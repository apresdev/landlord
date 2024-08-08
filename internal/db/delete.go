package db

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func DeleteTables(ctx context.Context, ddb AWSDynamoDB) error {
	for _, tableName := range tableNames {
		if tableExists(ctx, ddb, tableName) {
			fmt.Printf("Initiating delete of table: %s\n", tableName)
			if err := deleteTable(ctx, ddb, tableName); err != nil {
				return err
			}
		} else {
			fmt.Printf("Table does not exist: %s\n", tableName)
		}
	}
	for {
		allDeleted := true
		for _, tableName := range tableNames {
			if tableExists(ctx, ddb, tableName) {
				fmt.Printf("Waiting for table %s to be deleted\n", tableName)
				allDeleted = false
			}
		}
		if allDeleted {
			break
		}
		time.Sleep(2 * time.Second)
	}
	return nil
}

func deleteTable(ctx context.Context, ddb AWSDynamoDB, table string) error {
	input := &dynamodb.DeleteTableInput{
		TableName: aws.String(table),
	}
	_, err := ddb.DeleteTable(ctx, input)
	if err != nil {
		return err
	}
	return nil
}
