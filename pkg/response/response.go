// Package response defines the JSON envelope returned by every endpoint.
package response

type Status string

const (
	StatusSuccess       Status = "SUCCESS"
	BadRequest          Status = "BAD_REQUEST"
	Unauthorized        Status = "UNAUTHORIZED"
	Forbidden           Status = "FORBIDDEN"
	DataNotFound        Status = "DATA_NOT_FOUND"
	MethodNotAllowed    Status = "METHOD_NOT_ALLOWED"
	DataConflict        Status = "DATA_CONFLICT"
	PayloadTooLarge     Status = "PAYLOAD_TOO_LARGE"
	UnprocessableEntity Status = "UNPROCESSABLE_ENTITY"
	TooManyRequests     Status = "TOO_MANY_REQUESTS"
	InternalServerError Status = "SERVER_ERROR"
	ServiceUnavailable  Status = "SERVICE_UNAVAILABLE"
)

func (s Status) String() string { return string(s) }

// Response is the standard envelope for all API responses.
type Response struct {
	Status     string            `json:"status" example:"SUCCESS"`
	Message    string            `json:"message" example:"OK"`
	Data       any               `json:"data,omitempty"`
	Meta       any               `json:"meta,omitempty"`
	ErrorField map[string]string `json:"errorField,omitempty"`
	RequestID  string            `json:"requestId,omitempty" example:"REQ-01932c7e6f2a7c3b9f1d2e3a4b5c6d7e"`
}

// Success builds a success response.
func Success(message string, data any, requestID string) Response {
	return Response{Status: StatusSuccess.String(), Message: message, Data: data, RequestID: requestID}
}

// SuccessWithMeta builds a success response with metadata (e.g. pagination).
func SuccessWithMeta(message string, data, meta any, requestID string) Response {
	r := Success(message, data, requestID)
	r.Meta = meta
	return r
}

// Error builds an error response. fields may be nil.
func Error(status Status, message string, fields map[string]string, requestID string) Response {
	return Response{Status: status.String(), Message: message, ErrorField: fields, RequestID: requestID}
}

// StatusFromHTTP maps an HTTP status code to the envelope status.
func StatusFromHTTP(code int) Status {
	switch code {
	case 400:
		return BadRequest
	case 401:
		return Unauthorized
	case 403:
		return Forbidden
	case 404:
		return DataNotFound
	case 405:
		return MethodNotAllowed
	case 409:
		return DataConflict
	case 413:
		return PayloadTooLarge
	case 422:
		return UnprocessableEntity
	case 429:
		return TooManyRequests
	case 503:
		return ServiceUnavailable
	}
	if code >= 200 && code < 300 {
		return StatusSuccess
	}
	if code >= 400 && code < 500 {
		return BadRequest
	}
	return InternalServerError
}
