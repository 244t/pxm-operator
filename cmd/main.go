package main

import (
	"fmt"
	"log"
	"os"

	"pxm-operator/internal/proxmox"
	"pxm-operator/internal/monitor"
	"pxm-operator/internal/qmp"

	"github.com/joho/godotenv"
)

func main() {
	// .env ファイルを読み込み
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// 環境変数から設定取得
	proxmoxURL := os.Getenv("PROXMOX_URL")
	proxmoxToken := os.Getenv("PROXMOX_TOKEN")

	if proxmoxURL == "" || proxmoxToken == "" {
		log.Fatalf("PROXMOX_URL and PROXMOX_TOKEN must be set in .env file")
	}

	client := proxmox.NewClient(proxmoxURL, proxmoxToken)

	fmt.Println("Fetching nodes...")
	nodes, err := client.GetNodes()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Found %d node(s):\n", len(nodes))
	for _, node := range nodes {
		fmt.Printf("- Node: %s\n", node.Name)
		fmt.Printf("  CPU: %.2f%% (%d cores)\n", node.CPU*100, node.MaxCPU)
		fmt.Printf("  Memory: %.2f GB / %.2f GB\n",
			float64(node.Mem)/1024/1024/1024,
			float64(node.MaxMem)/1024/1024/1024)
		fmt.Println()
	}

	fmt.Println("Fetching  all VMs...")
	allVMs, err := client.GetAllVMs()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Found %d VM(s):\n", len(allVMs))
	for _, vm := range allVMs {
		fmt.Printf("- VM: %s (ID: %d)\n", vm.Name, vm.VMID)
		fmt.Printf("  Node: %s\n", vm.Node)
		fmt.Printf("  Status: %s\n", vm.Status)
		fmt.Printf("  CPU: %.2f%% (%d cores)\n", vm.CPU*100, vm.CPUs)
		fmt.Printf("  Memory: %.2f GB / %.2f GB\n", float64(vm.Mem)/1024/1024/1024, float64(vm.MaxMem)/1024/1024/1024)
		fmt.Println()
	}

	targets, err := client.GetMigrationTargets(100, "tanishi")
	if err != nil {
		log.Printf("Error getting migration targets: %v", err)
	} else {
		fmt.Printf("Migration targets for VM100: %v\n", targets)
	}


	m := monitor.NewMemoryMonitor(client, 0.9)
	highMemoryNodes, err := m.CheckMemoryPressure()
	if err != nil {
		log.Printf("Error check memory pressure: %v", err)
	}
	if len(highMemoryNodes) > 0{
		fmt.Printf("High memory nodes: %v\n", highMemoryNodes)
	} else {
		fmt.Printf("All nodes have sufficient memory\n")
	}

	fmt.Println("QMP Dirty Rate Test")
    fmt.Println("=====================")
    
    // QMP接続
    fmt.Println("Connecting to QMP...")
    qmpClient, err := qmp.NewQMPClient("192.168.1.100", 4444)
    if err != nil {
        log.Fatalf("Failed to connect to QMP: %v", err)
    }
    defer qmpClient.Close()
    
    fmt.Println("✅ QMP connection established!")
    
    // Dirty Rate測定
    fmt.Println("📊 Measuring Dirty Rate (10 seconds)...")
    result, err := qmpClient.GetDirtyRate(10)
    if err != nil {
        log.Fatalf("Failed to get dirty rate: %v", err)
    }
    
    // 結果表示
    fmt.Println("✅ Measurement completed!")
    fmt.Printf("📈 Results:\n")
    fmt.Printf("   Status: %s\n", result.Status)
    fmt.Printf("   Dirty Rate: %.2f MB/s\n", result.DirtyRate)
    fmt.Printf("   Calc Time: %d %s\n", result.CalcTime, result.CalcTimeUnit)
    fmt.Printf("   Mode: %s\n", result.Mode)
    fmt.Printf("   Sample Pages: %d\n", result.SamplePages)
    
    fmt.Println("🎉 Test completed successfully!")
}
