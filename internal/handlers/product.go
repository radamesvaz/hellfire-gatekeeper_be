package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	productsRepository "github.com/radamesvaz/bakery-app/internal/repository/products"
)

type ProductHandler struct {
	Repo *productsRepository.ProductRepository
}

func (h *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	allProducts, err := h.Repo.GetAll()
	if err != nil {
		http.Error(w, "Error getting products", http.StatusInternalServerError)
		return
	}

	response := []productsRepository.ProductResponse{}

	for _, product := range allProducts {
		entry := productsRepository.Marshal(&product)

		response = append(response, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idProduct, err := strconv.ParseUint(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	product, err := h.Repo.GetProductByID(idProduct)
	if err != nil {
		http.Error(w, "Error getting product", http.StatusInternalServerError)
		return
	}

	response := productsRepository.Marshal(&product)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
