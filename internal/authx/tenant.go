package authx

type Tenant struct {
	ID           TenantID `json:"id"`
	ShortName    string   `json:"short_name"`
	Name         string   `json:"name"`
	EmailDomains []string `json:"email_domains"`
}

func NewTenant() Tenant {
	return Tenant{
		ID: MustNewTenantID(),
	}
}
