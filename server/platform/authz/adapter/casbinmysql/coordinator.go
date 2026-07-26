package casbinmysql

import (
	"context"
	"fmt"
	"time"

	"github.com/oublie6/awesome-zero-platform/server/foundation/database"
	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
)

// Coordinator is the deployed mutation boundary for a clustered Engine. It
// embeds all read and Enforce behavior unchanged while serializing every policy
// write with account bootstrap/status operations that protect the same super
// administrator invariant.
type Coordinator struct {
	*Engine
}

func NewCoordinator(engine *Engine) (*Coordinator, error) {
	if engine == nil || engine.db == nil {
		return nil, fmt.Errorf("casbin engine is required")
	}
	return &Coordinator{Engine: engine}, nil
}

func (c *Coordinator) AddRoleForUser(ctx context.Context, accountID, role string) (bool, error) {
	return c.withBoolMutation(ctx, func(lockCtx context.Context) (bool, error) {
		return c.Engine.AddRoleForUser(lockCtx, accountID, role)
	})
}

func (c *Coordinator) DeleteRoleForUser(ctx context.Context, accountID, role string) (bool, error) {
	return c.withBoolMutation(ctx, func(lockCtx context.Context) (bool, error) {
		return c.Engine.DeleteRoleForUser(lockCtx, accountID, role)
	})
}

func (c *Coordinator) AddPermissionForRole(ctx context.Context, role, resource, action string) (bool, error) {
	return c.withBoolMutation(ctx, func(lockCtx context.Context) (bool, error) {
		return c.Engine.AddPermissionForRole(lockCtx, role, resource, action)
	})
}

func (c *Coordinator) DeletePermissionForRole(ctx context.Context, role, resource, action string) (bool, error) {
	return c.withBoolMutation(ctx, func(lockCtx context.Context) (bool, error) {
		return c.Engine.DeletePermissionForRole(lockCtx, role, resource, action)
	})
}

func (c *Coordinator) DeleteUser(ctx context.Context, accountID string) (bool, error) {
	return c.withBoolMutation(ctx, func(lockCtx context.Context) (bool, error) {
		return c.Engine.DeleteUser(lockCtx, accountID)
	})
}

func (c *Coordinator) DeleteRole(ctx context.Context, role string) (bool, error) {
	return c.withBoolMutation(ctx, func(lockCtx context.Context) (bool, error) {
		return c.Engine.DeleteRole(lockCtx, role)
	})
}

func (c *Coordinator) ReplaceRawRules(ctx context.Context, rules []authz.RawRule) error {
	return c.withMutation(ctx, func(lockCtx context.Context) error {
		return c.Engine.ReplaceRawRules(lockCtx, rules)
	})
}

func (c *Coordinator) ReplacePermissionsForRole(ctx context.Context, role string, permissions []authz.Permission) error {
	return c.withMutation(ctx, func(lockCtx context.Context) error {
		return c.Engine.ReplacePermissionsForRole(lockCtx, role, permissions)
	})
}

func (c *Coordinator) ReplaceRolesForUser(ctx context.Context, accountID string, roles []string) error {
	return c.withMutation(ctx, func(lockCtx context.Context) error {
		return c.Engine.ReplaceRolesForUser(lockCtx, accountID, roles)
	})
}

func (c *Coordinator) withBoolMutation(ctx context.Context, fn func(context.Context) (bool, error)) (bool, error) {
	var changed bool
	err := c.withMutation(ctx, func(lockCtx context.Context) error {
		var err error
		changed, err = fn(lockCtx)
		return err
	})
	return changed, err
}

func (c *Coordinator) withMutation(ctx context.Context, fn func(context.Context) error) error {
	return database.WithAdvisoryLock(
		ctx,
		c.Engine.db,
		authz.SuperAdminMutationLock,
		10*time.Second,
		fn,
	)
}

var (
	_ authz.Authorizer    = (*Coordinator)(nil)
	_ authz.PolicyManager = (*Coordinator)(nil)
	_ authz.Administrator = (*Coordinator)(nil)
)
