package webapp

import (
	"context"
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/perms"
	"github.com/gin-gonic/gin"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/db"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func handleAdminActivityReport(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	return deliverActivityReport(a, c, sub, targetTenant)
}

func handleAdminSummaryReport(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	return deliverSummaryReport(a, c, sub, targetTenant)
}

type SummaryReportRow struct {
	Username            string
	LastSuccessfulLogin time.Time
	LoginCountSuccess   int
	LastFailedLogin     time.Time
	LoginCountFail      int
	RecoveryAttempts    int
}

func (row SummaryReportRow) Header() []string {
	return []string{
		"Username",
		"Last Successful Login",
		"Login Count Success",
		"Last Failed Login",
		"Login Count Fail",
		"Recovery Attempts",
	}
}

func (row SummaryReportRow) ToRow() []string {
	return []string{
		row.Username,
		row.LastSuccessfulLogin.Format(time.RFC3339),
		strconv.Itoa(row.LoginCountSuccess),
		row.LastFailedLogin.Format(time.RFC3339),
		strconv.Itoa(row.LoginCountFail),
		strconv.Itoa(row.RecoveryAttempts),
	}
}

type ActivityReportRow struct {
	Username             string
	EventTime            time.Time
	EventID              string
	EventType            string
	EventResponse        string
	LoginLocationCity    string
	LoginLocationCountry string
	LoginIPAddress       string
}

func (row ActivityReportRow) Header() []string {
	return []string{
		"Username",
		"Event Time",
		"Event ID",
		"Event Type",
		"Event Response",
		"Login Location City",
		"Login Location Country",
		"Login IP Address",
	}
}

func (row ActivityReportRow) ToRow() []string {
	return []string{
		row.Username,
		row.EventTime.Format(time.RFC3339),
		row.EventID,
		row.EventType,
		row.EventResponse,
		row.LoginLocationCity,
		row.LoginLocationCountry,
		row.LoginIPAddress,
	}
}

const historyDays = 32

func deliverActivityReport(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	_ = sub // TODO: permission to read Cognito events?

	events, err := getCognitoEvents(c.Request.Context(), a, targetTenant.ID)
	if err != nil {
		return err
	}
	c.Writer.Header().Set("Content-Type", "text/csv")
	filename := fmt.Sprintf("cognito-activity-report-%s-%s.csv",
		events.tenantName, time.Now().UTC().Format("2006-01-02"))
	c.Writer.Header().Set("Content-Disposition", "attachment; filename="+filename)
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	if err := writer.Write(ActivityReportRow{}.Header()); err != nil {
		return err
	}
	rows := newActivityReport(events)
	for _, row := range rows {
		err := writer.Write(row.ToRow())
		if err != nil {
			zap.L().Error("failed to write row", zap.Error(err))
		}
	}
	writer.Flush()
	return nil
}

func deliverSummaryReport(
	a App, c *gin.Context, sub *perms.Subject, targetTenant *multi.Tenant) error {

	_ = sub // TODO: permission to read Cognito events?

	events, err := getCognitoEvents(c.Request.Context(), a, targetTenant.ID)
	if err != nil {
		return err
	}
	c.Writer.Header().Set("Content-Type", "text/csv")
	filename := fmt.Sprintf("cognito-summary-report-%s-%s.csv",
		events.tenantName, time.Now().UTC().Format("2006-01-02"))
	c.Writer.Header().Set("Content-Disposition", "attachment; filename="+filename)
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	if err := writer.Write(SummaryReportRow{}.Header()); err != nil {
		return err
	}
	rows := newSummaryReport(events)
	for _, row := range rows {
		err := writer.Write(row.ToRow())
		if err != nil {
			zap.L().Error("failed to write row", zap.Error(err))
		}
	}
	return nil
}

type cognitoEvents struct {
	tenantName string
	usernames  []string
	events     map[string][]types.AuthEventType
}

func getCognitoEvents(ctx context.Context, a App, tenantID uuid.UUID) (cognitoEvents, error) {
	ten, err := db.GetTenantByID(ctx, a.DDB(), tenantID)
	if err != nil {
		return cognitoEvents{},
			apperr.NewNotFoundE(
				fmt.Errorf("tenant not found: %s %w", tenantID, err))
	}
	users, err := db.GetUsersForTenant(ctx, a.DDB(), tenantID)
	if err != nil {
		return cognitoEvents{},
			apperr.NewInternalE(
				fmt.Errorf("failed to get users for tenant %s: %w", tenantID, err))
	}
	result := cognitoEvents{
		tenantName: ten.Name,
		usernames:  make([]string, 0, len(users)),
		events:     make(map[string][]types.AuthEventType),
	}
	var nextToken *string
	for _, user := range users {
		result.usernames = append(result.usernames, user.Username)
		for {
			o, err := a.CogIdp().AdminListUserAuthEvents(ctx, &cognitoidentityprovider.AdminListUserAuthEventsInput{
				UserPoolId: &ten.UserPoolID,
				Username:   aws.String(user.Username),
				NextToken:  nextToken,
			})
			if err != nil {
				zap.L().Error("failed to list user auth events",
					zap.String("username", user.Username),
					zap.Error(err))
				continue
			}
			for _, event := range o.AuthEvents {
				if time.Since(aws.ToTime(event.CreationDate)) > historyDays*24*time.Hour {
					break
				}
				result.events[user.Username] = append(result.events[user.Username], event)
			}
			if o.NextToken == nil {
				break
			}
			nextToken = o.NextToken
		}
	}
	sort.Slice(result.usernames, func(i, j int) bool {
		return result.usernames[i] < result.usernames[j]
	})
	return result, nil
}

func newActivityReport(events cognitoEvents) []ActivityReportRow {
	result := make([]ActivityReportRow, 0)
	for _, username := range events.usernames {
		for _, event := range events.events[username] {
			row := ActivityReportRow{
				Username:      username,
				EventTime:     aws.ToTime(event.CreationDate),
				EventID:       aws.ToString(event.EventId),
				EventType:     string(event.EventType),
				EventResponse: string(event.EventResponse),
			}
			if event.EventContextData != nil {
				row.LoginLocationCity = aws.ToString(event.EventContextData.City)
				row.LoginLocationCountry = aws.ToString(event.EventContextData.Country)
				row.LoginIPAddress = aws.ToString(event.EventContextData.IpAddress)
			}
			result = append(result, row)
		}
	}
	return result
}

func newSummaryReport(events cognitoEvents) []SummaryReportRow {
	result := make([]SummaryReportRow, 0)
	for _, username := range events.usernames {
		row := SummaryReportRow{
			Username: username,
		}
		for _, event := range events.events[username] {
			if event.EventType == types.EventTypeSignIn {
				if event.EventResponse == types.EventResponseTypePass {
					if row.LastSuccessfulLogin.Before(aws.ToTime(event.CreationDate)) {
						row.LastSuccessfulLogin = aws.ToTime(event.CreationDate)
					}
					row.LoginCountSuccess++
				} else if event.EventResponse == types.EventResponseTypeFail {
					if row.LastFailedLogin.Before(aws.ToTime(event.CreationDate)) {
						row.LastFailedLogin = aws.ToTime(event.CreationDate)
					}
					row.LoginCountFail++
				}
			} else if event.EventType == types.EventTypeForgotPassword ||
				event.EventType == types.EventTypeResendCode {
				row.RecoveryAttempts++
			}
		}
		result = append(result, row)
	}
	return result
}
