package main

import (
    "bufio"
    "context"
    "crypto/tls"
    "flag"
    "fmt"
    "net/http"
    "os"
    "strings"
    "sync"
    "time"
)

var (
    outputFile *os.File
    domainSet  = make(map[string]struct{})
    domainMux  sync.Mutex
    outputMux  sync.Mutex
    cleanWildcards bool // Новый флаг для очистки *. префикса
)

// extractDomainsFromCSP парсит CSP и извлекает домены из всех директив
func extractDomainsFromCSP(csp string) []string {
    result := []string{}
    tokens := strings.Fields(csp)

    for _, token := range tokens {
        // Пропускаем директивы (оканчивающиеся на :) и ключевые слова в кавычках
        if strings.HasSuffix(token, ":") || (strings.HasPrefix(token, "'") && strings.HasSuffix(token, "'")) {
            continue
        }

        cleaned := token

        // Удаляем схемы, если есть
        cleaned = strings.TrimPrefix(cleaned, "https://")
        cleaned = strings.TrimPrefix(cleaned, "http://")

        // Удаляем ведущие '*.' если есть
        original := cleaned
        cleaned = strings.TrimPrefix(cleaned, "*.")

        // Удаляем конечный символ ';' если есть
        cleaned = strings.TrimSuffix(cleaned, ";")

        // Убираем путь, если есть (оставляем только домен)
        if idx := strings.Index(cleaned, "/"); idx != -1 {
            cleaned = cleaned[:idx]
        }

        // Убираем порт, если есть
        if idx := strings.Index(cleaned, ":"); idx != -1 {
            cleaned = cleaned[:idx]
        }

        // Фильтруем пустые и не доменные строки
        if cleaned == "" || !strings.Contains(cleaned, ".") {
            continue
        }

        domainMux.Lock()
        if _, exists := domainSet[cleaned]; !exists {
            domainSet[cleaned] = struct{}{}
            if cleanWildcards {
                result = append(result, cleaned)
            } else {
                result = append(result, original)
            }
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
        return
    }

    resp, err := client.Do(req)
    if err != nil {
        return
    }
    defer resp.Body.Close()

    csp := resp.Header.Get("Content-Security-Policy")
    if csp == "" {
        return
    }

    domains := extractDomainsFromCSP(csp)
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
    flag.BoolVar(&cleanWildcards, "clean", false, "Remove *. prefix from domains") // Новый флаг

    flag.Parse()

    if *filePath == "" && *singleURL == "" {
        fmt.Println("Usage:")
        fmt.Println("  -f urls.txt   (for list of URLs)")
        fmt.Println("  -u https://example.com   (for single URL)")
        fmt.Println("  -clean        (remove *. prefix from domains)")
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

    if *singleURL != "" {
        fetchCSPDomains(*singleURL, client)
        return
    }

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