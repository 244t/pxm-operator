package qmp

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"
)

type QMPClient struct {
	conn net.Conn
	enc *json.Encoder
	dec *json.Decoder
}

type QMPGreeting struct {
    QMP struct {
        Version struct {
            QEMU struct {
                Major int `json:"major"`
                Minor int `json:"minor"`
                Micro int `json:"micro"`
            } `json:"qemu"`
            Package string `json:"package"`
        } `json:"version"`
        Capabilities []string `json:"capabilities"`
    } `json:"QMP"`
}

type QMPCommand struct {
    Execute   string                 `json:"execute"`
    Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type QMPResponse struct {
    Return interface{} `json:"return,omitempty"`
    Error  *QMPError   `json:"error,omitempty"`
}

type QMPError struct {
    Class string `json:"class"`
    Desc  string `json:"desc"`
}

type DirtyRateResult struct {
    Status       string  `json:"status"`
    DirtyRate    float64 `json:"dirty-rate"`
    CalcTime     int     `json:"calc-time"`
    CalcTimeUnit string  `json:"calc-time-unit"`
    SamplePages  int     `json:"sample-pages"`
    Mode         string  `json:"mode"`
    StartTime    int64   `json:"start-time"`
}

func NewQMPClient(host string, port int) (*QMPClient, error) {
    address := net.JoinHostPort(host, strconv.Itoa(port))
    conn, err := net.Dial("tcp", address)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to QMP: %v", err)
    }
    
    client := &QMPClient{
        conn: conn,
        enc:  json.NewEncoder(conn),
        dec:  json.NewDecoder(conn),
    }
    
    // 初期化実行
    if err := client.handshake(); err != nil {
        client.Close()
        return nil, err
    }
    
    return client, nil
}

func (q *QMPClient) Close() error {
    return q.conn.Close()
}

func (q *QMPClient) handshake() error {
    // QMP Greeting受信
    var greeting QMPGreeting
    if err := q.dec.Decode(&greeting); err != nil {
        return fmt.Errorf("failed to receive QMP greeting: %v", err)
    }
    
    // qmp_capabilities送信
    cmd := QMPCommand{Execute: "qmp_capabilities"}
    if err := q.enc.Encode(cmd); err != nil {
        return fmt.Errorf("failed to send qmp_capabilities: %v", err)
    }
    
    // レスポンス受信
    var resp QMPResponse
    if err := q.dec.Decode(&resp); err != nil {
        return fmt.Errorf("failed to receive capabilities response: %v", err)
    }
    
    if resp.Error != nil {
        return fmt.Errorf("QMP capabilities error: %s - %s", resp.Error.Class, resp.Error.Desc)
    }
    
    return nil
}

func (q *QMPClient) CalcDirtyRate(calcTime int) error {
    cmd := QMPCommand{
        Execute: "calc-dirty-rate",
        Arguments: map[string]interface{}{
            "calc-time": calcTime,
        },
    }
    
    if err := q.enc.Encode(cmd); err != nil {
        return fmt.Errorf("failed to send calc-dirty-rate: %v", err)
    }
    
    var resp QMPResponse
    if err := q.dec.Decode(&resp); err != nil {
        return fmt.Errorf("failed to receive calc-dirty-rate response: %v", err)
    }
    
    if resp.Error != nil {
        return fmt.Errorf("calc-dirty-rate error: %s - %s", resp.Error.Class, resp.Error.Desc)
    }
    
    return nil
}

func (q *QMPClient) QueryDirtyRate() (*DirtyRateResult, error) {
    cmd := QMPCommand{Execute: "query-dirty-rate"}
    
    if err := q.enc.Encode(cmd); err != nil {
        return nil, fmt.Errorf("failed to send query-dirty-rate: %v", err)
    }
    
    var resp QMPResponse
    if err := q.dec.Decode(&resp); err != nil {
        return nil, fmt.Errorf("failed to receive query-dirty-rate response: %v", err)
    }
    
    if resp.Error != nil {
        return nil, fmt.Errorf("query-dirty-rate error: %s - %s", resp.Error.Class, resp.Error.Desc)
    }
    
    // レスポンスを構造体に変換
    resultBytes, _ := json.Marshal(resp.Return)
    var result DirtyRateResult
    if err := json.Unmarshal(resultBytes, &result); err != nil {
        return nil, fmt.Errorf("failed to parse dirty rate result: %v", err)
    }
    
    return &result, nil
}

func (q *QMPClient) GetDirtyRate(calcTime int) (*DirtyRateResult, error) {
    // 1. 測定開始
    if err := q.CalcDirtyRate(calcTime); err != nil {
        return nil, err
    }
    
    // 2. 測定完了まで待機
    time.Sleep(time.Duration(calcTime) * time.Second*2)
    
    // 3. 結果取得
    result, err := q.QueryDirtyRate()
    if err != nil {
        return nil, err
    }
    
    // 4. 測定完了確認
    if result.Status != "measured" {
        return nil, fmt.Errorf("measurement not completed: status=%s", result.Status)
    }
    
    return result, nil
}
