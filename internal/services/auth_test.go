package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/sbilibin2017/gw-currency-wallet/internal/models"
	"github.com/sbilibin2017/gw-currency-wallet/internal/services"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReader := services.NewMockUserReader(ctrl)
	mockWriter := services.NewMockUserWriter(ctrl)
	mockJWT := services.NewMockJWTGenerator(ctrl)

	svc := services.NewAuthService(mockReader, mockWriter, mockJWT)

	tests := []struct {
		name         string
		username     string
		password     string
		email        string
		existingUser *models.UserDB
		readerErr    error
		writerErr    error
		wantErr      error
	}{
		{
			name:         "successful registration",
			username:     "alice",
			password:     "pass123",
			email:        "alice@example.com",
			existingUser: nil,
			wantErr:      nil,
		},
		{
			name:         "user already exists",
			username:     "bob",
			password:     "pass123",
			email:        "bob@example.com",
			existingUser: &models.UserDB{UserID: uuid.New()},
			wantErr:      services.ErrUserAlreadyExists,
		},
		{
			name:      "reader error",
			username:  "eve",
			password:  "pass123",
			email:     "eve@example.com",
			readerErr: errors.New("db error"),
			wantErr:   errors.New("db error"),
		},
		{
			name:      "writer error",
			username:  "carol",
			password:  "pass123",
			email:     "carol@example.com",
			writerErr: errors.New("save error"),
			wantErr:   errors.New("save error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReader.EXPECT().
				GetByUsernameOrEmail(gomock.Any(), &tt.username, &tt.email).
				Return(tt.existingUser, tt.readerErr)

			if tt.existingUser == nil && tt.readerErr == nil {
				mockWriter.EXPECT().
					Save(gomock.Any(), tt.username, gomock.Any(), tt.email).
					Return(tt.writerErr)
			}

			err := svc.Register(context.Background(), tt.username, tt.password, tt.email)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReader := services.NewMockUserReader(ctrl)
	mockWriter := services.NewMockUserWriter(ctrl)
	mockJWT := services.NewMockJWTGenerator(ctrl)

	svc := services.NewAuthService(mockReader, mockWriter, mockJWT)

	password := "secret"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	userID := uuid.New()

	tests := []struct {
		name      string
		username  string
		user      *models.UserDB
		readerErr error
		jwtErr    error
		wantErr   error
		expectJWT string
		loginPass string
	}{
		{
			name:      "successful login",
			username:  "alice",
			user:      &models.UserDB{UserID: userID, Username: "alice", PasswordHash: string(hashed)},
			expectJWT: "token123",
			loginPass: password,
		},
		{
			name:      "user does not exist",
			username:  "bob",
			user:      nil,
			wantErr:   services.ErrUserDoesNotExist,
			loginPass: password,
		},
		{
			name:      "invalid password",
			username:  "carol",
			user:      &models.UserDB{UserID: uuid.New(), Username: "carol", PasswordHash: string(hashed)},
			wantErr:   services.ErrInvalidCredentials,
			loginPass: "wrongpass",
		},
		{
			name:      "reader error",
			username:  "eve",
			user:      nil,
			readerErr: errors.New("db error"),
			wantErr:   errors.New("db error"),
			loginPass: password,
		},
		{
			name:      "JWT generation error",
			username:  "dan",
			user:      &models.UserDB{UserID: userID, Username: "dan", PasswordHash: string(hashed)},
			jwtErr:    errors.New("jwt error"),
			wantErr:   errors.New("jwt error"),
			loginPass: password,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReader.EXPECT().
				GetByUsernameOrEmail(gomock.Any(), &tt.username, (*string)(nil)).
				Return(tt.user, tt.readerErr)

			if tt.user != nil && tt.readerErr == nil && tt.loginPass == password {
				mockJWT.EXPECT().
					Generate(gomock.Any(), tt.user.UserID).
					Return(tt.expectJWT, tt.jwtErr)
			}

			token, err := svc.Login(context.Background(), tt.username, tt.loginPass)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectJWT, token)
			}
		})
	}
}
