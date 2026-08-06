// Package config decodes the opaque node configuration owned by the Xray plugin.
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Relayward/relayward-sdk/contract"
)

type Configuration struct {
	XrayVersion string          `json:"xray_version"`
	XrayConfig  json.RawMessage `json:"xray_config"`
}

func Decode(raw []byte) (Configuration, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var value Configuration
	if err := decoder.Decode(&value); err != nil {
		return Configuration{}, fmt.Errorf("decode configuration: %w", err)
	}
	if err := requireEOF(decoder); err != nil {
		return Configuration{}, err
	}
	if err := contract.ValidateSemanticVersion(value.XrayVersion); err != nil {
		return Configuration{}, fmt.Errorf("xray_version: %w", err)
	}
	if strings.ContainsAny(value.XrayVersion, "-+") {
		return Configuration{}, fmt.Errorf("xray_version: pre-release and build versions are not supported")
	}
	if err := validateJSONObject(value.XrayConfig); err != nil {
		return Configuration{}, fmt.Errorf("xray_config: %w", err)
	}
	value.XrayConfig = append(json.RawMessage(nil), value.XrayConfig...)
	return value, nil
}

func validateJSONObject(raw []byte) error {
	if len(raw) == 0 {
		return fmt.Errorf("required")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var object map[string]json.RawMessage
	if err := decoder.Decode(&object); err != nil {
		return fmt.Errorf("must be a JSON object")
	}
	if object == nil {
		return fmt.Errorf("must be a JSON object")
	}
	if err := requireEOF(decoder); err != nil {
		return fmt.Errorf("must contain one JSON object")
	}
	return nil
}

func requireEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("configuration contains a trailing JSON value")
		}
		return fmt.Errorf("decode trailing configuration: %w", err)
	}
	return nil
}
