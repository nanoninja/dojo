// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/platform/database"
	"github.com/nanoninja/dojo/internal/platform/security"
)

// userRow is used to scan rows that contain encrypted fields.
// Encrypted columns are stored as VARCHAR in the database and must be
// scanned as *string before being decrypted into their real Go types.
type userRow struct {
	model.User
	BirthDate    *string `db:"birth_date"`
	AddressLine1 *string `db:"address_line1"`
	AddressLine2 *string `db:"address_line2"`
	VATNumber    *string `db:"vat_number"`
	LastLoginIP  *string `db:"last_login_ip"`
}

// UserFilter holds the criteria used to query the user list.
type UserFilter struct {
	Status    string
	Search    string // matches email, first_name, last_name
	SortOrder string // "asc" or "desc", defaults to "desc"
	Limit     int
	Offset    int
}

// UserStore defines the data access contract for user accounts.
type UserStore interface {
	// List returns a paginated list of non-deleted users matching the filter.
	List(ctx context.Context, f UserFilter) ([]model.User, int, error)

	// FindByID returns a user by ID, or nil if not found or deleted.
	FindByID(ctx context.Context, id string) (*model.User, error)

	// FindByEmail returns a user with credentials for authentication purposes.
	FindByEmail(ctx context.Context, email string) (*model.User, error)

	FindCredentialsByID(ctx context.Context, id string) (*model.User, error)

	// Create inserts a new user and populates u.ID with the generated UUID.
	Create(ctx context.Context, u *model.User) error

	// Update updates the profile fields of an existing user.
	Update(ctx context.Context, u *model.User) error

	// UpdatePassword replaces the hashed password for the given user.
	UpdatePassword(ctx context.Context, id, passwordHash string) error

	// UpdateLastLogin records the login timestamp, IP and increments login count.
	UpdateLastLogin(ctx context.Context, id, ip string) error

	// UpdateVerified marks a user account as verified.
	UpdateVerified(ctx context.Context, id string) error

	// Delete soft-deletes a user by setting status and deleted_at.
	Delete(ctx context.Context, id string) error

	// IncrementFailedLogin increments the failed login counter for the given user.
	IncrementFailedLogin(ctx context.Context, id string) error

	// LockAccount sets locked_until to prevent login until the given time.
	LockAccount(ctx context.Context, id string, until time.Time) error

	// ResetFailedLogin clears the failed login counter and any active lock.
	ResetFailedLogin(ctx context.Context, id string) error
}

type userStore struct {
	db     database.Querier
	cipher *security.Cipher
}

// NewUserStore creates a new UserStore.
func NewUserStore(db database.Querier, cipher *security.Cipher) UserStore {
	return &userStore{
		db:     db,
		cipher: cipher,
	}
}

