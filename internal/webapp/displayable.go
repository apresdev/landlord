package webapp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"

	"github.com/golang-jwt/jwt/v5"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/db"

	"go.uber.org/zap"

	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/multi"
)

func displayTimestamp(t string) string {
	parsedTime, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return t
	}
	return parsedTime.Format("2006-01-02 15:04:05") + " UTC"
}

// DisplayableUser wraps a multi.User and provides convenience methods for
// displaying a user in a web page.
type DisplayableUser struct {
	User                  *multi.User
	subjectCanImpersonate bool
}

// NewDisplayableUser creates a new DisplayableUser.
func NewDisplayableUser(
	ctx context.Context,
	app App,
	sub *perms.Subject,
	ten *multi.Tenant,
	user *multi.User) (*DisplayableUser, error) {

	e, err := perms.Eval(
		ctx,
		app,
		sub,
		perms.ActionWrite,
		perms.NewResource(perms.ResourceKindImpersonation, perms.ResourceNameAny, ten, user))
	if err != nil {
		return nil, err
	}

	return &DisplayableUser{User: user, subjectCanImpersonate: e.IsAllowed()}, nil
}

// NewDisplayableUsers creates a slice of DisplayableUser instances from a
// slice of multi.Users.
func NewDisplayableUsers(
	ctx context.Context,
	app App,
	sub *perms.Subject,
	ten *multi.Tenant,
	users []*multi.User) ([]*DisplayableUser, error) {

	dus := make([]*DisplayableUser, len(users))
	var err error
	for i, user := range users {
		dus[i], err = NewDisplayableUser(ctx, app, sub, ten, user)
		if err != nil {
			return nil, err
		}
	}
	return dus, nil
}

// MFAStatus returns a string representation of the MFA status of the wrapped
// user.
func (du *DisplayableUser) MFAStatus() string {
	switch du.User.MFAStatus {
	case multi.MFAStatusNone:
		return "No MFA required"
	case multi.MFAStatusPending:
		return "Pending MFA setup"
	case multi.MFAStatusActive:
		return "Active"
	default:
		return "Unknown"
	}
}

// Status returns a string representation of status of the wrapped user
func (du *DisplayableUser) Status() string {
	switch du.User.Status {
	case multi.UserStatusInvited:
		return "Invited"
	case multi.UserStatusInactive:
		return "Inactive"
	case multi.UserStatusActive:
		return "Active"
	default:
		return "Unknown"
	}
}

// SubjectCanImpersonate returns true if the current subject can impersonate
// the wrapped user.
func (du *DisplayableUser) SubjectCanImpersonate() bool {
	return du.subjectCanImpersonate
}

type DisplayableCognitoUser struct {
	ctx    context.Context
	app    App
	o      *cognitoidentityprovider.AdminGetUserOutput
	cogURL string
}

func NewDisplayableCognitoUser(
	ctx context.Context,
	app App,
	userPoolID string,
	username string) (*DisplayableCognitoUser, error) {

	o, err := app.CogIdp().AdminGetUser(ctx, &cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: aws.String(userPoolID),
		Username:   aws.String(username),
	})
	if err != nil {
		return nil, err
	}
	cogURL := authx.CognitoUserDetailsURL(
		app.Cfg().Region, userPoolID, aws.ToString(o.Username))

	return &DisplayableCognitoUser{ctx: ctx, app: app, o: o, cogURL: cogURL}, nil
}

func (dcu *DisplayableCognitoUser) IsImpersonating() bool {
	for _, attr := range dcu.o.UserAttributes {
		if aws.ToString(attr.Name) == authx.CustomImpersonateTenantClaim {
			return true
		}
	}
	return false
}

func (dcu *DisplayableCognitoUser) ImpersonationName() string {
	impersonateEmail := ""
	impersonateTenant := ""
	impersonateRole := ""
	for _, attr := range dcu.o.UserAttributes {
		if aws.ToString(attr.Name) == authx.CustomImpersonateGroupClaim {
			role, err := authx.ParseRoleFromGroup(aws.ToString(attr.Value))
			if err != nil {
				zap.L().Error("error parsing role from group",
					zap.String("group", aws.ToString(attr.Value)),
					zap.Error(err))
				return "error"
			}
			impersonateRole = role.Name
		}
		if aws.ToString(attr.Name) == authx.CustomImpersonateEmailClaim {
			impersonateEmail = aws.ToString(attr.Value)
		}
		if aws.ToString(attr.Name) == authx.CustomImpersonateTenantClaim {
			impersonateTenantID := aws.ToString(attr.Value)
			impersonateTenantUUID, err := uuid.Parse(impersonateTenantID)
			if err != nil {
				zap.L().Error("error parsing tenant ID",
					zap.String("tenantID", impersonateTenantID),
					zap.Error(err))
				impersonateTenant = "error"
			} else {
				ten, err := db.GetTenantByID(dcu.ctx, dcu.app.DDB(), impersonateTenantUUID)
				if err != nil {
					zap.L().Error("error resolving tenant",
						zap.String("tenantID", impersonateTenantID),
						zap.Error(err))
					impersonateTenant = "error"
				} else {
					impersonateTenant = ten.Name
				}
			}
		}
	}
	return fmt.Sprintf("%s | %s | %s", impersonateEmail, impersonateRole, impersonateTenant)
}

