package business

import (
	"context"
	"fmt"
	"log"
	"time"
)

// User represents a user entity
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetID returns the user ID
func (u *User) GetID() string {
	return u.ID
}

// UserBackend defines the backend for user operations
type UserBackend interface {
	Create(ctx context.Context, user *User) (*User, error)
	Read(ctx context.Context, id string) (*User, error)
	Update(ctx context.Context, user *User) (*User, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*User, error)
	Query(ctx context.Context, filter map[string]any, limit, offset int) ([]*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}

// UserService provides user management operations
type UserService struct {
	*AbstractService
	backend UserBackend
}

// NewUserService creates a new user service
func NewUserService(backend UserBackend, cache ServiceCache) *UserService {
	abstractService := NewAbstractService("user-service", &userServiceAdapter{backend: backend}, cache)
	return &UserService{
		AbstractService: abstractService,
		backend:         backend,
	}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, user *User) (*User, error) {
	entity, err := s.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	u, ok := entity.(*User)
	if !ok {
		return nil, fmt.Errorf("unexpected entity type: %T", entity)
	}
	return u, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
	entity, err := s.Read(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	u, ok := entity.(*User)
	if !ok {
		return nil, fmt.Errorf("unexpected entity type: %T", entity)
	}
	return u, nil
}

// UpdateUser updates an existing user
func (s *UserService) UpdateUser(ctx context.Context, user *User) (*User, error) {
	entity, err := s.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	u, ok := entity.(*User)
	if !ok {
		return nil, fmt.Errorf("unexpected entity type: %T", entity)
	}
	return u, nil
}

// DeleteUser deletes a user
func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	return s.Delete(ctx, id)
}

// ListUsers lists all users with pagination
func (s *UserService) ListUsers(ctx context.Context, limit, offset int) ([]*User, error) {
	entities, err := s.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]*User, len(entities))
	for i, entity := range entities {
		u, ok := entity.(*User)
		if !ok {
			return nil, fmt.Errorf("unexpected entity type: %T", entity)
		}
		users[i] = u
	}
	return users, nil
}

// QueryUsers queries users with filtering
func (s *UserService) QueryUsers(ctx context.Context, filter map[string]any, limit, offset int) ([]*User, error) {
	entities, err := s.Query(ctx, filter, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}

	users := make([]*User, len(entities))
	for i, entity := range entities {
		u, ok := entity.(*User)
		if !ok {
			return nil, fmt.Errorf("unexpected entity type: %T", entity)
		}
		users[i] = u
	}
	return users, nil
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("user-service:email:%s", email)
	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != nil {
			u, ok := cached.(*User)
			if !ok {
				return nil, fmt.Errorf("unexpected cached type: %T", cached)
			}
			return u, nil
		}
	}

	// Query backend
	user, err := s.backend.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	// Cache result
	if s.cache != nil && user != nil {
		if err := s.cache.Set(ctx, cacheKey, user, 1*time.Hour); err != nil {
			log.Printf("UserService: cache set error for key %s: %v", cacheKey, err)
		}
	}

	return user, nil
}

// userServiceAdapter adapts UserBackend to ServiceBackend
type userServiceAdapter struct {
	backend UserBackend
}

func (a *userServiceAdapter) Create(ctx context.Context, entity Entity) (Entity, error) {
	user, ok := entity.(*User)
	if !ok {
		return nil, fmt.Errorf("expected *User, got %T", entity)
	}
	return a.backend.Create(ctx, user)
}

func (a *userServiceAdapter) Read(ctx context.Context, id string) (Entity, error) {
	return a.backend.Read(ctx, id)
}

func (a *userServiceAdapter) Update(ctx context.Context, entity Entity) (Entity, error) {
	user, ok := entity.(*User)
	if !ok {
		return nil, fmt.Errorf("expected *User, got %T", entity)
	}
	return a.backend.Update(ctx, user)
}

func (a *userServiceAdapter) Delete(ctx context.Context, id string) error {
	return a.backend.Delete(ctx, id)
}

func (a *userServiceAdapter) List(ctx context.Context, limit, offset int) ([]Entity, error) {
	users, err := a.backend.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users from backend: %w", err)
	}

	entities := make([]Entity, len(users))
	for i, user := range users {
		entities[i] = user
	}
	return entities, nil
}

func (a *userServiceAdapter) Query(ctx context.Context, filter map[string]any, limit, offset int) ([]Entity, error) {
	users, err := a.backend.Query(ctx, filter, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query users from backend: %w", err)
	}

	entities := make([]Entity, len(users))
	for i, user := range users {
		entities[i] = user
	}
	return entities, nil
}
