package account

import (
	"context"
	"strings"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/mail"
)

type Service struct {
	store  *Store
	cipher auth.Cipher
	tokens auth.TokenIssuer
	mailer *mail.Sender
	webURL string
}

func NewService(store *Store, cipher auth.Cipher, tokens auth.TokenIssuer, mailer *mail.Sender, webURL string) *Service {
	return &Service{store: store, cipher: cipher, tokens: tokens, mailer: mailer, webURL: webURL}
}

func (s *Service) Login(ctx context.Context, request domain.LoginRequest) (domain.LoginResponse, error) {
	user, err := s.store.FindLoginUser(ctx, request.Username, s.cipher.Encrypt(request.Password))
	if err != nil {
		return domain.LoginResponse{}, err
	}
	if user == nil {
		return domain.LoginResponse{}, domain.InvalidOperation("Incorrect username or password")
	}
	userRole, err := s.store.FindUserRole(ctx, user.Id)
	if err != nil {
		return domain.LoginResponse{}, err
	}
	var role *domain.Role
	if userRole != nil {
		if role, err = s.store.FindRole(ctx, userRole.RoleId); err != nil {
			return domain.LoginResponse{}, err
		}
	}
	if role == nil {
		return domain.LoginResponse{}, domain.InvalidOperation("User role not found")
	}
	token, expires, err := s.tokens.Issue(*user, userRole.RoleId, role.RoleName)
	if err != nil {
		return domain.LoginResponse{}, err
	}
	accesses, err := s.store.AccessOfUser(ctx, user.Id)
	if err != nil {
		return domain.LoginResponse{}, err
	}
	accessData := make([]domain.AccessData, 0, len(accesses))
	for _, access := range accesses {
		accessData = append(accessData, domain.AccessData{Modul: access.Modul, IsAdd: access.IsAdd, IsEdit: access.IsEdit, IsDelete: access.IsDelete, IsView: access.IsView, IsDownload: access.IsDownload})
	}
	return domain.LoginResponse{
		Id:         user.Id,
		UserName:   user.UserName,
		FullName:   user.Fullname,
		Email:      user.Email,
		RoleName:   role.RoleName,
		Token:      token,
		ValidUntil: expires,
		AccessData: accessData,
	}, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]domain.UserData, error) {
	rows, err := s.store.ListUsersWithRoles(ctx)
	if err != nil {
		return nil, err
	}
	users := make([]domain.UserData, 0, len(rows))
	for _, row := range rows {
		roleId := domain.EmptyGuid
		if row.RoleId != nil {
			roleId = *row.RoleId
		}
		users = append(users, domain.UserData{Id: row.Id, UserName: row.UserName, FullName: row.Fullname, Email: row.Email, RoleId: roleId, RoleName: row.RoleName})
	}
	return users, nil
}

func (s *Service) UserData(ctx context.Context, id domain.Guid) (domain.UserData, error) {
	row, err := s.store.FindUserWithRole(ctx, id)
	if err != nil {
		return domain.UserData{}, err
	}
	if row == nil {
		return domain.UserData{}, domain.InvalidOperation("Incorrect username or password")
	}
	if row.RoleName == nil {
		return domain.UserData{}, domain.InvalidOperation("User role not found")
	}
	roleId := domain.EmptyGuid
	if row.RoleId != nil {
		roleId = *row.RoleId
	}
	return domain.UserData{Id: row.Id, UserName: row.UserName, FullName: row.Fullname, Email: row.Email, RoleId: roleId, RoleName: row.RoleName}, nil
}

func (s *Service) AddUser(ctx context.Context, request domain.UserCreate) error {
	existingName, err := s.store.UserByName(ctx, request.UserName)
	if err != nil {
		return err
	}
	if existingName != nil {
		return domain.InvalidOperation("Username '" + request.UserName + "' already exists.")
	}
	existingEmail, err := s.store.UserByEmail(ctx, request.Email)
	if err != nil {
		return err
	}
	if existingEmail != nil {
		return domain.InvalidOperation("Email '" + request.Email + "' already exists.")
	}
	role, err := s.store.FindRole(ctx, request.RoleId)
	if err != nil {
		return err
	}
	if role == nil {
		return domain.InvalidOperation("Role Id :'" + request.RoleId.String() + "' not found.")
	}
	user := domain.User{Id: domain.NewGuid(), Fullname: request.Fullname, UserName: request.UserName, Email: request.Email, Password: s.cipher.Encrypt(request.Password)}
	if err := s.store.InsertUser(ctx, user); err != nil {
		return err
	}
	return s.store.InsertUserRole(ctx, user.Id, role.Id)
}

