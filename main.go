package main

import (
    "bufio"
    "context"
    "crypto/tls"
    "flag"
    "fmt"
    "net/http"
    "os"
    "regexp"
    "strings"
    "sync"
    "time"
)

var (
    domainRegex = regexp.MustCompile(`https?://([^ ;]+)`)
    outputFile  *os.File
    domainSet   = make(map[string]struct{})
    domainMux   sync.Mutex
    outputMux   sync.Mutex
)

func extractDomains(csp string) []string {
    matches := domainRegex.FindAllStringSubmatch(csp, -1)
    result := []string{}
    for _, match := range matches {
        domain := match[1]
        domainMux.Lock()
        if _, exists := domainSet[domain]; !exists {
            domainSet[domain] = struct{}{}
            result = append(result, domain)
        }
        domainMux.Unlock()
    }
    return result
}

func fetchCSPDomains(target string, client *http.Client) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
    if err != nil {
        return // игнорируем ошибку
    }

    resp, err := client.Do(req)
    if err != nil {
        return // игнорируем ошибку
    }
    defer resp.Body.Close()

    csp := resp.Header.Get("Content-Security-Policy")
    if csp == "" {
        return
    }

    domains := extractDomains(csp)
    for _, d := range domains {
        fmt.Println(d)
        if outputFile != nil {
            outputMux.Lock()
            fmt.Fprintln(outputFile, d)
            outputMux.Unlock()
        }
    }
}

func worker(jobs <-chan string, wg *sync.WaitGroup, client *http.Client, rate time.Duration) {
    defer wg.Done()
    for url := range jobs {
        fetchCSPDomains(url, client)
        time.Sleep(rate)
    }
}

func main() {
    filePath := flag.String("f", "", "Path to file with list of URLs")
    singleURL := flag.String("u", "", "Single URL to scan")
    outPath := flag.String("o", "", "Output file to save found domains")
    concurrency := flag.Int("c", 5, "Number of concurrent workers")
    rateLimit := flag.Float64("r", 0.5, "Rate limit in seconds between requests per worker")

    flag.Parse()

    if *filePath == "" && *singleURL == "" {
        fmt.Println("Usage:")
        fmt.Println("  -f urls.txt   (for list of URLs)")
        fmt.Println("  -u https://example.com   (for single URL)")
        os.Exit(1)
    }

    var err error
    if *outPath != "" {
        outputFile, err = os.Create(*outPath)
        if err != nil {
            fmt.Printf("Failed to open output file: %v\n", err)
            os.Exit(1)
        }
        defer outputFile.Close()
    }

    client := &http.Client{
        Timeout: 15 * time.Second,
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
        },
    }

    // Обработка одного URL
    if *singleURL != "" {
        fetchCSPDomains(*singleURL, client)
        return
    }

    // Чтение из файла и распределение по worker-ам
    file, err := os.Open(*filePath)
    if err != nil {
        fmt.Printf("Failed to open input file: %v\n", err)
        os.Exit(1)
    }
    defer file.Close()

    jobs := make(chan string, *concurrency)
    var wg sync.WaitGroup

    for i := 0; i < *concurrency; i++ {
        wg.Add(1)
        go worker(jobs, &wg, client, time.Duration(float64(time.Second)*(*rateLimit)))
    }

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        jobs <- line
    }
    close(jobs)
    wg.Wait()

    if err := scanner.Err(); err != nil {
        fmt.Printf("Error reading file: %v\n", err)
    }
}
