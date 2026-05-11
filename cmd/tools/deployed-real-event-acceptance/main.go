package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	defaultRPCURL      = "http://127.0.0.1:8545"
	defaultAPIURL      = "http://127.0.0.1:8080"
	defaultPrivateKey  = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	defaultEmitValue   = int64(42)
	defaultTimeoutSecs = 60
	eventEmitterABI    = `[{"type":"function","name":"emitPing","inputs":[{"name":"value","type":"uint256","internalType":"uint256"}],"outputs":[],"stateMutability":"nonpayable"},{"type":"event","name":"Ping","inputs":[{"name":"sender","type":"address","indexed":true,"internalType":"address"},{"name":"value","type":"uint256","indexed":false,"internalType":"uint256"}],"anonymous":false}]`
	eventEmitterBin    = "0x6080604052348015600e575f5ffd5b5061015a8061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610029575f3560e01c806329f51a931461002d575b5f5ffd5b610047600480360381019061004291906100d1565b610049565b005b3373ffffffffffffffffffffffffffffffffffffffff167ffd8d0c1dc3ab254ec49463a1192bb2423b3b851adedec1aa94dcd362dc063c9d8260405161008f919061010b565b60405180910390a250565b5f5ffd5b5f819050919050565b6100b08161009e565b81146100ba575f5ffd5b50565b5f813590506100cb816100a7565b92915050565b5f602082840312156100e6576100e561009a565b5b5f6100f3848285016100bd565b91505092915050565b6101058161009e565b82525050565b5f60208201905061011e5f8301846100fc565b9291505056fea26469706673582212204ba781633048d3b0701a699d87fcc004c48bd1ae013228a782969b6ee67db5be64736f6c63430008210033"
)

type queryResponse struct {
	Events json.RawMessage `json:"events"`
	Data   json.RawMessage `json:"data"`
}

type eventResponse struct {
	TransactionHash string `json:"transactionHash"`
	ContractAddress string `json:"contractAddress"`
	EventName       string `json:"eventName"`
	EventID         string `json:"eventId"`
}

