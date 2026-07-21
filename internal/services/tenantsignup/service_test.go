package tenantsignup

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	tenantSignupRepo "github.com/radamesvaz/bakery-app/internal/repository/tenantsignup"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	"github.com/radamesvaz/bakery-app/internal/services/email"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingSignupEmailSender struct {
	last  email.TenantSignupCodePayload
	calls int
	err   error
}

func (r *recordingSignupEmailSender) SendPasswordReset(context.Context, email.PasswordResetPayload) error {
	return nil
}

func (r *recordingSignupEmailSender) SendTenantInvitation(context.Context, email.TenantInvitationPayload) error {
	return nil
}

func (r *recordingSignupEmailSender) SendTenantSignupCode(_ context.Context, payload email.TenantSignupCodePayload) error {
	r.calls++
	r.last = payload
	return r.err
}

func TestTenantSignupService_CreateSignupCode_SendsEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sender := &recordingSignupEmailSender{}
	svc := &TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authService.New("testingsecret", 60),
		EmailSender: sender,
		AppBaseURL:  "https://admin.example.com",
	}

	mock.ExpectQuery(`INSERT INTO tenant_signup_codes`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), uint64(99), "ana@panaderia.com", "notes").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint64(7)))

	resp, err := svc.CreateSignupCode(context.Background(), uint64(uModel.UserRoleSuperAdmin), 99, authModel.CreateSignupCodeRequest{
		Email:            "Ana@Panaderia.com",
		ExpiresInMinutes: 60,
		Notes:            "notes",
	})
	require.NoError(t, err)
	assert.Equal(t, uint64(7), resp.ID)
	assert.NotEmpty(t, resp.Code)
	assert.Equal(t, "ana@panaderia.com", resp.Email)
	assert.Equal(t, "Signup code sent successfully", resp.Message)
	assert.Equal(t, 1, sender.calls)
	assert.Equal(t, "ana@panaderia.com", sender.last.ToEmail)
	assert.Contains(t, sender.last.RegisterURL, "https://admin.example.com/tenant-register?code=")
	assert.Contains(t, sender.last.RegisterURL, resp.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantSignupService_CreateSignupCode_EmailSendFailure_DoesNotPersist(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sender := &recordingSignupEmailSender{err: errors.New("brevo down")}
	svc := &TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authService.New("testingsecret", 60),
		EmailSender: sender,
		AppBaseURL:  "https://admin.example.com",
	}

	_, err = svc.CreateSignupCode(context.Background(), uint64(uModel.UserRoleSuperAdmin), 99, authModel.CreateSignupCodeRequest{
		Email: "ana@panaderia.com",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmailDeliveryFailed)
	assert.Equal(t, 1, sender.calls)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantSignupService_CreateSignupCode_MissingAppBaseURL(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := &TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authService.New("testingsecret", 60),
		EmailSender: &recordingSignupEmailSender{},
		AppBaseURL:  "",
	}

	_, err = svc.CreateSignupCode(context.Background(), uint64(uModel.UserRoleSuperAdmin), 99, authModel.CreateSignupCodeRequest{
		Email: "ana@panaderia.com",
	})
	require.ErrorIs(t, err, ErrAppBaseURLRequired)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantSignupService_CreateSignupCode_Forbidden(t *testing.T) {
	svc := &TenantSignupService{
		AuthService: authService.New("testingsecret", 60),
		EmailSender: &recordingSignupEmailSender{},
		AppBaseURL:  "https://admin.example.com",
	}
	_, err := svc.CreateSignupCode(context.Background(), uint64(uModel.UserRoleAdmin), 1, authModel.CreateSignupCodeRequest{
		Email: "ana@panaderia.com",
	})
	require.ErrorIs(t, err, ErrForbidden)
}

func TestTenantSignupService_CreateSignupCode_DefaultTTL(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sender := &recordingSignupEmailSender{}
	svc := &TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authService.New("testingsecret", 60),
		EmailSender: sender,
		AppBaseURL:  "https://admin.example.com",
	}

	mock.ExpectQuery(`INSERT INTO tenant_signup_codes`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), uint64(99), "ana@panaderia.com", "").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint64(7)))

	resp, err := svc.CreateSignupCode(context.Background(), uint64(uModel.UserRoleSuperAdmin), 99, authModel.CreateSignupCodeRequest{
		Email: "ana@panaderia.com",
	})
	require.NoError(t, err)
	want := time.Now().UTC().Add(120 * time.Minute)
	assert.WithinDuration(t, want, resp.ExpiresAt, 3*time.Minute)
	assert.NoError(t, mock.ExpectationsWereMet())
}
