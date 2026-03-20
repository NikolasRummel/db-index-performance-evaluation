package bench

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
)

var indexColorHex = []string{
	"#268bd2", // blue  — btree
	"#859900", // green — bptree
	"#dc322f", // red   — lsm
}

func PlotAll(outDir string) error {
	allPlots := []struct {
		file  string
		label string
		fn    func(string) error
	}{
		{"t1_point_query.csv", "T1", PlotT1},
		{"t2_range_query.csv", "T2", PlotT2},
		{"t3_write_throughput.csv", "T3", PlotT3},
		{"t4_read_heavy.csv", "T4", PlotT4},
		{"t5_write_heavy.csv", "T5", PlotT5},
	}

	for _, p := range allPlots {
		path := filepath.Join(outDir, p.file)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("[Plotter] Skipping %s: %s not found\n", p.label, p.file)
			continue
		}

		if err := p.fn(outDir); err != nil {
			return fmt.Errorf("failed to plot %s: %w", p.label, err)
		}
	}
	return nil
}

func PlotT1(outDir string) error {
	f, err := os.Open(fmt.Sprintf("%s/t1_point_query.csv", outDir))
	if err != nil {
		return err
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return err
	}

	var labels []string
	var boxItems []opts.BoxPlotData
	var barItems []opts.BarData

	for i, rec := range records[1:] {
		minNs, _ := strconv.ParseFloat(rec[3], 64)
		q1Ns, _ := strconv.ParseFloat(rec[4], 64)
		p50Ns, _ := strconv.ParseFloat(rec[5], 64)
		q3Ns, _ := strconv.ParseFloat(rec[6], 64)
		p99Ns, _ := strconv.ParseFloat(rec[10], 64)
		tput, _ := strconv.ParseFloat(rec[11], 64)

		labels = append(labels, rec[0])
		boxItems = append(boxItems, opts.BoxPlotData{
			Value: []interface{}{minNs, q1Ns, p50Ns, q3Ns, p99Ns},
			ItemStyle: &opts.ItemStyle{
				Color:       indexColorHex[i%len(indexColorHex)],
				BorderColor: indexColorHex[i%len(indexColorHex)],
			},
		})
		barItems = append(barItems, opts.BarData{
			Value: tput,
			ItemStyle: &opts.ItemStyle{
				Color: indexColorHex[i%len(indexColorHex)],
			},
		})
	}

	box := charts.NewBoxPlot()
	box.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "T1 — Point Query Latency",
			Subtitle: "Box: Q1/median/Q3 — Whiskers: min/p99",
		}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Latency (ns)", Type: "log"}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true)}),
		charts.WithInitializationOpts(opts.Initialization{Width: "900px", Height: "500px"}),
	)
	box.SetXAxis(labels).AddSeries("Latency", boxItems)

	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: "T1 — Point Query Throughput"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Ops/sec"}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true)}),
		charts.WithInitializationOpts(opts.Initialization{Width: "900px", Height: "500px"}),
	)
	bar.SetXAxis(labels).AddSeries("Throughput", barItems)

	page := components.NewPage()
	page.AddCharts(box, bar)

	return renderPage(page, fmt.Sprintf("%s/t1.html", outDir), "[T1]")
}

func PlotT2(outDir string) error {
	f, err := os.Open(fmt.Sprintf("%s/t2_range_query.csv", outDir))
	if err != nil {
		return err
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return err
	}

	type point struct {
		rangeSize int
		totalMs   int64
	}

	byIndex := make(map[string][]point)
	var indexOrder []string
	seen := make(map[string]bool)

	for _, rec := range records[1:] {
		idxName := rec[0]
		rangeSize, _ := strconv.Atoi(rec[1])
		totalMs, _ := strconv.ParseInt(rec[3], 10, 64)

		if !seen[idxName] {
			indexOrder = append(indexOrder, idxName)
			seen[idxName] = true
		}
		byIndex[idxName] = append(byIndex[idxName], point{rangeSize, totalMs})
	}

	var xLabels []string
	if len(indexOrder) > 0 {
		for _, p := range byIndex[indexOrder[0]] {
			xLabels = append(xLabels, strconv.Itoa(p.rangeSize))
		}
	}

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "T2 — Range Query Latency",
			Subtitle: "Total time to scan all keys in range",
		}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Total time (ms)"}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Range size (keys)"}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true)}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true)}),
		charts.WithInitializationOpts(opts.Initialization{Width: "900px", Height: "500px"}),
	)
	line.SetXAxis(xLabels)

	for i, idxName := range indexOrder {
		var lineItems []opts.LineData
		for _, p := range byIndex[idxName] {
			lineItems = append(lineItems, opts.LineData{Value: p.totalMs})
		}
		line.AddSeries(idxName, lineItems,
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: indexColorHex[i%len(indexColorHex)],
				Width: 2,
			}),
			charts.WithItemStyleOpts(opts.ItemStyle{
				Color: indexColorHex[i%len(indexColorHex)],
			}),
		)
	}

	page := components.NewPage()
	page.AddCharts(line)

	return renderPage(page, fmt.Sprintf("%s/t2.html", outDir), "[T2]")
}

