package query

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// PageCursor is a stable, opaque pagination cursor derived from the event's
// on-chain sort key (block number + log index) and its ID for verification.
// It is base64-encoded on the wire so clients treat it as an opaque token.
type PageCursor struct {
	BlockNumber uint64 `json:"bn"`
	LogIndex    uint64 `json:"li"`
	EventID     string `json:"id"`
}

// EncodePageCursor serialises a PageCursor to an opaque base64 string.
func EncodePageCursor(blockNumber uint64, logIndex uint64, eventID string) string {
	pc := PageCursor{BlockNumber: blockNumber, LogIndex: logIndex, EventID: eventID}
	b, err := json.Marshal(&pc)
	if err != nil {
		// Fallback: should never happen with simple fields
		return fmt.Sprintf("%d:%d:%s", blockNumber, logIndex, eventID)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodePageCursor deserialises an opaque cursor string back into a PageCursor.
// Returns false for the second value if the cursor cannot be decoded.
func DecodePageCursor(cursor string) (PageCursor, bool) {
	if cursor == "" {
		return PageCursor{}, false
	}
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return PageCursor{}, false
	}
	var pc PageCursor
	if err := json.Unmarshal(b, &pc); err != nil {
		return PageCursor{}, false
	}
	return pc, true
}
