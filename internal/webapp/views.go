package webapp

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/jwx"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/google/uuid"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/appenv"
	"go.uber.org/zap"
)

const unknownViewValue = "unknown"

type UnauthenticatedView struct {
	Settings appenv.Settings
	// This username should not be trusted, and is only for usability -- e.g.,
	// when a user has hit certain errors, we can pre-populate the "email" field
	// for a login. Only use this as a hint.
	PotentialUsername string
	OTPImageLink      string
	ErrMsg            string
}

func NewUnauthenticatedView(settings appenv.Settings) UnauthenticatedView {
	return UnauthenticatedView{
		Settings: settings,
	}
}

func (uv UnauthenticatedView) WithPotentialUsername(username string) UnauthenticatedView {
	uv.PotentialUsername = username
	return uv
}

func (uv UnauthenticatedView) WithOTPImageLink(otpImageLink string) UnauthenticatedView {
	uv.OTPImageLink = otpImageLink
	return uv
}

func (uv UnauthenticatedView) WithErrMsg(errMsg string) UnauthenticatedView {
	uv.ErrMsg = errMsg
	return uv
}

type SubjectView struct {
	ctx context.Context
	app App
	sub *perms.Subject
}

func NewSubjectView(ctx context.Context, app App, sub *perms.Subject) SubjectView {
	return SubjectView{ctx: ctx, app: app, sub: sub}
}

func (sv SubjectView) Subject() *perms.Subject {
	return sv.sub
}

func (sv SubjectView) ID() string {
	if sv.sub == nil {
		return unknownViewValue
	}
	return sv.sub.SubID.String()
}

func (sv SubjectView) IsImpersonating() bool {
	return sv.sub != nil && sv.sub.IsImpersonating()
}

func (sv SubjectView) Username() string {
	if sv.sub == nil {
		return ""
	}
	return sv.sub.Username
}

func (sv SubjectView) TenantName() string {
	if sv.sub == nil {
		return unknownViewValue
	}
	subjectTenant, err := sv.sub.ResolveTenant(sv.ctx, sv.app)
	if err != nil {
		zap.L().Error("error resolving tenant",
			zap.String("tenantID", sv.sub.TenantID.String()),
			zap.Error(err))
		return unknownViewValue
	}
	return subjectTenant.Name
}

func (sv SubjectView) TenantLink() string {
	if sv.sub == nil {
		return "/"
	}
	return "/ll-auth/app/admin/tenants/" + sv.sub.TenantID.String()
}

func (sv SubjectView) AccessToken() string {
	if sv.sub == nil {
		return ""
	}
	rawDesc, err := jwx.Describe("Access Token", sv.sub.AccessToken)
	if err != nil {
		zap.L().Error("error describing token", zap.Error(err))
		return "invalid-token"
	}
	return rawDesc
}

