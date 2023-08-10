// Package sdk exports application names for external use.
package sdk

const (
	CodeExistsSubscriber = "exists_subscriber"
	CodeNoSubscribers    = "no_subscribers"
)

type Event struct {
	Code string `json:"code"`
}