func (s *Service) UpdateUser(ctx context.Context, id domain.Guid, request domain.UserUpdate) error {
	user, err := s.store.FindUser(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return domain.InvalidOperation("Data null")
	}
	sameName, err := s.store.UserByName(ctx, request.UserName)
	if err != nil {
		return err
	}
	role, err := s.store.FindRole(ctx, request.RoleId)
	if err != nil {
		return err
	}
	userRole, err := s.store.FindUserRole(ctx, id)
	if err != nil {
		return err
	}
	if role == nil {
		return domain.InvalidOperation("Role Id : '" + request.RoleId.String() + "' Not Found.")
	}
	if userRole == nil {
		return domain.InvalidOperation("Object reference not set to an instance of an object.")
	}
	if err := s.store.UpdateUserRole(ctx, id, role.Id); err != nil {
		return err
	}
	if sameName != nil && sameName.Id != user.Id {
		return domain.InvalidOperation("Username with the name '" + request.UserName + "' already exists.")
	}
	return s.store.UpdateUserNames(ctx, id, request.Fullname, request.UserName)
}

func (s *Service) UpdatePassword(ctx context.Context, id domain.Guid, request domain.PasswordUpdate) error {
	user, err := s.store.FindUser(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return domain.InvalidOperation("Data null")
	}
	return s.storeConfirmedPassword(ctx, id, request)
}

func (s *Service) ResetPassword(ctx context.Context, id domain.Guid, request domain.PasswordUpdate) error {
	user, err := s.store.FindUser(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return domain.KeyNotFound("Data not found")
	}
	return s.storeConfirmedPassword(ctx, id, request)
}

func (s *Service) storeConfirmedPassword(ctx context.Context, id domain.Guid, request domain.PasswordUpdate) error {
	encrypted := s.cipher.Encrypt(request.PasswordNew)
	if encrypted != s.cipher.Encrypt(request.ReconfirmPassword) {
		return domain.InvalidOperation("new password is not the same as the confirmed password")
	}
	return s.store.UpdateUserPassword(ctx, id, encrypted)
}

func (s *Service) DeleteUser(ctx context.Context, id domain.Guid) error {
	if err := s.store.DeleteUserRole(ctx, id); err != nil {
		return err
	}
	return s.store.DeleteUser(ctx, id)
}

func (s *Service) ListRoles(ctx context.Context) ([]domain.Role, error) {
	return s.store.ListRoles(ctx)
}

func (s *Service) FindRole(ctx context.Context, id domain.Guid) (*domain.Role, error) {
	return s.store.FindRole(ctx, id)
}

func (s *Service) AddRole(ctx context.Context, request domain.RoleCreate) error {
	existing, err := s.store.RoleByName(ctx, request.RoleName)
	if err != nil {
		return err
	}
	if existing != nil {
		return domain.InvalidOperation("Role with the name '" + request.RoleName + "' already exists.")
	}
	return s.store.InsertRole(ctx, domain.Role{Id: domain.NewGuid(), RoleName: strings.ToLower(request.RoleName)})
}

func (s *Service) UpdateRole(ctx context.Context, id domain.Guid, request domain.RoleCreate) error {
	role, err := s.store.FindRole(ctx, id)
	if err != nil {
		return err
	}
	if role == nil {
		return domain.InvalidOperation("Data null")
	}
	return s.store.UpdateRoleName(ctx, id, strings.ToLower(request.RoleName))
}

func (s *Service) DeleteRole(ctx context.Context, id domain.Guid) error {
	role, err := s.store.FindRole(ctx, id)
	if err != nil {
		return err
	}
	inUse, err := s.store.RoleInUse(ctx, id)
	if err != nil {
		return err
	}
	if inUse {
		name := ""
		if role != nil {
			name = role.RoleName
		}
		return domain.InvalidOperation("can't be deleted, please delete the user with role name :'" + name + "'")
	}
	if role == nil {
		return nil
	}
	return s.store.DeleteRole(ctx, id)
}