func PlotT3(outDir string) error {
	f, err := os.Open(fmt.Sprintf("%s/t3_write_throughput.csv", outDir))
	if err != nil {
		return err
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return err
	}

	type point struct {
		totalOps  int
		opsPerSec float64
	}

	byIndex := make(map[string][]point)
	var indexOrder []string
	seen := make(map[string]bool)

	for _, rec := range records[1:] {
		idxName := rec[0]
		totalOps, _ := strconv.Atoi(rec[1])
		opsPerSec, _ := strconv.ParseFloat(rec[2], 64)

		if !seen[idxName] {
			indexOrder = append(indexOrder, idxName)
			seen[idxName] = true
		}
		byIndex[idxName] = append(byIndex[idxName], point{totalOps, opsPerSec})
	}

	var xLabels []string
	if len(indexOrder) > 0 {
		for _, p := range byIndex[indexOrder[0]] {
			xLabels = append(xLabels, strconv.Itoa(p.totalOps))
		}
	}

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "T3 — Write Throughput vs. Data Growth",
			Subtitle: "Ops/sec measured as dataset grows",
		}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Throughput (Ops/sec)"}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Total Operations (Growth)"}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true)}),
		charts.WithInitializationOpts(opts.Initialization{Width: "900px", Height: "500px"}),
	)
	line.SetXAxis(xLabels)

	for i, idxName := range indexOrder {
		var items []opts.LineData
		for _, p := range byIndex[idxName] {
			items = append(items, opts.LineData{Value: p.opsPerSec})
		}
		line.AddSeries(idxName, items,
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: indexColorHex[i%len(indexColorHex)],
				Width: 2,
			}),
		)
	}

	page := components.NewPage()
	page.AddCharts(line)
	return renderPage(page, fmt.Sprintf("%s/t3.html", outDir), "[T3]")
}

func PlotT4(outDir string) error {
	return plotMixedWorkload(
		outDir,
		"t4_read_heavy.csv",
		"T4 — Read-Heavy Workload (95% Read / 5% Write)",
		"t4.html",
	)
}

func PlotT5(outDir string) error {
	return plotMixedWorkload(
		outDir,
		"t5_write_heavy.csv",
		"T5 — Write-Heavy Workload (5% Read / 95% Write)",
		"t5.html",
	)
}
func plotMixedWorkload(outDir, fileName, title, outHtml string) error {
	f, err := os.Open(filepath.Join(outDir, fileName))
	if err != nil {
		return err
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return err
	}

	type point struct {
		opCount int
		latNs   int64
	}

	byIndex := make(map[string][]point)
	var indexOrder []string
	seen := make(map[string]bool)

	for _, rec := range records[1:] {
		idxName := rec[0]
		opCount, _ := strconv.Atoi(rec[1])
		latNs, _ := strconv.ParseInt(rec[2], 10, 64)

		if !seen[idxName] {
			indexOrder = append(indexOrder, idxName)
			seen[idxName] = true
		}
		byIndex[idxName] = append(byIndex[idxName], point{opCount, latNs})
	}

	var xLabels []string
	if len(indexOrder) > 0 {
		for _, p := range byIndex[indexOrder[0]] {
			xLabels = append(xLabels, strconv.Itoa(p.opCount))
		}
	}

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    title,
			Subtitle: "Latency (ns) over operation sequence",
		}),
		// Use log axis if B-Tree and LSM results are orders of magnitude apart
		charts.WithYAxisOpts(opts.YAxis{Name: "Latency (ns)", Type: "value"}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Operation Count"}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true)}),
		charts.WithDataZoomOpts(opts.DataZoom{Type: "slider", Start: 0, End: 100}),
		charts.WithInitializationOpts(opts.Initialization{Width: "1000px", Height: "600px"}),
	)
	line.SetXAxis(xLabels)

	for i, idxName := range indexOrder {
		var items []opts.LineData
		for _, p := range byIndex[idxName] {
			items = append(items, opts.LineData{Value: p.latNs})
		}
		line.AddSeries(idxName, items,
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: indexColorHex[i%len(indexColorHex)],
				Width: 2,
			}),
		)
	}

	page := components.NewPage()
	page.AddCharts(line)
	return renderPage(page, filepath.Join(outDir, outHtml), "["+title[:2]+"]")
}

// ---
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
