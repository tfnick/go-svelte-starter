package routes

import (
	"github.com/labstack/echo/v4"
	fwcontext "github.com/tfnick/go-svelte-starter/api/framework/http/context"
	httpresponse "github.com/tfnick/go-svelte-starter/api/framework/http/response"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

type ProductResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Price       int64  `json:"price"`
	Stock       int    `json:"stock"`
}

func ToProductResponse(product usecase.ProductCo) ProductResponse {
	return ProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
	}
}

func ToProductResponses(products []usecase.ProductCo) []ProductResponse {
	responses := make([]ProductResponse, 0, len(products))
	for i := range products {
		responses = append(responses, ToProductResponse(products[i]))
	}
	return responses
}

func ListProducts(c echo.Context) error {
	ctx := fwcontext.InternalUsecaseContext(c)
	products, err := usecase.ListProducts(ctx)
	if err != nil {
		return httpresponse.InternalUsecaseError(c, err)
	}
	return httpresponse.OK(c, ToProductResponses(products))
}
