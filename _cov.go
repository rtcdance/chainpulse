package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	out, _ := exec.Command("go", "tool", "cover", "-func=coverage.out").Output()
	pkg := make(map[string][]int)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "total:") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		pctStr := strings.TrimSuffix(parts[len(parts)-1], "%")
		stmtsStr := parts[len(parts)-2]
		funcPath := parts[0]

		pct, err := strconv.ParseFloat(pctStr, 64)
		if err != nil {
			continue
		}
		stmts, err := strconv.Atoi(stmtsStr)
		if err != nil {
			continue
		}

		p := funcPath
		if idx := strings.LastIndex(p, ":"); idx >= 0 {
			p = p[:idx]
		}
		if idx := strings.LastIndex(p, "/"); idx >= 0 {
			p = p[:idx]
		}
		if _, ok := pkg[p]; !ok {
			pkg[p] = []int{0, 0}
		}
		pkg[p][0] += stmts
		pkg[p][1] += int(float64(stmts) * pct / 100.0)
	}

	type entry struct {
		name      string
		total     int
		covered   int
		uncovered int
		pct       float64
	}
	var entries []entry
	for p, v := range pkg {
		total := v[0]
		cov := v[1]
		uncov := total - cov
		entries = append(entries, entry{p, total, cov, uncov, float64(cov) / float64(total) * 100})
	}

	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].uncovered > entries[i].uncovered {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	for i, e := range entries {
		if i >= 20 || e.uncovered < 20 {
			break
		}
		fmt.Printf("%5d uncovered, %5.1f%% (%5d total): %s\n", e.uncovered, e.pct, e.total, e.name)
	}
}
