package regression

import (
	"fmt"
	"io"
	"math"
	"os"
	"time"
)

const (
	Memory = "memory"
	Time   = "time"
)

// Result struct holds resource usage from a run.
type Result struct {
	Memory int64
	Wtime  time.Duration
	Stime  time.Duration
	Utime  time.Duration
}

// Comparison has the percentages of change between two runs.
type Comparison struct {
	Memory float64
	Wtime  float64
	Stime  float64
	Utime  float64
}

// Compare returns percentage difference between this and another run.
func (p *Result) Compare(q *Result) Comparison {
	return Comparison{
		Memory: Percent(p.Memory, q.Memory),
		Wtime:  Percent(int64(p.Wtime), int64(q.Wtime)),
		Stime:  Percent(int64(p.Stime), int64(q.Stime)),
		Utime:  Percent(int64(p.Utime), int64(q.Utime)),
	}
}

var CompareFormat = "%s: %v -> %v (%v), %v\n"

// ComparePrint does a result comparison, prints the result in human readable
// form and returns a bool if change is within allowance.
func (p *Result) ComparePrint(q *Result, allowance float64) bool {
	ok := true
	c := p.Compare(q)

	if c.Memory > allowance {
		ok = false
	}
	fmt.Printf(CompareFormat,
		"Memory",
		p.Memory,
		q.Memory,
		c.Memory,
		allowance > c.Memory,
	)

	if c.Wtime > allowance {
		ok = false
	}
	fmt.Printf(CompareFormat,
		"Wtime",
		p.Wtime,
		q.Wtime,
		c.Wtime,
		allowance > c.Wtime,
	)

	fmt.Printf(CompareFormat,
		"Stime",
		p.Stime,
		q.Stime,
		c.Stime,
		allowance > c.Stime,
	)

	fmt.Printf(CompareFormat,
		"Utime",
		p.Utime,
		q.Utime,
		c.Utime,
		allowance > c.Utime,
	)

	return ok
}

// Percent returns the percentage difference between to int64.
func Percent(a, b int64) float64 {
	diff := b - a
	return (float64(diff) / float64(a)) * 100
}

// Average gets the average consumption from a set of results. If the number of
// results is greater than 2 the first one is discarded as warmup run.
func Average(rs []*Result) *Result {
	agg := new(Result)

	// Discard first for warmup
	if len(rs) > 2 {
		rs = rs[1:]
	}

	for _, r := range rs {
		agg.Memory += r.Memory
		agg.Wtime += r.Wtime
		agg.Stime += r.Stime
		agg.Utime += r.Utime
	}

	agg.Memory = int64(math.Round(float64(agg.Memory) / float64(len(rs))))
	agg.Wtime = time.Duration(math.Round(float64(agg.Wtime) / float64(len(rs))))
	agg.Stime = time.Duration(math.Round(float64(agg.Stime) / float64(len(rs))))
	agg.Utime = time.Duration(math.Round(float64(agg.Utime) / float64(len(rs))))

	return agg
}

func ToMiB(i int64) float64 {
	return float64(i) / float64(1024*1024)
}

func (p *Result) SaveAllCSV(prefix string) error {
	for _, s := range []string{Memory, Time} {
		if err := p.SaveCSV(s, fmt.Sprintf("%s%s.csv", prefix, s)); err != nil {
			return err
		}
	}

	return nil
}

func (p *Result) SaveCSV(series string, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	if err := p.WriteCSV(series, f); err != nil {
		_ = f.Close()
		return err
	}

	return f.Close()
}

func (p *Result) WriteCSV(series string, w io.Writer) error {
	switch series {
	case Memory:
		_, err := fmt.Fprintf(w, "%s\n%f\n", Memory, ToMiB(p.Memory))
		return err
	case Time:
		_, err := fmt.Fprintf(w, "Wtime,Stime,Utime\n%f,%f,%f\n",
			p.Wtime.Seconds(), p.Stime.Seconds(), p.Utime.Seconds())
		return err
	default:
		return fmt.Errorf("unsupported series: %s", series)
	}
}