// DisplayableSubject wraps a perms.Subject and provides convenience methods
// for displaying a subject in a web page.
type DisplayableSubject struct {
	ctx context.Context
	app App
	sub *perms.Subject
}

// NewDisplayableSubject creates a new DisplayableSubject.
func NewDisplayableSubject(
	ctx context.Context, app App, sub *perms.Subject) *DisplayableSubject {
	return &DisplayableSubject{ctx: ctx, app: app, sub: sub}
}

// Username returns the username of the wrapped subject or the empty string
func (ds *DisplayableSubject) Username() string {
	if ds.sub == nil {
		return ""
	}
	return ds.sub.Username
}

func (ds *DisplayableSubject) CognitoSubID() uuid.UUID {
	if ds.sub == nil {
		return uuid.Nil
	}
	return ds.sub.SubID
}

func (ds *DisplayableSubject) RoleName() string {
	if ds.sub == nil && ds.sub.Role != nil {
		return "unknown"
	}
	return ds.sub.Role.Name
}

// TenantID returns the tenant ID of the wrapped subject or uuid.Nil
func (ds *DisplayableSubject) TenantID() uuid.UUID {
	if ds.sub == nil {
		return uuid.Nil
	}
	return ds.sub.TenantID
}

// TenantName returns the name of the tenant of the wrapped subject or "unknown"
func (ds *DisplayableSubject) TenantName() string {
	if ds.sub == nil {
		return "unknown"
	}
	subjectTenant, err := ds.sub.ResolveTenant(ds.ctx, ds.app)
	if err != nil {
		zap.L().Error("error resolving tenant",
			zap.String("tenantID", ds.sub.TenantID.String()),
			zap.Error(err))
		return "unknown"
	}
	return subjectTenant.Name
}

// IsImpersonating returns true if the wrapped subject is impersonating a user
// or a group.
func (ds *DisplayableSubject) IsImpersonating() bool {
	return ds.sub != nil && ds.sub.IsImpersonating()
}

// ImpersonationName returns a string suitable for displaying the impersonation
// status of the wrapped subject. A subject, if impersonating, can either impersonate
// a specific user or a group.
func (ds *DisplayableSubject) ImpersonationName() string {
	if !ds.IsImpersonating() {
		return ""
	}
	tenName := ds.sub.ImpersonateTenant.String()
	ten, err := db.GetTenantByID(ds.ctx, ds.app.DDB(), ds.sub.ImpersonateTenant)
	if err != nil {
		zap.L().Error("error resolving tenant",
			zap.String("tenantID", ds.sub.ImpersonateTenant.String()),
			zap.Error(err))
	} else {
		tenName = ten.Name
	}
	if ds.sub.ImpersonateSub != uuid.Nil {
		return fmt.Sprintf("%s | %s", ds.sub.ImpersonateEmail, tenName)
	}
	if ds.sub.ImpersonateGroup != "" {
		roleName := ds.sub.ImpersonateGroup
		role, err := authx.ParseRoleFromGroup(ds.sub.ImpersonateGroup)
		if err != nil {
			zap.L().Error("error parsing role from group",
				zap.String("group", ds.sub.ImpersonateGroup),
				zap.Error(err))
		} else {
			roleName = role.Name
		}
		return fmt.Sprintf("%s | %s", roleName, tenName)
	}
	return ""
}

// IsImpersonatingUser returns true if the wrapped subject is impersonating
// the given user.
func (ds *DisplayableSubject) IsImpersonatingUser(du *DisplayableUser) bool {
	return ds.IsImpersonating() && ds.sub.ImpersonateSub == du.User.ID
}

// IsUser returns true if the wrapped subject is the given user.
func (ds *DisplayableSubject) IsUser(du *DisplayableUser) bool {
	return ds.sub != nil && ds.sub.SubID == du.User.ID
}

// CreatedAt returns the creation timestamp of the wrapped subject or "unknown"
func (ds *DisplayableSubject) CreatedAt() string {
	if ds.sub == nil {
		return "unknown"
	}
	user, err := ds.sub.ResolveUser(ds.ctx, ds.app)
	if err != nil {
		zap.L().Error("error resolving user",
			zap.String("username", ds.sub.Username),
			zap.Error(err))
		return "unknown"
	}
	return user.CreatedAt
}

// State returns the status of the wrapped subject or "unknown"
func (ds *DisplayableSubject) State() string {
	if ds.sub == nil {
		return "unknown"
	}
	user, err := ds.sub.ResolveUser(ds.ctx, ds.app)
	if err != nil {
		zap.L().Error("error resolving user",
			zap.String("username", ds.sub.Username),
			zap.Error(err))
		return "unknown"
	}
	return user.Status.String()
}

type DisplayableJWT struct {
	TokenName string
	JWT       *jwt.Token
}

func NewDisplayableJWT(name string, token *jwt.Token) *DisplayableJWT {
	return &DisplayableJWT{TokenName: name, JWT: token}
}

func (dj *DisplayableJWT) RawDescription() string {
	rawDesc, err := jwx.Describe(dj.TokenName, dj.JWT)
	if err != nil {
		zap.L().Error("error describing token", zap.Error(err))
		return "invalid-token"
	}
	return rawDesc
}

func titleDisplay(s string) string {
	s = cases.Title(language.English).String(s)
	s = strings.ReplaceAll(s, "_", " ")
	return strings.ReplaceAll(s, "-", " ")
}