func main() {
	cfg := loadConfig()
	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

type config struct {
	RPCURL       string
	APIURL       string
	PrivateKey   string
	EmitValue    int64
	ExpectAPI    bool
	Timeout      time.Duration
	PollInterval time.Duration
}

func loadConfig() config {
	timeoutSecs := envInt("TIMEOUT_SECONDS", defaultTimeoutSecs)
	return config{
		RPCURL:       envString("RPC_URL", defaultRPCURL),
		APIURL:       strings.TrimRight(envString("API_URL", defaultAPIURL), "/"),
		PrivateKey:   envString("PRIVATE_KEY", defaultPrivateKey),
		EmitValue:    int64(envInt("EMIT_VALUE", int(defaultEmitValue))),
		ExpectAPI:    envBool("EXPECT_API", true),
		Timeout:      time.Duration(timeoutSecs) * time.Second,
		PollInterval: 2 * time.Second,
	}
}

func run(cfg config) error {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	client, err := ethclient.DialContext(ctx, cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("connect rpc %s: %w", cfg.RPCURL, err)
	}
	defer client.Close()

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("read chain id: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(eventEmitterABI))
	if err != nil {
		return fmt.Errorf("parse abi: %w", err)
	}

	auth, err := newTransactor(ctx, client, privateKey, chainID)
	if err != nil {
		return err
	}

	contractAddress, deployTx, _, err := bind.DeployContract(auth, parsedABI, common.FromHex(eventEmitterBin), client)
	if err != nil {
		return fmt.Errorf("deploy event emitter: %w", err)
	}

	deployReceipt, err := bind.WaitMined(ctx, client, deployTx)
	if err != nil {
		return fmt.Errorf("wait deploy receipt: %w", err)
	}
	if deployReceipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("deploy transaction failed: %s", deployTx.Hash().Hex())
	}

	auth, err = newTransactor(ctx, client, privateKey, chainID)
	if err != nil {
		return err
	}

	contract := bind.NewBoundContract(contractAddress, parsedABI, client, client, client)
	tx, err := contract.Transact(auth, "emitPing", big.NewInt(cfg.EmitValue))
	if err != nil {
		return fmt.Errorf("emit ping tx: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return fmt.Errorf("wait event receipt: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("event transaction failed: %s", tx.Hash().Hex())
	}

	if err := validatePingLog(parsedABI, contractAddress, receipt); err != nil {
		return err
	}

	log.Printf("chain event emitted: contract=%s tx=%s logs=%d", contractAddress.Hex(), tx.Hash().Hex(), len(receipt.Logs))

	if !cfg.ExpectAPI {
		log.Printf("chain-side validation passed; api validation skipped")
		return nil
	}

	if err := waitForEventInAPI(ctx, cfg.APIURL, contractAddress.Hex(), tx.Hash().Hex()); err != nil {
		return err
	}

	log.Printf("api observed emitted event: tx=%s", tx.Hash().Hex())
	return nil
}

func newTransactor(ctx context.Context, client *ethclient.Client, privateKey *ecdsa.PrivateKey, chainID *big.Int) (*bind.TransactOpts, error) {
	from := crypto.PubkeyToAddress(privateKey.PublicKey)

	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, fmt.Errorf("read pending nonce: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("suggest gas price: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("build transactor: %w", err)
	}
	auth.Context = ctx
	auth.Nonce = new(big.Int).SetUint64(nonce)
	auth.GasPrice = gasPrice
	auth.GasLimit = 1_500_000
	return auth, nil
}

func validatePingLog(parsedABI abi.ABI, contractAddress common.Address, receipt *types.Receipt) error {
	pingEvent, ok := parsedABI.Events["Ping"]
	if !ok {
		return fmt.Errorf("ping event missing from abi")
	}

	for _, lg := range receipt.Logs {
		if lg.Address != contractAddress {
			continue
		}
		if len(lg.Topics) == 0 {
			continue
		}
		if lg.Topics[0] == pingEvent.ID {
			return nil
		}
	}

	return fmt.Errorf("ping log not found in receipt")
}

func waitForEventInAPI(ctx context.Context, apiURL, contractAddress, txHash string) error {
	paths := []string{
		"/events/contract/" + contractAddress + "?limit=50",
		"/events/name/Ping?limit=50",
		"/events?limit=100",
		"/api/v1/events?limit=100",
	}

	client := &http.Client{Timeout: 5 * time.Second}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for api to observe tx %s", txHash)
		case <-ticker.C:
			for _, path := range paths {
				ok, err := apiContainsEvent(client, apiURL+path, contractAddress, txHash)
				if err != nil {
					continue
				}
				if ok {
					return nil
				}
			}
		}
	}
}

func apiContainsEvent(client *http.Client, url, contractAddress, txHash string) (bool, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("status %d", resp.StatusCode)
	}

	var payload queryResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false, err
	}

	var events []eventResponse
	if len(payload.Events) > 0 && string(payload.Events) != "null" {
		if err := json.Unmarshal(payload.Events, &events); err == nil {
			return containsEvent(events, contractAddress, txHash), nil
		}
	}
	if len(payload.Data) > 0 && string(payload.Data) != "null" {
		if err := json.Unmarshal(payload.Data, &events); err == nil {
			return containsEvent(events, contractAddress, txHash), nil
		}
	}

	return false, nil
}

func containsEvent(events []eventResponse, contractAddress, txHash string) bool {
	for _, item := range events {
		if strings.EqualFold(item.TransactionHash, txHash) {
			return true
		}
		if strings.EqualFold(item.ContractAddress, contractAddress) && strings.EqualFold(item.EventName, "Ping") {
			return true
		}
	}
	return false
}

func envString(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes"
}
