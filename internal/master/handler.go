package master

import (
	"errors"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/web"
)

const spreadsheetContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Mount(router chi.Router) {
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Get("/api/MasterManufacturer", h.listManufacturers)
		protected.Get("/api/MasterManufacturer/{vendorId}", h.manufacturerById)
		protected.Post("/api/MasterManufacturer", h.createManufacturer)
		protected.Put("/api/MasterManufacturer/{vendorId}", h.updateManufacturer)
		protected.Delete("/api/MasterManufacturer/{vendorId}", h.deleteManufacturer)
		protected.Get("/api/MasterComponent", h.listComponents)
		protected.Get("/api/MasterComponent/export", h.exportComponents)
		protected.Get("/api/MasterComponent/template", h.template)
		protected.Get("/api/MasterComponent/recent", h.recentComponents)
		protected.Get("/api/MasterComponent/{componentId}", h.componentById)
		protected.Post("/api/MasterComponent", h.createComponent)
		protected.Post("/api/MasterComponent/import", h.importComponents)
		protected.Put("/api/MasterComponent/{componentId}", h.updateComponent)
		protected.Delete("/api/MasterComponent/{componentId}", h.deleteComponent)
		protected.Get("/api/MasterProject", h.listProjects)
		protected.Get("/api/MasterProject/allSystem", h.listProjectSystems)
		protected.Get("/api/MasterProject/recent", h.recentProjects)
		protected.Get("/api/MasterProject/high-reliability", h.highReliabilityProjects)
		protected.Get("/api/MasterProject/{projectId}", h.projectById)
		protected.Get("/api/MasterProject/{projectId}/details", h.projectDetails)
		protected.Post("/api/MasterProject", h.createProject)
		protected.Put("/api/MasterProject/{projectId}", h.updateProject)
	})
}

type listQuery struct {
	page      *int
	pageSize  *int
	search    string
	sortBy    string
	sortOrder string
}

func readListQuery(w http.ResponseWriter, r *http.Request) (listQuery, bool) {
	page, err := web.QueryOptionalInt(r, "page")
	if err != nil {
		web.RespondFieldProblem(w, "page", err)
		return listQuery{}, false
	}
	pageSize, err := web.QueryOptionalInt(r, "pageSize")
	if err != nil {
		web.RespondFieldProblem(w, "pageSize", err)
		return listQuery{}, false
	}
	return listQuery{
		page:      page,
		pageSize:  pageSize,
		search:    web.QueryString(r, "search"),
		sortBy:    web.QueryString(r, "sortBy"),
		sortOrder: web.QueryStringOr(r, "sortOrder", "asc"),
	}, true
}

func respondList[T any](w http.ResponseWriter, items []T, query listQuery, message string) {
	page, meta := web.PageOptional(items, query.page, query.pageSize)
	web.Respond(w, http.StatusOK, web.SuccessWithMeta(page, message, meta))
}

func (h *Handler) listManufacturers(w http.ResponseWriter, r *http.Request) {
	query, ok := readListQuery(w, r)
	if !ok {
		return
	}
	rows, err := h.service.ListManufacturers(r.Context(), query.search, query.sortBy, query.sortOrder)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	respondList(w, rows, query, "Success")
}

