package identity

import (
	"context"
	"strings"
)

func (s *MySQLStore) ListAccounts(ctx context.Context, query AccountQuery) (AccountPage, error) {
	where := make([]string, 0, 2)
	args := make([]any, 0, 8)
	if query.Search != "" {
		pattern := "%" + strings.ToLower(query.Search) + "%"
		where = append(where, `(username_key LIKE ? OR email_key LIKE ? OR phone_key LIKE ? OR LOWER(display_name) LIKE ?)`)
		args = append(args, pattern, pattern, pattern, pattern)
	}
	if query.Status != "" {
		where = append(where, `status = ?`)
		args = append(args, query.Status)
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	var total int64
	if err := s.conn.QueryRowCtx(ctx, &total, `SELECT COUNT(*) FROM identity_accounts`+whereSQL, args...); err != nil {
		return AccountPage{}, wrapIdentityError(ErrPersistence, err)
	}

	listArgs := append([]any(nil), args...)
	listArgs = append(listArgs, query.PageSize, (query.Page-1)*query.PageSize)
	var records []accountQueryRow
	if err := s.conn.QueryRowsCtx(ctx, &records, `
        SELECT account_id, username, email, phone, display_name, status, created_at, updated_at
        FROM identity_accounts`+whereSQL+`
        ORDER BY created_at DESC, account_id DESC
        LIMIT ? OFFSET ?`, listArgs...); err != nil {
        return AccountPage{}, wrapIdentityError(ErrPersistence, err)
    }
    items := make([]Account, 0, len(records))
    for _, row := range records { items = append(items, row.account()) }
	return AccountPage{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}
