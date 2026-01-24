package deployment

import "time"

// ServiceInfo holds information about a service
type ServiceInfo struct {
	ID              string
	Name            string
	Version         string
	Status          HealthStatus
	Endpoint        string
	Port            int
	Protocol        string
	HealthCheckURL  string
	Metadata        map[string]string
	RegisteredAt    time.Time
	LastHeartbeat   time.Time
	Tags            []string
	DeregisterAfter string
}

// HealthStatus represents the health status of a service
type HealthStatus struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details"`
}

// NewServiceInfo creates a new ServiceInfo
func NewServiceInfo(id, name, version string) *ServiceInfo {
	return &ServiceInfo{
		ID:           id,
		Name:         name,
		Version:      version,
		Status:       HealthStatus{Status: "unknown"},
		Metadata:     make(map[string]string),
		RegisteredAt: time.Now(),
		Tags:         make([]string, 0),
	}
}

// IsHealthy returns true if the service is healthy
func (s *ServiceInfo) IsHealthy() bool {
	return s.Status.Status == "healthy"
}

// UpdateStatus updates the service status
func (s *ServiceInfo) UpdateStatus(status HealthStatus) {
	s.Status = status
	s.LastHeartbeat = time.Now()
}

// AddTag adds a tag to the service
func (s *ServiceInfo) AddTag(tag string) {
	s.Tags = append(s.Tags, tag)
}

// SetMetadata sets metadata for the service
func (s *ServiceInfo) SetMetadata(key, value string) {
	s.Metadata[key] = value
}

// GetMetadata gets metadata for the service
func (s *ServiceInfo) GetMetadata(key string) string {
	return s.Metadata[key]
}
