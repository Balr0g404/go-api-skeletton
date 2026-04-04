package database_test

import (
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Balr0g404/go-api-skeletton/internal/config"
	"github.com/Balr0g404/go-api-skeletton/internal/database"
)

func TestNewRedis_ConnectsSuccessfully(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	parts := strings.SplitN(mr.Addr(), ":", 2)
	cfg := &config.RedisConfig{
		Host: parts[0],
		Port: parts[1],
	}

	client, err := database.NewRedis(cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewRedis_ReturnsErrorWhenUnreachable(t *testing.T) {
	// Port fictif — connexion refusée immédiatement à chaque tentative.
	// NOTE : NewRedis effectue jusqu'à 5 tentatives avec 2s d'attente entre chacune ;
	// ce test est donc intentionnellement lent (~8s) et tagué "integration".
	// Pour l'exécuter : go test -tags=integration ./internal/database/...
	if testing.Short() {
		t.Skip("skipped in short mode: NewRedis retry loop takes ~8s")
	}

	cfg := &config.RedisConfig{
		Host: "127.0.0.1",
		Port: "19999",
	}

	client, err := database.NewRedis(cfg)
	assert.Nil(t, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to redis")
}
