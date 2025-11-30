package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// bcrypt cost factor (12 is a good balance of security and performance)
const bcryptCost = 12

// ErrAccountNotFound is returned when an account lookup fails.
var ErrAccountNotFound = errors.New("account not found")

// ErrAccountExists is returned when trying to create a duplicate account.
var ErrAccountExists = errors.New("account already exists")

// ErrInvalidCredentials is returned when login credentials are incorrect.
var ErrInvalidCredentials = errors.New("invalid username or password")

// ErrAccountBanned is returned when a banned account tries to login.
var ErrAccountBanned = errors.New("account is banned")

// Account represents a player account.
type Account struct {
	ID           int64
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	LastLogin    *time.Time
	LastIP       string
	Banned       bool
	IsAdmin      bool
}

// CreateAccount creates a new account with the given username and password.
// The password is hashed with bcrypt before storage.
func (d *Database) CreateAccount(username, password string) (*Account, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, errors.New("username cannot be empty")
	}
	if len(password) < 4 {
		return nil, errors.New("password must be at least 4 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	result, err := d.db.Exec(
		"INSERT INTO accounts (username, password_hash) VALUES (?, ?)",
		username, string(hash),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrAccountExists
		}
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get account ID: %w", err)
	}

	return &Account{
		ID:           id,
		Username:     username,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}, nil
}

// ValidateLogin checks if the username and password are correct.
// Returns the account if valid, or ErrInvalidCredentials if not.
// The ipAddress parameter is used to log the connection IP.
func (d *Database) ValidateLogin(username, password, ipAddress string) (*Account, error) {
	account, err := d.GetAccountByUsername(username)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Check if account is banned
	if account.Banned {
		return nil, ErrAccountBanned
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last login time and IP
	if err := d.UpdateLastLoginAndIP(account.ID, ipAddress); err != nil {
		// Log but don't fail the login
		fmt.Printf("warning: failed to update last login: %v\n", err)
	}

	return account, nil
}

// GetAccountByUsername retrieves an account by username (case-insensitive).
func (d *Database) GetAccountByUsername(username string) (*Account, error) {
	var account Account
	var lastLogin sql.NullTime
	var lastIP sql.NullString
	var banned int
	var isAdmin int

	err := d.db.QueryRow(
		"SELECT id, username, password_hash, created_at, last_login, last_ip, banned, is_admin FROM accounts WHERE username = ?",
		username,
	).Scan(&account.ID, &account.Username, &account.PasswordHash, &account.CreatedAt, &lastLogin, &lastIP, &banned, &isAdmin)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if lastLogin.Valid {
		account.LastLogin = &lastLogin.Time
	}
	if lastIP.Valid {
		account.LastIP = lastIP.String
	}
	account.Banned = banned != 0
	account.IsAdmin = isAdmin != 0

	return &account, nil
}

// UpdateLastLogin updates the last_login timestamp for an account.
func (d *Database) UpdateLastLogin(accountID int64) error {
	_, err := d.db.Exec(
		"UPDATE accounts SET last_login = CURRENT_TIMESTAMP WHERE id = ?",
		accountID,
	)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// UpdateLastLoginAndIP updates the last_login timestamp and IP address for an account.
func (d *Database) UpdateLastLoginAndIP(accountID int64, ipAddress string) error {
	_, err := d.db.Exec(
		"UPDATE accounts SET last_login = CURRENT_TIMESTAMP, last_ip = ? WHERE id = ?",
		ipAddress, accountID,
	)
	if err != nil {
		return fmt.Errorf("failed to update last login and IP: %w", err)
	}
	return nil
}

// BanAccount sets the banned flag to true for an account.
func (d *Database) BanAccount(accountID int64) error {
	_, err := d.db.Exec(
		"UPDATE accounts SET banned = 1 WHERE id = ?",
		accountID,
	)
	if err != nil {
		return fmt.Errorf("failed to ban account: %w", err)
	}
	return nil
}

// UnbanAccount sets the banned flag to false for an account.
func (d *Database) UnbanAccount(accountID int64) error {
	_, err := d.db.Exec(
		"UPDATE accounts SET banned = 0 WHERE id = ?",
		accountID,
	)
	if err != nil {
		return fmt.Errorf("failed to unban account: %w", err)
	}
	return nil
}

// IsAccountBanned checks if an account is banned.
func (d *Database) IsAccountBanned(accountID int64) (bool, error) {
	var banned int
	err := d.db.QueryRow("SELECT banned FROM accounts WHERE id = ?", accountID).Scan(&banned)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrAccountNotFound
		}
		return false, fmt.Errorf("failed to check ban status: %w", err)
	}
	return banned != 0, nil
}

