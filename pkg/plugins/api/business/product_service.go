package business

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Product represents a product entity
type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Stock       int64     `json:"stock"`
	Category    string    `json:"category"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GetID returns the product ID
func (p *Product) GetID() string {
	return p.ID
}

// ProductBackend defines the backend for product operations
type ProductBackend interface {
	Create(ctx context.Context, product *Product) (*Product, error)
	Read(ctx context.Context, id string) (*Product, error)
	Update(ctx context.Context, product *Product) (*Product, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*Product, error)
	Query(ctx context.Context, filter map[string]any, limit, offset int) ([]*Product, error)
	GetByCategory(ctx context.Context, category string, limit, offset int) ([]*Product, error)
	UpdateStock(ctx context.Context, id string, quantity int64) error
}

// ProductService provides product management operations
type ProductService struct {
	*AbstractService
	backend ProductBackend
}

// NewProductService creates a new product service
func NewProductService(backend ProductBackend, cache ServiceCache) *ProductService {
	abstractService := NewAbstractService("product-service", &productServiceAdapter{backend: backend}, cache)
	return &ProductService{
		AbstractService: abstractService,
		backend:         backend,
	}
}

// CreateProduct creates a new product
func (s *ProductService) CreateProduct(ctx context.Context, product *Product) (*Product, error) {
	entity, err := s.Create(ctx, product)
	if err != nil {
		return nil, err
	}
	product, ok := entity.(*Product)
	if !ok {
		return nil, fmt.Errorf("unexpected entity type: %T", entity)
	}
	return product, nil
}

// GetProduct retrieves a product by ID
func (s *ProductService) GetProduct(ctx context.Context, id string) (*Product, error) {
	entity, err := s.Read(ctx, id)
	if err != nil {
		return nil, err
	}
	product, ok := entity.(*Product)
	if !ok {
		return nil, fmt.Errorf("unexpected entity type: %T", entity)
	}
	return product, nil
}

// UpdateProduct updates an existing product
func (s *ProductService) UpdateProduct(ctx context.Context, product *Product) (*Product, error) {
	entity, err := s.Update(ctx, product)
	if err != nil {
		return nil, err
	}
	p, ok := entity.(*Product)
	if !ok {
		return nil, fmt.Errorf("unexpected entity type: %T", entity)
	}
	return p, nil
}

// DeleteProduct deletes a product
func (s *ProductService) DeleteProduct(ctx context.Context, id string) error {
	return s.Delete(ctx, id)
}

// ListProducts lists all products with pagination
func (s *ProductService) ListProducts(ctx context.Context, limit, offset int) ([]*Product, error) {
	entities, err := s.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	products := make([]*Product, len(entities))
	for i, entity := range entities {
		p, ok := entity.(*Product)
		if !ok {
			return nil, fmt.Errorf("unexpected entity type: %T", entity)
		}
		products[i] = p
	}
	return products, nil
}

// QueryProducts queries products with filtering
func (s *ProductService) QueryProducts(ctx context.Context, filter map[string]any, limit, offset int) ([]*Product, error) {
	entities, err := s.Query(ctx, filter, limit, offset)
	if err != nil {
		return nil, err
	}

	products := make([]*Product, len(entities))
	for i, entity := range entities {
		p, ok := entity.(*Product)
		if !ok {
			return nil, fmt.Errorf("unexpected entity type: %T", entity)
		}
		products[i] = p
	}
	return products, nil
}

// GetProductsByCategory retrieves products by category
func (s *ProductService) GetProductsByCategory(ctx context.Context, category string, limit, offset int) ([]*Product, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("product-service:category:%s:limit:%d:offset:%d", category, limit, offset)
	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != nil {
			products, ok := cached.([]*Product)
			if !ok {
				return nil, fmt.Errorf("unexpected cached type: %T", cached)
			}
			return products, nil
		}
	}

	// Query backend
	products, err := s.backend.GetByCategory(ctx, category, limit, offset)
	if err != nil {
		return nil, err
	}

	// Cache result
	if s.cache != nil && products != nil {
		if err := s.cache.Set(ctx, cacheKey, products, 5*time.Minute); err != nil {
			log.Printf("ProductService: cache set error for key %s: %v", cacheKey, err)
		}
	}

	return products, nil
}

// UpdateProductStock updates product stock
func (s *ProductService) UpdateProductStock(ctx context.Context, id string, quantity int64) error {
	// Update in backend
	err := s.backend.UpdateStock(ctx, id, quantity)
	if err != nil {
		return err
	}

	// Invalidate cache for this product
	if s.cache != nil {
		cacheKey := fmt.Sprintf("product-service:read:%s", id)
		if err := s.cache.Delete(ctx, cacheKey); err != nil {
			log.Printf("ProductService: cache delete error for key %s: %v", cacheKey, err)
		}
	}

	return nil
}

// productServiceAdapter adapts ProductBackend to ServiceBackend
type productServiceAdapter struct {
	backend ProductBackend
}

func (a *productServiceAdapter) Create(ctx context.Context, entity Entity) (Entity, error) {
	product, ok := entity.(*Product)
	if !ok {
		return nil, fmt.Errorf("expected *Product, got %T", entity)
	}
	return a.backend.Create(ctx, product)
}

func (a *productServiceAdapter) Read(ctx context.Context, id string) (Entity, error) {
	return a.backend.Read(ctx, id)
}

func (a *productServiceAdapter) Update(ctx context.Context, entity Entity) (Entity, error) {
	product, ok := entity.(*Product)
	if !ok {
		return nil, fmt.Errorf("expected *Product, got %T", entity)
	}
	return a.backend.Update(ctx, product)
}

func (a *productServiceAdapter) Delete(ctx context.Context, id string) error {
	return a.backend.Delete(ctx, id)
}

func (a *productServiceAdapter) List(ctx context.Context, limit, offset int) ([]Entity, error) {
	products, err := a.backend.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	entities := make([]Entity, len(products))
	for i, product := range products {
		entities[i] = product
	}
	return entities, nil
}

func (a *productServiceAdapter) Query(ctx context.Context, filter map[string]any, limit, offset int) ([]Entity, error) {
	products, err := a.backend.Query(ctx, filter, limit, offset)
	if err != nil {
		return nil, err
	}

	entities := make([]Entity, len(products))
	for i, product := range products {
		entities[i] = product
	}
	return entities, nil
}
