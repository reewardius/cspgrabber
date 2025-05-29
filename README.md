# cspgrabber

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
./cspgrabber -f urls.txt -o out.txt -c 20 -r 0.1
```

#### CSP takeover
```
awk '{gsub(/^\*\./, "", $0); print}' out.txt > temp && mv temp input.txt
nuclei -l input.txt -profile subdomain-takeovers -nh -o csp_takeovers.txt
```
