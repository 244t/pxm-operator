package proxmox

import (
    "crypto/tls"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
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

type NodesResponse struct {
    Data []Node `json:"data"`
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
