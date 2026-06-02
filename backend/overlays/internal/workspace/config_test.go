package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type configFixture struct {
	RawConfig        map[string]any `json:"raw_config"`
	NormalisedConfig Config         `json:"normalised_config"`
	FormPayload      FormPayload    `json:"form_payload"`
	ConfigFromForm   Config         `json:"config_from_form"`
}

func TestNormaliseConfigMatchesPythonBaseline(t *testing.T) {
	fixture := loadConfigFixture(t)
	fixture.NormalisedConfig.ChannelBuses = []ChannelBusConfig{}
	fixture.NormalisedConfig.LimiterEnabled = true

	normalised, err := NormaliseConfig(fixture.RawConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(normalised, fixture.NormalisedConfig) {
		t.Fatalf("normalised config mismatch\nactual:   %#v\nexpected: %#v", normalised, fixture.NormalisedConfig)
	}
}

func TestConfigToFormPayloadMatchesPythonBaseline(t *testing.T) {
	fixture := loadConfigFixture(t)

	payload, err := ConfigToFormPayload(fixture.NormalisedConfig)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Rate != fixture.FormPayload.Rate {
		t.Fatalf("rate = %q, want %q", payload.Rate, fixture.FormPayload.Rate)
	}

	var actualLayers any
	var expectedLayers any
	if err := json.Unmarshal([]byte(payload.LayersJSON), &actualLayers); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(fixture.FormPayload.LayersJSON), &expectedLayers); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actualLayers, expectedLayers) {
		t.Fatalf("layers_json mismatch\nactual:   %s\nexpected: %s", payload.LayersJSON, fixture.FormPayload.LayersJSON)
	}
}

func TestNormaliseConfigRejectsInvalidSchema(t *testing.T) {
	_, err := NormaliseConfig(map[string]any{"schema": "wrong"})
	if err == nil || err.Error() != "Workspace config schema must be octabit.workspace_config.v1." {
		t.Fatalf("error = %v", err)
	}
}

func loadConfigFixture(t *testing.T) configFixture {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "python-baseline", "config", "workspace_config_normalisation.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixture configFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}
