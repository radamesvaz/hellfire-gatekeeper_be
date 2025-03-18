package handlers

import (
	"encoding/json"
	"net/http"

	productsRepository "github.com/radamesvaz/bakery-app/repository/products"
)

type ProductHandler struct {
	Repo *productsRepository.ProductRepository
}

func (h *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.Repo.GetAll()
	if err != nil {
		http.Error(w, "Error obteniendo productos", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}
