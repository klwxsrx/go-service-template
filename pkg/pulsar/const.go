package pulsar

type contextKey int

const (
	pulsarMessageIDContextKey contextKey = iota
)

const (
	messageIDPropertyName = "messageID"
)
