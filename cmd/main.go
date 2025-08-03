package main

import (
    "fmt"
    "log"
    "os"
    
    "github.com/joho/godotenv"
    "pxm-operator/internal/proxmox"
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
}