package multi

import "github.com/google/uuid"

type UserStatus string

const (
	UserStatusInactive UserStatus = "inactive"
	UserStatusInvited  UserStatus = "invited"
	UserStatusActive   UserStatus = "active"
)

type User struct {
	Username string     `dynamodbav:"username"`
	Role     string     `dynamodbav:"role"`
	Status   UserStatus `dynamodbav:"status"`
	TenantID uuid.UUID  `dynamodbav:"tenant"`
}
