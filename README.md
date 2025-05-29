# cspgrabber
#### Usage
```
Usage of ./cspgrabber:

  -c int
        Number of concurrent workers (default 5)
  -f string
        Path to file with list of URLs
  -o string
        Output file to save found domains
  -r float
        Rate limit in seconds between requests per worker (default 0.5)
  -u string
        Single URL to scan
```
#### Build
```
go build -o cspgrabber main.go
```

#### Single domain
```
./cspgrabber -u https://example.com
```

#### List of domains
```
./cspgrabber -f urls.txt -c 20 -r 0.1 -o out.txt
```

#### CSP takeover
```
awk '{gsub(/^\*\./, "", $0); print}' out.txt > temp && mv temp input.txt
nuclei -l input.txt -profile subdomain-takeovers -nh -o csp_takeovers.txt
```
