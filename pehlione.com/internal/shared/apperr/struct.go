package apperr

type Kind string

type AppError struct {
	Kind      Kind
	PublicMsg string            // kullanıcıya gösterilebilir mesaj
	Fields    map[string]string // form/validation alan hataları (opsiyonel)
	Err       error             // internal hata (log için)
}