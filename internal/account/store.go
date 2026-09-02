package account

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/Galang17061/strata-api/internal/domain"
)

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) FindLoginUser(ctx context.Context, username, encryptedPassword string) (*domain.User, error) {
	var user domain.User
	err := s.db.GetContext(ctx, &user, `SELECT TOP 1 Id, Fullname, UserName, Email, Password FROM dbo.Users WHERE UserName = @p1 AND Password = @p2 ORDER BY Id`, username, encryptedPassword)
	return optional(&user, err)
}

func (s *Store) FindUser(ctx context.Context, id domain.Guid) (*domain.User, error) {
	var user domain.User
	err := s.db.GetContext(ctx, &user, `SELECT Id, Fullname, UserName, Email, Password FROM dbo.Users WHERE Id = @p1`, id)
	return optional(&user, err)
}

func (s *Store) UserByName(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	err := s.db.GetContext(ctx, &user, `SELECT TOP 1 Id, Fullname, UserName, Email, Password FROM dbo.Users WHERE UserName = @p1 ORDER BY Id`, username)
	return optional(&user, err)
}

func (s *Store) UserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := s.db.GetContext(ctx, &user, `SELECT TOP 1 Id, Fullname, UserName, Email, Password FROM dbo.Users WHERE Email = @p1 ORDER BY Id`, email)
	return optional(&user, err)
}

type passwordReset struct {
	Id     string      `db:"password_reset_id"`
	UserId domain.Guid `db:"user_id"`
}

func (s *Store) InsertPasswordReset(ctx context.Context, id string, userId domain.Guid, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO dbo.PasswordReset (password_reset_id, user_id, token_hash, expires_at) VALUES (@p1, @p2, @p3, @p4)`, id, userId, tokenHash, expiresAt)
	return err
}

func (s *Store) ActivePasswordReset(ctx context.Context, tokenHash string) (*passwordReset, error) {
	var row passwordReset
	err := s.db.GetContext(ctx, &row, `SELECT TOP 1 password_reset_id, user_id FROM dbo.PasswordReset WHERE token_hash = @p1 AND used_at IS NULL AND expires_at > GETDATE() ORDER BY created_at DESC`, tokenHash)
	return optional(&row, err)
}

func (s *Store) MarkPasswordResetUsed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE dbo.PasswordReset SET used_at = GETDATE() WHERE password_reset_id = @p1`, id)
	return err
}

type userWithRole struct {
	Id       domain.Guid  `db:"Id"`
	Fullname string       `db:"Fullname"`
	UserName string       `db:"UserName"`
	Email    string       `db:"Email"`
	RoleId   *domain.Guid `db:"RoleId"`
	RoleName *string      `db:"RoleName"`
}

func (s *Store) ListUsersWithRoles(ctx context.Context) ([]userWithRole, error) {
	rows := []userWithRole{}
	err := s.db.SelectContext(ctx, &rows, `SELECT u.Id, u.Fullname, u.UserName, u.Email, ur.RoleId, r.RoleName FROM dbo.Users u LEFT JOIN dbo.UserRole ur ON ur.UserId = u.Id LEFT JOIN dbo.Roles r ON r.Id = ur.RoleId ORDER BY u.Id`)
	return rows, err
}

func (s *Store) FindUserWithRole(ctx context.Context, id domain.Guid) (*userWithRole, error) {
	var row userWithRole
	err := s.db.GetContext(ctx, &row, `SELECT u.Id, u.Fullname, u.UserName, u.Email, ur.RoleId, r.RoleName FROM dbo.Users u LEFT JOIN dbo.UserRole ur ON ur.UserId = u.Id LEFT JOIN dbo.Roles r ON r.Id = ur.RoleId WHERE u.Id = @p1`, id)
	return optional(&row, err)
}

func (s *Store) InsertUser(ctx context.Context, user domain.User) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO dbo.Users (Id, Fullname, UserName, Email, Password) VALUES (@p1, @p2, @p3, @p4, @p5)`, user.Id, user.Fullname, user.UserName, user.Email, user.Password)
	return err
}

func (s *Store) UpdateUserNames(ctx context.Context, id domain.Guid, fullname, username string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE dbo.Users SET Fullname = @p2, UserName = @p3 WHERE Id = @p1`, id, fullname, username)
	return err
}

func (s *Store) UpdateUserPassword(ctx context.Context, id domain.Guid, password string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE dbo.Users SET Password = @p2 WHERE Id = @p1`, id, password)
	return err
}

func (s *Store) DeleteUser(ctx context.Context, id domain.Guid) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dbo.Users WHERE Id = @p1`, id)
	return err
}

func (s *Store) FindUserRole(ctx context.Context, userId domain.Guid) (*domain.UserRole, error) {
	var role domain.UserRole
	err := s.db.GetContext(ctx, &role, `SELECT UserId, RoleId FROM dbo.UserRole WHERE UserId = @p1`, userId)
	return optional(&role, err)
}

func (s *Store) InsertUserRole(ctx context.Context, userId, roleId domain.Guid) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO dbo.UserRole (UserId, RoleId) VALUES (@p1, @p2)`, userId, roleId)
	return err
}

func (s *Store) UpdateUserRole(ctx context.Context, userId, roleId domain.Guid) error {
	_, err := s.db.ExecContext(ctx, `UPDATE dbo.UserRole SET RoleId = @p2 WHERE UserId = @p1`, userId, roleId)
	return err
}

func (s *Store) DeleteUserRole(ctx context.Context, userId domain.Guid) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dbo.UserRole WHERE UserId = @p1`, userId)
	return err
}

