package proxmox

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"bytes"
)

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

type Node struct {
	Name   string  `json:"node"`
	CPU    float64 `json:"cpu"`
	MaxCPU int     `json:"maxcpu"`
	Mem    int64   `json:"mem"`
	MaxMem int64   `json:"maxmem"`
}

type VM struct {
	VMID   int     `json:"vmid"`
	Name   string  `json:"name"`
	Status string  `json:"status"`
	CPU    float64 `json:"cpu"`
	CPUs   int     `json:"cpus"`
	Mem    int64   `json:"mem"`
	MaxMem int64   `json:"maxmem"`
	Node   string
}

type Migration struct {
	VMID int `json:"vmid"`
	Name string `json:"name"`
	SourceNode string `json:"source_node"`
	TargetNode string `json:"target_node"`
}

type NodesResponse struct {
	Data []Node `json:"data"`
}

type VMsResponse struct {
	Data []VM `json:"data"`
}

type MigrationResponse struct {
    AllowedNodes []string `json:"allowed_nodes"`
    Running int `json:"running"`
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func (c *Client) GetNodes() ([]Node, error) {
	url := fmt.Sprintf("%s/api2/json/nodes", c.BaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s", c.Token))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var nodesResp NodesResponse
	err = json.Unmarshal(body, &nodesResp)
	if err != nil {
		return nil, err
	}

	return nodesResp.Data, nil
}

func (c *Client) GetNodeVMs(nodeName string) ([]VM, error) {
	url := fmt.Sprintf("%s/api2/json/nodes/%s/qemu", c.BaseURL, nodeName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s", c.Token))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var vmsResp VMsResponse
	err = json.Unmarshal(body, &vmsResp)
	if err != nil {
		return nil, err
	}

	return vmsResp.Data, nil
}

func (c *Client) GetAllVMs() ([]VM, error) {
	nodes, err := c.GetNodes()
	if err != nil {
		return nil, err // ノード一覧取得失敗は致命的
	}

	var allVMs []VM
	var errors []string // エラーを蓄積

	for _, node := range nodes {
		vms, err := c.GetNodeVMs(node.Name)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Node %s: %v", node.Name, err))
			continue // このノードはスキップ、他は継続
		}

		for i := range vms {
			vms[i].Node = node.Name
		}

		allVMs = append(allVMs, vms...)
	}

	if len(errors) > 0 {
		log.Printf("Warning: Failed to get VMs from some nodes: %v", errors)
	}

	return allVMs, nil
}

func (c *Client) GetMigrationTargets(vmid int, sourceNode string) ([]string, error) {
    url := fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%d/migrate", c.BaseURL, sourceNode, vmid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s", c.Token))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var migrationResp MigrationResponse
	err = json.Unmarshal(body, &migrationResp)
	if err != nil {
		return nil, err
	}

	return migrationResp.AllowedNodes, nil
}

func (c *Client) ExecuteMigration (m Migration) error {
	url := fmt.Sprintf(
		"%s/api2/json/nodes/%s/qemu/%d/migrate",
        c.BaseURL, m.SourceNode, m.VMID,
	)

	 // Body用の構造体
    body := struct {
        Target string `json:"target"`
        Online bool   `json:"online"`
    }{
        Target: m.TargetNode,
        Online: true, // Live Migration
    }

	jsonData, err := json.Marshal(body)
    if err != nil {
        return err
    }

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    
    req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s", c.Token))
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("migration failed: status %d, response: %s", 
            resp.StatusCode, string(body))
    }
    
    fmt.Printf("Migration started: VM%d %s → %s\n", 
        m.VMID, m.SourceNode, m.TargetNode)
    
    return nil
}