func (s *userStore) List(ctx context.Context, f UserFilter) ([]model.User, int, error) {
	args := make([]any, 0, 6)
	where := `WHERE deleted_at IS NULL`

	if f.Status != "" {
		where += ` AND status = ?`
		args = append(args, f.Status)
	}
	if f.Search != "" {
		where += ` AND (email ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?)`
		param := "%" + f.Search + "%"
		args = append(args, param, param, param)
	}

	// COUNT uses the same filters but without ORDER BY / LIMIT / OFFSET
	var total int
	countQuery := s.db.Rebind(`SELECT COUNT(*) FROM users ` + where)
	if err := s.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	order := "DESC"
	if f.SortOrder == "asc" {
		order = "ASC"
	}

	query := `
		SELECT
			id, email, status, role, is_verified,
			first_name, last_name, company_name, headline, bio, avatar_url, website,
			address_line1, address_line2, city, postal_code, latitude, longitude,
			vat_number, country_code, language, timezone, birth_date,
			last_login_at, last_login_ip, login_count,
			created_at, updated_at, banned_at, deleted_at
		FROM users ` + where + ` ORDER BY created_at ` + order

	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	query += ` LIMIT ?`
	args = append(args, f.Limit)

	if f.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, f.Offset)
	}

	fullQuery := s.db.Rebind(query)
	rows, err := s.db.QueryxContext(ctx, fullQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close() //nolint:errcheck

	users := make([]model.User, 0, limit)
	for rows.Next() {
		var row userRow
		if err := rows.StructScan(&row); err != nil {
			return nil, 0, err
		}
		u, err := s.decryptUser(row)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	return users, total, rows.Err()
}

func (s *userStore) FindByID(ctx context.Context, id string) (*model.User, error) {
	var row userRow
	err := s.db.GetContext(ctx, &row, `
		SELECT
			id, email, status, role, is_verified,
			first_name, last_name, company_name, headline, bio, avatar_url, website,
			address_line1, address_line2, city, postal_code, latitude, longitude,
			vat_number, country_code, language, timezone, birth_date,
			last_login_at, last_login_ip, login_count,
			failed_login_attempts, locked_until,
			created_at, updated_at, banned_at, deleted_at
		FROM users
		WHERE id = $1
		  AND deleted_at IS NULL`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u, err := s.decryptUser(row)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *userStore) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	err := s.db.GetContext(ctx, &u, `
		SELECT
			id,
			email,
			password_hash,
			status,
			role,
			is_verified,
			failed_login_attempts,
			locked_until
		FROM users
		WHERE email = $1
		  AND deleted_at IS NULL`, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}

func (s *userStore) FindCredentialsByID(ctx context.Context, id string) (*model.User, error) {
	var u model.User
	err := s.db.GetContext(ctx, &u, `
		SELECT
			id,
			email,
			password_hash,
			status,
			role,
			is_verified,
			failed_login_attempts,
			locked_until
		FROM users
		WHERE id = $1
			AND deleted_at IS NULL`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}

func (s *userStore) Create(ctx context.Context, u *model.User) error {
	return s.create(ctx, s.db, u)
}

func (s *userStore) Update(ctx context.Context, u *model.User) error {
	addrLine1, err := encrypt(s.cipher, u.AddressLine1)
	if err != nil {
		return err
	}
	addrLine2, err := encrypt(s.cipher, u.AddressLine2)
	if err != nil {
		return err
	}
	vatNumber, err := encrypt(s.cipher, u.VATNumber)
	if err != nil {
		return err
	}
	birthDate, err := encryptTime(s.cipher, u.BirthDate)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE users
			SET first_name    = $1,
				last_name     = $2,
				company_name  = $3,
				headline      = $4,
				bio           = $5,
				avatar_url    = $6,
				website       = $7,
				address_line1 = $8,
				address_line2 = $9,
				city          = $10,
				postal_code   = $11,
				country_code  = $12,
				vat_number    = $13,
				language      = $14,
				timezone      = $15,
				birth_date    = $16
			WHERE id = $17
			  AND deleted_at IS NULL`,
		u.FirstName,
		u.LastName,
		u.CompanyName,
		u.Headline,
		u.Bio,
		u.AvatarURL,
		u.Website,
		addrLine1,
		addrLine2,
		u.City,
		u.PostalCode,
		u.CountryCode,
		vatNumber,
		u.Language,
		u.Timezone,
		birthDate,
		u.ID,
	)
	return err
}

func (s *userStore) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users
			SET
				password_hash = $1
		WHERE id = $2
		  AND deleted_at IS NULL`,
		passwordHash,
		id,
	)
	return err
}

func (s *userStore) UpdateLastLogin(ctx context.Context, id, ip string) error {
	encIP, err := encrypt(s.cipher, &ip)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE users
			SET
				last_login_at = NOW(),
				last_login_ip = $1,
				login_count   = login_count + 1
		WHERE id = $2
		  AND deleted_at IS NULL`,
		encIP,
		id,
	)
	return err
}

func (s *userStore) UpdateVerified(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users
			SET is_verified = true
		WHERE id = $1
		  AND deleted_at IS NULL`,
		id,
	)
	return err
}

func (s *userStore) Delete(ctx context.Context, id string) error {
	anonymisedEmail := "deleted_" + id + "@deleted"

	_, err := s.db.ExecContext(ctx, `
		UPDATE users
			SET
				status        = $1,
				deleted_at    = NOW(),
				email         = $2,
				first_name    = '',
				last_name     = '',
				password_hash = '',
				address_line1 = NULL,
				address_line2 = NULL,
				vat_number    = NULL,
				birth_date    = NULL,
				last_login_ip = NULL
			WHERE id = $3
			  AND deleted_at IS NULL`,
		model.UserStatusDeleted,
		anonymisedEmail,
		id,
	)
	return err
}

func (s *userStore) create(ctx context.Context, q database.Querier, u *model.User) error {
	return q.QueryRowContext(ctx, `
		INSERT INTO users (
			email,
			password_hash,
			first_name,
			last_name,
			status,
			is_verified,
			is_2fa_enabled,
			role,
			language,
			timezone
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		) RETURNING id`,
		u.Email,
		u.PasswordHash,
		u.FirstName,
		u.LastName,
		u.Status,
		u.IsVerified,
		u.Is2FAEnabled,
		u.Role.String(),
		u.Language,
		u.Timezone,
	).Scan(&u.ID)
}

func (s *userStore) IncrementFailedLogin(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users
		   SET failed_login_attempts = failed_login_attempts + 1
		WHERE id = $1
		  AND deleted_at IS NULL
	`, id)
	return err
}

func (s *userStore) LockAccount(ctx context.Context, id string, until time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users
		   SET locked_until = $1
		WHERE id = $2
		  AND deleted_at IS NULL
	`, until, id)
	return err
}

func (s *userStore) ResetFailedLogin(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users
		   SET failed_login_attempts = 0,
		       locked_until          = NULL
		WHERE id = $1
		  AND deleted_at IS NULL
	`, id)
	return err
}

// decryptUser decrypts all encrypted fields from a userRow into a model.User.
func (s *userStore) decryptUser(row userRow) (model.User, error) {
	u := row.User

	birthDate, err := decryptTime(s.cipher, row.BirthDate)
	if err != nil {
		return model.User{}, err
	}
	u.BirthDate = birthDate

	addrLine1, err := decrypt(s.cipher, row.AddressLine1)
	if err != nil {
		return model.User{}, err
	}
	u.AddressLine1 = addrLine1

	addrLine2, err := decrypt(s.cipher, row.AddressLine2)
	if err != nil {
		return model.User{}, err
	}
	u.AddressLine2 = addrLine2

	vatNumber, err := decrypt(s.cipher, row.VATNumber)
	if err != nil {
		return model.User{}, err
	}
	u.VATNumber = vatNumber

	lastLoginIP, err := decrypt(s.cipher, row.LastLoginIP)
	if err != nil {
		return model.User{}, err
	}
	u.LastLoginIP = lastLoginIP

	return u, nil
}
