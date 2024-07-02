package authx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/service/iam/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

const RoleGuest = "guest"

type PolicyStatement struct {
	Effect    string                       `json:"Effect,omitempty"`
	Action    []string                     `json:"Action,omitempty"`
	Principal map[string]string            `json:"Principal,omitempty"`
	Condition map[string]map[string]string `json:"Condition,omitempty"`
}

type TrustPolicyDocument struct {
	Version   string
	Statement []PolicyStatement
}

func NewTrustPolicyDocument() *TrustPolicyDocument {
	return &TrustPolicyDocument{
		Version: "2012-10-17",
	}
}

func (t *TrustPolicyDocument) AddStatement(s PolicyStatement) {
	t.Statement = append(t.Statement, s)
}

func (t *TrustPolicyDocument) ToJSONString() (string, error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetIndent("  ", "  ")
	err := enc.Encode(t)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func NewTrustPolicyDocumentForTenantRole(identityPoolID string) *TrustPolicyDocument {
	tp := NewTrustPolicyDocument()
	tp.Statement = append(tp.Statement, PolicyStatement{
		Effect: "Allow",
		Principal: map[string]string{
			"Federated": "cognito-identity.amazonaws.com",
		},
		Action: []string{"sts:AssumeRoleWithWebIdentity"},
		Condition: map[string]map[string]string{
			"StringEquals": {
				"cognito-identity.amazonaws.com:aud": identityPoolID,
			},
		},
	})
	return tp
}

const DefaultTenantRolePath = "/tenant/"

func CreateTenantRole(
	ctx context.Context,
	i AWSIAM,
	tenantID uuid.UUID) (*types.Role, error) {

	// TODO: this is the wrong ID for this policy document
	tp := NewTrustPolicyDocumentForTenantRole(tenantID.String())
	roleName := fmt.Sprintf("tenant-%s", tenantID.String())
	tags := []types.Tag{
		{Key: aws.String("tenant-id"), Value: aws.String(tenantID.String())},
	}
	return CreateIAMRole(ctx, i, tp, DefaultTenantRolePath, roleName, tags)
}

func CreateIAMRole(
	ctx context.Context,
	i AWSIAM,
	tp *TrustPolicyDocument,
	path string,
	name string,
	tags []types.Tag) (*types.Role, error) {

	trustPolicy, err := tp.ToJSONString()
	if err != nil {
		return nil, err
	}
	createRoleInput := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(trustPolicy),
		RoleName:                 aws.String(name),
		Path:                     aws.String(path),
		Tags:                     tags,
	}
	result, err := i.CreateRole(ctx, createRoleInput)
	if err != nil {
		return nil, err
	}
	return result.Role, nil
}

func GetIAMRoles(ctx context.Context, i AWSIAM, prefix string) ([]types.Role, error) {
	input := &iam.ListRolesInput{
		MaxItems: aws.Int32(MaxPagingResults),
	}
	if prefix != "" {
		input.PathPrefix = aws.String(prefix)
	}
	result := make([]types.Role, 0)
	for {
		o, err := i.ListRoles(ctx, input)
		if err != nil {
			return nil, err
		}
		result = append(result, o.Roles...)
		if o.IsTruncated {
			input.Marker = o.Marker
		} else {
			break
		}
	}
	return result, nil
}

func DeleteIAMRoleByPrefix(ctx context.Context, i AWSIAM, prefix string) ([]string, error) {
	roles, err := GetIAMRoles(ctx, i, prefix)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0)
	for _, role := range roles {
		err := DeleteIAMRole(ctx, i, *role.RoleName)
		if err != nil {
			return nil, err
		}
		result = append(result, *role.RoleName)
	}
	return result, nil
}

func DeleteIAMRole(ctx context.Context, i AWSIAM, roleName string) error {
	input := &iam.DeleteRoleInput{
		RoleName: aws.String(roleName),
	}
	_, err := i.DeleteRole(ctx, input)
	if err != nil {
		return err
	}
	return nil
}
