package configmodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigValidate_Valid(t *testing.T) {
	t.Parallel()
	c := &Config{
		WorkerPoolSize:  10,
		BatchSize:       100,
		MaxRetries:      3,
		RetryBackoff:    1000,
		BlockChunkSize:  500,
		ShutdownTimeout: 30,
		ReorgThreshold:  64,
	}
	assert.NoError(t, c.Validate())
}

func TestConfigValidate_AllDefaults(t *testing.T) {
	t.Parallel()
	c := &Config{}
	assert.NoError(t, c.Validate())
}

func TestConfigValidate_NegativeWorkerPoolSize(t *testing.T) {
	t.Parallel()
	c := &Config{WorkerPoolSize: -1}
	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worker_pool_size")
}

func TestConfigValidate_NegativeBatchSize(t *testing.T) {
	t.Parallel()
	c := &Config{ReorgThreshold: 0, ConfirmationDepth: 0, BatchSize: -1}
	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch_size")
}

func TestConfigValidate_NegativeMaxRetries(t *testing.T) {
	t.Parallel()
	c := &Config{MaxRetries: -1}
	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_retries")
}

func TestConfigValidate_NegativeRetryBackoff(t *testing.T) {
	t.Parallel()
	c := &Config{RetryBackoff: -1}
	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retry_backoff")
}

func TestConfigValidate_NegativeBlockChunkSize(t *testing.T) {
	t.Parallel()
	c := &Config{BlockChunkSize: -1}
	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "block_chunk_size")
}

func TestConfigValidate_NegativeShutdownTimeout(t *testing.T) {
	t.Parallel()
	c := &Config{ShutdownTimeout: -1}
	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "shutdown_timeout")
}

func TestConfigValidate_MissingReorgWhenIndexing(t *testing.T) {
	t.Parallel()
	c := &Config{
		Blockchains:  map[string]BlockchainConfig{"ethereum": {}},
		ActiveChains: []string{"ethereum"},
	}
	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reorg_threshold")
}

func TestConfigValidate_MissingReorgNoBlockchains(t *testing.T) {
	t.Parallel()
	c := &Config{}
	c.Blockchains = nil
	c.ActiveChains = nil
	c.ReorgThreshold = 0
	c.ConfirmationDepth = 0
	assert.NoError(t, c.Validate())
}

func TestConfigValidate_MultipleErrors(t *testing.T) {
	t.Parallel()
	c := &Config{
		WorkerPoolSize:    -1,
		BatchSize:         -1,
		MaxRetries:        -1,
		Blockchains:       map[string]BlockchainConfig{"ethereum": {}},
		ActiveChains:      []string{"ethereum"},
		ReorgThreshold:    0,
		ConfirmationDepth: 0,
	}
	err := c.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worker_pool_size")
	assert.Contains(t, err.Error(), "batch_size")
	assert.Contains(t, err.Error(), "max_retries")
	assert.Contains(t, err.Error(), "reorg_threshold")
}

func TestActiveChainFields_MultiChain(t *testing.T) {
	t.Parallel()
	c := &Config{
		Blockchains: map[string]BlockchainConfig{
			"ethereum": {NodeURL: "http://eth-node"},
		},
		ActiveChains: []string{"ethereum"},
	}
	result := c.activeChainFields()
	assert.NotNil(t, result)
	assert.Equal(t, "ethereum", result.ChainID)
	assert.Equal(t, "http://eth-node", result.NodeURL)
}

func TestActiveChainFields_Legacy(t *testing.T) {
	t.Parallel()
	c := &Config{
		ChainID:           "1",
		BlockchainNodeURL: "http://legacy-node",
		StartBlock:        100,
		Network:           "mainnet",
		EventSignatures:   []string{"Transfer(address,address,uint256)"},
		ConfirmationDepth: 64,
	}
	result := c.activeChainFields()
	assert.NotNil(t, result)
	assert.Equal(t, "1", result.ChainID)
	assert.Equal(t, "http://legacy-node", result.NodeURL)
	assert.Equal(t, uint64(100), result.StartBlock)
	assert.Equal(t, "mainnet", result.Network)
	assert.Equal(t, uint64(64), result.ConfirmationBlocks)
}

func TestActiveChainFields_Nil(t *testing.T) {
	t.Parallel()
	c := &Config{}
	result := c.activeChainFields()
	assert.Nil(t, result)
}

func TestActiveChainFields_MultiChainMiss(t *testing.T) {
	t.Parallel()
	c := &Config{
		Blockchains:  map[string]BlockchainConfig{},
		ActiveChains: []string{"polygon"},
		ChainID:      "",
	}
	result := c.activeChainFields()
	assert.Nil(t, result)
}

func TestActiveChainFields_MultiChainOverLegacy(t *testing.T) {
	t.Parallel()
	c := &Config{
		Blockchains: map[string]BlockchainConfig{
			"ethereum": {NodeURL: "http://multi-node"},
		},
		ActiveChains:      []string{"ethereum"},
		ChainID:           "1",
		BlockchainNodeURL: "http://legacy-node",
	}
	result := c.activeChainFields()
	assert.NotNil(t, result)
	assert.Equal(t, "ethereum", result.ChainID)
	assert.Equal(t, "http://multi-node", result.NodeURL)
}

func TestSecrectString_Value(t *testing.T) {
	t.Parallel()
	s := SecretString("my-secret-password")
	assert.Equal(t, "my-secret-password", s.Value())
}

func TestSecrectString_String(t *testing.T) {
	t.Parallel()
	s := SecretString("my-secret-password")
	assert.Equal(t, "***", s.String())
}

func TestSecrectString_StringEmpty(t *testing.T) {
	t.Parallel()
	s := SecretString("")
	assert.Equal(t, "", s.String())
}

func TestSecrectString_GoString(t *testing.T) {
	t.Parallel()
	s := SecretString("password")
	assert.Equal(t, "***", s.GoString())
}

func TestSecrectString_GoStringEmpty(t *testing.T) {
	t.Parallel()
	s := SecretString("")
	assert.Equal(t, "", s.GoString())
}

func TestSecrectString_MarshalJSON(t *testing.T) {
	t.Parallel()
	s := SecretString("secret")
	data, err := s.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `"***"`, string(data))
}

func TestSecrectString_MarshalText(t *testing.T) {
	t.Parallel()
	s := SecretString("secret")
	data, err := s.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, "***", string(data))
}

func TestSecrectString_UnmarshalJSON(t *testing.T) {
	t.Parallel()
	var s SecretString
	err := s.UnmarshalJSON([]byte(`"new-secret"`))
	assert.NoError(t, err)
	assert.Equal(t, "new-secret", s.Value())
}

func TestSecrectString_UnmarshalJSONInvalid(t *testing.T) {
	t.Parallel()
	var s SecretString
	err := s.UnmarshalJSON([]byte(`invalid`))
	assert.Error(t, err)
}

func TestToSecretStrings(t *testing.T) {
	t.Parallel()
	result := ToSecretStrings([]string{"a", "b", "c"})
	assert.Len(t, result, 3)
	assert.Equal(t, "a", result[0].Value())
	assert.Equal(t, "b", result[1].Value())
	assert.Equal(t, "c", result[2].Value())
}

func TestToSecretStrings_Nil(t *testing.T) {
	t.Parallel()
	result := ToSecretStrings(nil)
	assert.Nil(t, result)
}

func TestToSecretStrings_Empty(t *testing.T) {
	t.Parallel()
	result := ToSecretStrings([]string{})
	assert.Empty(t, result)
}
