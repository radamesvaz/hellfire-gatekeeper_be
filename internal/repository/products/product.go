package products

import (
	"database/sql"
	"fmt"

	"github.com/radamesvaz/bakery-app/internal/errors"
	pModel "github.com/radamesvaz/bakery-app/model/products"
)

type ProductRepository struct {
	DB *sql.DB
}

// GetAll gets all the products from the table
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

// Getting a product by its ID
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
			return product, errors.NewNotFound(errors.ErrProductNotFound)
		}
		fmt.Printf("Error retrieving the product: %v", err)
		return product, errors.NewNotFound(errors.ErrCouldNotGetTheProduct)
	}

	return product, nil
}

// Creating a product
func (r *ProductRepository) CreateProduct(
	productName string,
	productDescription string,
	productPrice float64,
	available bool,
) (pModel.Product, error) {
	fmt.Printf("Creating product: %v", productName)

	createdProduct := pModel.Product{}

	if productName == "" || productDescription == "" || productPrice == 0 {
		return createdProduct, errors.NewBadRequest(errors.ErrCreatingProduct)
	}

	row := r.DB.QueryRow(
		`INSERT INTO products 
		(name, description, price, available) 
		VALUES (?, ?, ?, ?) 
		RETURNING 
		id_product, 
		name, 
		description, 
		price,
		available, 
		created_on`,
		productName, productDescription, productPrice, available)
	err := row.Scan(
		&createdProduct.ID,
		&createdProduct.Name,
		&createdProduct.Description,
		&createdProduct.Price,
		&createdProduct.Available,
		&createdProduct.CreatedOn,
	)

	if err != nil {
		return createdProduct, errors.NewInternalServerError(errors.ErrCreatingProduct)
	}

	return createdProduct, nil

}
