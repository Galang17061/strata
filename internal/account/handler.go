package account

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/web"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Mount(router chi.Router) {
	router.Post("/api/Auth/Login", h.login)
	router.Post("/api/Auth/logout", h.logout)
	router.Get("/api/User/me", h.currentUser)
	router.Get("/api/User", h.listUsers)
	router.Get("/api/User/DetailUser", h.userDetail)
	router.Post("/api/User", h.createUser)
	router.Put("/api/User/ChangePassword", h.changePassword)
	router.Put("/api/User/{UserId}", h.updateUser)
	router.Delete("/api/User/{UserId}", h.deleteUser)
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Get("/api/Role", h.listRoles)
		protected.Get("/api/Role/{RoleId}", h.roleById)
		protected.Post("/api/Role", h.createRole)
		protected.Put("/api/Role/{RoleId}", h.updateRole)
		protected.Delete("/api/Role/{RoleId}", h.deleteRole)
		protected.Get("/api/UserAccess", h.listAccess)
		protected.Get("/api/UserAccess/{UserId}", h.accessOfUser)
		protected.Post("/api/UserAccess", h.createAccess)
		protected.Post("/api/UserAccess/CreateOrUpdateMany", h.createOrUpdateMany)
		protected.Put("/api/UserAccess/{Id}", h.updateAccess)
		protected.Delete("/api/UserAccess/{Id}", h.deleteAccess)
		protected.Get("/api/Settings", h.settings)
	})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var request domain.LoginRequest
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	if problems := requiredFields(map[string]string{"Username": request.Username, "Password": request.Password}); problems != nil {
		web.RespondValidation(w, problems)
		return
	}
	response, err := h.service.Login(r.Context(), request)
	if err != nil {
		web.RespondMessage(w, http.StatusUnauthorized, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Success(response, "Success"))
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	web.Respond(w, http.StatusOK, web.Success(nil, "Successfully logged out."))
}

func (h *Handler) currentUser(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok || identity.Id == "" {
		web.RespondMessage(w, http.StatusUnauthorized, "User not found or token is invalid.")
		return
	}
	web.WriteJSON(w, http.StatusOK, struct {
		Id       string `json:"id"`
		Fullname string `json:"fullname"`
		Username string `json:"username"`
		Email    string `json:"email"`
		RoleId   string `json:"roleId"`
		Role     string `json:"role"`
	}{identity.Id, identity.Fullname, identity.Username, identity.Email, identity.RoleId, identity.Role})
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	page, pageSize, ok := pagingQuery(w, r)
	if !ok {
		return
	}
	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	items, meta := web.Page(users, page, pageSize)
	web.Respond(w, http.StatusOK, web.SuccessWithMeta(items, "Success", meta))
}

