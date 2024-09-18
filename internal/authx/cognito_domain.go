package authx

import (
	"fmt"
	"strings"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"go.uber.org/zap"
)

func CognitoOauth2TokenEndpoint(up *types.UserPoolType) (string, error) {
	if up.Domain == nil {
		zap.L().Error("user pool is missing required domain configuration",
			zap.String("user_pool_id", aws.ToString(up.Id)))
		return "", apperr.NewInternal("user pool is missing required domain configuration")
	}
	splits := strings.Split(aws.ToString(up.Arn), ":")
	if len(splits) != 6 {
		zap.L().Error("unexpected user pool ARN format",
			zap.String("user_pool_id", aws.ToString(up.Id)),
			zap.String("arn", aws.ToString(up.Arn)))
		return "", apperr.NewInternal("unexpected user pool ARN format")
	}
	// The region is the fourth element of the ARN
	region := splits[3]
	return fmt.Sprintf("https://%s.auth.%s.amazoncognito.com/oauth2/token", aws.ToString(up.Domain), region), nil
}
