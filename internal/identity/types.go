package identity

import "time"

type User struct {
	ID          string    `json:"id"`
	Issuer      string    `json:"issuer"`
	Subject     string    `json:"subject"`
	Email       string    `json:"email,omitempty"`
	DisplayName string    `json:"displayName,omitempty"`
	AvatarURL   string    `json:"avatarUrl,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Organization struct {
	ID         string    `json:"id"`
	LogtoOrgID string    `json:"logtoOrgId"`
	Slug       string    `json:"slug"`
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	SyncedAt   time.Time `json:"syncedAt"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Membership struct {
	OrganizationID string    `json:"organizationId"`
	UserID         string    `json:"userId"`
	Roles          []string  `json:"roles"`
	SyncedAt       time.Time `json:"syncedAt"`
}

type Claims struct {
	Issuer            string
	Subject           string
	ClientID          string
	Email             string
	EmailVerified     bool
	Name              string
	Picture           string
	Scope             string
	OrganizationID    string
	Organizations     []string
	OrganizationRoles []string
	Roles             []string
	ExpiresAt         time.Time
	NotBefore         time.Time
}

type Me struct {
	User               User               `json:"user"`
	ActiveOrganization *Organization      `json:"activeOrganization,omitempty"`
	Organizations      []OrganizationRole `json:"organizations"`
	Scopes             []string           `json:"scopes"`
	Personal           PersonalContext    `json:"personal"`
}

type OrganizationRole struct {
	Organization
	Role string `json:"role"`
}

type PersonalContext struct {
	DraftsEnabled bool `json:"draftsEnabled"`
}
