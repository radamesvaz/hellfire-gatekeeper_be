package validators

import (
	"testing"

	oModel "github.com/radamesvaz/bakery-app/model/orders"
)

func TestValidateCreateOrderPayload(t *testing.T) {
	type args struct {
		payload oModel.CreateOrderPayload
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateCreateOrderPayload(tt.args.payload); (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreateOrderPayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
