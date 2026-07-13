package main

import (
	"encoding/json"
	"io"
	"os"
)

type Identity struct {
	Name     string `json:"name"`
	Nature   string `json:"nature"`
	Purpose  string `json:"purpose"`
	City     string `json:"city"`
	Timezone string `json:"timezone"`
}

const IdentityFile = "/root/identity.json"

func LoadIdentity() (*Identity, error) {
	file, err := os.Open(IdentityFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var identity Identity
	if err := json.Unmarshal(body, &identity); err != nil {
		return nil, err
	}

	return &identity, nil
}

func SaveIdentity(identity *Identity) error {
	data, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(IdentityFile, data, 0644)
}

func DeleteIdentity() error {
	return os.Remove(IdentityFile)
}
