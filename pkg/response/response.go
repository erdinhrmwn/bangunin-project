// Package response provides the standard API envelope:
// {success, message, data, meta, errors}.
package response

type Meta struct {
	Page       int    `json:"page,omitempty"`
	PerPage    int    `json:"per_page,omitempty"`
	Total      int    `json:"total,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type Envelope struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

func Success(message string, data any) Envelope {
	return Envelope{Success: true, Message: message, Data: data}
}

func SuccessPaginated(message string, data any, meta Meta) Envelope {
	return Envelope{Success: true, Message: message, Data: data, Meta: &meta}
}

func Error(message string, errs any) Envelope {
	return Envelope{Success: false, Message: message, Errors: errs}
}
