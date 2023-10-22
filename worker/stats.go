package worker

// import (
// 	"github.com/c9s/goprocinfo/linux"
// 	"log"
// )

// type Stats struct {
// 	MemStats  *linux.MemInfo
// 	DiskStats *linux.Disk
// 	CpuStats  *linux.CPUStat
// 	LoadStats *linux.LoadAvg
// 	TaskCount int
// }

// // Stats related to Memory - Used by a worker
// func (s *Stats) MemUsedKb() uint64 {
// 	return s.MemStats.MemTotal - s.MemStats.MemAvailable
// }

// func (s *Stats) MemUsedPercent() uint64 {
// 	return s.MemStats.MemAvailable / s.MemStats.MemTotal
// }

// func (s *Stats) MemAvailableKb() uint64 {
// 	return s.MemStats.MemAvailable
// }

// func (s *Stats) MemToalKb() uint64 {
// 	return s.MemStats.MemTotal
// }

// // Stats related to Disk - Used by a worker
// func (s *Stats) DiskTotal() uint64 {
// 	return s.DiskStats.All
// }

// func (s *Stats) DiskFree() uint64 {
// 	return s.DiskStats.Free
// }

// func (s *Stats) DiskUsed() uint64 {
// 	return s.DiskStats.Used
// }

// // Stats related to CPU - Used by a worker
// func (s *Stats) CpuUsage() float64 {
// 	// Accurate calculation of CPU usage given in percentage on Linux(https://stackoverflow.com/questions/23367857/accurate-calculation-of-cpu-usage-given-in-percentage-in-linux)
// 	/*
// 		ALGORITHM - CPU USAGE CALCULATION
// 		1. Sum of the values for idle states
// 		2. Sum of the values for non-idle states
// 		3. Total = SUM(idle states) + SUM(non-idle states)
// 		4. Usage = (Total - Idle) / Total 
// 	*/
	
// 	idle := s.CpuStats.Idle + s.CpuStats.IOWait
// 	nonIdle := s.CpuStats.User + s.CpuStats.Nice + s.CpuStats.System + s.CpuStats.IRQ + s.CpuStats.SoftIRQ + s.CpuStats.Steal
// 	total := idle + nonIdle

// 	if total == 0 {
// 		return 0.00
// 	}

// 	return (float64(total) - float64(idle)) / float64(total)
// }

// // Compact Stats
// func GetStats() *Stats {
// 	return &Stats{
// 		MemStats:  GetMemoryInfo(),
// 		DiskStats: GetDiskInfo(),
// 		CpuStats:  GetCpuStats(),
// 		LoadStats: GetLoadAvg(),
// 	}
// }

// // Helper functions with fully populated stat values - Can use on worker's API
// func GetMemoryInfo() *linux.MemInfo {
// 	memstats, err := linux.ReadMemInfo("/proc/meminfo")

// 	if err != nil {
// 		log.Printf("Error reading from /proc/meminfo")
// 		return &linux.MemInfo{}
// 	}

// 	return memstats
// }

// func GetDiskInfo() *linux.Disk {
// 	diskstats, err := linux.ReadDisk("/")

// 	if err != nil {
// 		log.Printf("Error reading from /")
// 		return &linux.Disk{}
// 	}

// 	return diskstats
// }

// func GetCpuStats() *linux.CPUStat {
// 	stats, err := linux.ReadStat("/proc/stat")

// 	if err != nil {
// 		log.Printf("Error reading from /proc/stat")
// 		return &linux.CPUStat{}
// 	}

// 	return &stats.CPUStatAll
// }

// func GetLoadAvg() *linux.LoadAvg {
// 	loadavg, err := linux.ReadLoadAvg("/proc/loadavg")

// 	if err != nil {
// 		log.Printf("Error reading from /proc/loading")
// 		return &linux.LoadAvg{}
// 	}

// 	return loadavg
// }
