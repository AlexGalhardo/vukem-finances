package models

import (
	"context"
	"strconv"

	"github.com/AlexGalhardo/vukem/tenancy"
)

// This file is where the models meet the tenancy package: every domain
// model takes its workspace_id from the context tenant on insert, whatever
// the request sent, and fails without one (tenancy.ErrNoTenant).
// app/workspaces/tenant.go holds the request side.

func tenantID(id tenancy.ID) int64 {
	n, _ := strconv.ParseInt(string(id), 10, 64)
	return n
}

// SetTenant implements tenancy.Scoped.
func (m *Account) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant and, for a row made by the web UI, its uid.
func (m *Account) BeforeCreate(ctx context.Context) error {
	if err := CheckCents(m.OpeningCents); err != nil {
		return err
	}

	m.UID = stampUID(m.UID)

	return tenancy.Assign(ctx, m)
}

// SetTenant implements tenancy.Scoped.
func (m *Category) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *Category) BeforeCreate(ctx context.Context) error {
	m.UID = stampUID(m.UID)

	return tenancy.Assign(ctx, m)
}

// SetTenant implements tenancy.Scoped.
func (m *Transaction) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *Transaction) BeforeCreate(ctx context.Context) error {
	if err := CheckCents(m.AmountCents); err != nil {
		return err
	}

	if m.TagUIDs == "" {
		m.TagUIDs = "[]"
	}

	m.UID = stampUID(m.UID)

	return tenancy.Assign(ctx, m)
}

// SetTenant implements tenancy.Scoped.
func (m *Budget) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *Budget) BeforeCreate(ctx context.Context) error {
	if err := CheckCents(m.LimitCents); err != nil {
		return err
	}

	m.UID = stampUID(m.UID)

	return tenancy.Assign(ctx, m)
}

// SetTenant implements tenancy.Scoped.
func (m *Tag) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant and the uid.
func (m *Tag) BeforeCreate(ctx context.Context) error {
	m.UID = stampUID(m.UID)

	return tenancy.Assign(ctx, m)
}

// SetTenant implements tenancy.Scoped.
func (m *TransactionTag) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *TransactionTag) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }

// SetTenant implements tenancy.Scoped.
func (m *Import) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *Import) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }

// SetTenant implements tenancy.Scoped.
func (m *Invite) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *Invite) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }

// SetTenant implements tenancy.Scoped.
func (m *MonthlySummarySent) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *MonthlySummarySent) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }

// SetTenant implements tenancy.Scoped.
func (m *AuditEvent) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *AuditEvent) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }

// SetTenant implements tenancy.Scoped.
func (m *TransactionMonthTotal) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *TransactionMonthTotal) BeforeCreate(ctx context.Context) error {
	return tenancy.Assign(ctx, m)
}
