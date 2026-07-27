package domain

type AggregateType string

const (
	AggregateRoom AggregateType = "room"
	AggregateHand AggregateType = "hand"
)

type Event struct {
	AggregateType AggregateType
	AggregateID   string
	Name          string
	Version       uint64
	Payload       any
}
