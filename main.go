package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"fyne.io/fyne/v2/dialog"
	"log"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	_ "github.com/lib/pq"
)

// Config structures remain unchanged
type SqlMonitorApp struct {
	AreaList          []*AppArea  `json:"areas"`
	RefreshTimeout    string      `json:"refreshTimeout"`
	DefaultConnection *Connection `json:"defaultConnection"`
}

type AppArea struct {
	Title          string     `json:"title"`
	TabList        []*AreaTab `json:"tabs"`
	RefreshTimeout string     `json:"refreshTimeout"`
}

type AreaTab struct {
	Title          string `json:"title"`
	Query          string `json:"query"`
	RefreshTimeout string `json:"refreshTimeout"`
}

type Connection struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type TabRuntime struct {
	Tab          *AreaTab
	DB           *sql.DB
	Table        *widget.Table
	Data         [][]string
	LastUpdated  binding.String
	ColumnWidths []float32
	SelectedCell *widget.TableCellID
}

func main() {
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	fyneApp := app.New()
	window := fyneApp.NewWindow("SQL Monitor")
	window.Resize(fyne.NewSize(1200, 800))

	tabs := container.NewAppTabs()
	var activeTabs []*TabRuntime // Track all tab runtimes

	db, err := connectDB(config.DefaultConnection)
	if err != nil {
		showCriticalError(fyneApp, window, "Error connecting to database")
		log.Fatal("Database connection error:", err)
	}
	defer db.Close()

	// Register hotkeys
	copyShortcut := &fyne.ShortcutCopy{
		Clipboard: fyneApp.Driver().AllWindows()[0].Clipboard(), // Get the clipboard
	}

	window.Canvas().AddShortcut(copyShortcut, func(shortcut fyne.Shortcut) {
		copySelectedCell(window, tabs, activeTabs)
	})

	for _, area := range config.AreaList {
		areaTabs := container.NewAppTabs()

		for _, tab := range area.TabList {
			refreshTimeout := getRefreshTimeout(tab.RefreshTimeout, area.RefreshTimeout, config.RefreshTimeout)

			tabRuntime := &TabRuntime{
				Tab:          tab,
				DB:           db,
				Data:         [][]string{},
				LastUpdated:  binding.NewString(),
				ColumnWidths: []float32{200, 150, 100},
			}
			activeTabs = append(activeTabs, tabRuntime)

			// Create table
			tabRuntime.Table = widget.NewTable(
				func() (int, int) {
					if len(tabRuntime.Data) == 0 {
						return 0, 0
					}
					return len(tabRuntime.Data), len(tabRuntime.Data[0])
				},
				func() fyne.CanvasObject {
					label := widget.NewLabel("")
					label.Wrapping = fyne.TextTruncate
					return container.NewPadded(label)
				},
				func(id widget.TableCellID, obj fyne.CanvasObject) {
					cellContainer := obj.(*fyne.Container)
					label := cellContainer.Objects[0].(*widget.Label)

					if id.Row < len(tabRuntime.Data) && id.Col < len(tabRuntime.Data[id.Row]) {
						text := fmt.Sprintf(" %s ", tabRuntime.Data[id.Row][id.Col])
						label.SetText(text)

						if id.Row == 0 {
							label.TextStyle = fyne.TextStyle{Bold: true}
						}
					}
				},
			)
			// Set column widths
			for i := 0; i < len(tabRuntime.ColumnWidths); i++ {
				tabRuntime.Table.SetColumnWidth(i, tabRuntime.ColumnWidths[i])
			}

			// Cell selection handlers
			tabRuntime.Table.OnSelected = func(id widget.TableCellID) {
				tabRuntime.SelectedCell = &id
			}

			tabRuntime.Table.OnUnselected = func(id widget.TableCellID) {
				tabRuntime.SelectedCell = nil
			}

			// Header with controls
			header := container.NewHBox(
				widget.NewLabelWithStyle(tab.Title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				container.NewHBox(
					widget.NewLabelWithData(tabRuntime.LastUpdated),
					widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
						refreshTabData(tabRuntime)
					}),
					widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
						if tabRuntime.SelectedCell != nil {
							row, col := tabRuntime.SelectedCell.Row, tabRuntime.SelectedCell.Col
							if row >= 0 && row < len(tabRuntime.Data) && col >= 0 && col < len(tabRuntime.Data[row]) {
								window.Clipboard().SetContent(tabRuntime.Data[row][col])
							}
						}
					}),
				),
			)

			// Scrollable table area
			tableContainer := container.NewHScroll(
				container.NewVScroll(tabRuntime.Table),
			)
			tableContainer.SetMinSize(fyne.NewSize(1100, 600))

			content := container.NewBorder(
				header,
				nil,
				nil,
				nil,
				tableContainer,
			)

			areaTabs.Append(container.NewTabItem(tab.Title, content))
			go startTabRefresher(tabRuntime, refreshTimeout)
		}

		tabs.Append(container.NewTabItem(area.Title, areaTabs))
	}

	window.SetContent(tabs)
	window.Show()
	fyneApp.Run()
}

