package authz

import (
	"context"
	"fmt"
	"strings"
)

type Authorizer interface {
	Enforce(context.Context, string, string, string) (bool, error)
}

type PolicyManager interface {
	AddRoleForUser(context.Context, string, string) (bool, error)
	DeleteRoleForUser(context.Context, string, string) (bool, error)
	AddPermissionForRole(context.Context, string, string, string) (bool, error)
	DeletePermissionForRole(context.Context, string, string, string) (bool, error)
	DeleteUser(context.Context, string) (bool, error)
	DeleteRole(context.Context, string) (bool, error)
}

type Service struct {
	authorizer Authorizer
	policies   PolicyManager
}

func NewService(authorizer Authorizer, policies PolicyManager) (*Service, error) {
	if authorizer == nil {
		return nil, fmt.Errorf("authorizer is required")
	}
	if policies == nil {
		return nil, fmt.Errorf("policy manager is required")
	}
	return &Service{authorizer: authorizer, policies: policies}, nil
}

func (s *Service) Enforce(ctx context.Context, subject, resource, action string) (bool, error) {
	subject, resource, action, err := normalizePermission(subject, resource, action)
	if err != nil {
		return false, err
	}
	return s.authorizer.Enforce(ctx, subject, resource, action)
}

func (s *Service) AssignRole(ctx context.Context, accountID, role string) (bool, error) {
	accountID, role, err := normalizePair("account id", accountID, "role", role)
	if err != nil {
		return false, err
	}
	return s.policies.AddRoleForUser(ctx, accountID, role)
}

func (s *Service) RevokeRole(ctx context.Context, accountID, role string) (bool, error) {
	accountID, role, err := normalizePair("account id", accountID, "role", role)
	if err != nil {
		return false, err
	}
	return s.policies.DeleteRoleForUser(ctx, accountID, role)
}

func (s *Service) GrantPermission(ctx context.Context, role, resource, action string) (bool, error) {
	role, resource, action, err := normalizePermission(role, resource, action)
	if err != nil {
		return false, err
	}
	return s.policies.AddPermissionForRole(ctx, role, resource, action)
}

func (s *Service) RevokePermission(ctx context.Context, role, resource, action string) (bool, error) {
	role, resource, action, err := normalizePermission(role, resource, action)
	if err != nil {
		return false, err
	}
	return s.policies.DeletePermissionForRole(ctx, role, resource, action)
}

func (s *Service) RemoveAccountPolicies(ctx context.Context, accountID string) (bool, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return false, fmt.Errorf("account id must not be empty")
	}
	return s.policies.DeleteUser(ctx, accountID)
}

func (s *Service) RemoveRole(ctx context.Context, role string) (bool, error) {
	role = strings.TrimSpace(role)
	if role == "" {
		return false, fmt.Errorf("role must not be empty")
	}
	return s.policies.DeleteRole(ctx, role)
}

func normalizePermission(subject, resource, action string) (string, string, string, error) {
	subject = strings.TrimSpace(subject)
	resource = strings.TrimSpace(resource)
	action = strings.TrimSpace(action)
	if subject == "" {
		return "", "", "", fmt.Errorf("subject must not be empty")
	}
	if resource == "" {
		return "", "", "", fmt.Errorf("resource must not be empty")
	}
	if action == "" {
		return "", "", "", fmt.Errorf("action must not be empty")
	}
	return subject, resource, action, nil
}

func normalizePair(leftName, left, rightName, right string) (string, string, error) {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" {
		return "", "", fmt.Errorf("%s must not be empty", leftName)
	}
	if right == "" {
		return "", "", fmt.Errorf("%s must not be empty", rightName)
	}
	return left, right, nil
}
