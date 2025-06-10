# cspgrabber

`cspgrabber` is a Go tool designed to extract domains from Content Security Policy (CSP) headers of websites. It supports scanning a single URL, list of URLs with configurable concurrency, rate limiting, and can optionally remove wildcard prefixes (*.) from extracted domains.

#### Install
```
git clone https://github.com/reewardius/cspgrabber && cd cspgrabber/
go mod init main.go && go mod tidy && go build -o cspgrabber
```
#### Usage
```
Usage of ./cspgrabber:

  -c int
        Number of concurrent workers (default 5)
  -clean
        Remove *. prefix from domains
  -f string
        Path to file with list of URLs
  -o string
        Output file to save found domains
  -r float
        Rate limit in seconds between requests per worker (default 0.5)
  -u string
        Single URL to scan
```

#### Scan a single URL
Extract domains from the CSP header of a single website:
```
./cspgrabber -u https://example.com
```

#### Scan a List of URLs
Extract domains from CSP headers of multiple URLs listed in a file:
```
./cspgrabber -f alive_http_services.txt -c 20 -r 0.1 -o csp_domains.txt
```

#### Scan and Clean Wildcard Prefixes
Extract domains from a list of URLs and remove *. prefixes from the output:
```
./cspgrabber -f alive_http_services.txt -c 20 -r 0.1 -o csp_domains.txt -clean
```

#### CSP takeover
```
./cspgrabber -f alive_http_services.txt -c 20 -r 0.1 -clean -o csp_domains.txt
nuclei -l csp_domains.txt -profile subdomain-takeovers -nh -o csp_takeovers.txt
```
