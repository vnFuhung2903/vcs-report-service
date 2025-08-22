package entities

type ContainerStatus string

const (
	ContainerOn  ContainerStatus = "ON"
	ContainerOff ContainerStatus = "OFF"
)

type ContainerWithStatus struct {
	ContainerId string          `json:"container_id"`
	Status      ContainerStatus `json:"status"`
}