// ChangePassword updates the password for an account after verifying the old password.
func (d *Database) ChangePassword(accountID int64, newPassword string) error {
	if len(newPassword) < 4 {
		return errors.New("password must be at least 4 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = d.db.Exec(
		"UPDATE accounts SET password_hash = ? WHERE id = ?",
		string(hash), accountID,
	)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

// ChangePasswordWithVerify updates the password after verifying the old password.
func (d *Database) ChangePasswordWithVerify(accountID int64, oldPassword, newPassword string) error {
	// Get current password hash
	var currentHash string
	err := d.db.QueryRow("SELECT password_hash FROM accounts WHERE id = ?", accountID).Scan(&currentHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAccountNotFound
		}
		return fmt.Errorf("failed to get account: %w", err)
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	// Change to new password
	return d.ChangePassword(accountID, newPassword)
}

// AccountExists checks if an account with the given username exists.
func (d *Database) AccountExists(username string) (bool, error) {
	var count int
	err := d.db.QueryRow(
		"SELECT COUNT(*) FROM accounts WHERE username = ?",
		username,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check account existence: %w", err)
	}
	return count > 0, nil
}

// SetAdmin sets or removes admin status for an account.
func (d *Database) SetAdmin(accountID int64, isAdmin bool) error {
	adminValue := 0
	if isAdmin {
		adminValue = 1
	}
	_, err := d.db.Exec(
		"UPDATE accounts SET is_admin = ? WHERE id = ?",
		adminValue, accountID,
	)
	if err != nil {
		return fmt.Errorf("failed to set admin status: %w", err)
	}
	return nil
}

// IsAdmin checks if an account has admin privileges.
func (d *Database) IsAdmin(accountID int64) (bool, error) {
	var isAdmin int
	err := d.db.QueryRow("SELECT is_admin FROM accounts WHERE id = ?", accountID).Scan(&isAdmin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrAccountNotFound
		}
		return false, fmt.Errorf("failed to check admin status: %w", err)
	}
	return isAdmin != 0, nil
}

// GetAllAdmins returns all admin accounts.
func (d *Database) GetAllAdmins() ([]*Account, error) {
	rows, err := d.db.Query(
		"SELECT id, username, password_hash, created_at, last_login, last_ip, banned, is_admin FROM accounts WHERE is_admin = 1",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query admins: %w", err)
	}
	defer rows.Close()

	var admins []*Account
	for rows.Next() {
		var account Account
		var lastLogin sql.NullTime
		var lastIP sql.NullString
		var banned int
		var isAdmin int

		if err := rows.Scan(&account.ID, &account.Username, &account.PasswordHash, &account.CreatedAt, &lastLogin, &lastIP, &banned, &isAdmin); err != nil {
			return nil, fmt.Errorf("failed to scan admin account: %w", err)
		}

		if lastLogin.Valid {
			account.LastLogin = &lastLogin.Time
		}
		if lastIP.Valid {
			account.LastIP = lastIP.String
		}
		account.Banned = banned != 0
		account.IsAdmin = isAdmin != 0

		admins = append(admins, &account)
	}

	return admins, nil
}

// GetAccountByID retrieves an account by ID.
func (d *Database) GetAccountByID(accountID int64) (*Account, error) {
	var account Account
	var lastLogin sql.NullTime
	var lastIP sql.NullString
	var banned int
	var isAdmin int

	err := d.db.QueryRow(
		"SELECT id, username, password_hash, created_at, last_login, last_ip, banned, is_admin FROM accounts WHERE id = ?",
		accountID,
	).Scan(&account.ID, &account.Username, &account.PasswordHash, &account.CreatedAt, &lastLogin, &lastIP, &banned, &isAdmin)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if lastLogin.Valid {
		account.LastLogin = &lastLogin.Time
	}
	if lastIP.Valid {
		account.LastIP = lastIP.String
	}
	account.Banned = banned != 0
	account.IsAdmin = isAdmin != 0

	return &account, nil
}

// GetTotalAccounts returns the total number of accounts.
func (d *Database) GetTotalAccounts() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM accounts").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count accounts: %w", err)
	}
	return count, nil
}

// GetTotalCharacters returns the total number of characters.
func (d *Database) GetTotalCharacters() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM characters").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count characters: %w", err)
	}
	return count, nil
}
