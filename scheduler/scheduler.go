package scheduler

import (
	"crewmen/node"
	"crewmen/task"
	"log"
	"math"
	"time"
)

/*
	<--- Scheduler : Assign a task for each worker --->
	* Goal - 	Spread the tasks across the cluster of worker machines such that,
			it minimize the resource consumption(Memory, CPU, Disk etc.)
	* Optimization - All workers must do some task. Neither overloaded nor starved.
*/

const (
	LIEB = 1.53960071783900203869
)

type Scheduler interface {
	// To select set of nodes that potentially able to run a given task
	SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node

	// To prioritize the selected candidate nodes
	Score(t task.Task, nodes []*node.Node) map[string]float64

	// To pick the best node to run the task
	Pick(scores map[string]float64, candidates []*node.Node) *node.Node
}

/*
---------------------
Round Robin Strategy
---------------------
*/
type RoundRobin struct {
	Name       string
	LastWorker int
}

// In RR Scheduling we don't need to specifically filter out candidates from set of nodes
// Threfore we can simply return the input set of nodes
func (r *RoundRobin) SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node {
	return nodes
}

func (r *RoundRobin) Score(t task.Task, nodes []*node.Node) map[string]float64 {
	/*
		This is a map with node name and associated score. For instance
		{
			"node1": 1.0,
			"node2": 0.1,
			"node3": 1.0
		}
	*/
	nodeScores := make(map[string]float64)

	var newWorker int
	if r.LastWorker+1 < len(nodes) {
		newWorker = r.LastWorker + 1
		r.LastWorker++
	} else {
		newWorker = 0
		r.LastWorker = 0
	}

	// Simply check index is equal to the newWorker
	for idx, node := range nodes {
		if idx == newWorker {
			nodeScores[node.Name] = 0.1
		} else {
			nodeScores[node.Name] = 1.0
		}
	}

	return nodeScores
}

// Best score is lowesest score from the candidate's scores map
// For instance if the map is 1.0, 0.1, 1.0 then the node with 0.1 score will be selected
func (r *RoundRobin) Pick(scores map[string]float64, candidates []*node.Node) *node.Node {
	var bestNode *node.Node
	var lowestScore float64
	for idx, node := range candidates {
		if idx == 0 {
			bestNode = node
			lowestScore = scores[node.Name]
			continue
		}

		if scores[node.Name] < lowestScore {
			bestNode = node
			lowestScore = scores[node.Name]
		}
	}

	return bestNode
}

/*
---------------------
Gready Strategy
---------------------
*/
type Greedy struct {
	Name string
}

func (g *Greedy) SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node {
	return selectCandidateNodes(t, nodes)
}

func (g *Greedy) Score(t task.Task, nodes []*node.Node) map[string]float64 {
	nodeScores := make(map[string]float64)

	for _, node := range nodes {
		cpuUsage, err := calculateCpuUsage(node)
		if err != nil {
			log.Printf("error calculating CPU usage for node %s, skipping: %v", node.Name, err)
			continue
		}

		cpuLoad := calculateLoad(float64(*cpuUsage), math.Pow(2, 0.8))
		nodeScores[node.Name] = cpuLoad
	}

	return nodeScores
}

func (g *Greedy) Pick(candidates map[string]float64, nodes []*node.Node) *node.Node {
	minCpu := 0.00
	var bestNode *node.Node
	for idx, node := range nodes {
		if idx == 0 {
			minCpu = candidates[node.Name]
			bestNode = node
			continue
		}

		if candidates[node.Name] < minCpu {
			minCpu = candidates[node.Name]
			bestNode = node
		}
	}

	return bestNode
}

/*
---------------------
E-PVM Strategy
---------------------
*/
type Epvm struct {
	Name string
}

func (g *Epvm) SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node {
	return selectCandidateNodes(t, nodes)
}