// ImpersonationName returns a string suitable for displaying the impersonation
// status of the wrapped subject. A subject, if impersonating, can either impersonate
// a specific user or a group.
func (sv SubjectView) ImpersonationName() string {
	if !sv.IsImpersonating() {
		return ""
	}
	tenName := sv.sub.ImpersonateTenant.String()
	ten, err := db.GetTenantByID(sv.ctx, sv.app.DDB(), sv.sub.ImpersonateTenant)
	if err != nil {
		zap.L().Error("error resolving tenant",
			zap.String("tenantID", sv.sub.ImpersonateTenant.String()),
			zap.Error(err))
	} else {
		tenName = ten.Name
	}
	if sv.sub.ImpersonateSub != uuid.Nil {
		return fmt.Sprintf("%s | %s", sv.sub.ImpersonateEmail, tenName)
	}
	if sv.sub.ImpersonateGroup != "" {
		roleName := sv.sub.ImpersonateGroup
		role, err := authx.ParseRoleFromGroup(sv.sub.ImpersonateGroup)
		if err != nil {
			zap.L().Error("error parsing role from group",
				zap.String("group", sv.sub.ImpersonateGroup),
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
func (sv SubjectView) IsImpersonatingUser(user UserView) bool {
	return sv.IsImpersonating() && sv.sub.ImpersonateSub.String() == user.ID()
}

type ProfileTab struct {
	Link   string
	Active bool
	Title  string
}

type TenantView struct {
	ctx context.Context
	a   App
	ten *multi.Tenant
}

func NewTenantView(ctx context.Context, a App, ten *multi.Tenant) TenantView {
	return TenantView{ctx: ctx, a: a, ten: ten}
}

func (tv TenantView) Tenant() *multi.Tenant {
	return tv.ten
}

func (tv TenantView) Name() string {
	if tv.ten == nil {
		return unknownViewValue
	}
	return tv.ten.Name
}

func (tv TenantView) ID() string {
	if tv.ten == nil {
		return unknownViewValue
	}
	return tv.ten.ID.String()
}

func (tv TenantView) State() string {
	if tv.ten == nil {
		return unknownViewValue
	}
	return tv.ten.State.String()
}

func (tv TenantView) Description() string {
	if tv.ten == nil {
		return unknownViewValue
	}
	return tv.ten.Description
}

func (tv TenantView) UserPoolName() string {
	if tv.ten == nil {
		return unknownViewValue
	}
	return tv.ten.UserPoolName
}

func (tv TenantView) UserPoolID() string {
	if tv.ten == nil {
		return unknownViewValue
	}
	return tv.ten.UserPoolID
}

func (tv TenantView) UserPoolCognitoURL() string {
	if tv.ten == nil {
		return unknownViewValue
	}
	return authx.CognitoUserPoolURL(tv.a.Cfg().Region, tv.ten.UserPoolID)
}

func (tv TenantView) IsRoot() bool {
	return tv.ten != nil && tv.ten.IsRoot()
}

func (tv TenantView) IsActive() bool {
	return tv.ten != nil && tv.ten.State == multi.TenantStateActive
}

func (tv TenantView) Link() string {
	if tv.ten == nil {
		return ""
	}
	return "/ll-auth/app/admin/tenants/" + tv.ten.ID.String()
}

func (tv TenantView) Roles() RolesView {
	if tv.ten == nil {
		return RolesView{}
	}
	return NewRolesView(tv.ctx, tv.a, tv.ten)
}

func (tv TenantView) SubTenantRoles() RolesView {
	if tv.ten == nil {
		return RolesView{}
	}
	return NewSubRolesView(tv.ten)
}

func (tv TenantView) MFAPolicyStrings() []string {
	return multi.MFAPolicyStrings()
}

func (tv TenantView) UserProfileDefinition() *multi.ProfileDefinition {
	if tv.ten == nil {
		return &multi.ProfileDefinition{}
	}
	return tv.ten.GetUserProfileDefinition()
}

func (tv TenantView) CreatedAt() string {
	if tv.ten == nil {
		return unknownViewValue
	}
	return displayTimestamp(tv.ten.CreatedAt)
}

func (tv TenantView) Profile() ProfileFieldsView {
	if tv.ten == nil {
		return ProfileFieldsView{}
	}
	return NewProfileFieldsView(
		fmt.Sprintf("Tenant profile for '%s'", tv.ten.Name),
		fmt.Sprintf("/ll-auth/app/admin/tenants/%s", tv.ten.ID.String()),
		tv.ten.GetTenantProfileDefinition(),
		tv.ten.Profile())
}

type UserView struct {
	ctx     context.Context
	app     App
	sub     *perms.Subject
	ten     *multi.Tenant
	user    *multi.User
	cogUser *cognitoidentityprovider.AdminGetUserOutput
}

func NewUserView(
	ctx context.Context,
	app App,
	sub *perms.Subject,
	ten *multi.Tenant,
	user *multi.User) *UserView {

	return &UserView{
		ctx:  ctx,
		app:  app,
		sub:  sub,
		ten:  ten,
		user: user,
	}
}

func (uv *UserView) ID() string {
	if uv.user == nil {
		return ""
	}
	return uv.user.ID.String()
}

func (uv *UserView) Username() string {
	if uv.user == nil {
		return unknownViewValue
	}
	return uv.user.Username
}

func (uv *UserView) IsActive() bool {
	return uv.user != nil && uv.user.Status == multi.UserStatusActive
}

func (uv *UserView) IsInvited() bool {
	return uv.user != nil && uv.user.Status == multi.UserStatusInvited
}

func (uv *UserView) Link() string {
	if uv.user == nil {
		return unknownViewValue
	}
	return fmt.Sprintf("/ll-auth/app/admin/tenants/%s/users/%s",
		uv.ten.ID.String(), uv.user.ID.String())
}

func (uv *UserView) MFAPolicy() string {
	if uv.user == nil {
		return unknownViewValue
	}
	return uv.user.MFAPolicy.String()
}

func (uv *UserView) HasMFAPolicyNone() bool {
	return uv.user != nil && uv.user.MFAPolicy == multi.MFAPolicyNone
}

func (uv *UserView) MFAStatus() string {
	if uv.user == nil {
		return unknownViewValue
	}
	return uv.user.MFAStatus.String()
}

func (uv *UserView) IsMFAActive() bool {
	return uv.user != nil && uv.user.MFAStatus == multi.MFAStatusActive
}

func (uv *UserView) CogUser() *cognitoidentityprovider.AdminGetUserOutput {
	if uv.user == nil {
		return nil
	}
	if uv.cogUser != nil {
		return uv.cogUser
	}
	userPoolID := uv.ten.UserPoolID
	o, err := uv.app.CogIdp().AdminGetUser(
		uv.ctx, &cognitoidentityprovider.AdminGetUserInput{
			UserPoolId: aws.String(userPoolID),
			Username:   aws.String(uv.user.Username),
		})
	if err != nil {
		zap.L().Error("error getting CogIDP user",
			zap.String("userPoolID", userPoolID),
			zap.String("user", uv.user.Username),
			zap.Error(err))
		return nil
	}
	uv.cogUser = o
	return o
}

func (uv *UserView) CogURL() string {
	if uv.user == nil {
		return unknownViewValue
	}
	o := uv.CogUser()
	if o == nil {
		return unknownViewValue
	}
	userPoolID := uv.ten.UserPoolID
	return authx.CognitoUserDetailsURL(
		uv.app.Cfg().Region, userPoolID, aws.ToString(o.Username))
}

func (uv *UserView) Role() string {
	if uv.user == nil {
		return ""
	}
	return uv.user.Role
}

func (uv *UserView) Tenant() TenantView {
	if uv.ten == nil {
		return TenantView{}
	}
	return NewTenantView(uv.ctx, uv.app, uv.ten)
}

func (uv *UserView) IsImpersonating() bool {
	if uv.user == nil {
		return false
	}
	cu := uv.CogUser()
	if cu != nil {
		for _, attr := range cu.UserAttributes {
			if aws.ToString(attr.Name) == authx.CustomImpersonateTenantClaim {
				return true
			}
		}
	}
	return false
}

func (uv *UserView) ImpersonationName() string {
	if uv.user == nil {
		return ""
	}
	impersonateEmail := ""
	impersonateTenant := ""
	impersonateRole := ""
	cu := uv.CogUser()
	if cu != nil {
		for _, attr := range cu.UserAttributes {
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
					ten, err := db.GetTenantByID(uv.ctx, uv.app.DDB(), impersonateTenantUUID)
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
	}
	return fmt.Sprintf("%s | %s | %s", impersonateEmail, impersonateRole, impersonateTenant)
}

func (uv *UserView) CanSubjectImpersonate() bool {
	e, err := perms.Eval(
		uv.ctx,
		uv.app,
		uv.sub,
		perms.ActionWrite,
		perms.NewResource(
			perms.ResourceKindImpersonation,
			perms.ResourceNameAny,
			uv.ten,
			uv.user))
	if err != nil {
		zap.L().Error("error evaluating permissions",
			zap.Error(err),
			zap.Any("UserView", uv))
		return false
	}
	return e.IsAllowed()
}

func (uv *UserView) IsSubjectImpersonating() bool {
	if uv.sub.IsImpersonating() {
		return uv.IsImpersonating() && uv.sub.ImpersonateSub == uv.user.ID
	}
	return false
}

func (uv *UserView) IsTheSubject() bool {
	return uv.sub.SubID == uv.user.ID
}

func (uv *UserView) ImpersonateLink() string {
	return fmt.Sprintf("/ll-auth/app/admin/tenants/%s/users/%s/impersonate",
		uv.ten.ID.String(), uv.user.ID.String())
}

func (uv *UserView) ActivateLink() string {
	return fmt.Sprintf("/ll-auth/app/admin/tenants/%s/users/%s/activate",
		uv.ten.ID.String(), uv.user.ID.String())
}

func (uv *UserView) DeactivateLink() string {
	return fmt.Sprintf("/ll-auth/app/admin/tenants/%s/users/%s/deactivate",
		uv.ten.ID.String(), uv.user.ID.String())
}

func (uv *UserView) ReInviteLink() string {
	return fmt.Sprintf("/ll-auth/app/admin/tenants/%s/users/%s/reinvite",
		uv.ten.ID.String(), uv.user.ID.String())
}

func (uv *UserView) CancelInviteLink() string {
	return fmt.Sprintf("/ll-auth/app/admin/tenants/%s/users/%s/cancel-invite",
		uv.ten.ID.String(), uv.user.ID.String())
}

func (uv *UserView) MFAEnableLink() string {
	return fmt.Sprintf("/ll-auth/app/admin/tenants/%s/users/%s/mfa/enable",
		uv.ten.ID.String(), uv.user.ID.String())
}

func (uv *UserView) MFADisableLink() string {
	return fmt.Sprintf("/ll-auth/app/admin/tenants/%s/users/%s/mfa/disable",
		uv.ten.ID.String(), uv.user.ID.String())
}

func displayTimestamp(t string) string {
	parsedTime, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return t
	}
	return parsedTime.Format("2006-01-02 15:04:05") + " UTC"
}

func (uv *UserView) CreatedAt() string {
	if uv.user == nil {
		return ""
	}
	return displayTimestamp(uv.user.CreatedAt)
}

func (uv *UserView) Status() string {
	if uv.user == nil {
		return unknownViewValue
	}
	return uv.user.Status.String()
}

func (uv *UserView) Profile() ProfileFieldsView {
	if uv.user == nil {
		return ProfileFieldsView{}
	}
	return NewProfileFieldsView(
		fmt.Sprintf("User profile for '%s'", uv.user.Username),
		fmt.Sprintf("/ll-auth/app/admin/tenants/%s/users/%s",
			uv.ten.ID.String(), uv.user.ID.String()),
		uv.ten.GetUserProfileDefinition(),
		uv.user.Profile())
}

type RoleView struct {
	tenantID         uuid.UUID
	name             string
	count            int
	removableToolTip string
}

func NewRoleView(tenantID uuid.UUID, roleName string, memberCount int, removableToolTip string) RoleView {
	return RoleView{
		tenantID:         tenantID,
		name:             roleName,
		count:            memberCount,
		removableToolTip: removableToolTip}
}

func (rv RoleView) Name() string {
	return rv.name
}

func (rv RoleView) Count() string {
	return strconv.Itoa(rv.count)
}

func (rv RoleView) TenantID() string {
	return rv.tenantID.String()
}

func (rv RoleView) RemovableToolTip() string {
	return rv.removableToolTip
}

type RolesView struct {
	roles []RoleView
}

func NewRolesView(ctx context.Context, a App, ten *multi.Tenant) RolesView {
	users, err := db.GetUsersForTenant(ctx, a.DDB(), ten.ID)
	if err != nil {
		zap.L().Error("error getting users for tenant",
			zap.String("tenantID", ten.ID.String()),
			zap.Error(err))
		users = nil
	}
	roles := ten.Roles
	sort.Strings(roles)
	roleCounts := make(map[string]int)
	for _, u := range users {
		roleCounts[u.Role]++
	}
	result := RolesView{
		roles: make([]RoleView, 0, len(roles)),
	}
	for _, r := range roles {
		tooltip := ""
		if multi.IsBuiltinRole(r) {
			tooltip = "Cannot remove built-in role"
		} else if roleCounts[r] > 0 {
			tooltip = "Cannot remove role that is in use"
		}
		result.roles = append(result.roles,
			NewRoleView(ten.ID, r, roleCounts[r], tooltip))
	}
	sort.Slice(result.roles, func(i, j int) bool {
		return result.roles[i].Name() < result.roles[j].Name()
	})
	return result
}

func NewSubRolesView(ten *multi.Tenant) RolesView {
	roles := ten.SubTenantRoles
	result := RolesView{
		roles: make([]RoleView, 0, len(roles)),
	}
	for _, r := range roles {
		tooltip := ""
		if multi.IsBuiltinRole(r) {
			tooltip = "Cannot remove built-in role"
		}
		result.roles = append(result.roles, NewRoleView(ten.ID, r, 0, tooltip))
	}
	sort.Slice(result.roles, func(i, j int) bool {
		return result.roles[i].Name() < result.roles[j].Name()
	})
	return result
}

func (rsv RolesView) Views() []RoleView {
	return rsv.roles
}

func (rsv RolesView) Display() string {
	names := make([]string, 0, len(rsv.roles))
	for _, r := range rsv.roles {
		names = append(names, r.Name())
	}
	return strings.Join(names, ", ")
}

type FragmentView struct {
	ErrMsg string
}

func NewFragmentView() FragmentView {
	return FragmentView{}
}

func (fv FragmentView) WithErrMsg(s string) FragmentView {
	fv.ErrMsg = s
	return fv
}

type AuthenticatedSubjectView struct {
	ctx          context.Context
	a            App
	sub          SubjectView
	errMsg       string
	profileTab   profileTab
	tenTab       TenantTab
	userTab      UserTab
	tens         []TenantView
	selectedTen  TenantView
	users        []*UserView
	selectedUser *UserView
}

func NewAuthenticatedSubjectView(
	ctx context.Context,
	a App,
	sub *perms.Subject) AuthenticatedSubjectView {

	return AuthenticatedSubjectView{
		ctx:        ctx,
		a:          a,
		sub:        NewSubjectView(ctx, a, sub),
		errMsg:     "",
		profileTab: profileTabSummary,
	}
}

func (asv AuthenticatedSubjectView) WithErrMsg(msg string) AuthenticatedSubjectView {
	asv.errMsg = msg
	return asv
}

func (asv AuthenticatedSubjectView) WithProfileTab(tab profileTab) AuthenticatedSubjectView {
	asv.profileTab = tab
	return asv
}

func (asv AuthenticatedSubjectView) WithTenants(tens []*multi.Tenant) AuthenticatedSubjectView {
	asv.tens = make([]TenantView, 0, len(tens))
	for _, ten := range tens {
		tv := NewTenantView(asv.ctx, asv.a, ten)
		asv.tens = append(asv.tens, tv)
	}
	return asv
}

func (asv AuthenticatedSubjectView) WithSelectedTen(ten *multi.Tenant) AuthenticatedSubjectView {
	asv.selectedTen = NewTenantView(asv.ctx, asv.a, ten)
	return asv
}

func (asv AuthenticatedSubjectView) WithTenantTab(tab TenantTab) AuthenticatedSubjectView {
	asv.tenTab = tab
	return asv
}

func (asv AuthenticatedSubjectView) WithUserTab(tab UserTab) AuthenticatedSubjectView {
	asv.userTab = tab
	return asv
}

func (asv AuthenticatedSubjectView) WithUsers(users []*multi.User) AuthenticatedSubjectView {
	asv.users = make([]*UserView, 0, len(users))
	for _, user := range users {
		uv := NewUserView(
			asv.ctx,
			asv.a,
			asv.sub.Subject(),
			asv.selectedTen.Tenant(),
			user)
		asv.users = append(asv.users, uv)
	}
	return asv
}

func (asv AuthenticatedSubjectView) WithSelectedUser(user *multi.User) AuthenticatedSubjectView {
	asv.selectedUser = NewUserView(
		asv.ctx,
		asv.a,
		asv.sub.Subject(),
		asv.selectedTen.Tenant(),
		user)
	return asv
}

func (asv AuthenticatedSubjectView) SelectedUser() *UserView {
	return asv.selectedUser
}

func (asv AuthenticatedSubjectView) Settings() appenv.Settings {
	return asv.a.Settings()
}

func (asv AuthenticatedSubjectView) Subject() SubjectView {
	return asv.sub
}

func (asv AuthenticatedSubjectView) ErrMsg() string {
	return asv.errMsg
}

func (asv AuthenticatedSubjectView) ProfileTab() string {
	return string(asv.profileTab)
}

func titleFromSlug(slug string) string {
	return strings.TrimSpace(
		strings.ReplaceAll(
			cases.Title(language.English).String(slug), "-", " "))
}

func (asv AuthenticatedSubjectView) ProfileTabs() []ProfileTab {
	result := make([]ProfileTab, 0)
	for _, tab := range profileTabs {
		result = append(result, ProfileTab{
			Link:   fmt.Sprintf("/ll-auth/app/admin/profile/%s", tab),
			Active: tab == asv.profileTab,
			Title:  tab.title(),
		})
	}
	return result
}

func (asv AuthenticatedSubjectView) Tenants() []TenantView {
	return asv.tens
}

func (asv AuthenticatedSubjectView) Tenant() TenantView {
	return asv.selectedTen
}

func (asv AuthenticatedSubjectView) Users() []*UserView {
	return asv.users
}

func (asv AuthenticatedSubjectView) TenantTabs() []TenantTabView {
	result := make([]TenantTabView, 0, len(asv.tens))
	for _, tab := range tenantTabs {
		active := false
		if tab == asv.tenTab {
			active = true
		}
		result = append(result, TenantTabView{
			Name:   tab,
			Active: active,
			Link: fmt.Sprintf("/ll-auth/app/admin/tenants/%s/tabs/%s",
				asv.selectedTen.ID(), tab),
		})
	}
	return result
}

func (asv AuthenticatedSubjectView) TenantTab() TenantTab {
	return asv.tenTab
}

func (asv AuthenticatedSubjectView) UserTabs() []UserTabView {
	result := make([]UserTabView, 0, len(userTabs))
	for _, tab := range userTabs {
		active := false
		if tab == asv.userTab {
			active = true
		}
		result = append(result, UserTabView{
			Name:   tab,
			Active: active,
			Link: fmt.Sprintf("/ll-auth/app/admin/tenants/%s/users/%s/tabs/%s",
				asv.selectedTen.ID(), asv.selectedUser.ID(), tab),
		})
	}
	return result
}

func (asv AuthenticatedSubjectView) UserTab() UserTab {
	return asv.userTab
}

type TenantTabView struct {
	Name   TenantTab
	Active bool
	Link   string
}

func (ttv TenantTabView) Title() string {
	return cases.Title(language.English).String(string(ttv.Name))
}

type ErrorPageView struct {
	settings appenv.Settings
	errMsg   string
}

func NewErrorPageView(a App, errMsg string) ErrorPageView {
	return ErrorPageView{
		settings: a.Settings(),
		errMsg:   errMsg,
	}
}

func (epv ErrorPageView) Settings() appenv.Settings {
	return epv.settings
}

func (epv ErrorPageView) ErrMsg() string {
	return epv.errMsg
}

type ProfileFieldView struct {
	baseURL string
	def     multi.ProfileFieldDefinition
	value   *multi.ProfileField
}

func NewProfileFieldView(
	baseURL string,
	def multi.ProfileFieldDefinition,
	value *multi.ProfileField) ProfileFieldView {

	return ProfileFieldView{baseURL: baseURL, def: def, value: value}
}

func (pfv ProfileFieldView) Name() string {
	return pfv.def.Name
}

func (pfv ProfileFieldView) UIName() string {
	return pfv.def.UIName
}

func (pfv ProfileFieldView) HTMLInputType() string {
	return pfv.def.Format.HTMLInputType()
}

func (pfv ProfileFieldView) Value() string {
	if pfv.value == nil {
		return ""
	}
	return pfv.value.Value
}

func (pfv ProfileFieldView) EditLink() string {
	return fmt.Sprintf("%s/field/edit?fieldName=%s", pfv.baseURL, pfv.Name())
}

func (pfv ProfileFieldView) FormLink() string {
	return fmt.Sprintf("%s/field/form?fieldName=%s", pfv.baseURL, pfv.Name())
}

type ProfileFieldsView struct {
	Title  string
	Fields []ProfileFieldView
}

func NewProfileFieldsView(
	title string,
	baseURL string,
	profileDef *multi.ProfileDefinition,
	values *multi.Profile) ProfileFieldsView {

	result := ProfileFieldsView{Title: title}
	for _, field := range profileDef.Fields {
		if values == nil {
			result.Fields = append(result.Fields,
				NewProfileFieldView(baseURL, field, nil))
		} else {
			result.Fields = append(result.Fields,
				NewProfileFieldView(baseURL, field, values.GetField(field.Name)))
		}
	}
	return result
}

// SettingsProvider is an interface used to type check templates that take
// a view that is only required to provide Settings. For instance, the "header"
// template can be used in an unauthenticated state, but should still have
// access to the AppName stored in Settings.
type SettingsProvider interface {
	Settings() appenv.Settings
}

type UserTabView struct {
	Name   UserTab
	Active bool
	Link   string
}

func (utv UserTabView) Title() string {
	return cases.Title(language.English).String(string(utv.Name))
}
