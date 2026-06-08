package usecase

import (
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
)

type ProductCo struct {
	ID          string
	Name        string
	Description string
	Price       int64
	Stock       int
}

func ListProducts(ctx fwusecase.Context) ([]ProductCo, error) {
	products, err := models.ListProducts(ctx.Std())
	if err != nil {
		return nil, fwusecase.E(fwusecase.CodeInternal, "failed to load products", err)
	}

	result := make([]ProductCo, 0, len(products))
	for i := range products {
		result = append(result, productCoFromModel(&products[i]))
	}
	return result, nil
}

func productCoFromModel(product *models.Product) ProductCo {
	if product == nil {
		return ProductCo{}
	}

	return ProductCo{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
	}
}
