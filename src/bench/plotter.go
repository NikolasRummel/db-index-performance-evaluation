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
	var p95Items []opts.BarData
	var barItems []opts.BarData
	counters := make(map[string]int)

	for _, rec := range records[1:] {
		p95, _ := strconv.ParseFloat(rec[9], 64) // p95 is index 9
		tput, _ := strconv.ParseFloat(rec[11], 64)

		color := pickColor(rec[0], counters)
		labels = append(labels, rec[0])
		p95Items = append(p95Items, opts.BarData{
			Value:     p95,
			ItemStyle: &opts.ItemStyle{Color: color},
		})
		barItems = append(barItems, opts.BarData{
			Value:     tput,
			ItemStyle: &opts.ItemStyle{Color: color},
		})
	}

	p95Bar := charts.NewBar()
	p95Bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: "T1 — Point Query P95 Response Time"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "ns", Type: "value"}),
		charts.WithInitializationOpts(opts.Initialization{Width: chartWidth, Height: chartHeight}),
	)
	p95Bar.SetXAxis(labels).AddSeries("P95 Response Time", p95Items)

	bar := charts.NewBar()
	bar.SetGlobalOptions(charts.WithTitleOpts(opts.Title{Title: "T1 — Point Query Throughput"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Ops/sec"}),
		charts.WithInitializationOpts(opts.Initialization{Width: chartWidth, Height: chartHeight}),
	)
	bar.SetXAxis(labels).AddSeries("Throughput", barItems)

	page := components.NewPage()
	page.SetLayout(components.PageFlexLayout)
	page.AddCharts(p95Bar, bar)
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

func PlotMixed(outDir, file, title, outHtml string) error {
	// Detailed line plot
	linePage := components.NewPage()
	linePage.SetLayout(components.PageFlexLayout)

	lineChart := charts.NewLine()
	lineChart.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: title}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Response Time (ns)", Type: "value"}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Op Count"}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true), Top: "8%"}),
		charts.WithInitializationOpts(opts.Initialization{Width: chartWidth, Height: chartHeight}),
		charts.WithDataZoomOpts(opts.DataZoom{Type: "slider", Start: 0, End: 100}),
	)

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
		if !seen[idxName] {
			indexOrder = append(indexOrder, idxName)
			seen[idxName] = true
		}
		byIndex[idxName] = append(byIndex[idxName], opts.LineData{Value: yVal})
		if idxName == indexOrder[0] {
			xLabels = append(xLabels, xVal)
		}
	}
	lineChart.SetXAxis(xLabels)
	counters := make(map[string]int)
	for _, name := range indexOrder {
		color := pickColor(name, counters)
		lineChart.AddSeries(name, byIndex[name],
			charts.WithLineStyleOpts(opts.LineStyle{Color: color, Width: 2}),
		)
	}

	// Summary bar chart
	sumFile := file[:len(file)-len(".csv")] + "_summary.csv"
	sf, _ := os.Open(filepath.Join(outDir, sumFile))
	defer sf.Close()
	sumRecords, _ := csv.NewReader(sf).ReadAll()

	var sumLabels []string
	var p50Read, p95Read, p50Write, p95Write []opts.BarData
	sumSeen := make(map[string]bool)

	for _, rec := range sumRecords[1:] {
		idxName, opType := rec[0], rec[1]
		p50, _ := strconv.ParseFloat(rec[4], 64)
		p95, _ := strconv.ParseFloat(rec[5], 64)

		if !sumSeen[idxName] {
			sumLabels = append(sumLabels, idxName)
			sumSeen[idxName] = true
		}

		if opType == "read" {
			p50Read = append(p50Read, opts.BarData{Value: p50})
			p95Read = append(p95Read, opts.BarData{Value: p95})
		} else {
			p50Write = append(p50Write, opts.BarData{Value: p50})
			p95Write = append(p95Write, opts.BarData{Value: p95})
		}
	}

	sumBar := charts.NewBar()
	sumBar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: title + " — Summary"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "ns"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true), Top: "8%"}),
		charts.WithInitializationOpts(opts.Initialization{Width: chartWidth, Height: chartHeight}),
	)
	sumBar.SetXAxis(sumLabels).
		AddSeries("Read P50", p50Read).
		AddSeries("Read P95", p95Read).
		AddSeries("Write P50", p50Write).
		AddSeries("Write P95", p95Write)

	page := components.NewPage()
	page.SetLayout(components.PageFlexLayout)
	page.AddCharts(sumBar, lineChart)
	return renderPage(page, filepath.Join(outDir, outHtml), "["+title[:2]+"]")
}

func PlotT4(outDir string) error {
	return PlotMixed(outDir, "t4_read_heavy.csv", "T4 — Read-Heavy (95/5) Response Time", "t4.html")
}

func PlotT5(outDir string) error {
	return PlotMixed(outDir, "t5_write_heavy.csv", "T5 — Write-Heavy (5/95) Response Time", "t5.html")
}
