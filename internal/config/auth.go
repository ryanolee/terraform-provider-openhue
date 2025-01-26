package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openhue/openhue-go"
	"github.com/spf13/viper"
)

const cachePath = ".openhue-credentials.json"

type AuthConfig struct {
	BridgeIp     string
	BridgeApiKey string
}

// GetAuthConfig returns the Hue bridge IP and API key to use for the provider
// If the bridge IP is not provided, it will attempt to discover one on the local network
// If the API key is not provided, it will attempt to create one but you will need to press the link button on the Hue bridge to authenticate
func GetAuthConfig(ctx context.Context, tfBridgeIp string, tfBridgeApiKey string, useCache bool) (*AuthConfig, error) {
	cfg := viper.New()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	cfg.SetConfigName("credentials")
	cfg.SetConfigType("json")
	viper.SetDefault("hue_bridge_ip", tfBridgeIp)
	viper.SetDefault("hue_bridge_api_key", tfBridgeApiKey)

	if useCache {
		tflog.Info(ctx, "Using cache for Hue bridge credentials")
		cfg.SetConfigFile(filepath.Join(homeDir, cachePath))
	}
	cfg.AutomaticEnv()
	cfg.ReadInConfig()

	// Fallback to discovery and authentication
	hueBridgeIp := cfg.GetString("hue_bridge_ip")
	if hueBridgeIp == "" || tfBridgeIp == "discover" {
		tflog.Info(ctx, "No Hue bridge ip provided ot set to always discover, attempting to discover one on the local network")

		discoveryMetadata, err := openhue.NewBridgeDiscovery(
			openhue.WithTimeout(5 * time.Second),
		).Discover()

		if err != nil {
			return nil, fmt.Errorf("failed to discover Hue Bridge: %w", err)
		}

		ctx = tflog.SetField(ctx, "bridge_ip", discoveryMetadata.IpAddress)
		ctx = tflog.SetField(ctx, "bridge_instance", discoveryMetadata.Instance)
		tflog.Info(ctx, "Successfully discovered Hue Bridge")

		cfg.Set("hue_bridge_ip", discoveryMetadata.IpAddress)
	}

	hueBridgeApiKey := cfg.GetString("hue_bridge_api_key")

	if hueBridgeApiKey == "" {
		tflog.Warn(ctx, "No Hue bridge api key provided, attempting to create one. You will need to press the link button on the Hue Bridge to authenticate!")

		tfBridgeApiKey, err = authenticateWithBridgeIp(ctx, cfg.GetString("hue_bridge_ip"))
		if err != nil {
			return nil, fmt.Errorf("failed to authenticate with Hue Bridge: %w", err)
		}

		cfg.Set("hue_bridge_api_key", tfBridgeApiKey)
	}

	if useCache {
		if err := cfg.SafeWriteConfigAs(filepath.Join(homeDir, cachePath)); err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Failed to write cache file: %s", err))
		}
	}

	return &AuthConfig{
		BridgeIp:     cfg.GetString("hue_bridge_ip"),
		BridgeApiKey: cfg.GetString("hue_bridge_api_key"),
	}, nil
}

func authenticateWithBridgeIp(ctx context.Context, bridgeIpAddress string) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	authenticator, err := openhue.NewAuthenticator(bridgeIpAddress)
	if err != nil {
		return "", fmt.Errorf("failed to create authenticator: %w", err)
	}

	var apiKey string

	for apiKey == "" {
		select {
		case <-timeoutCtx.Done():
			return "", fmt.Errorf("timed out waiting for Hue Bridge to be authenticated")
		case <-ticker.C:
			fmt.Println("Waiting for hue bridge button to be pressed")
			apiKey, err = authenticateWithBridgeIpOnce(ctx, authenticator)
			if err != nil {
				return "", err
			}
		}

	}

	return apiKey, nil

}

func authenticateWithBridgeIpOnce(ctx context.Context, authenticator openhue.Authenticator) (string, error) {
	// try to authenticate
	apiKey, retry, err := authenticator.Authenticate()
	if err != nil && retry {
		return "", nil
	} else if err != nil && !retry {
		return "", err
	} else {
		return apiKey, nil
	}

}
