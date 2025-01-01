package auth_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zyrterviews/chirpy/internal/auth"
)

func TestHashPassword(t *testing.T) {
	t.Parallel()

	t.Run(
		"should return a different string (hash) than the given plain text password",
		func(t *testing.T) {
			t.Parallel()

			pwd := "strongpassword123"

			hash, err := auth.HashPassword(pwd)
			if err != nil {
				t.Fatalf("unexpected error during hashing: %v", err)
			}

			if hash == pwd {
				t.Fatal(
					"expected hash to be different than plain text password",
				)
			}
		},
	)

	t.Run(
		"should return an error if the password length is greater than 72 bytes",
		func(t *testing.T) {
			t.Parallel()

			pwd := "This is a string that contains more than seventy-two bytes of data for testing purposes right now."

			if _, err := auth.HashPassword(pwd); err == nil {
				t.Fatal("expected an error, but got none")
			}
		},
	)

	t.Run(
		"should return an error if the password string is empty",
		func(t *testing.T) {
			t.Parallel()

			if _, err := auth.HashPassword(""); err == nil {
				t.Fatal("expected an error, but got none")
			}
		},
	)
}

func TestCheckPasswordHash(t *testing.T) {
	t.Parallel()

	t.Run(
		"should successfully check password hash",
		func(t *testing.T) {
			t.Parallel()

			pwd := "strongpassword123"

			hash, err := auth.HashPassword(pwd)
			if err != nil {
				t.Fatalf("unexpected error during hashing: %v", err)
			}

			if err := auth.CheckPasswordHash(pwd, hash); err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		},
	)

	t.Run(
		"should return an error if the password length is greater than 72 bytes",
		func(t *testing.T) {
			t.Parallel()

			pwd := "This is a string that contains more than seventy-two bytes of data for testing purposes right now."

			if err := auth.CheckPasswordHash(pwd, "irrelevant"); err == nil {
				t.Fatal("expected an error, but got none")
			}
		},
	)

	t.Run(
		"should return an error if the hash string is empty",
		func(t *testing.T) {
			t.Parallel()

			if err := auth.CheckPasswordHash("123", ""); err == nil {
				t.Fatal("expected an error, but got none")
			}
		},
	)
}

func TestMakeJWT(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	secret := "secret"

	t.Run("should return a valid JWT", func(t *testing.T) {
		t.Parallel()

		token, err := auth.MakeJWT(id, secret, 5*time.Second)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if token == "" {
			t.Fatal("expected a token, got an empty string")
		}
	})

	t.Run("should error if the user ID is nil", func(t *testing.T) {
		t.Parallel()

		token, err := auth.MakeJWT(uuid.Nil, secret, 5*time.Second)
		if err == nil {
			t.Fatal("expected an error, but got none")
		}

		if token != "" {
			t.Fatalf("expected an empty string, got: %q", token)
		}
	})

	t.Run("should error if the secret string is empty", func(t *testing.T) {
		t.Parallel()

		id := uuid.New()

		token, err := auth.MakeJWT(id, "", 5*time.Second)
		if err == nil {
			t.Fatal("expected an error, but got none")
		}

		if token != "" {
			t.Fatalf("expected an empty string, got: %q", token)
		}
	})
}

func TestValidateJWT(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	secret := "secret"

	t.Run("should return the expected UUID", func(t *testing.T) {
		t.Parallel()

		token, _ := auth.MakeJWT(id, secret, 5*time.Second)

		resID, err := auth.ValidateJWT(token, secret)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resID != id {
			t.Fatalf("expected result ID to match %q, got %q", id, resID)
		}
	})

	t.Run("should error if signing key is invalid", func(t *testing.T) {
		t.Parallel()

		token, _ := auth.MakeJWT(id, secret, 5*time.Second)

		resID, err := auth.ValidateJWT(token, "wrongkey")
		if err == nil {
			t.Fatal("expected an error, but got none")
		}

		if resID != uuid.Nil {
			t.Fatalf("expected a nil UUID, got: %q (original: %q)", resID, id)
		}
	})

	t.Run("should error if JWT is expired", func(t *testing.T) {
		t.Parallel()

		token, _ := auth.MakeJWT(id, secret, 1*time.Nanosecond)

		time.Sleep(1 * time.Nanosecond)

		resID, err := auth.ValidateJWT(token, secret)
		if err == nil {
			t.Fatal("expected an error, but got none")
		}

		if resID != uuid.Nil {
			t.Fatalf("expected a nil UUID, got: %q (original: %q)", resID, id)
		}
	})

	t.Run("should error if the token string is empty", func(t *testing.T) {
		t.Parallel()

		resID, err := auth.ValidateJWT("", secret)
		if err == nil {
			t.Fatal("expected an error, but got none")
		}

		if resID != uuid.Nil {
			t.Fatalf("expected a nil UUID, got: %q (original: %q)", resID, id)
		}
	})

	t.Run("should error if the secret string is empty", func(t *testing.T) {
		t.Parallel()

		resID, err := auth.ValidateJWT("123", "")
		if err == nil {
			t.Fatal("expected an error, but got none")
		}

		if resID != uuid.Nil {
			t.Fatalf("expected a nil UUID, got: %q (original: %q)", resID, id)
		}
	})
}

func TestGetBearerToken(t *testing.T) {
	t.Parallel()

	t.Run("should return the bearer token", func(t *testing.T) {
		t.Parallel()

		token := "123"
		headers := http.Header{}
		headers.Set("Authorization", "Bearer "+token)

		tk, err := auth.GetBearerToken(headers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tk != token {
			t.Fatalf("unexpected token: got %q, want %q", tk, token)
		}
	})

	t.Run("should error if no bearer token is present", func(t *testing.T) {
		t.Parallel()

		headers := http.Header{}

		tk, err := auth.GetBearerToken(headers)
		if err == nil {
			t.Fatal("expected an error, but got none")
		}

		if tk != "" {
			t.Fatalf("expected an empty string, got %q", tk)
		}
	})
}

func TestMakeRefreshToken(t *testing.T) {
	token, err := auth.MakeRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token == "" {
		t.Fatal("expected token, got an empty string")
	}
}
