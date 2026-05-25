package business

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockUserBackend implements UserBackend for testing
type MockUserBackend struct {
	users map[string]*User
	mu    sync.RWMutex
}

func NewMockUserBackend() *MockUserBackend {
	return &MockUserBackend{
		users: make(map[string]*User),
	}
}

func (m *MockUserBackend) Create(ctx context.Context, user *User) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if user.ID == "" {
		user.ID = fmt.Sprintf("user-%d", len(m.users)+1)
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return user, nil
}

func (m *MockUserBackend) Read(ctx context.Context, id string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	user, ok := m.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (m *MockUserBackend) Update(ctx context.Context, user *User) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[user.ID]; !ok {
		return nil, fmt.Errorf("user not found")
	}
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return user, nil
}

func (m *MockUserBackend) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[id]; !ok {
		return fmt.Errorf("user not found")
	}
	delete(m.users, id)
	return nil
}

func (m *MockUserBackend) List(ctx context.Context, limit, offset int) ([]*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	users := make([]*User, 0)
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

func (m *MockUserBackend) Query(ctx context.Context, filter map[string]any, limit, offset int) ([]*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	users := make([]*User, 0)
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

func (m *MockUserBackend) GetByEmail(ctx context.Context, email string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

// MockServiceCache implements ServiceCache for testing
type MockServiceCache struct {
	cache map[string]any
	mu    sync.RWMutex
}

func NewMockServiceCache() *MockServiceCache {
	return &MockServiceCache{
		cache: make(map[string]any),
	}
}

func (m *MockServiceCache) Get(ctx context.Context, key string) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.cache[key]
	if !ok {
		return nil, fmt.Errorf("key not found")
	}
	return val, nil
}

func (m *MockServiceCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[key] = value
	return nil
}

func (m *MockServiceCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, key)
	return nil
}

func TestUserServiceCreate(t *testing.T) {
	t.Parallel()
	backend := NewMockUserBackend()
	cache := NewMockServiceCache()
	service := NewUserService(backend, cache)

	user := &User{
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   "user",
		Active: true,
	}

	created, err := service.CreateUser(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if created.ID == "" {
		t.Fatal("created user has no ID")
	}

	if created.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, created.Email)
	}
}

func TestUserServiceRead(t *testing.T) {
	t.Parallel()
	backend := NewMockUserBackend()
	cache := NewMockServiceCache()
	service := NewUserService(backend, cache)

	user := &User{
		ID:     "user-1",
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   "user",
		Active: true,
	}

	if _, err := backend.Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	retrieved, err := service.GetUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}

	if retrieved.ID != user.ID {
		t.Errorf("expected ID %s, got %s", user.ID, retrieved.ID)
	}
}

func TestUserServiceUpdate(t *testing.T) {
	t.Parallel()
	backend := NewMockUserBackend()
	cache := NewMockServiceCache()
	service := NewUserService(backend, cache)

	user := &User{
		ID:     "user-1",
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   "user",
		Active: true,
	}

	if _, err := backend.Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	user.Name = "Updated User"
	updated, err := service.UpdateUser(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	if updated.Name != "Updated User" {
		t.Errorf("expected name 'Updated User', got %s", updated.Name)
	}
}

func TestUserServiceDelete(t *testing.T) {
	t.Parallel()
	backend := NewMockUserBackend()
	cache := NewMockServiceCache()
	service := NewUserService(backend, cache)

	user := &User{
		ID:     "user-1",
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   "user",
		Active: true,
	}

	if _, err := backend.Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	err := service.DeleteUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	_, err = backend.Read(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected error when reading deleted user")
	}
}

func TestUserServiceGetByEmail(t *testing.T) {
	t.Parallel()
	backend := NewMockUserBackend()
	cache := NewMockServiceCache()
	service := NewUserService(backend, cache)

	user := &User{
		ID:     "user-1",
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   "user",
		Active: true,
	}

	if _, err := backend.Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	retrieved, err := service.GetUserByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("failed to get user by email: %v", err)
	}

	if retrieved.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, retrieved.Email)
	}
}

func TestUserServiceMetrics(t *testing.T) {
	t.Parallel()
	backend := NewMockUserBackend()
	cache := NewMockServiceCache()
	service := NewUserService(backend, cache)

	user := &User{
		ID:     "user-1",
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   "user",
		Active: true,
	}

	if _, err := backend.Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if _, err := service.GetUser(context.Background(), "user-1"); err != nil {
		t.Fatalf("failed to get user: %v", err)
	}

	metrics := service.GetMetrics()
	if metrics["service_name"] != "user-service" {
		t.Errorf("expected service_name 'user-service', got %v", metrics["service_name"])
	}

	if metrics["total_operations"].(int64) < 1 {
		t.Errorf("expected total_operations >= 1, got %v", metrics["total_operations"])
	}
}

func TestUserServiceList(t *testing.T) {
	t.Parallel()
	backend := NewMockUserBackend()
	cache := NewMockServiceCache()
	service := NewUserService(backend, cache)

	for i := 1; i <= 3; i++ {
		user := &User{
			ID:     fmt.Sprintf("user-%d", i),
			Email:  fmt.Sprintf("user%d@example.com", i),
			Name:   fmt.Sprintf("User %d", i),
			Role:   "user",
			Active: true,
		}
		if _, err := backend.Create(context.Background(), user); err != nil {
			t.Fatalf("failed to create user: %v", err)
		}
	}

	users, err := service.ListUsers(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}

	if len(users) != 3 {
		t.Errorf("expected 3 users, got %d", len(users))
	}
}

func TestUserServiceQuery(t *testing.T) {
	t.Parallel()
	backend := NewMockUserBackend()
	cache := NewMockServiceCache()
	service := NewUserService(backend, cache)

	user := &User{
		ID:     "user-1",
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   "user",
		Active: true,
	}
	if _, err := backend.Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	filter := map[string]any{"role": "user"}
	users, err := service.QueryUsers(context.Background(), filter, 10, 0)
	if err != nil {
		t.Fatalf("failed to query users: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
}

func TestUserServiceGetByEmail_Caching(t *testing.T) {
	t.Parallel()
	backend := NewMockUserBackend()
	cache := NewMockServiceCache()
	service := NewUserService(backend, cache)

	user := &User{
		ID:     "user-1",
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   "user",
		Active: true,
	}
	if _, err := backend.Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	_, err := service.GetUserByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	retrieved, err := service.GetUserByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("cached call failed: %v", err)
	}

	if retrieved.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, retrieved.Email)
	}
}

func TestUserServiceGetByEmail_NoCache(t *testing.T) {
	t.Parallel()
	backend := NewMockUserBackend()
	service := NewUserService(backend, nil)

	user := &User{
		ID:     "user-1",
		Email:  "test@example.com",
		Name:   "Test User",
		Role:   "user",
		Active: true,
	}
	if _, err := backend.Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	retrieved, err := service.GetUserByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("failed to get user by email: %v", err)
	}

	if retrieved.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, retrieved.Email)
	}
}
