package view

type FlashKind string

const (
	FlashInfo    FlashKind = "info"
	FlashSuccess FlashKind = "success"
	FlashWarning FlashKind = "warning"
	FlashError   FlashKind = "error"
)

type Flash struct {
	Kind    FlashKind `json:"kind"`
	Message string    `json:"message"`
}
