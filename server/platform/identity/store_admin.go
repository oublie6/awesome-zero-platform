package identity

import (
	"context"
	"database/sql"
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
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM identity_accounts`+whereSQL, args...).Scan(&total); err != nil {
		return AccountPage{}, wrapIdentityError(ErrPersistence, err)
	}

	listArgs := append([]any(nil), args...)
	listArgs = append(listArgs, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.db.QueryContext(ctx, `
		SELECT account_id, username, email, phone, display_name, status, created_at, updated_at
		FROM identity_accounts`+whereSQL+`
		ORDER BY created_at DESC, account_id DESC
		LIMIT ? OFFSET ?`, listArgs...)
	if err != nil {
		return AccountPage{}, wrapIdentityError(ErrPersistence, err)
	}
	defer rows.Close()

	items := make([]Account, 0, query.PageSize)
	for rows.Next() {
		var account Account
		var username, email, phone sql.NullString
		if err := rows.Scan(
			&account.ID,
			&username,
			&email,
			&phone,
			&account.DisplayName,
			&account.Status,
			&account.CreatedAt,
			&account.UpdatedAt,
		); err != nil {
			return AccountPage{}, wrapIdentityError(ErrPersistence, err)
		}
		account.Username = nullStringValue(username)
		account.Email = nullStringValue(email)
		account.Phone = nullStringValue(phone)
		items = append(items, account)
	}
	if err := rows.Err(); err != nil {
		return AccountPage{}, wrapIdentityError(ErrPersistence, err)
	}
	return AccountPage{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}
