package orders

import (
	"context"
	"testing"

	"github.com/radamesvaz/bakery-app/internal/errors"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
)

type MockUserRepo struct {
	ShouldCreate   bool
	UserWasCreated bool
	CreateUserErr  error
}

func (m *MockUserRepo) GetUserByEmail(email string) (uModel.User, error) {
	if m.ShouldCreate {
		return uModel.User{}, errors.ErrUserNotFound
	}
	return uModel.User{ID: 1, Email: email}, nil
}

func (m *MockUserRepo) CreateUser(ctx context.Context, input uModel.CreateUserRequest) (uint64, error) {
	m.UserWasCreated = true
	return 2, nil
}

func TestFindOrCreateUser_CreatesUserIfNotExists(t *testing.T) {
	mockRepo := &MockUserRepo{ShouldCreate: true}
	service := Creator{UserRepo: mockRepo}

	ctx := context.Background()
	input := oModel.CreateOrderPayload{
		Name:  "Nuevo Cliente",
		Email: "nuevo@example.com",
		Phone: "12345678",
	}

	user, err := service.GetOrCreateUser(ctx, input)

	assert.NoError(t, err)
	assert.Equal(t, uint64(2), user.ID)
	assert.True(t, mockRepo.UserWasCreated)
}

func TestFindOrCreateUser_DoesNotCreateAnUser(t *testing.T) {
	mockRepo := &MockUserRepo{ShouldCreate: false}
	service := Creator{UserRepo: mockRepo}

	ctx := context.Background()
	input := oModel.CreateOrderPayload{
		Name:  "Existente Cliente",
		Email: "existente@example.com",
		Phone: "12345678",
	}

	user, err := service.GetOrCreateUser(ctx, input)

	assert.NoError(t, err)
	assert.Equal(t, uint64(1), user.ID)
	assert.False(t, mockRepo.UserWasCreated)
}
