package serviceapi

// Service represents a system service like Redis or PostgreSQL
type Service interface {
	Start() error
	Stop() error
	IsRunning() bool
	RequiredBy() []string
	Name() string
}

// BaseService provides common functionality for services
type BaseService struct {
	name         string
	dependencies []string
}

func (s *BaseService) Name() string {
	return s.name
}

func (s *BaseService) RequiredBy() []string {
	return s.dependencies
}

// NewBaseService creates a new base service
func NewBaseService(name string, dependencies []string) BaseService {
	return BaseService{
		name:         name,
		dependencies: dependencies,
	}
}
