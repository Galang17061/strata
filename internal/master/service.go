package master

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Galang17061/strata-api/internal/domain"
)

const (
	logoFolder   = "logo"
	maxLogoBytes = 3145728
)

type Service struct {
	store     *Store
	uploadDir string
}

func NewService(store *Store, uploadDir string) *Service {
	return &Service{store: store, uploadDir: uploadDir}
}

func (s *Service) ListManufacturers(ctx context.Context, search, sortBy, sortOrder string) ([]domain.MasterManufacturer, error) {
	rows, err := s.store.ListManufacturers(ctx, search)
	if err != nil {
		return nil, err
	}
	if sortBy != "" && sortOrder != "" {
		compare := manufacturerComparer(sortBy)
		domain.OrderBy(rows, strings.ToLower(sortOrder) == "desc", compare)
	} else {
		domain.OrderBy(rows, false, func(a, b domain.MasterManufacturer) int { return domain.CompareText(a.ManufacturerName, b.ManufacturerName) })
	}
	return rows, nil
}

func manufacturerComparer(sortBy string) func(a, b domain.MasterManufacturer) int {
	switch strings.ToLower(sortBy) {
	case "vendorid":
		return func(a, b domain.MasterManufacturer) int { return domain.CompareText(a.VendorId, b.VendorId) }
	case "createdat":
		return func(a, b domain.MasterManufacturer) int {
			return domain.CompareInt64(a.CreatedAt.UnixNano(), b.CreatedAt.UnixNano())
		}
	default:
		return func(a, b domain.MasterManufacturer) int { return domain.CompareText(a.ManufacturerName, b.ManufacturerName) }
	}
}

func (s *Service) FindManufacturer(ctx context.Context, vendorId string) (*domain.MasterManufacturer, error) {
	return s.store.FindManufacturer(ctx, vendorId)
}

type UploadedFile struct {
	Header *multipart.FileHeader
}

func (s *Service) AddManufacturer(ctx context.Context, name string, logo *multipart.FileHeader, validUntil *domain.DateTime, currentUser string) error {
	upload := ""
	if logo != nil {
		if err := checkImage(logo, "file bukan berupa image"); err != nil {
			return err
		}
		stored, err := s.storeLogo(logo)
		if err != nil {
			return err
		}
		upload = stored
	}
	existing, err := s.store.ManufacturerByName(ctx, name)
	if err != nil {
		return err
	}
	if existing != nil {
		return domain.InvalidOperation("Manufacturer Name Already Exist")
	}
	lastId, err := s.store.LastManufacturerId(ctx)
	if err != nil {
		return err
	}
	newId, err := nextPrefixedId(lastId, "VD-")
	if err != nil {
		return err
	}
	now := domain.Now()
	return s.store.InsertManufacturer(ctx, domain.MasterManufacturer{
		VendorId:         newId,
		ManufacturerName: name,
		LogoImage:        domain.StringPtr(upload),
		ValidUntil:       validUntil,
		CreatedAt:        now,
		UpdatedAt:        now,
		CreatedBy:        domain.StringPtr(currentUser),
		UpdatedBy:        domain.StringPtr(currentUser),
	})
}

func (s *Service) UpdateManufacturer(ctx context.Context, vendorId, name string, logo *multipart.FileHeader, validUntil *domain.DateTime, currentUser string) error {
	manufacturer, err := s.store.FindManufacturer(ctx, vendorId)
	if err != nil || manufacturer == nil {
		return err
	}
	if logo != nil {
		if err := checkImage(logo, "file bukan berupa dokumen"); err != nil {
			return err
		}
		if manufacturer.LogoImage != nil && *manufacturer.LogoImage != "" {
			_ = os.Remove(filepath.Join(s.uploadDir, filepath.FromSlash(*manufacturer.LogoImage)))
		}
		stored, err := s.storeLogo(logo)
		if err != nil {
			return err
		}
		manufacturer.LogoImage = domain.StringPtr(stored)
	}
	manufacturer.ManufacturerName = name
	manufacturer.ValidUntil = validUntil
	manufacturer.UpdatedAt = domain.Now()
	manufacturer.UpdatedBy = domain.StringPtr(currentUser)
	return s.store.UpdateManufacturer(ctx, *manufacturer)
}

