# SRE
SRE home exercise
# Fetch SRE Take-Home Exercise (Go)

This project is a Go-based monitoring service that checks the availability of configured HTTP endpoints every 15 seconds and reports domain-level availability percentages.

---

## How to Run

###  Prerequisites

- Go 1.18 or later installed on your system (https://go.dev/dl/)
- A terminal or shell environment
- A YAML configuration file matching the sample format (e.g., `sample.yaml`)

### Run the Program

```bash
go run main.go sample.yaml
