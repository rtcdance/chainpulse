package core

// ProtocolType represents the type of protocol
type ProtocolType int

const (
	ProtocolHTTP ProtocolType = iota
	ProtocolWebSocket
	ProtocolGRPC
	ProtocolGraphQL
	ProtocolUnknown
)

// HTTPMethod represents HTTP methods
type HTTPMethod string

const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	DELETE HTTPMethod = "DELETE"
	PATCH  HTTPMethod = "PATCH"
	HEAD   HTTPMethod = "HEAD"
)

// StatusCode represents HTTP status codes
type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusCreated             StatusCode = 201
	StatusAccepted            StatusCode = 202
	StatusNoContent           StatusCode = 204
	StatusBadRequest          StatusCode = 400
	StatusUnauthorized        StatusCode = 401
	StatusForbidden           StatusCode = 403
	StatusNotFound            StatusCode = 404
	StatusMethodNotAllowed    StatusCode = 405
	StatusConflict            StatusCode = 409
	StatusInternalServerError StatusCode = 500
	StatusNotImplemented      StatusCode = 501
	StatusServiceUnavailable  StatusCode = 503
)

// ContentType represents content types
type ContentType string

const (
	ContentTypeJSON      ContentType = "application/json"
	ContentTypeXML       ContentType = "application/xml"
	ContentTypeText      ContentType = "text/plain"
	ContentTypeHTML      ContentType = "text/html"
	ContentTypeProtobuf  ContentType = "application/protobuf"
	ContentTypeGraphQL   ContentType = "application/graphql"
)



// RequestMetadata holds metadata about a request
type RequestMetadata struct {
	Protocol      ProtocolType
	ClientIP      string
	UserAgent     string
	RequestID     string
	Timestamp     int64
	ContentLength int64
}

// ResponseMetadata holds metadata about a response
type ResponseMetadata struct {
	Protocol      ProtocolType
	ContentLength int64
	Duration      int64 // milliseconds
	Timestamp     int64
}