func (s *Service) DeleteManufacturer(ctx context.Context, vendorId string) error {
	manufacturer, err := s.store.FindManufacturer(ctx, vendorId)
	if err != nil || manufacturer == nil {
		return err
	}
	if manufacturer.LogoImage != nil && *manufacturer.LogoImage != "" {
		_ = os.Remove(filepath.Join(s.uploadDir, filepath.FromSlash(*manufacturer.LogoImage)))
	}
	return s.store.DeleteManufacturer(ctx, vendorId)
}

func checkImage(header *multipart.FileHeader, notImageMessage string) error {
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(header.Filename)))
	if contentType == "" {
		return domain.Argument("tidak dapat menemukan Content-Type dari " + header.Filename + ".")
	}
	switch strings.Split(contentType, ";")[0] {
	case "image/tiff", "image/jpeg", "image/bmp", "image/gif", "image/png", "image/svg+xml", "image/webp":
	default:
		return domain.InvalidOperation(notImageMessage)
	}
	if header.Size > maxLogoBytes {
		return domain.InvalidOperation("File size exceeds the maximum limit of 3MB.")
	}
	return nil
}

func (s *Service) storeLogo(header *multipart.FileHeader) (string, error) {
	if header.Size <= 0 {
		return "", domain.InvalidOperation("File kosong")
	}
	extensionIndex := strings.LastIndex(header.Filename, ".")
	if extensionIndex < 0 {
		return "", domain.Argument("Value cannot be null. (Parameter 'path2')")
	}
	random := make([]byte, 16)
	_, _ = rand.Read(random)
	fileName := hex.EncodeToString(random) + header.Filename[extensionIndex:]
	folder := filepath.Join(s.uploadDir, logoFolder)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return "", err
	}
	source, err := header.Open()
	if err != nil {
		return "", err
	}
	defer source.Close()
	target, err := os.Create(filepath.Join(folder, fileName))
	if err != nil {
		return "", err
	}
	defer target.Close()
	if _, err := io.Copy(target, source); err != nil {
		return "", err
	}
	return logoFolder + "/" + fileName, nil
}

func nextPrefixedId(lastId *string, prefix string) (string, error) {
	if lastId != nil && strings.HasPrefix(*lastId, prefix) {
		number, err := strconv.Atoi((*lastId)[len(prefix):])
		if err != nil {
			return "", domain.Argument("Input string was not in a correct format.")
		}
		return fmt.Sprintf("%s%05d", prefix, number+1), nil
	}
	return prefix + "00001", nil
}

func (s *Service) ListComponents(ctx context.Context, search, sortBy, sortOrder string) ([]domain.MasterComponentView, error) {
	rows, err := s.store.ListComponentViews(ctx, search)
	if err != nil {
		return nil, err
	}
	if sortBy != "" && sortOrder != "" {
		domain.OrderBy(rows, strings.ToLower(sortOrder) == "desc", componentComparer(sortBy))
	} else {
		domain.OrderBy(rows, false, func(a, b domain.MasterComponentView) int { return domain.CompareText(a.ComponentName, b.ComponentName) })
	}
	return rows, nil
}

func componentComparer(sortBy string) func(a, b domain.MasterComponentView) int {
	switch strings.ToLower(sortBy) {
	case "manufacturername":
		return func(a, b domain.MasterComponentView) int { return domain.CompareText(a.ManufacturerName, b.ManufacturerName) }
	case "failurerate":
		return func(a, b domain.MasterComponentView) int { return domain.CompareNumber(a.FailureRate, b.FailureRate) }
	case "createdat":
		return func(a, b domain.MasterComponentView) int {
			return domain.CompareInt64(a.CreatedAt.UnixNano(), b.CreatedAt.UnixNano())
		}
	default:
		return func(a, b domain.MasterComponentView) int { return domain.CompareText(a.ComponentName, b.ComponentName) }
	}
}

func (s *Service) FindComponent(ctx context.Context, componentId string) (*domain.MasterComponentView, error) {
	return s.store.FindComponentView(ctx, componentId)
}

