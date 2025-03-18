package repository

import (
	"database/sql"
	"fmt"

	pModel "github.com/radamesvaz/bakery-app/model/products"
)

type ProductRepository struct {
	DB *sql.DB
}

func (r *ProductRepository) GetAll() ([]pModel.Product, error) {
	fmt.Println(
		"Getting all products",
	)
	rows, err := r.DB.Query("SELECT id_product, name, description, price, available, created_on FROM products")
	if err != nil {
		fmt.Printf("Error getting the products: %v", err)
		return nil, err
	}
	defer rows.Close()

	var products []pModel.Product
	for rows.Next() {
		var product pModel.Product
		if err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.Price,
			&product.Available,
			&product.CreatedOn,
		); err != nil {
			fmt.Printf("Error mapping the products: %v", err)
			return nil, err
		}
		products = append(products, product)
	}
	return products, nil
}

func (r *ProductRepository) GetProductByID(idProduct uint64) (pModel.Product, error) {
	fmt.Printf(
		"Getting product by id = %v",
		idProduct,
	)

	product := pModel.Product{}

	err := r.DB.QueryRow(
		"SELECT id_product, name, description, price, available, created_on FROM products WHERE id_product = ?",
		idProduct,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.Available,
		&product.CreatedOn,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Product not found")
			return product, nil
		}
		fmt.Printf("Error retrieving the product: %v", err)
		return product, nil
	}

	return product, nil
}