func (h *Handler) userDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := guidQuery(w, r, "IdUser")
	if !ok {
		return
	}
	data, err := h.service.UserData(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(data, "Success"))
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var request domain.UserCreate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	if problems := requiredFields(map[string]string{"Fullname": request.Fullname, "UserName": request.UserName, "Email": request.Email, "Password": request.Password}); problems != nil {
		web.RespondValidation(w, problems)
		return
	}
	if err := h.service.AddUser(r.Context(), request); err != nil {
		web.RespondException(w, http.StatusBadRequest, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Created(request, "Data successfully "))
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := guidParam(w, r, "UserId")
	if !ok {
		return
	}
	var request domain.UserUpdate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	if problems := requiredFields(map[string]string{"Fullname": request.Fullname, "UserName": request.UserName}); problems != nil {
		web.RespondValidation(w, problems)
		return
	}
	if _, err := h.service.UserData(r.Context(), id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := h.service.UpdateUser(r.Context(), id, request); err != nil {
		web.RespondException(w, http.StatusBadRequest, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Success(request, "Data updated successfully"))
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	id, ok := guidQuery(w, r, "UserId")
	if !ok {
		return
	}
	var request domain.PasswordUpdate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	if problems := requiredFields(map[string]string{"PasswordNew": request.PasswordNew, "ReconfirmPassword": request.ReconfirmPassword}); problems != nil {
		web.RespondValidation(w, problems)
		return
	}
	if _, err := h.service.UserData(r.Context(), id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := h.service.UpdatePassword(r.Context(), id, request); err != nil {
		web.RespondException(w, http.StatusBadRequest, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Success(request, "Data updated successfully"))
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := guidParam(w, r, "UserId")
	if !ok {
		return
	}
	if _, err := h.service.UserData(r.Context(), id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := h.service.DeleteUser(r.Context(), id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "Data deleted successfully"))
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	page, pageSize, ok := pagingQuery(w, r)
	if !ok {
		return
	}
	roles, err := h.service.ListRoles(r.Context())
	if err != nil {
		web.RespondException(w, http.StatusInternalServerError, err.Error())
		return
	}
	items, meta := web.Page(roles, page, pageSize)
	web.Respond(w, http.StatusOK, web.SuccessWithMeta(items, "Success", meta))
}

func (h *Handler) roleById(w http.ResponseWriter, r *http.Request) {
	id, ok := guidParam(w, r, "RoleId")
	if !ok {
		return
	}
	role, err := h.service.FindRole(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if role == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(role, "Success"))
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	var request domain.RoleCreate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	if problems := requiredFields(map[string]string{"RoleName": request.RoleName}); problems != nil {
		web.RespondValidation(w, problems)
		return
	}
	if err := h.service.AddRole(r.Context(), request); err != nil {
		web.RespondException(w, http.StatusBadRequest, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Created(request, "Data successfully "))
}

func (h *Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	id, ok := guidParam(w, r, "RoleId")
	if !ok {
		return
	}
	var request domain.RoleCreate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	if problems := requiredFields(map[string]string{"RoleName": request.RoleName}); problems != nil {
		web.RespondValidation(w, problems)
		return
	}
	role, err := h.service.FindRole(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if role == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	if err := h.service.UpdateRole(r.Context(), id, request); err != nil {
		web.RespondException(w, http.StatusBadRequest, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Success(request, "Data updated successfully"))
}

func (h *Handler) deleteRole(w http.ResponseWriter, r *http.Request) {
	id, ok := guidParam(w, r, "RoleId")
	if !ok {
		return
	}
	role, err := h.service.FindRole(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if role == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	if err := h.service.DeleteRole(r.Context(), id); err != nil {
		web.RespondException(w, http.StatusBadRequest, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "Data deleted successfully"))
}

func (h *Handler) listAccess(w http.ResponseWriter, r *http.Request) {
	page, pageSize, ok := pagingQuery(w, r)
	if !ok {
		return
	}
	rows, err := h.service.ListAccessData(r.Context())
	if err != nil {
		web.RespondException(w, http.StatusInternalServerError, err.Error())
		return
	}
	items, meta := web.Page(rows, page, pageSize)
	web.Respond(w, http.StatusOK, web.SuccessWithMeta(items, "Success", meta))
}

func (h *Handler) accessOfUser(w http.ResponseWriter, r *http.Request) {
	id, ok := guidParam(w, r, "UserId")
	if !ok {
		return
	}
	rows, err := h.service.AccessOfUser(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(rows) == 0 {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(rows, "Success"))
}

func (h *Handler) createAccess(w http.ResponseWriter, r *http.Request) {
	var request domain.UserAccessCreate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	if problems := requiredFields(map[string]string{"Modul": request.Modul}); problems != nil {
		web.RespondValidation(w, problems)
		return
	}
	if err := h.service.AddAccess(r.Context(), request); err != nil {
		web.RespondException(w, http.StatusBadRequest, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Created(request, "Data successfully "))
}

func (h *Handler) createOrUpdateMany(w http.ResponseWriter, r *http.Request) {
	var requests []domain.UserAccessCreate
	if err := web.DecodeBody(r, &requests); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	problems := map[string][]string{}
	for index, request := range requests {
		if strings.TrimSpace(request.Modul) == "" {
			problems["["+itoa(index)+"].Modul"] = []string{web.RequiredMessage("Modul")}
		}
	}
	if len(problems) > 0 {
		web.RespondValidation(w, problems)
		return
	}
	if err := h.service.CreateOrUpdateMany(r.Context(), requests); err != nil {
		var invalid domain.InvalidOperationError
		if errors.As(err, &invalid) {
			web.RespondException(w, http.StatusBadRequest, "Validation error: "+err.Error())
			return
		}
		web.RespondException(w, http.StatusInternalServerError, "An error occurred: "+err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Created(web.NonNil(requests), "Data successfully added"))
}

func (h *Handler) updateAccess(w http.ResponseWriter, r *http.Request) {
	id, ok := guidParam(w, r, "Id")
	if !ok {
		return
	}
	var request domain.UserAccessEdit
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	if problems := requiredFields(map[string]string{"Modul": request.Modul}); problems != nil {
		web.RespondValidation(w, problems)
		return
	}
	if err := h.service.UpdateAccess(r.Context(), id, request); err != nil {
		web.RespondException(w, http.StatusBadRequest, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Success(request, "Data updated successfully"))
}

func (h *Handler) deleteAccess(w http.ResponseWriter, r *http.Request) {
	id, ok := guidParam(w, r, "Id")
	if !ok {
		return
	}
	if err := h.service.DeleteAccess(r.Context(), id); err != nil {
		web.RespondException(w, http.StatusBadRequest, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "Data deleted successfully"))
}

func (h *Handler) settings(w http.ResponseWriter, r *http.Request) {
	web.Respond(w, http.StatusOK, web.Success(map[string]string{"message": "Settings endpoint"}, "Success"))
}

func pagingQuery(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	page, err := web.QueryInt(r, "page", 1)
	if err != nil {
		web.RespondFieldProblem(w, "page", err)
		return 0, 0, false
	}
	pageSize, err := web.QueryInt(r, "pageSize", 10)
	if err != nil {
		web.RespondFieldProblem(w, "pageSize", err)
		return 0, 0, false
	}
	return page, pageSize, true
}

func guidParam(w http.ResponseWriter, r *http.Request, name string) (domain.Guid, bool) {
	raw := chi.URLParam(r, name)
	id, err := domain.ParseGuid(raw)
	if err != nil {
		web.RespondFieldProblem(w, name, err)
		return domain.Guid{}, false
	}
	return id, true
}

func guidQuery(w http.ResponseWriter, r *http.Request, name string) (domain.Guid, bool) {
	raw, present := web.Query(r, name)
	if !present || raw == "" {
		return domain.EmptyGuid, true
	}
	id, err := domain.ParseGuid(raw)
	if err != nil {
		web.RespondFieldProblem(w, name, err)
		return domain.Guid{}, false
	}
	return id, true
}

func requiredFields(fields map[string]string) map[string][]string {
	problems := map[string][]string{}
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			problems[name] = []string{web.RequiredMessage(name)}
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return problems
}

func itoa(value int) string {
	digits := []byte{}
	if value == 0 {
		return "0"
	}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}
