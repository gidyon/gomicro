package grpcauth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGenTokenV2(t *testing.T) {
	api := NewAPI([]byte("secret"), "issuer", "audience")
	payload := &Payload{
		ID:        "user-1",
		ProjectID: "project-1",
	}
	claims := &Claims{
		Payload: payload,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		},
	}

	// This is what GenTokenFromClaims does
	expires := time.Now().Add(2 * time.Hour).Unix()
	tokenStr, err := api.genTokenV2(context.Background(), claims, expires, api.signingKey)
	if err != nil {
		t.Fatalf("failed to gen token: %v", err)
	}

	parsedClaims, err := api.parseToken(tokenStr, api.signingKey)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	if parsedClaims.ExpiresAt != expires {
		t.Errorf("expected ExpiresAt %d, got %d", expires, parsedClaims.ExpiresAt)
	}
}

func TestAuthenticator_NilPayload(t *testing.T) {
	api := NewAPI([]byte("secret"), "issuer", "audience")

	// Create a token with NO payload fields
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Subject: "test",
	})
	tokenStr, _ := token.SignedString([]byte("secret"))

	claims, err := api.parseToken(tokenStr, []byte("secret"))
	if err != nil {
		t.Fatalf("Parse token failed: %v", err)
	}

	if claims.Payload == nil {
		t.Error("claims.Payload is nil, expected pre-allocated Payload")
	}

	// This should NOT panic
	_ = claims.ID
}

func TestAuthorizeGroups(t *testing.T) {
	api := NewAPI([]byte("secret"), "issuer", "audience")

	tests := []struct {
		name          string
		claims        *Claims
		allowedGroups []string
		wantErr       bool
		errCode       codes.Code
	}{
		{
			name: "Authorized via Group",
			claims: &Claims{
				Payload: &Payload{Group: "ADMIN"},
			},
			allowedGroups: []string{"ADMIN", "USER"},
			wantErr:       false,
		},
		{
			name: "Authorized via Roles",
			claims: &Claims{
				Payload: &Payload{Group: "GUEST", Roles: []string{"ADMIN"}},
			},
			allowedGroups: []string{"ADMIN"},
			wantErr:       false,
		},
		{
			name: "Unauthorized",
			claims: &Claims{
				Payload: &Payload{Group: "GUEST", Roles: []string{"USER"}},
			},
			allowedGroups: []string{"ADMIN"},
			wantErr:       true,
			errCode:       codes.PermissionDenied,
		},
		{
			name:          "No claims",
			claims:        nil,
			allowedGroups: []string{"ADMIN"},
			wantErr:       true,
			errCode:       codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.claims != nil {
				ctx = context.WithValue(ctx, claimsKey, tt.claims)
			}

			payload, err := api.AuthorizeGroups(ctx, tt.allowedGroups...)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthorizeGroups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if status.Code(err) != tt.errCode {
					t.Errorf("AuthorizeGroups() error code = %v, want %v", status.Code(err), tt.errCode)
				}
				if tt.errCode == codes.PermissionDenied && !strings.Contains(err.Error(), "allowed list") {
					t.Errorf("AuthorizeGroups() error message should contain 'allowed list', got: %v", err)
				}
			} else if payload == nil {
				t.Error("AuthorizeGroups() returned nil payload on success")
			}
		})
	}
}

func TestAuthorizeIds(t *testing.T) {
	api := NewAPI([]byte("secret"), "issuer", "audience")

	tests := []struct {
		name       string
		claims     *Claims
		allowedIds []string
		wantErr    bool
		errCode    codes.Code
	}{
		{
			name: "Authorized",
			claims: &Claims{
				Payload: &Payload{ID: "user-1"},
			},
			allowedIds: []string{"user-1", "user-2"},
			wantErr:    false,
		},
		{
			name: "Unauthorized",
			claims: &Claims{
				Payload: &Payload{ID: "user-3"},
			},
			allowedIds: []string{"user-1", "user-2"},
			wantErr:    true,
			errCode:    codes.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), claimsKey, tt.claims)

			payload, err := api.AuthorizeIds(ctx, tt.allowedIds...)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthorizeIds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if status.Code(err) != tt.errCode {
					t.Errorf("AuthorizeIds() error code = %v, want %v", status.Code(err), tt.errCode)
				}
				if tt.errCode == codes.PermissionDenied && !strings.Contains(err.Error(), "allowed list") {
					t.Errorf("AuthorizeIds() error message should contain 'allowed list', got: %v", err)
				}
			} else if payload == nil {
				t.Error("AuthorizeIds() returned nil payload on success")
			}
		})
	}
}

type mockPayloadProvider struct {
	getPayloadFunc func(ctx context.Context, id string) (*Payload, error)
}

func (m *mockPayloadProvider) GetPayload(ctx context.Context, id string) (*Payload, error) {
	return m.getPayloadFunc(ctx, id)
}

func TestAuthenticator(t *testing.T) {
	api := NewAPI([]byte("secret"), "issuer", "audience")

	t.Run("Valid Token", func(t *testing.T) {
		tokenStr, _ := api.GenToken(context.Background(), &Payload{ID: "user-1"}, time.Now().Add(time.Hour))
		md := metadata.Pairs("authorization", "Bearer "+tokenStr)
		ctx := metadata.NewIncomingContext(context.Background(), md)

		newCtx, err := api.Authenticator(ctx)
		if err != nil {
			t.Fatalf("Authenticator() error = %v", err)
		}

		claims, err := api.GetClaims(newCtx)
		if err != nil {
			t.Fatalf("GetClaims() error = %v", err)
		}
		if claims.ID != "user-1" {
			t.Errorf("expected ID user-1, got %s", claims.ID)
		}
	})

	t.Run("Expired Token", func(t *testing.T) {
		tokenStr, _ := api.genToken(context.Background(), &Payload{ID: "user-1"}, time.Now().Add(-time.Hour).Unix())
		md := metadata.Pairs("authorization", "Bearer "+tokenStr)
		ctx := metadata.NewIncomingContext(context.Background(), md)

		_, err := api.Authenticator(ctx)
		if err == nil {
			t.Fatal("Authenticator() expected error for expired token, got nil")
		}
		if status.Code(err) != codes.Unauthenticated {
			t.Errorf("expected Unauthenticated, got %v", status.Code(err))
		}
	})

	t.Run("With PayloadProvider", func(t *testing.T) {
		api.SetPayloadProvider(&mockPayloadProvider{
			getPayloadFunc: func(ctx context.Context, id string) (*Payload, error) {
				return &Payload{ID: id, Group: "DYNAMIC_GROUP", ExternalID: "external-123"}, nil
			},
		})
		defer api.SetPayloadProvider(nil)

		tokenStr, _ := api.GenToken(context.Background(), &Payload{ID: "user-2"}, time.Now().Add(time.Hour))
		md := metadata.Pairs("authorization", "Bearer "+tokenStr)
		ctx := metadata.NewIncomingContext(context.Background(), md)

		newCtx, err := api.Authenticator(ctx)
		if err != nil {
			t.Fatalf("Authenticator() error = %v", err)
		}

		claims, _ := api.GetClaims(newCtx)
		if claims.Group != "DYNAMIC_GROUP" {
			t.Errorf("expected Group DYNAMIC_GROUP, got %s", claims.Group)
		}
		if claims.ExternalID != "external-123" {
			t.Errorf("expected ExternalID external-123, got %s", claims.ExternalID)
		}
	})
}