func (h *Handler) manufacturerById(w http.ResponseWriter, r *http.Request) {
	row, err := h.service.FindManufacturer(r.Context(), chi.URLParam(r, "vendorId"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if row == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(row, "Success"))
}

type manufacturerInput struct {
	name       string
	logo       *multipart.FileHeader
	validUntil *domain.DateTime
	echo       domain.ManufacturerForm
}

func readManufacturerForm(w http.ResponseWriter, r *http.Request) (manufacturerInput, bool) {
	form, err := web.ParseForm(r)
	if err != nil {
		web.RespondUnsupportedMediaType(w)
		return manufacturerInput{}, false
	}
	problems := map[string][]string{}
	name, _ := form.Value("ManufacturerName")
	if strings.TrimSpace(name) == "" {
		problems["ManufacturerName"] = []string{web.RequiredMessage("ManufacturerName")}
	}
	var validUntil *domain.DateTime
	if raw, present := form.Value("ValidUntil"); present && strings.TrimSpace(raw) != "" {
		parsed, err := domain.ParseDateTime(strings.TrimSpace(raw))
		if err != nil {
			problems["ValidUntil"] = []string{web.InvalidValue(raw).Error()}
		} else {
			validUntil = &parsed
		}
	}
	if len(problems) > 0 {
		web.RespondValidation(w, problems)
		return manufacturerInput{}, false
	}
	logo := form.File("LogoImage")
	input := manufacturerInput{name: name, logo: logo, validUntil: validUntil}
	input.echo = domain.ManufacturerForm{ManufacturerName: name, ValidUntil: validUntil}
	if logo != nil {
		headers := map[string][]string{}
		for key, values := range logo.Header {
			headers[key] = values
		}
		input.echo.LogoImage = &domain.FormFileInfo{
			ContentDisposition: logo.Header.Get("Content-Disposition"),
			ContentType:        logo.Header.Get("Content-Type"),
			Headers:            headers,
			Length:             logo.Size,
			Name:               "LogoImage",
			FileName:           logo.Filename,
		}
	}
	return input, true
}

func (h *Handler) createManufacturer(w http.ResponseWriter, r *http.Request) {
	input, ok := readManufacturerForm(w, r)
	if !ok {
		return
	}
	if err := h.service.AddManufacturer(r.Context(), input.name, input.logo, input.validUntil, auth.CurrentUserName(r.Context())); err != nil {
		var invalid domain.InvalidOperationError
		if errors.As(err, &invalid) {
			web.Respond(w, http.StatusBadRequest, web.BadRequest(err.Error()))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(input.echo, "Data successfully"))
}

func (h *Handler) updateManufacturer(w http.ResponseWriter, r *http.Request) {
	vendorId := chi.URLParam(r, "vendorId")
	input, ok := readManufacturerForm(w, r)
	if !ok {
		return
	}
	existing, err := h.service.FindManufacturer(r.Context(), vendorId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if existing == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	if err := h.service.UpdateManufacturer(r.Context(), vendorId, input.name, input.logo, input.validUntil, auth.CurrentUserName(r.Context())); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(input.echo, "Data updated successfully"))
}

func (h *Handler) deleteManufacturer(w http.ResponseWriter, r *http.Request) {
	vendorId := chi.URLParam(r, "vendorId")
	existing, err := h.service.FindManufacturer(r.Context(), vendorId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if existing == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	if err := h.service.DeleteManufacturer(r.Context(), vendorId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "Data deleted successfully"))
}

func (h *Handler) listComponents(w http.ResponseWriter, r *http.Request) {
	query, ok := readListQuery(w, r)
	if !ok {
		return
	}
	rows, err := h.service.ListComponents(r.Context(), query.search, query.sortBy, query.sortOrder)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	respondList(w, rows, query, "Success")
}

func (h *Handler) componentById(w http.ResponseWriter, r *http.Request) {
	row, err := h.service.FindComponent(r.Context(), chi.URLParam(r, "componentId"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if row == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(row, "Success"))
}

func (h *Handler) createComponent(w http.ResponseWriter, r *http.Request) {
	var request domain.MasterComponentCreate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	if problems := requiredFields(map[string]string{"ComponentName": request.ComponentName, "VendorId": request.VendorId}); problems != nil {
		web.RespondValidation(w, problems)
		return
	}
	if err := h.service.AddComponent(r.Context(), request, auth.CurrentUserName(r.Context())); err != nil {
		var invalid domain.InvalidOperationError
		if errors.As(err, &invalid) {
			web.Respond(w, http.StatusBadRequest, web.BadRequest(err.Error()))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(request, "Data created successfully"))
}

func (h *Handler) updateComponent(w http.ResponseWriter, r *http.Request) {
	componentId := chi.URLParam(r, "componentId")
	var request domain.MasterComponentUpdate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	if problems := requiredFields(map[string]string{"ComponentName": request.ComponentName, "VendorId": request.VendorId}); problems != nil {
		web.RespondValidation(w, problems)
		return
	}
	existing, err := h.service.FindComponent(r.Context(), componentId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if existing == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	if err := h.service.UpdateComponent(r.Context(), componentId, request, auth.CurrentUserName(r.Context())); err != nil {
		var invalid domain.InvalidOperationError
		if errors.As(err, &invalid) {
			web.Respond(w, http.StatusBadRequest, web.BadRequest(err.Error()))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(request, "Data updated successfully"))
}

func (h *Handler) deleteComponent(w http.ResponseWriter, r *http.Request) {
	componentId := chi.URLParam(r, "componentId")
	existing, err := h.service.FindComponent(r.Context(), componentId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if existing == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	if err := h.service.DeleteComponent(r.Context(), componentId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "Data deleted successfully"))
}

func (h *Handler) importComponents(w http.ResponseWriter, r *http.Request) {
	form, err := web.ParseForm(r)
	if err != nil {
		web.RespondUnsupportedMediaType(w)
		return
	}
	file := form.File("file")
	if file == nil || file.Size == 0 {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("No file uploaded."))
		return
	}
	extension := strings.ToLower(filepath.Ext(file.Filename))
	if extension != ".xlsx" && extension != ".xls" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("Invalid file format. Please upload an Excel file (.xlsx or .xls)."))
		return
	}
	source, err := file.Open()
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
		return
	}
	defer source.Close()
	result, err := h.service.ImportWorkbook(r.Context(), source, auth.CurrentUserName(r.Context()))
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
		return
	}
	if len(result.FailedRows) > 0 {
		web.Respond(w, http.StatusOK, web.Success(result, "Import completed. Success: "+itoa(result.SuccessCount)+", Updated: "+itoa(result.UpdatedCount)+", Failed: "+itoa(len(result.FailedRows))))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(result, "Data imported successfully. "+itoa(result.SuccessCount)+" new records, "+itoa(result.UpdatedCount)+" updated records."))
}

func (h *Handler) exportComponents(w http.ResponseWriter, r *http.Request) {
	content, err := h.service.ExportWorkbook(r.Context())
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
		return
	}
	web.AttachmentHeader(w, spreadsheetContentType, "MasterComponent_"+time.Now().UTC().Format("20060102_150405")+".xlsx")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (h *Handler) template(w http.ResponseWriter, r *http.Request) {
	content, err := h.service.TemplateWorkbook()
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
		return
	}
	web.AttachmentHeader(w, spreadsheetContentType, "MasterComponent_Template.xlsx")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (h *Handler) recentComponents(w http.ResponseWriter, r *http.Request) {
	rows, err := h.service.RecentComponents(r.Context(), 3)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(rows, "Recent components retrieved successfully"))
}

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	query, ok := readListQuery(w, r)
	if !ok {
		return
	}
	rows, err := h.service.ListProjects(r.Context(), query.search, query.sortBy, query.sortOrder)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	respondList(w, rows, query, "Success")
}

