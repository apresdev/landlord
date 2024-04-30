package authx

import (
	"fmt"

	"github.com/apresdev/landlord/internal/session"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// TODO: add user-pool sub-command
func listUserPools() error {
	s := session.Must()
	cc := cognitoidentityprovider.NewFromConfig(s.AWS())
	maxResults := int32(10)
	nextToken := newNextToken()
	args := cognitoidentityprovider.ListUserPoolsInput{
		MaxResults: &maxResults,
		NextToken:  &nextToken,
	}
	u, err := cc.ListUserPools(s.Ctx(), &args)
	if err != nil {
		return err
	}
	for _, up := range u.UserPools {
		fmt.Printf("User Pool: %s\n", *up.Name)
	}
	return nil
}

func getUserPool(up *types.UserPoolDescriptionType) (*types.UserPoolType, error) {
	s := session.Must()
	args := cognitoidentityprovider.DescribeUserPoolInput{
		UserPoolId: up.Id,
	}
	cc := cognitoidentityprovider.NewFromConfig(s.AWS())
	o, err := cc.DescribeUserPool(s.Ctx(), &args)
	if err != nil {
		return nil, err
	}
	return o.UserPool, nil
}

func getUserPoolDescriptionByName(userPoolName string) (*types.UserPoolDescriptionType, error) {
	s := session.Must()
	cc := cognitoidentityprovider.NewFromConfig(s.AWS())
	maxResults := int32(10)
	nextToken := newNextToken()
	args := cognitoidentityprovider.ListUserPoolsInput{
		MaxResults: &maxResults,
		NextToken:  &nextToken,
	}
	u, err := cc.ListUserPools(s.Ctx(), &args)
	if err != nil {
		return nil, err
	}
	for _, up := range u.UserPools {
		if *up.Name == userPoolName {
			return &up, nil
		}
	}
	return nil, fmt.Errorf("not found: %s", userPoolName)
}

func GetUserPoolByName(userPoolName string) (*types.UserPoolType, error) {
	up, err := getUserPoolDescriptionByName(userPoolName)
	if err != nil {
		return nil, err
	}
	return getUserPool(up)
}

func getUserPoolClients(up *types.UserPoolType) ([]types.UserPoolClientDescription, error) {
	s := session.Must()
	cc := cognitoidentityprovider.NewFromConfig(s.AWS())
	args := cognitoidentityprovider.ListUserPoolClientsInput{
		UserPoolId: up.Id,
	}
	o, err := cc.ListUserPoolClients(s.Ctx(), &args)
	if err != nil {
		return nil, err
	}
	return o.UserPoolClients, nil
}

func getSingletonUserPoolClientDescription(up *types.UserPoolType) (*types.UserPoolClientDescription, error) {
	c, err := getUserPoolClients(up)
	if err != nil {
		return nil, err
	}
	if len(c) == 0 {
		return nil, fmt.Errorf("no user pool clients found")
	}
	if len(c) > 1 {
		return nil, fmt.Errorf("more than one user pool client found")
	}
	return &c[0], nil
}

func getSingletonUserPoolClient(up *types.UserPoolType) (*types.UserPoolClientType, error) {
	cd, err := getSingletonUserPoolClientDescription(up)
	if err != nil {
		return nil, err
	}
	s := session.Must()
	cc := cognitoidentityprovider.NewFromConfig(s.AWS())
	args := cognitoidentityprovider.DescribeUserPoolClientInput{
		ClientId:   cd.ClientId,
		UserPoolId: cd.UserPoolId,
	}
	o, err := cc.DescribeUserPoolClient(s.Ctx(), &args)
	if err != nil {
		return nil, err
	}
	return o.UserPoolClient, nil
}

type Users struct {
	Entries []types.UserType
}

func GetUsersInUserPool(userPoolName string) (Users, error) {
	up, err := getUserPoolDescriptionByName(userPoolName)
	if err != nil {
		return Users{}, err
	}
	s := session.Must()
	cc := cognitoidentityprovider.NewFromConfig(s.AWS())
	args := cognitoidentityprovider.ListUsersInput{
		UserPoolId: up.Id,
	}
	o, err := cc.ListUsers(s.Ctx(), &args)
	if err != nil {
		return Users{}, err
	}
	return Users{Entries: o.Users}, nil
}
