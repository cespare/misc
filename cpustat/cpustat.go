package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	log.SetFlags(log.Lshortfile)

	cur := parseCPUStats()
	prev := loadCPUStats()
	if prev != nil {
		if len(prev) != len(cur) {
			log.Fatalf("prev has %d entries; cur has %d", len(prev), len(cur))
		}
		for i := range cur {
			deltaIdle := float32(cur[i].idle - prev[i].idle)
			deltaTotal := float32(cur[i].total - prev[i].total)
			pct := int(100 * (1 - deltaIdle/deltaTotal))
			name := "cpu"
			if i > 0 {
				name = fmt.Sprintf("cpu%d", i-1)
			}
			fmt.Printf("%s %d%%\n", name, pct)
		}
	}
	storeCPUStats(cur)
}

type cpuStat struct {
	idle  int64
	total int64
}

func parseCPUStats() []cpuStat {
	f, err := os.Open("/proc/stat")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var stats []cpuStat
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if !strings.HasPrefix(fields[0], "cpu") {
			break
		}
		if len(stats) == 0 {
			if fields[0] != "cpu" {
				log.Fatal("unexpected header:", fields[0])
			}
		} else {
			if fields[0] != fmt.Sprintf("cpu%d", len(stats)-1) {
				log.Fatal("unexpected header:", fields[0])
			}
		}
		var st cpuStat
		for i := 1; i < len(fields); i++ {
			n, err := strconv.ParseInt(fields[i], 10, 64)
			if err != nil {
				log.Fatal(err)
			}
			st.total += n
			if i == 4 || i == 5 {
				st.idle += n
			}
		}
		stats = append(stats, st)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return stats
}

func storeCPUStats(stats []cpuStat) {
	f, err := os.Create("stats.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	for _, st := range stats {
		fmt.Fprintf(f, "%d %d\n", st.idle, st.total)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}

func loadCPUStats() []cpuStat {
	f, err := os.Open("stats.txt")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		log.Fatal(err)
	}
	defer f.Close()

	var stats []cpuStat
	for {
		var st cpuStat
		_, err := fmt.Fscanf(f, "%d %d\n", &st.idle, &st.total)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		stats = append(stats, st)
	}
	return stats
}
