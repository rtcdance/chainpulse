package business

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockProductBackend implements ProductBackend for testing
type MockProductBackend struct {
	products map[string]*Product
	mu       sync.RWMutex
}

func NewMockProductBackend() *MockProductBackend {
	return &MockProductBackend{
		products: make(map[string]*Product),
	}
}

func (m *MockProductBackend) Create(ctx context.Context, product *Product) (*Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if product.ID == "" {
		product.ID = fmt.Sprintf("product-%d", len(m.products)+1)
	}
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()
	m.products[product.ID] = product
	return product, nil
}

func (m *MockProductBackend) Read(ctx context.Context, id string) (*Product, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	product, ok := m.products[id]
	if !ok {
		return nil, fmt.Errorf("product not found")
	}
	return product, nil
}

func (m *MockProductBackend) Update(ctx context.Context, product *Product) (*Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.products[product.ID]; !ok {
		return nil, fmt.Errorf("product not found")
	}
	product.UpdatedAt = time.Now()
	m.products[product.ID] = product
	return product, nil
}

func (m *MockProductBackend) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.products[id]; !ok {
		return fmt.Errorf("product not found")
	}
	delete(m.products, id)
	return nil
}

func (m *MockProductBackend) List(ctx context.Context, limit, offset int) ([]*Product, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	products := make([]*Product, 0)
	for _, product := range m.products {
		products = append(products, product)
	}
	return products, nil
}

func (m *MockProductBackend) Query(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]*Product, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	products := make([]*Product, 0)
	for _, product := range m.products {
		products = append(products, product)
	}
	return products, nil
}

func (m *MockProductBackend) GetByCategory(ctx context.Context, category string, limit, offset int) ([]*Product, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	products := make([]*Product, 0)
	for _, product := range m.products {
		if product.Category == category {
			products = append(products, product)
		}
	}
	return products, nil
}

func (m *MockProductBackend) UpdateStock(ctx context.Context, id string, quantity int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	product, ok := m.products[id]
	if !ok {
		return fmt.Errorf("product not found")
	}
	product.Stock += quantity
	return nil
}

func TestProductServiceCreate(t *testing.T) {
	backend := NewMockProductBackend()
	cache := NewMockServiceCache()
	service := NewProductService(backend, cache)

	product := &Product{
		Name:        "Test Product",
		Description: "A test product",
		Price:       99.99,
		Stock:       100,
		Category:    "electronics",
		Active:      true,
	}

	created, err := service.CreateProduct(context.Background(), product)
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	if created.ID == "" {
		t.Fatal("created product has no ID")
	}

	if created.Name != product.Name {
		t.Errorf("expected name %s, got %s", product.Name, created.Name)
	}
}

func TestProductServiceRead(t *testing.T) {
	backend := NewMockProductBackend()
	cache := NewMockServiceCache()
	service := NewProductService(backend, cache)

	product := &Product{
		ID:          "product-1",
		Name:        "Test Product",
		Description: "A test product",
		Price:       99.99,
		Stock:       100,
		Category:    "electronics",
		Active:      true,
	}

	if _, err := backend.Create(context.Background(), product); err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	retrieved, err := service.GetProduct(context.Background(), "product-1")
	if err != nil {
		t.Fatalf("failed to get product: %v", err)
	}

	if retrieved.ID != product.ID {
		t.Errorf("expected ID %s, got %s", product.ID, retrieved.ID)
	}
}

func TestProductServiceUpdate(t *testing.T) {
	backend := NewMockProductBackend()
	cache := NewMockServiceCache()
	service := NewProductService(backend, cache)

	product := &Product{
		ID:          "product-1",
		Name:        "Test Product",
		Description: "A test product",
		Price:       99.99,
		Stock:       100,
		Category:    "electronics",
		Active:      true,
	}

	if _, err := backend.Create(context.Background(), product); err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	product.Price = 149.99
	updated, err := service.UpdateProduct(context.Background(), product)
	if err != nil {
		t.Fatalf("failed to update product: %v", err)
	}

	if updated.Price != 149.99 {
		t.Errorf("expected price 149.99, got %f", updated.Price)
	}
}

func TestProductServiceDelete(t *testing.T) {
	backend := NewMockProductBackend()
	cache := NewMockServiceCache()
	service := NewProductService(backend, cache)

	product := &Product{
		ID:          "product-1",
		Name:        "Test Product",
		Description: "A test product",
		Price:       99.99,
		Stock:       100,
		Category:    "electronics",
		Active:      true,
	}

	if _, err := backend.Create(context.Background(), product); err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	err := service.DeleteProduct(context.Background(), "product-1")
	if err != nil {
		t.Fatalf("failed to delete product: %v", err)
	}

	_, err = backend.Read(context.Background(), "product-1")
	if err == nil {
		t.Fatal("expected error when reading deleted product")
	}
}

func TestProductServiceGetByCategory(t *testing.T) {
	backend := NewMockProductBackend()
	cache := NewMockServiceCache()
	service := NewProductService(backend, cache)

	product := &Product{
		ID:          "product-1",
		Name:        "Test Product",
		Description: "A test product",
		Price:       99.99,
		Stock:       100,
		Category:    "electronics",
		Active:      true,
	}

	if _, err := backend.Create(context.Background(), product); err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	products, err := service.GetProductsByCategory(context.Background(), "electronics", 10, 0)
	if err != nil {
		t.Fatalf("failed to get products by category: %v", err)
	}

	if len(products) != 1 {
		t.Errorf("expected 1 product, got %d", len(products))
	}
}

func TestProductServiceUpdateStock(t *testing.T) {
	backend := NewMockProductBackend()
	cache := NewMockServiceCache()
	service := NewProductService(backend, cache)

	product := &Product{
		ID:          "product-1",
		Name:        "Test Product",
		Description: "A test product",
		Price:       99.99,
		Stock:       100,
		Category:    "electronics",
		Active:      true,
	}

	if _, err := backend.Create(context.Background(), product); err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	err := service.UpdateProductStock(context.Background(), "product-1", 50)
	if err != nil {
		t.Fatalf("failed to update stock: %v", err)
	}

	updated, _ := backend.Read(context.Background(), "product-1")
	if updated.Stock != 150 {
		t.Errorf("expected stock 150, got %d", updated.Stock)
	}
}

func TestProductServiceMetrics(t *testing.T) {
	backend := NewMockProductBackend()
	cache := NewMockServiceCache()
	service := NewProductService(backend, cache)

	product := &Product{
		ID:          "product-1",
		Name:        "Test Product",
		Description: "A test product",
		Price:       99.99,
		Stock:       100,
		Category:    "electronics",
		Active:      true,
	}

	if _, err := backend.Create(context.Background(), product); err != nil {
		t.Fatalf("failed to create product: %v", err)
	}
	if _, err := service.GetProduct(context.Background(), "product-1"); err != nil {
		t.Fatalf("failed to get product: %v", err)
	}

	metrics := service.GetMetrics()
	if metrics["service_name"] != "product-service" {
		t.Errorf("expected service_name 'product-service', got %v", metrics["service_name"])
	}

	if metrics["total_operations"].(int64) < 1 {
		t.Errorf("expected total_operations >= 1, got %v", metrics["total_operations"])
	}
}