func (h *Handler) listProjectSystems(w http.ResponseWriter, r *http.Request) {
	query, ok := readListQuery(w, r)
	if !ok {
		return
	}
	rows, err := h.service.ListProjectSystems(r.Context(), query.search, query.sortBy, query.sortOrder)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	respondList(w, rows, query, "Success")
}

func (h *Handler) projectById(w http.ResponseWriter, r *http.Request) {
	row, err := h.service.FindProject(r.Context(), chi.URLParam(r, "projectId"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if row == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(row, "Success"))
}

func (h *Handler) projectDetails(w http.ResponseWriter, r *http.Request) {
	detail, err := h.service.ProjectDetail(r.Context(), chi.URLParam(r, "projectId"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if detail == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(detail, "Success"))
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	var request domain.MasterProjectCreate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	if problems := requiredFields(map[string]string{"ProjectName": request.ProjectName}); problems != nil {
		web.RespondValidation(w, problems)
		return
	}
	if err := h.service.AddProject(r.Context(), &request, auth.CurrentUserName(r.Context())); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", web.AbsoluteURL(r, "/api/MasterProject/"+domain.Deref(request.ProjectId)))
	web.Respond(w, http.StatusCreated, web.Created(request, "Data created successfully"))
}

func (h *Handler) updateProject(w http.ResponseWriter, r *http.Request) {
	projectId := chi.URLParam(r, "projectId")
	var request domain.MasterProjectUpdate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	if problems := requiredFields(map[string]string{"ProjectName": request.ProjectName}); problems != nil {
		web.RespondValidation(w, problems)
		return
	}
	existing, err := h.service.FindProject(r.Context(), projectId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if existing == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	if err := h.service.UpdateProject(r.Context(), projectId, request, auth.CurrentUserName(r.Context())); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(request, "Data updated successfully"))
}

func (h *Handler) recentProjects(w http.ResponseWriter, r *http.Request) {
	rows, err := h.service.RecentProjectSystems(r.Context(), 3)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(rows, "Success"))
}

func (h *Handler) highReliabilityProjects(w http.ResponseWriter, r *http.Request) {
	count, err := web.QueryInt(r, "count", 10)
	if err != nil {
		web.RespondFieldProblem(w, "count", err)
		return
	}
	rows, err := h.service.HighReliabilitySystems(r.Context(), count)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(rows, "Success"))
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
	return web.Itoa(value)
}