func showCriticalError(app fyne.App, parent fyne.Window, message string) {
	dialog.ShowError(
		errors.New(message),
		parent,
	)

	parent.SetOnClosed(func() {
		app.Quit()
	})
}

func copySelectedCell(window fyne.Window, tabs *container.AppTabs, activeTabs []*TabRuntime) {
	if tabs.Selected() != nil {
		if areaTabs, ok := tabs.Selected().Content.(*container.AppTabs); ok {
			if areaTabs.Selected() != nil {
				for _, rt := range activeTabs {
					if rt.Tab.Title == areaTabs.Selected().Text {
						copyFromTab(rt, window)
						return
					}
				}
			}
		}
	}
}

func copyFromTab(tabRuntime *TabRuntime, window fyne.Window) {
	if tabRuntime.SelectedCell != nil {
		row, col := tabRuntime.SelectedCell.Row, tabRuntime.SelectedCell.Col
		if row >= 0 && row < len(tabRuntime.Data) && col >= 0 && col < len(tabRuntime.Data[row]) {
			window.Clipboard().SetContent(tabRuntime.Data[row][col])
		}
	}
}

func loadConfig(filename string) (*SqlMonitorApp, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config SqlMonitorApp
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if config.RefreshTimeout == "" {
		config.RefreshTimeout = "1m"
	}

	return &config, nil
}

func connectDB(conn *Connection) (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		conn.Host, conn.Port, conn.User, conn.Password, conn.Database)
	return sql.Open("postgres", psqlInfo)
}

func parseDuration(durationStr string, defaultDuration time.Duration) time.Duration {
	if durationStr == "" {
		return defaultDuration
	}
	dur, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Printf("Invalid duration '%s', using default %v: %v", durationStr, defaultDuration, err)
		return defaultDuration
	}
	return dur
}

func getRefreshTimeout(tabTimeout, areaTimeout, defaultTimeout string) time.Duration {
	defaultDur := parseDuration(defaultTimeout, time.Minute)

	if tabTimeout != "" {
		return parseDuration(tabTimeout, defaultDur)
	}
	if areaTimeout != "" {
		return parseDuration(areaTimeout, defaultDur)
	}
	return defaultDur
}

func startTabRefresher(tabRuntime *TabRuntime, interval time.Duration) {
	refreshTabData(tabRuntime)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		refreshTabData(tabRuntime)
	}
}

func refreshTabData(tabRuntime *TabRuntime) {
	data, err := executeQuery(tabRuntime.DB, tabRuntime.Tab.Query)
	if err != nil {
		log.Printf("Error executing query for tab %s: %v", tabRuntime.Tab.Title, err)
		fyne.CurrentApp().Driver().DoFromGoroutine(func() {
			tabRuntime.LastUpdated.Set("Error: " + err.Error())
		}, false)
		return
	}

	fyne.CurrentApp().Driver().DoFromGoroutine(func() {
		tabRuntime.Data = data

		if len(data) > 0 && len(data[0]) > 0 {
			// Reset column widths if column count changed
			if len(data[0]) != len(tabRuntime.ColumnWidths) {
				tabRuntime.ColumnWidths = make([]float32, len(data[0]))
			}

			// Calculate max width for each column including header
			for col := 0; col < len(data[0]); col++ {
				maxWidth := float32(0)

				// Measure header with bold style (headers are in first row)
				if len(data) > 0 {
					headerText := data[0][col]
					headerWidth := fyne.MeasureText(headerText, theme.TextSize(), fyne.TextStyle{Bold: true}).Width
					if headerWidth > maxWidth {
						maxWidth = headerWidth
					}
				}

				// Measure content in data rows
				for row := 1; row < len(data); row++ {
					if col < len(data[row]) {
						text := data[row][col]
						contentWidth := fyne.MeasureText(text, theme.TextSize(), fyne.TextStyle{}).Width
						if contentWidth > maxWidth {
							maxWidth = contentWidth
						}
					}
				}

				// Add padding (20px on each side) and set minimum width
				padding := float32(40)
				minWidth := float32(100) // Minimum column width
				calculatedWidth := maxWidth + padding

				if calculatedWidth < minWidth {
					calculatedWidth = minWidth
				}

				if col < len(tabRuntime.ColumnWidths) {
					tabRuntime.ColumnWidths[col] = calculatedWidth
					tabRuntime.Table.SetColumnWidth(col, calculatedWidth)
				}
			}
		}

		tabRuntime.Table.Refresh()
		tabRuntime.LastUpdated.Set(time.Now().Format("2006-01-02 15:04:05"))
	}, false)
}

func executeQuery(db *sql.DB, query string) ([][]string, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results [][]string
	results = append(results, columns) // Заголовки столбцов

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))

	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		var row []string
		for i := range columns {
			var val string
			switch v := values[i].(type) {
			case nil:
				val = "NULL"
			case []byte:
				val = string(v)
			default:
				val = fmt.Sprintf("%v", v)
			}
			row = append(row, val)
		}
		results = append(results, row)
	}

	return results, nil
}
