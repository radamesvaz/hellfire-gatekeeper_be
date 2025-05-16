package validators

import (
	"testing"

	oModel "github.com/radamesvaz/bakery-app/model/orders"
)

func TestValidateCreateOrderPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload oModel.CreateOrderPayload
		wantErr bool
	}{
		{
			name: "Happy path: All fields are valid",
			payload: oModel.CreateOrderPayload{
				Name:         "usuario uno",
				Email:        "usuario1@gmail.com",
				Phone:        "55-555",
				Note:         "entregar temprano",
				DeliveryDate: "2025-05-20",
				Items: []oModel.CreateOrderItemInput{
					{
						IdProduct: 1,
						Quantity:  1,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Sad path: missing name",
			payload: oModel.CreateOrderPayload{
				Name:         "",
				Email:        "usuario1@gmail.com",
				Phone:        "55-555",
				Note:         "entregar temprano",
				DeliveryDate: "2025-05-20",
				Items: []oModel.CreateOrderItemInput{
					{
						IdProduct: 1,
						Quantity:  1,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Sad path: empty email",
			payload: oModel.CreateOrderPayload{
				Name:         "usuario uno",
				Email:        "",
				Phone:        "55-555",
				Note:         "entregar temprano",
				DeliveryDate: "2025-05-20",
				Items: []oModel.CreateOrderItemInput{
					{IdProduct: 1, Quantity: 1},
				},
			},
			wantErr: true,
		},
		{
			name: "Sad path: empty phone",
			payload: oModel.CreateOrderPayload{
				Name:         "usuario uno",
				Email:        "usuario1@gmail.com",
				Phone:        "",
				Note:         "entregar temprano",
				DeliveryDate: "2025-05-20",
				Items: []oModel.CreateOrderItemInput{
					{IdProduct: 1, Quantity: 1},
				},
			},
			wantErr: true,
		},
		{
			name: "Sad path: empty items",
			payload: oModel.CreateOrderPayload{
				Name:         "usuario uno",
				Email:        "usuario1@gmail.com",
				Phone:        "55-555",
				Note:         "entregar temprano",
				DeliveryDate: "2025-05-20",
				Items:        []oModel.CreateOrderItemInput{},
			},
			wantErr: true,
		},
		{
			name: "Sad path: item with zero quantity",
			payload: oModel.CreateOrderPayload{
				Name:         "usuario uno",
				Email:        "usuario1@gmail.com",
				Phone:        "55-555",
				Note:         "entregar temprano",
				DeliveryDate: "2025-05-20",
				Items: []oModel.CreateOrderItemInput{
					{IdProduct: 1, Quantity: 0},
				},
			},
			wantErr: true,
		},
		{
			name: "Sad path: item with zero id_product",
			payload: oModel.CreateOrderPayload{
				Name:         "usuario uno",
				Email:        "usuario1@gmail.com",
				Phone:        "55-555",
				Note:         "entregar temprano",
				DeliveryDate: "2025-05-20",
				Items: []oModel.CreateOrderItemInput{
					{IdProduct: 0, Quantity: 2},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				if err := ValidateCreateOrderPayload(tt.payload); err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err := ValidateCreateOrderPayload(tt.payload); err != nil {
					t.Errorf("did not expect error but got: %v", err)
				}
			}
		})
	}
}
