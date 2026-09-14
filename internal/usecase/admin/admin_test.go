package admin_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/firereach/api/internal/domain"
	"github.com/firereach/api/internal/usecase/admin"
	"github.com/firereach/api/internal/usecase/mocks"
)

const secret = "unit-test-secret"

var errDatabaseDown = errors.New("connection refused")

// hashOf makes a real bcrypt hash at the minimum cost so the suite stays
// fast; CompareHashAndPassword accepts a hash of any cost.
func hashOf(t *testing.T, password string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(h)
}

func repoWith(a *domain.AdminUser) *mocks.AdminRepo {
	return &mocks.AdminRepo{
		GetByEmailFunc: func(ctx context.Context, email string) (*domain.AdminUser, error) {
			if a != nil && email == a.Email {
				return a, nil
			}
			return nil, domain.ErrNotFound
		},
	}
}

// ---------------------------------------------------------------- login

func TestLogin_IssuesATokenTheAuthMiddlewareAccepts(t *testing.T) {
	a := &domain.AdminUser{ID: "admin-1", Email: "a@firereach.test", PasswordHash: hashOf(t, "correct-horse-battery")}
	token, err := admin.NewLogin(repoWith(a), secret).Execute(context.Background(), "a@firereach.test", "correct-horse-battery")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	// The checks middleware.Auth makes: HMAC-signed with the shared secret,
	// with a non-empty string subject.
	parsed, err := jwt.Parse(token, func(*jwt.Token) (interface{}, error) { return []byte(secret), nil },
		jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !parsed.Valid {
		t.Fatalf("token does not verify: %v", err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["sub"] != "admin-1" {
		t.Errorf("sub = %v, want admin-1", claims["sub"])
	}
	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil {
		t.Fatalf("no exp claim: %v", err)
	}
	if d := time.Until(exp.Time); d < 23*time.Hour || d > 25*time.Hour {
		t.Errorf("token expires in %v, want about 24h", d)
	}
	if iat, err := claims.GetIssuedAt(); err != nil || iat == nil {
		t.Error("token has no iat claim")
	}
}

func TestLogin_UnknownEmailIsUnauthorized(t *testing.T) {
	_, err := admin.NewLogin(repoWith(nil), secret).Execute(context.Background(), "nobody@firereach.test", "a-long-password")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestLogin_WrongPasswordIsUnauthorized(t *testing.T) {
	a := &domain.AdminUser{ID: "admin-1", Email: "a@firereach.test", PasswordHash: hashOf(t, "correct-horse-battery")}
	_, err := admin.NewLogin(repoWith(a), secret).Execute(context.Background(), "a@firereach.test", "wrong-password")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

// The first bug this refactor fixes: a database failure used to be reported
// as "invalid credentials", so an outage looked like a wrong password.
func TestLogin_DatabaseFailureIsNotReportedAsUnauthorized(t *testing.T) {
	repo := &mocks.AdminRepo{GetByEmailFunc: func(ctx context.Context, email string) (*domain.AdminUser, error) {
		return nil, errDatabaseDown
	}}
	_, err := admin.NewLogin(repo, secret).Execute(context.Background(), "a@firereach.test", "a-long-password")
	if errors.Is(err, domain.ErrUnauthorized) {
		t.Fatal("a database failure was reported as bad credentials")
	}
	if !errors.Is(err, errDatabaseDown) {
		t.Fatalf("err = %v, want the database error preserved", err)
	}
}

// --------------------------------------------------------- create admin

func TestCreateAdmin_StoresAHashThatMatchesThePassword(t *testing.T) {
	var gotEmail, gotHash string
	repo := &mocks.AdminRepo{CreateFunc: func(ctx context.Context, email, hash string) (*domain.AdminUser, error) {
		gotEmail, gotHash = email, hash
		return &domain.AdminUser{ID: "admin-2", Email: email}, nil
	}}
	created, err := admin.NewCreateAdmin(repo).Execute(context.Background(), "b@firereach.test", "a-long-password")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID != "admin-2" || gotEmail != "b@firereach.test" {
		t.Fatalf("created %+v with email %q", created, gotEmail)
	}
	if gotHash == "a-long-password" {
		t.Fatal("the plaintext password was stored")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(gotHash), []byte("a-long-password")); err != nil {
		t.Fatalf("the stored hash does not match the password: %v", err)
	}
}

func TestCreateAdmin_DuplicateEmailIsAlreadyExists(t *testing.T) {
	repo := &mocks.AdminRepo{CreateFunc: func(ctx context.Context, email, hash string) (*domain.AdminUser, error) {
		return nil, domain.ErrAlreadyExists
	}}
	_, err := admin.NewCreateAdmin(repo).Execute(context.Background(), "b@firereach.test", "a-long-password")
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("err = %v, want ErrAlreadyExists", err)
	}
}

// bcrypt reads at most 72 bytes. A longer password is the caller's mistake, not
// a server fault, and nothing may be written.
func TestCreateAdmin_PasswordOver72BytesIsInvalidInput(t *testing.T) {
	repo := &mocks.AdminRepo{CreateFunc: func(ctx context.Context, email, hash string) (*domain.AdminUser, error) {
		t.Fatal("nothing may be stored when the password cannot be hashed")
		return nil, nil
	}}
	_, err := admin.NewCreateAdmin(repo).Execute(context.Background(), "b@firereach.test", strings.Repeat("x", 73))
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
}

// ----------------------------------------------------------------- setup

func TestSetup_CompletedReflectsWhetherAnyAdminExists(t *testing.T) {
	for _, tc := range []struct {
		count int
		want  bool
	}{{0, false}, {1, true}, {3, true}} {
		count := tc.count
		repo := &mocks.AdminRepo{CountFunc: func(ctx context.Context) (int, error) { return count, nil }}
		got, err := admin.NewSetup(repo).Completed(context.Background())
		if err != nil || got != tc.want {
			t.Errorf("with %d admins: Completed = %v (err %v), want %v", tc.count, got, err, tc.want)
		}
	}
}

func TestSetup_CompletedPassesADatabaseFailureOn(t *testing.T) {
	repo := &mocks.AdminRepo{CountFunc: func(ctx context.Context) (int, error) { return 0, errDatabaseDown }}
	if _, err := admin.NewSetup(repo).Completed(context.Background()); !errors.Is(err, errDatabaseDown) {
		t.Fatalf("err = %v, want the database error", err)
	}
}

// Setup must go through the atomic CreateFirst, never the plain Create that
// the race used to slip through.
func TestSetup_CreatesThroughTheAtomicPath(t *testing.T) {
	var gotHash string
	repo := &mocks.AdminRepo{
		CreateFirstFunc: func(ctx context.Context, email, hash string) (*domain.AdminUser, error) {
			gotHash = hash
			return &domain.AdminUser{ID: "admin-1", Email: email}, nil
		},
		CreateFunc: func(ctx context.Context, email, hash string) (*domain.AdminUser, error) {
			t.Fatal("setup used the non-atomic Create")
			return nil, nil
		},
	}
	created, err := admin.NewSetup(repo).Execute(context.Background(), "first@firereach.test", "a-long-password")
	if err != nil || created.ID != "admin-1" {
		t.Fatalf("setup: %+v, %v", created, err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(gotHash), []byte("a-long-password")); err != nil {
		t.Fatalf("the stored hash does not match the password: %v", err)
	}
}

func TestSetup_LosingTheRaceIsSetupCompleted(t *testing.T) {
	repo := &mocks.AdminRepo{CreateFirstFunc: func(ctx context.Context, email, hash string) (*domain.AdminUser, error) {
		return nil, domain.ErrSetupCompleted
	}}
	_, err := admin.NewSetup(repo).Execute(context.Background(), "first@firereach.test", "a-long-password")
	if !errors.Is(err, domain.ErrSetupCompleted) {
		t.Fatalf("err = %v, want ErrSetupCompleted", err)
	}
}

func TestSetup_PasswordOver72BytesIsInvalidInput(t *testing.T) {
	repo := &mocks.AdminRepo{CreateFirstFunc: func(ctx context.Context, email, hash string) (*domain.AdminUser, error) {
		t.Fatal("nothing may be stored when the password cannot be hashed")
		return nil, nil
	}}
	_, err := admin.NewSetup(repo).Execute(context.Background(), "first@firereach.test", strings.Repeat("x", 73))
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
}
