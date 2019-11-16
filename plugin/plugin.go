package plugin

// Plugin generates a part of the wakeup message
type Plugin interface {
	Text() (string, error)
}