func (s *Service) AddComponent(ctx context.Context, request domain.MasterComponentCreate, currentUser string) error {
	existing, err := s.store.ComponentByNameAndVendor(ctx, request.ComponentName, request.VendorId, "")
	if err != nil {
		return err
	}
	if existing != nil {
		return domain.InvalidOperation("Component with same Vendor Already Exist")
	}
	lastId, err := s.store.LastComponentId(ctx)
	if err != nil {
		return err
	}
	newId, err := nextPrefixedId(lastId, "CM-")
	if err != nil {
		return err
	}
	now := domain.Now()
	failureRate := request.FailureRate
	return s.store.InsertComponent(ctx, domain.MasterComponent{
		ComponentId:   newId,
		ComponentName: request.ComponentName,
		VendorId:      request.VendorId,
		FailureRate:   &failureRate,
		Cost:          request.Cost,
		Compatibility: request.Compatibility,
		SerialNumber:  request.SerialNumber,
		CreatedAt:     now,
		UpdatedAt:     now,
		CreatedBy:     domain.StringPtr(currentUser),
		UpdatedBy:     domain.StringPtr(currentUser),
	})
}

func (s *Service) UpdateComponent(ctx context.Context, componentId string, request domain.MasterComponentUpdate, currentUser string) error {
	component, err := s.store.FindComponent(ctx, componentId)
	if err != nil || component == nil {
		return err
	}
	existing, err := s.store.ComponentByNameAndVendor(ctx, request.ComponentName, request.VendorId, componentId)
	if err != nil {
		return err
	}
	if existing != nil {
		return domain.InvalidOperation("Validasi relasi antar vendor dan component")
	}
	failureRate := request.FailureRate
	component.ComponentName = request.ComponentName
	component.VendorId = request.VendorId
	component.FailureRate = &failureRate
	component.Cost = request.Cost
	component.Compatibility = request.Compatibility
	component.SerialNumber = request.SerialNumber
	component.UpdatedAt = domain.Now()
	component.UpdatedBy = domain.StringPtr(currentUser)
	return s.store.UpdateComponent(ctx, *component)
}

func (s *Service) DeleteComponent(ctx context.Context, componentId string) error {
	component, err := s.store.FindComponent(ctx, componentId)
	if err != nil || component == nil {
		return err
	}
	return s.store.DeleteComponent(ctx, componentId)
}

func (s *Service) RecentComponents(ctx context.Context, count int) ([]domain.SystemComponentProperties, error) {
	return s.store.RecentComponentProperties(ctx, count)
}

func (s *Service) ListProjects(ctx context.Context, search, sortBy, sortOrder string) ([]domain.MasterProject, error) {
	rows, err := s.store.ListProjects(ctx, search)
	if err != nil {
		return nil, err
	}
	if sortBy != "" && sortOrder != "" {
		domain.OrderBy(rows, strings.ToLower(sortOrder) == "desc", projectComparer(sortBy))
	} else {
		domain.OrderBy(rows, false, func(a, b domain.MasterProject) int { return domain.CompareText(a.ProjectName, b.ProjectName) })
	}
	return rows, nil
}

func projectComparer(sortBy string) func(a, b domain.MasterProject) int {
	switch strings.ToLower(sortBy) {
	case "projectid":
		return func(a, b domain.MasterProject) int { return domain.CompareText(a.ProjectId, b.ProjectId) }
	case "createdat":
		return func(a, b domain.MasterProject) int {
			return domain.CompareInt64(a.CreatedAt.UnixNano(), b.CreatedAt.UnixNano())
		}
	default:
		return func(a, b domain.MasterProject) int { return domain.CompareText(a.ProjectName, b.ProjectName) }
	}
}

func (s *Service) ListProjectSystems(ctx context.Context, search, sortBy, sortOrder string) ([]domain.MasterProjectRbd, error) {
	rows, err := s.store.ListProjectSystems(ctx, search)
	if err != nil {
		return nil, err
	}
	if sortBy != "" && sortOrder != "" {
		domain.OrderBy(rows, strings.ToLower(sortOrder) == "desc", projectSystemComparer(sortBy))
	} else {
		domain.OrderBy(rows, false, func(a, b domain.MasterProjectRbd) int { return domain.CompareText(a.ProjectName, b.ProjectName) })
	}
	return rows, nil
}