func (s *Store) RoleInUse(ctx context.Context, roleId domain.Guid) (bool, error) {
	var count int
	err := s.db.GetContext(ctx, &count, `SELECT COUNT(1) FROM dbo.UserRole WHERE RoleId = @p1`, roleId)
	return count > 0, err
}

func (s *Store) ListRoles(ctx context.Context) ([]domain.Role, error) {
	roles := []domain.Role{}
	err := s.db.SelectContext(ctx, &roles, `SELECT Id, RoleName FROM dbo.Roles ORDER BY Id`)
	return roles, err
}

func (s *Store) FindRole(ctx context.Context, id domain.Guid) (*domain.Role, error) {
	var role domain.Role
	err := s.db.GetContext(ctx, &role, `SELECT Id, RoleName FROM dbo.Roles WHERE Id = @p1`, id)
	return optional(&role, err)
}

func (s *Store) RoleByName(ctx context.Context, name string) (*domain.Role, error) {
	var role domain.Role
	err := s.db.GetContext(ctx, &role, `SELECT TOP 1 Id, RoleName FROM dbo.Roles WHERE LOWER(RoleName) = @p1 ORDER BY Id`, strings.ToLower(name))
	return optional(&role, err)
}

func (s *Store) InsertRole(ctx context.Context, role domain.Role) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO dbo.Roles (Id, RoleName) VALUES (@p1, @p2)`, role.Id, role.RoleName)
	return err
}

func (s *Store) UpdateRoleName(ctx context.Context, id domain.Guid, name string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE dbo.Roles SET RoleName = @p2 WHERE Id = @p1`, id, name)
	return err
}

func (s *Store) DeleteRole(ctx context.Context, id domain.Guid) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dbo.Roles WHERE Id = @p1`, id)
	return err
}

func (s *Store) ListAccessData(ctx context.Context) ([]domain.UserAccessData, error) {
	rows := []domain.UserAccessData{}
	err := s.db.SelectContext(ctx, &rows, `SELECT ua.Id, ua.UserId, u.UserName, r.RoleName, ua.Modul, ua.is_add, ua.is_edit, ua.is_delete, ua.is_view, ua.is_download FROM dbo.UserAccess ua INNER JOIN dbo.Users u ON u.Id = ua.UserId INNER JOIN dbo.UserRole ur ON ur.UserId = u.Id INNER JOIN dbo.Roles r ON r.Id = ur.RoleId ORDER BY ua.Id`)
	return rows, err
}

func (s *Store) AccessOfUser(ctx context.Context, userId domain.Guid) ([]domain.UserAccess, error) {
	rows := []domain.UserAccess{}
	err := s.db.SelectContext(ctx, &rows, `SELECT Id, UserId, Modul, is_add, is_edit, is_delete, is_view, is_download FROM dbo.UserAccess WHERE UserId = @p1 ORDER BY Id`, userId)
	return rows, err
}

func (s *Store) AccessOfUsers(ctx context.Context, userIds []domain.Guid) ([]domain.UserAccess, error) {
	rows := []domain.UserAccess{}
	for _, userId := range userIds {
		part, err := s.AccessOfUser(ctx, userId)
		if err != nil {
			return nil, err
		}
		rows = append(rows, part...)
	}
	return rows, nil
}

func (s *Store) FindAccess(ctx context.Context, id domain.Guid) (*domain.UserAccess, error) {
	var access domain.UserAccess
	err := s.db.GetContext(ctx, &access, `SELECT Id, UserId, Modul, is_add, is_edit, is_delete, is_view, is_download FROM dbo.UserAccess WHERE Id = @p1`, id)
	return optional(&access, err)
}

func (s *Store) AccessByUserAndModul(ctx context.Context, userId domain.Guid, modul string) (*domain.UserAccess, error) {
	var access domain.UserAccess
	err := s.db.GetContext(ctx, &access, `SELECT TOP 1 Id, UserId, Modul, is_add, is_edit, is_delete, is_view, is_download FROM dbo.UserAccess WHERE UserId = @p1 AND Modul = @p2 ORDER BY Id`, userId, modul)
	return optional(&access, err)
}

func (s *Store) InsertAccess(ctx context.Context, access domain.UserAccess) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO dbo.UserAccess (Id, UserId, Modul, is_add, is_edit, is_delete, is_view, is_download) VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8)`, access.Id, access.UserId, access.Modul, access.IsAdd, access.IsEdit, access.IsDelete, access.IsView, access.IsDownload)
	return err
}

func (s *Store) UpdateAccess(ctx context.Context, access domain.UserAccess) error {
	_, err := s.db.ExecContext(ctx, `UPDATE dbo.UserAccess SET Modul = @p2, is_add = @p3, is_edit = @p4, is_delete = @p5, is_view = @p6, is_download = @p7 WHERE Id = @p1`, access.Id, access.Modul, access.IsAdd, access.IsEdit, access.IsDelete, access.IsView, access.IsDownload)
	return err
}

func (s *Store) DeleteAccess(ctx context.Context, id domain.Guid) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dbo.UserAccess WHERE Id = @p1`, id)
	return err
}

func optional[T any](value *T, err error) (*T, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return value, nil
}