func (s *Service) ListAccessData(ctx context.Context) ([]domain.UserAccessData, error) {
	return s.store.ListAccessData(ctx)
}

func (s *Service) AccessOfUser(ctx context.Context, userId domain.Guid) ([]domain.UserAccessView, error) {
	rows, err := s.store.AccessOfUser(ctx, userId)
	if err != nil {
		return nil, err
	}
	views := make([]domain.UserAccessView, 0, len(rows))
	for _, row := range rows {
		views = append(views, domain.UserAccessView{Id: row.Id, Modul: row.Modul, IsAdd: row.IsAdd, IsEdit: row.IsEdit, IsDelete: row.IsDelete, IsView: row.IsView, IsDownload: row.IsDownload})
	}
	return views, nil
}

func (s *Service) AddAccess(ctx context.Context, request domain.UserAccessCreate) error {
	existing, err := s.store.AccessByUserAndModul(ctx, request.UserId, request.Modul)
	if err != nil {
		return err
	}
	if existing != nil {
		return domain.InvalidOperation("Modul Access already exists.")
	}
	return s.store.InsertAccess(ctx, domain.UserAccess{Id: domain.NewGuid(), UserId: request.UserId, Modul: request.Modul, IsAdd: request.IsAdd, IsEdit: request.IsEdit, IsDelete: request.IsDelete, IsView: request.IsView, IsDownload: request.IsDownload})
}

func (s *Service) CreateOrUpdateMany(ctx context.Context, requests []domain.UserAccessCreate) error {
	if len(requests) == 0 {
		return domain.InvalidOperation("The request list is empty.")
	}
	seen := map[domain.Guid]bool{}
	userIds := []domain.Guid{}
	for _, request := range requests {
		if !seen[request.UserId] {
			seen[request.UserId] = true
			userIds = append(userIds, request.UserId)
		}
	}
	existing, err := s.store.AccessOfUsers(ctx, userIds)
	if err != nil {
		return err
	}
	for _, request := range requests {
		var match *domain.UserAccess
		for index := range existing {
			if existing[index].UserId == request.UserId && existing[index].Modul == request.Modul {
				match = &existing[index]
				break
			}
		}
		if match != nil {
			match.IsAdd, match.IsEdit, match.IsDelete, match.IsView, match.IsDownload = request.IsAdd, request.IsEdit, request.IsDelete, request.IsView, request.IsDownload
			if err := s.store.UpdateAccess(ctx, *match); err != nil {
				return err
			}
			continue
		}
		created := domain.UserAccess{Id: domain.NewGuid(), UserId: request.UserId, Modul: request.Modul, IsAdd: request.IsAdd, IsEdit: request.IsEdit, IsDelete: request.IsDelete, IsView: request.IsView, IsDownload: request.IsDownload}
		if err := s.store.InsertAccess(ctx, created); err != nil {
			return err
		}
		existing = append(existing, created)
	}
	return nil
}

func (s *Service) UpdateAccess(ctx context.Context, id domain.Guid, request domain.UserAccessEdit) error {
	access, err := s.store.FindAccess(ctx, id)
	if err != nil {
		return err
	}
	if access == nil {
		return domain.InvalidOperation("Data null")
	}
	existing, err := s.store.AccessByUserAndModul(ctx, access.UserId, request.Modul)
	if err != nil {
		return err
	}
	target := access
	if existing != nil {
		target = existing
	}
	target.Modul = request.Modul
	target.IsAdd, target.IsEdit, target.IsDelete, target.IsView, target.IsDownload = request.IsAdd, request.IsEdit, request.IsDelete, request.IsView, request.IsDownload
	return s.store.UpdateAccess(ctx, *target)
}

func (s *Service) DeleteAccess(ctx context.Context, id domain.Guid) error {
	access, err := s.store.FindAccess(ctx, id)
	if err != nil {
		return err
	}
	if access == nil {
		return nil
	}
	return s.store.DeleteAccess(ctx, id)
}
