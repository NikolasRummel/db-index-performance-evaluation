package bench

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
)

var indexColorHex = []string{
	"#9ecae1", "#4292c6", "#084594", // B-Tree (Blue)
	"#a1d99b", "#41ab5d", "#00441b", // B+ Tree (Green)
	"#fc9272", "#ef3b2c", "#67000d", // LSM (Red)
}

var colorBaseIndex = map[string]int{
	"btree":  0,
	"bptree": 3,
	"lsm":    6,
}

const (
	chartWidth  = "1000px"
	chartHeight = "500px"
)

// --- Helpers ---

func colorKey(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.HasPrefix(n, "bptree"), strings.HasPrefix(n, "b+"):
		return "bptree"
	case strings.HasPrefix(n, "btree"), strings.HasPrefix(n, "b-"):
		return "btree"
	case strings.HasPrefix(n, "lsm"), strings.HasPrefix(n, "pebble"):
		return "lsm"
	default:
		return "btree"
	}
}

func pickColor(name string, counters map[string]int) string {
	key := colorKey(name)
	idx := colorBaseIndex[key] + (counters[key] % 3)
	counters[key]++
	if idx >= len(indexColorHex) {
		idx = colorBaseIndex[key]
	}
	return indexColorHex[idx]
}

func renderPage(page *components.Page, path, label string) error {
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer out.Close()
	if err := page.Render(io.Writer(out)); err != nil {
		return fmt.Errorf("render %s: %w", path, err)
	}
	fmt.Printf("%s plot written to %s\n", label, path)
	return nil
}

// --- Plotters ---

func PlotAll(outDir string) error {
	plots := []struct {
		file, label string
		fn          func(string) error
	}{
		{"t1_point_query.csv", "T1", PlotT1},
		{"t2_range_query.csv", "T2", PlotT2},
		{"t3_write_throughput.csv", "T3", PlotT3},
		{"t4_read_heavy.csv", "T4", PlotT4},
		{"t5_write_heavy.csv", "T5", PlotT5},
	}
	for _, p := range plots {
		if _, err := os.Stat(filepath.Join(outDir, p.file)); os.IsNotExist(err) {
			fmt.Printf("[Plotter] Skipping %s: %s not found\n", p.label, p.file)
			continue
		}
		if err := p.fn(outDir); err != nil {
			return err
		}
	}
	return nil
}

func PlotT1(outDir string) error {
	f, _ := os.Open(filepath.Join(outDir, "t1_point_query.csv"))
	defer f.Close()
	records, _ := csv.NewReader(f).ReadAll()

	var labels []string
	var boxItems []opts.BoxPlotData
	var barItems []opts.BarData
	counters := make(map[string]int)

	for _, rec := range records[1:] {
		min, _ := strconv.ParseFloat(rec[3], 64)
		q1, _ := strconv.ParseFloat(rec[4], 64)
		p50, _ := strconv.ParseFloat(rec[5], 64)
		q3, _ := strconv.ParseFloat(rec[6], 64)
		p99, _ := strconv.ParseFloat(rec[10], 64)
		tput, _ := strconv.ParseFloat(rec[11], 64)

		color := pickColor(rec[0], counters)
		labels = append(labels, rec[0])
		boxItems = append(boxItems, opts.BoxPlotData{
			Value:     []interface{}{min, q1, p50, q3, p99},
			ItemStyle: &opts.ItemStyle{Color: color, BorderColor: color},
		})
		barItems = append(barItems, opts.BarData{
			Value:     tput,
			ItemStyle: &opts.ItemStyle{Color: color},
		})
	}

	box := charts.NewBoxPlot()
	box.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: "T1 — Point Query Latency", Subtitle: "min/Q1/median/Q3/p99"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "ns", Type: "log"}),
		charts.WithInitializationOpts(opts.Initialization{Width: chartWidth, Height: chartHeight}),
	)
	box.SetXAxis(labels).AddSeries("Latency", boxItems)

	bar := charts.NewBar()
	bar.SetGlobalOptions(charts.WithTitleOpts(opts.Title{Title: "T1 — Point Query Throughput"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Ops/sec"}),
		charts.WithInitializationOpts(opts.Initialization{Width: chartWidth, Height: chartHeight}),
	)
	bar.SetXAxis(labels).AddSeries("Throughput", barItems)

	page := components.NewPage()
	page.SetLayout(components.PageFlexLayout)
	page.AddCharts(box, bar)
	return renderPage(page, filepath.Join(outDir, "t1.html"), "[T1]")
}

// genericLinePlot handles CSV parsing and line chart generation for T2-T5
func genericLinePlot(outDir, file, title, yName, xName, outHtml string, isMixed bool) error {
	f, _ := os.Open(filepath.Join(outDir, file))
	defer f.Close()
	records, _ := csv.NewReader(f).ReadAll()

	byIndex := make(map[string][]opts.LineData)
	var indexOrder []string
	var xLabels []string
	seen := make(map[string]bool)

	for _, rec := range records[1:] {
		idxName, xVal := rec[0], rec[1]
		yVal, _ := strconv.ParseFloat(rec[2], 64)
		if file == "t2_range_query.csv" {
			yVal, _ = strconv.ParseFloat(rec[3], 64) // T2 uses totalMs at col 3
		}

		if !seen[idxName] {
			indexOrder = append(indexOrder, idxName)
			seen[idxName] = true
		}
		byIndex[idxName] = append(byIndex[idxName], opts.LineData{Value: yVal})
		if idxName == indexOrder[0] {
			xLabels = append(xLabels, xVal)
		}
	}

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: title}),
		charts.WithYAxisOpts(opts.YAxis{Name: yName, Type: "value"}),
		charts.WithXAxisOpts(opts.XAxis{Name: xName}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true), Top: "8%"}),
		charts.WithInitializationOpts(opts.Initialization{Width: chartWidth, Height: chartHeight}),
	)
	if isMixed {
		line.SetGlobalOptions(charts.WithDataZoomOpts(opts.DataZoom{Type: "slider", Start: 0, End: 100}))
	}

	counters := make(map[string]int)
	line.SetXAxis(xLabels)
	for _, name := range indexOrder {
		color := pickColor(name, counters)
		line.AddSeries(name, byIndex[name],
			charts.WithLineStyleOpts(opts.LineStyle{Color: color, Width: 2}),
			charts.WithItemStyleOpts(opts.ItemStyle{Color: color}), // Match dots to line
		)
	}

	page := components.NewPage()
	page.SetLayout(components.PageFlexLayout)
	page.AddCharts(line)
	return renderPage(page, filepath.Join(outDir, outHtml), "["+title[:2]+"]")
}

func PlotT2(outDir string) error {
	return genericLinePlot(outDir, "t2_range_query.csv", "T2 — Range Query Response Time", "Total ms", "Range Size", "t2.html", false)
}

func PlotT3(outDir string) error {
	return genericLinePlot(outDir, "t3_write_throughput.csv", "T3 — Write Throughput", "Ops/sec", "Dataset Growth", "t3.html", false)
}

func PlotT4(outDir string) error {
	return genericLinePlot(outDir, "t4_read_heavy.csv", "T4 — Read-Heavy (95/5)", "Latency (ns)", "Op Count", "t4.html", true)
}

func PlotT5(outDir string) error {
	return genericLinePlot(outDir, "t5_write_heavy.csv", "T5 — Write-Heavy (5/95)", "Latency (ns)", "Op Count", "t5.html", true)
}
