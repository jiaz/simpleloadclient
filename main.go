package main

import (
    "log"
    "flag"
    "time"
    "net"
    "net/http"
    "sync/atomic"
    "io/ioutil"
)

var defReq = "https://google.com"

var ops uint32 = 0

type DialerFunc func(net, addr string) (net.Conn, error)

func make_dialer(keepAlive bool) DialerFunc {
    return func(network, addr string) (net.Conn, error) {
        conn, err := (&net.Dialer{
            Timeout:   30 * time.Second,
            KeepAlive: 3000 * time.Second,
        }).Dial(network, addr)
        if err != nil {
            return conn, err
        }
        if !keepAlive {
            conn.(*net.TCPConn).SetLinger(0)
        }
        return conn, err
    }
}

func sendRequest(client *http.Client, req string, concurrency int) {
    resp, err := client.Get(req)
    if err != nil {
        log.Println(err)
    } else {
        _, err := ioutil.ReadAll(resp.Body)
        resp.Body.Close()
        if err != nil {
            log.Println(err)
        }
        atomic.AddUint32(&ops, 1)
    }
}

func main() {
    concurrency := flag.Int("concurrency", 10, "Concurent connection to the server")
    maxQPS := flag.Int("maxQPS", 1000, "Maximum QPS the client will generate")
    req := flag.String("req", defReq, "Request url")
    keepAlive := flag.Bool("keepAlive", true, "Whether to keep connection alive, if enabled keep alive for 5 min")

    flag.Parse()

    log.Printf("Current concurrency: %d\n", *concurrency)
    log.Printf("Current max QPS: %d\n", *maxQPS)
    log.Printf("KeepAlive: %v\n", *keepAlive)
    log.Printf("Url: %s\n", *req)

    fin := make(chan bool)
    bucket := make(chan bool, *maxQPS)

    go func() {
        // QPS calc
        for {
            currentOps := ops;
            time.Sleep(time.Second)
            var qps int32 = (int32)(ops - currentOps);
            log.Printf("QPS: %d\n", qps)
        }
    }()

    go func() {
        for {
            for i := 0; i < *maxQPS; i++ {
                select {
                case bucket <- true:
                default:
                }
            }
            time.Sleep(time.Second)
        }
    }()

    for i := 0; i < *concurrency; i++ {
        go func() {
            tr := &http.Transport{
                Dial: make_dialer(*keepAlive),
                TLSHandshakeTimeout: 10 * time.Second,
                DisableKeepAlives: !(*keepAlive),
                MaxIdleConnsPerHost: *concurrency,
            }
            client := &http.Client{Transport: tr}

            for {
                <- bucket

                sendRequest(client, *req, *concurrency)
            }
        }()
    }
    // make it never end
    <- fin
}
