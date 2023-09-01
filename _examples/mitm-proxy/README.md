## Mitm Proxy

This example implements Https as a man-in-the-middle proxy.

You can go to the `examples/generateCert` directory to regenerate the certificate files

## Steps

1. Go to `examples/generateCert` to generate the certificate file.

2. Import the `ca.crt` certificate file into your system. 

If you want to use the Go client to make HTTPS requests, you need to configure the certificate, 
for example:
```go
func main() {
    // Load ca.crt file
    certPEMBlock, err := os.ReadFile("ca.crt")
    if err != nil {
        panic("failed to load ca.crt file")
    }

    // client cert pool
    clientCertPool := x509.NewCertPool()
    ok := clientCertPool.AppendCertsFromPEM(certPEMBlock)
    if !ok {
        panic("failed to parse root certificate")
    }
    
    // set Transport
    http.DefaultClient.Transport = &http.Transport{
        Proxy: func(r *http.Request) (*url.URL, error) {
            // mitm proxy server address. eg. "http://localhost:8080"
            return url.Parse("http://localhost:8080")
        },
        TLSClientConfig: &tls.Config{
            Certificates: []tls.Certificate{cert.DefaultCertificate},
            ClientAuth:   tls.RequireAndVerifyClientCert,
            RootCAs:      clientCertPool,
        },
    }

    // To send request
    req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        t.Fatal(err)
    }
    defer resp.Body.Close()
}
```
    

