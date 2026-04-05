package domain

import "time"

// NodeStatus represents the current operational state of a node.
type NodeStatus string

const (
	NodeStatusUnknown  NodeStatus = "unknown"
	NodeStatusOnline   NodeStatus = "online"
	NodeStatusOffline  NodeStatus = "offline"
	NodeStatusDegraded NodeStatus = "degraded"
)

// Node represents a managed server instance.
type Node struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Host      string     `json:"host"`
	Status    NodeStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
