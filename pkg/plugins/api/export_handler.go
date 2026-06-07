package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rtcdance/chainpulse/pkg/chainid"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
)

type ExportHandler struct {
	logger     core.Logger
	eventStore domainquery.EventReader
}

func NewExportHandler(logger core.Logger, eventStore domainquery.EventReader) *ExportHandler {
	return &ExportHandler{
		logger:     logger,
		eventStore: eventStore,
	}
}

type exportFilter struct {
	Format    string
	Limit     int
	Offset    int
	ChainID   string
	EventName string
	Contract  string
	StartTime int64
	EndTime   int64
}

type exportRecord struct {
	ID              string `json:"id"`
	EventName       string `json:"eventName"`
	ChainID         string `json:"chainId"`
	ContractAddress string `json:"contractAddress"`
	BlockNumber     uint64 `json:"blockNumber"`
	TransactionHash string `json:"transactionHash"`
	Timestamp       int64  `json:"timestamp"`
	Status          string `json:"status"`
}

func (h *ExportHandler) HandleExport(w http.ResponseWriter, r *http.Request) {
	f := parseExportFilter(r)
	if err := validateExportFilter(f); err != "" {
		(&APIError{Code: "INVALID_REQUEST", Message: err, Status: http.StatusBadRequest}).WriteHTTP(w)
		return
	}

	ctx := r.Context()

	allEvents := make([]*blockchain.BlockchainEvent, 0)
	cursor := ""
	batchLimit := 500
	collected := 0

	for collected < f.Limit+f.Offset {
		events, hasMore, err := h.eventStore.GetEventsPaginated(ctx, cursor, batchLimit)
		if err != nil {
			h.logger.Error("Failed to get events for export", "error", err.Error())
			(&APIError{Code: "EXPORT_FAILED", Message: "Failed to retrieve events", Status: http.StatusInternalServerError}).WriteHTTP(w)
			return
		}

		for _, event := range events {
			if !h.matchesExportFilter(event, f) {
				continue
			}
			if collected >= f.Offset && collected < f.Offset+f.Limit {
				allEvents = append(allEvents, event)
			}
			collected++
			if collected >= f.Offset+f.Limit {
				break
			}
		}

		if !hasMore || len(events) == 0 {
			break
		}
		if collected >= f.Offset+f.Limit {
			break
		}
		if len(events) > 0 {
			cursor = events[len(events)-1].ID
		}
	}

	records := h.eventsToExportRecords(allEvents)

	switch f.Format {
	case "csv":
		h.ExportEventsCSV(w, records)
	case "json":
		h.ExportEventsJSON(w, records)
	default:
		(&APIError{Code: "INVALID_REQUEST", Message: "unsupported format, use csv or json", Status: http.StatusBadRequest}).WriteHTTP(w)
	}
}

func (h *ExportHandler) ExportEventsCSV(w http.ResponseWriter, records []exportRecord) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"events_%s.csv\"", time.Now().UTC().Format("20060102_150405")))

	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"id", "eventName", "chainId", "contractAddress", "blockNumber", "transactionHash", "timestamp", "status"})

	for _, rec := range records {
		_ = writer.Write([]string{
			rec.ID,
			rec.EventName,
			rec.ChainID,
			rec.ContractAddress,
			strconv.FormatUint(rec.BlockNumber, 10),
			rec.TransactionHash,
			strconv.FormatInt(rec.Timestamp, 10),
			rec.Status,
		})
	}

	writer.Flush()
}

func (h *ExportHandler) ExportEventsJSON(w http.ResponseWriter, records []exportRecord) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"events_%s.json\"", time.Now().UTC().Format("20060102_150405")))

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(records)
}

func (h *ExportHandler) eventsToExportRecords(events []*blockchain.BlockchainEvent) []exportRecord {
	records := make([]exportRecord, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		records = append(records, exportRecord{
			ID:              event.ID,
			EventName:       event.EventName,
			ChainID:         event.ChainID,
			ContractAddress: event.ContractAddress.Hex(),
			BlockNumber:     event.BlockNumber,
			TransactionHash: event.TransactionHash.Hex(),
			Timestamp:       event.BlockTimestamp,
			Status:          string(event.Status),
		})
	}
	return records
}

func (h *ExportHandler) matchesExportFilter(event *blockchain.BlockchainEvent, f exportFilter) bool {
	if f.ChainID != "" {
		resolvedID := chainid.ResolveChainID(f.ChainID)
		resolvedName := chainid.ResolveChainName(resolvedID)
		if event.ChainID != f.ChainID && strconv.Itoa(resolvedID) != event.ChainID && resolvedName != event.ChainID {
			return false
		}
	}
	if f.EventName != "" && event.EventName != f.EventName {
		return false
	}
	if f.Contract != "" && !strings.EqualFold(event.ContractAddress.Hex(), f.Contract) {
		return false
	}
	if f.StartTime > 0 && event.BlockTimestamp < f.StartTime {
		return false
	}
	if f.EndTime > 0 && event.BlockTimestamp > f.EndTime {
		return false
	}
	return true
}

func parseExportFilter(r *http.Request) exportFilter {
	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		format = "json"
	}
	return exportFilter{
		Format:    format,
		Limit:     parseIntParam(r, "limit", 1000),
		Offset:    parseIntParam(r, "offset", 0),
		ChainID:   strings.TrimSpace(r.URL.Query().Get("chainId")),
		EventName: strings.TrimSpace(r.URL.Query().Get("eventName")),
		Contract:  strings.TrimSpace(r.URL.Query().Get("contract")),
		StartTime: parseInt64Param(r, "start_time", 0),
		EndTime:   parseInt64Param(r, "end_time", 0),
	}
}

func validateExportFilter(f exportFilter) string {
	if f.Limit <= 0 || f.Limit > 10000 {
		return "limit must be between 1 and 10000"
	}
	if f.Offset < 0 {
		return "offset must be greater than or equal to 0"
	}
	if f.StartTime > 0 && f.EndTime > 0 && f.StartTime > f.EndTime {
		return "start_time must be less than or equal to end_time"
	}
	return ""
}

func parseIntParam(r *http.Request, name string, defaultValue int) int {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

func parseInt64Param(r *http.Request, name string, defaultValue int64) int64 {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}
	return intValue
}
