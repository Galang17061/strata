package domain

type User struct {
	Id       Guid   `db:"id" json:"id"`
	Fullname string `db:"fullname" json:"fullname"`
	UserName string `db:"username" json:"userName"`
	Email    string `db:"email" json:"email"`
	Password string `db:"password" json:"password"`
}

type Role struct {
	Id       Guid   `db:"id" json:"id"`
	RoleName string `db:"rolename" json:"roleName"`
}

type UserRole struct {
	UserId Guid `db:"userid" json:"userId"`
	RoleId Guid `db:"roleid" json:"roleId"`
}

type UserAccess struct {
	Id         Guid   `db:"id" json:"id"`
	UserId     Guid   `db:"userid" json:"userId"`
	Modul      string `db:"modul" json:"modul"`
	IsAdd      bool   `db:"is_add" json:"is_add"`
	IsEdit     bool   `db:"is_edit" json:"is_edit"`
	IsDelete   bool   `db:"is_delete" json:"is_delete"`
	IsView     bool   `db:"is_view" json:"is_view"`
	IsDownload bool   `db:"is_download" json:"is_download"`
}

type AccessData struct {
	Modul      string `json:"modul"`
	IsAdd      bool   `json:"is_add"`
	IsEdit     bool   `json:"is_edit"`
	IsDelete   bool   `json:"is_delete"`
	IsView     bool   `json:"is_view"`
	IsDownload bool   `json:"is_download"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Id         Guid         `json:"id"`
	UserName   string       `json:"userName"`
	FullName   string       `json:"fullName"`
	Email      string       `json:"email"`
	RoleName   string       `json:"roleName"`
	Token      string       `json:"token"`
	ValidUntil DateTime     `json:"validUntil"`
	AccessData []AccessData `json:"accessData"`
}

type UserData struct {
	Id       Guid    `json:"id"`
	UserName string  `json:"userName"`
	FullName string  `json:"fullName"`
	Email    string  `json:"email"`
	RoleId   Guid    `json:"roleId"`
	RoleName *string `json:"roleName"`
	Token    *string `json:"token"`
}

type UserCreate struct {
	Fullname string `json:"fullname"`
	UserName string `json:"userName"`
	Email    string `json:"email"`
	Password string `json:"password"`
	RoleId   Guid   `json:"roleId"`
}

type UserUpdate struct {
	Fullname string `json:"fullname"`
	UserName string `json:"userName"`
	RoleId   Guid   `json:"roleId"`
}

type PasswordUpdate struct {
	PasswordNew       string `json:"passwordNew"`
	ReconfirmPassword string `json:"reconfirmPassword"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token             string `json:"token"`
	PasswordNew       string `json:"passwordNew"`
	ReconfirmPassword string `json:"reconfirmPassword"`
}

type InviteCreate struct {
	Email  string `json:"email"`
	RoleId Guid   `json:"roleId"`
}

type InviteCreated struct {
	InviteUrl string   `json:"inviteUrl"`
	ExpiresAt DateTime `json:"expiresAt"`
	Delivered bool     `json:"delivered"`
}

type InviteView struct {
	Email    string `json:"email"`
	RoleName string `json:"roleName"`
}

type InviteAccept struct {
	Token             string `json:"token"`
	UserName          string `json:"userName"`
	Fullname          string `json:"fullname"`
	Password          string `json:"password"`
	ReconfirmPassword string `json:"reconfirmPassword"`
}

type RoleCreate struct {
	RoleName string `json:"roleName"`
}

type UserAccessCreate struct {
	UserId     Guid   `json:"userId"`
	Modul      string `json:"modul"`
	IsAdd      bool   `json:"is_add"`
	IsEdit     bool   `json:"is_edit"`
	IsDelete   bool   `json:"is_delete"`
	IsView     bool   `json:"is_view"`
	IsDownload bool   `json:"is_download"`
}

type UserAccessEdit struct {
	Modul      string `json:"modul"`
	IsAdd      bool   `json:"is_add"`
	IsEdit     bool   `json:"is_edit"`
	IsDelete   bool   `json:"is_delete"`
	IsView     bool   `json:"is_view"`
	IsDownload bool   `json:"is_download"`
}

type UserAccessData struct {
	Id         Guid   `db:"id" json:"id"`
	UserId     Guid   `db:"userid" json:"userId"`
	UserName   string `db:"username" json:"userName"`
	RoleName   string `db:"rolename" json:"roleName"`
	Modul      string `db:"modul" json:"modul"`
	IsAdd      bool   `db:"is_add" json:"is_add"`
	IsEdit     bool   `db:"is_edit" json:"is_edit"`
	IsDelete   bool   `db:"is_delete" json:"is_delete"`
	IsView     bool   `db:"is_view" json:"is_view"`
	IsDownload bool   `db:"is_download" json:"is_download"`
}

type UserAccessView struct {
	Id         Guid   `json:"id"`
	Modul      string `json:"modul"`
	IsAdd      bool   `json:"is_add"`
	IsEdit     bool   `json:"is_edit"`
	IsDelete   bool   `json:"is_delete"`
	IsView     bool   `json:"is_view"`
	IsDownload bool   `json:"is_download"`
}
