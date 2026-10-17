package database

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// PostgreSQL jsonb cannot contain U+0000 or arbitrary binary text. Raw log
// bytes use JSON's base64 byte encoding; the envelope also keeps replay inputs
// and decoded data that the original string-only row discarded.
type persistedEventPayload struct {
	Schema           string         `json:"schema"`
	Raw              []byte         `json:"raw"`
	Topics           []common.Hash  `json:"topics"`
	Signature        common.Hash    `json:"signature"`
	TransactionIndex uint64         `json:"transaction_index"`
	Decoded          map[string]any `json:"decoded"`
	Typed            any            `json:"typed"`
	DecoderVersion   string         `json:"decoder_version"`
}

func eventPayloadJSON(event *blockchain.BlockchainEvent) (string, error) {
	payload := persistedEventPayload{
		Schema: "chainpulse.event-payload.v1", Raw: event.EventData,
		Topics: event.EventTopic, Signature: event.EventSignature, TransactionIndex: event.TransactionIndex,
		Decoded: event.DecodedData, Typed: event.TypedData, DecoderVersion: "evm-v1",
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode persisted event payload: %w", err)
	}
	return string(encoded), nil
}

func restoreEventPayload(stored []byte, event *blockchain.BlockchainEvent) error {
	var payload persistedEventPayload
	if len(stored) > 0 && stored[0] == '{' {
		decoder := json.NewDecoder(bytes.NewReader(stored))
		decoder.UseNumber()
		if err := decoder.Decode(&payload); err != nil {
			return fmt.Errorf("decode persisted event payload: %w", err)
		}
		if len(bytes.TrimSpace(stored[decoder.InputOffset():])) != 0 {
			return fmt.Errorf("persisted event payload has trailing JSON")
		}
		if payload.Schema == "chainpulse.event-payload.v1" {
			event.EventData = payload.Raw
			event.EventTopic = payload.Topics
			event.EventSignature = payload.Signature
			event.TransactionIndex = payload.TransactionIndex
			event.DecodedData = payload.Decoded
			event.TypedData = payload.Typed
			return nil
		}
	}
	// Existing JSON-string rows remain readable during upgrade.
	event.EventData = eventDataFromJSON(stored)
	return nil
}