func (g *Epvm) Score(t task.Task, nodes []*node.Node) map[string]float64 {
	nodeScores := make(map[string]float64)
	maxJobs := 4.0 // Node can handle maximum 4 tasks at once

	for _, node := range nodes {
		// Calculations required to determine the marginal_cost
		cpuUsage, err := calculateCpuUsage(node)
		if err != nil {
			log.Printf("error calculating CPU usage for node %s, skipping: %v", node.Name, err)
			continue
		}

		cpuLoad := calculateLoad(float64(*cpuUsage), math.Pow(2, 0.8))

		memoryAllocated := float64(node.Stats.MemUsedKb()) + float64(node.MemoryAllocated)
		memoryPercentAllocated := memoryAllocated / float64(node.Memory)

		newMemPercent := (calculateLoad(memoryAllocated+float64(t.Memory/1000), float64(node.Memory)))
		memCost := math.Pow(LIEB, newMemPercent) + math.Pow(LIEB, (float64(node.TaskCount+1))/maxJobs) - math.Pow(LIEB, memoryPercentAllocated) - math.Pow(LIEB, float64(node.TaskCount)/float64(maxJobs))
		cpuCost := math.Pow(LIEB, cpuLoad) + math.Pow(LIEB, (float64(node.TaskCount+1))/maxJobs) - math.Pow(LIEB, cpuLoad) - math.Pow(LIEB, float64(node.TaskCount)/float64(maxJobs))

		nodeScores[node.Name] = memCost + cpuCost
	}

	return nodeScores
}

func (g *Epvm) Pick(scores map[string]float64, candidates []*node.Node) *node.Node {
	minCost := 0.00
	var bestNode *node.Node
	for idx, node := range candidates {
		if idx == 0 {
			minCost = scores[node.Name]
			bestNode = node
			continue
		}

		if scores[node.Name] < minCost {
			minCost = scores[node.Name]
			bestNode = node
		}
	}

	return bestNode
}

// UTILS
// Check that, the resrouces that the taskk is requesting is less than the resources available in the node
// In here we only check the disk availability
func selectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node {
	var candidates []*node.Node
	for node := range nodes {
		if checkDisk(t, nodes[node].Disk-nodes[node].DiskAllocated) {
			candidates = append(candidates, nodes[node])
		}
	}

	return candidates
}

func checkDisk(t task.Task, diskAvailable int64) bool {
	return t.Disk <= diskAvailable
}

func calculateLoad(usage float64, capacity float64) float64 {
	return usage / capacity
}

func calculateCpuUsage(node *node.Node) (*float64, error) {
	stat1, err := node.GetStats()
	if err != nil {
		return nil, err
	}

	time.Sleep(3 * time.Second)

	stat2, err := node.GetStats()
	if err != nil {
		return nil, err
	}

	stat1Idle := stat1.CpuStats.Idle + stat1.CpuStats.IOWait
	stat2Idle := stat2.CpuStats.Idle + stat2.CpuStats.IOWait

	stat1NonIdle := stat1.CpuStats.User + stat1.CpuStats.Nice + stat1.CpuStats.System + stat1.CpuStats.IRQ + stat1.CpuStats.SoftIRQ + stat1.CpuStats.Steal
	stat2NonIdle := stat2.CpuStats.User + stat2.CpuStats.Nice + stat2.CpuStats.System + stat2.CpuStats.IRQ + stat2.CpuStats.SoftIRQ + stat2.CpuStats.Steal

	stat1Total := stat1Idle + stat1NonIdle
	stat2Total := stat2Idle + stat2NonIdle

	total := stat2Total - stat1Total
	idle := stat2Idle - stat1Idle

	var cpuPercentUsage float64
	if total == 0 && idle == 0 {
		cpuPercentUsage = 0.00
	} else {
		cpuPercentUsage = (float64(total) - float64(idle)) / float64(total)
	}

	return &cpuPercentUsage, nil
}
