package config

import (
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	keyring.MockInit()
	os.Exit(m.Run())
}

func TestSave_StoresInKeyring(t *testing.T) {
	keyring.MockInit()
	s := NewStore("")
	creds := &Credentials{APIKey: "key1", Namespace: "acme"}

	if err := s.Save(creds); err != nil {
		t.Fatal(err)
	}

	data, err := keyring.Get(ServiceName, KeyringUser)
	if err != nil {
		t.Fatalf("expected keyring to have credentials: %v", err)
	}
	if data == "" {
		t.Error("expected non-empty keyring data")
	}
}

func TestLoad_ReturnsNilWhenNoCredentials(t *testing.T) {
	keyring.MockInit()
	s := NewStore("")
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	creds, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if creds != nil {
		t.Errorf("expected nil, got %+v", creds)
	}
}

func TestLoad_ReturnsCredsFromKeyring(t *testing.T) {
	keyring.MockInit()
	s := NewStore("")
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	s.Save(&Credentials{APIKey: "from-keyring", Namespace: "ns1"})

	creds, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if creds == nil || creds.APIKey != "from-keyring" || creds.Namespace != "ns1" {
		t.Errorf("unexpected creds: %+v", creds)
	}
}

func TestLoad_OverlaysAPIKeyEnvVar(t *testing.T) {
	keyring.MockInit()
	s := NewStore("")

	s.Save(&Credentials{APIKey: "keyring-key"})
	t.Setenv("RISKOS_API_KEY", "env-key")
	t.Setenv("RISKOS_EVAL_KEY", "")

	creds, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if creds.APIKey != "env-key" {
		t.Errorf("expected env-key, got %s", creds.APIKey)
	}
}

func TestLoad_OverlaysEvalKeyEnvVar(t *testing.T) {
	keyring.MockInit()
	s := NewStore("")

	s.Save(&Credentials{APIKey: "key1"})
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "env-eval-key")

	creds, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if creds.EvalKey != "env-eval-key" {
		t.Errorf("expected env-eval-key, got %s", creds.EvalKey)
	}
}

func TestLoad_ReturnsNilWhenAPIKeyMissing(t *testing.T) {
	keyring.MockInit()
	s := NewStore("")

	s.Save(&Credentials{Namespace: "ns"})
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	creds, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if creds != nil {
		t.Errorf("expected nil when api_key missing, got %+v", creds)
	}
}

func TestLoad_HandlesCorruptedKeyringGracefully(t *testing.T) {
	keyring.MockInit()
	keyring.Set(ServiceName, KeyringUser, "{{not valid json}}")

	s := NewStore("")
	t.Setenv("RISKOS_API_KEY", "")
	t.Setenv("RISKOS_EVAL_KEY", "")

	creds, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if creds != nil {
		t.Errorf("expected nil for corrupted keyring, got %+v", creds)
	}
}

func TestDelete_RemovesFromKeyring(t *testing.T) {
	keyring.MockInit()
	s := NewStore("")

	s.Save(&Credentials{APIKey: "key1"})

	if err := s.Delete(); err != nil {
		t.Fatal(err)
	}

	_, err := keyring.Get(ServiceName, KeyringUser)
	if err == nil {
		t.Error("expected keyring to be empty after delete")
	}
}

func TestSource_ReturnsCorrectSource(t *testing.T) {
	keyring.MockInit()
	s := NewStore("")
	t.Setenv("RISKOS_API_KEY", "")

	if src := s.Source(); src != SourceNone {
		t.Errorf("expected none, got %s", src)
	}

	s.Save(&Credentials{APIKey: "key1"})
	if src := s.Source(); src != SourceKeyring {
		t.Errorf("expected keyring, got %s", src)
	}

	t.Setenv("RISKOS_API_KEY", "env-key")
	if src := s.Source(); src != SourceEnvVar {
		t.Errorf("expected env, got %s", src)
	}
}