func projectSystemComparer(sortBy string) func(a, b domain.MasterProjectRbd) int {
	switch strings.ToLower(sortBy) {
	case "systemname":
		return func(a, b domain.MasterProjectRbd) int { return domain.CompareText(domain.Deref(a.SystemName), domain.Deref(b.SystemName)) }
	case "drawingname":
		return func(a, b domain.MasterProjectRbd) int {
			return domain.CompareText(domain.Deref(a.DrawingName), domain.Deref(b.DrawingName))
		}
	case "reliabilitytotal":
		return func(a, b domain.MasterProjectRbd) int { return domain.CompareNumber(a.ReliabilityTotal, b.ReliabilityTotal) }
	case "createdat":
		return func(a, b domain.MasterProjectRbd) int {
			return domain.CompareInt64(a.CreatedAt.UnixNano(), b.CreatedAt.UnixNano())
		}
	default:
		return func(a, b domain.MasterProjectRbd) int { return domain.CompareText(a.ProjectName, b.ProjectName) }
	}
}

func (s *Service) FindProject(ctx context.Context, projectId string) (*domain.MasterProject, error) {
	return s.store.FindProject(ctx, projectId)
}

func (s *Service) ProjectDetail(ctx context.Context, projectId string) (*domain.MasterProjectDetail, error) {
	project, err := s.store.FindProject(ctx, projectId)
	if err != nil || project == nil {
		return nil, err
	}
	systems, err := s.store.SystemsOfProject(ctx, projectId)
	if err != nil {
		return nil, err
	}
	detail := &domain.MasterProjectDetail{ProjectName: project.ProjectName, RbdSystems: []domain.RbdSystemSummary{}}
	for _, system := range systems {
		hierarchies, err := s.store.HierarchiesOfSystem(ctx, system.RbdSystemId)
		if err != nil {
			return nil, err
		}
		components, err := s.store.ComponentsOfSystem(ctx, system.RbdSystemId)
		if err != nil {
			return nil, err
		}
		summary := domain.RbdSystemSummary{SystemName: system.SystemName, Hierarchies: []domain.HierarchySummary{}}
		for _, hierarchy := range hierarchies {
			names := []domain.ComponentNameOnly{}
			for _, component := range components {
				if domain.Deref(component.ParentId) == hierarchy.HierarchyId {
					names = append(names, domain.ComponentNameOnly{ComponentName: component.ComponentName})
				}
			}
			summary.Hierarchies = append(summary.Hierarchies, domain.HierarchySummary{Level: hierarchy.Level, SubSystemName: hierarchy.SubSystemName, SystemComponents: names})
		}
		detail.RbdSystems = append(detail.RbdSystems, summary)
	}
	return detail, nil
}

func (s *Service) AddProject(ctx context.Context, request *domain.MasterProjectCreate, currentUser string) error {
	existing, err := s.store.ProjectByName(ctx, request.ProjectName)
	if err != nil {
		return err
	}
	if existing != nil {
		return domain.InvalidOperation("Project Name Already Exist")
	}
	lastId, err := s.store.LastProjectId(ctx)
	if err != nil {
		return err
	}
	newId, err := nextPrefixedId(lastId, "PJ-")
	if err != nil {
		return err
	}
	request.ProjectId = domain.StringPtr(newId)
	now := domain.Now()
	return s.store.InsertProject(ctx, domain.MasterProject{
		ProjectId:   newId,
		ProjectName: request.ProjectName,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   domain.StringPtr(currentUser),
		UpdatedBy:   domain.StringPtr(currentUser),
	})
}

func (s *Service) UpdateProject(ctx context.Context, projectId string, request domain.MasterProjectUpdate, currentUser string) error {
	project, err := s.store.FindProject(ctx, projectId)
	if err != nil || project == nil {
		return err
	}
	project.ProjectName = request.ProjectName
	project.UpdatedAt = domain.Now()
	project.UpdatedBy = domain.StringPtr(currentUser)
	return s.store.UpdateProject(ctx, *project)
}

func (s *Service) RecentProjectSystems(ctx context.Context, count int) ([]domain.MasterProjectRbd, error) {
	return s.store.RecentProjectSystems(ctx, count)
}

func (s *Service) HighReliabilitySystems(ctx context.Context, count int) ([]domain.MasterProjectRbd, error) {
	return s.store.HighReliabilitySystems(ctx, count)
}
