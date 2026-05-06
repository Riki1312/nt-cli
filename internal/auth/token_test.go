package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDeleteCredentialsRemovesSavedOAuthState(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := SaveToken(&Token{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveToken returned error: %v", err)
	}
	if err := SaveClientRegistration(&ClientRegistration{ClientID: "client"}); err != nil {
		t.Fatalf("SaveClientRegistration returned error: %v", err)
	}

	if err := DeleteCredentials(); err != nil {
		t.Fatalf("DeleteCredentials returned error: %v", err)
	}

	configPath := filepath.Join(os.Getenv("HOME"), ".config", "nt")
	for _, name := range []string{"token.json", "client.json"} {
		if _, err := os.Stat(filepath.Join(configPath, name)); !os.IsNotExist(err) {
			t.Fatalf("%s still exists or stat failed with non-not-exist error: %v", name, err)
		}
	}
}
