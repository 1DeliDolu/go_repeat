package products

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]Product, error) {
	return s.repo.ListActive(ctx, limit, offset)
}

func (s *Service) Detail(ctx context.Context, slug string) (Product, error) {
	return s.repo.GetBySlug(ctx, slug)
}
