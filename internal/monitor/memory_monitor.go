package monitor

import "pxm-operator/internal/proxmox"

type MemoryMonitor struct{
	Client *proxmox.Client 
	Threshold float64 
}

func NewMemoryMonitor(c *proxmox.Client, th float64) *MemoryMonitor{
	return &MemoryMonitor{
		Client: c,
		Threshold: th,
	}
}

func (m *MemoryMonitor) CheckMemoryPressure() ([]string, error) {
	nodes, err := m.Client.GetNodes()
	if err != nil{
		return nil, err
	}

	var highMemoryNodes []string
	for _, node := range nodes {
		if float64(node.Mem) / float64(node.MaxMem) > m.Threshold {
			highMemoryNodes = append(highMemoryNodes, node.Name)
		}
	}
	return highMemoryNodes, nil
}