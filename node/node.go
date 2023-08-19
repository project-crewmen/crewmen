package node

// Defines any machine(physical aspects) of the cluster
// For instance worker, manager are nodes since they run on PM
type Node struct {
	Name            string
	Ip              string
	Cores           int
	Memory          int
	MemoryAllocated int
	Disk            int
	DiskAllocated   int
	Role            string
	TaskCount       int
}
