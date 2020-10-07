package main

import (
	"testing"
)

func TestConfigParsing(t *testing.T) {
	config := getConfig("../example-lambda/config.yaml")
	if config.SecretsHome == "" {
		t.Error("SecretsHome not populated")
	}
}
