package datasource

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type TradeData struct {
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

type FeedEntry struct {
	TradeData
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
}

func PrintFeed(feed []FeedEntry, w io.Writer) {
	dic := make(map[string][]FeedEntry)

	for _, e := range feed {
		dic[e.Source] = append(dic[e.Source], e)
	}
	format := `
Source:			%s
+---------------------------------------------------------------------------+
|	Timestamp	|	Open	|	High	|	Low		|	Close	|	Volume	|
+---------------+-----------+-----------+-----------+-----------+-----------+
%s
	`
	lineFormat := `|	%s			|	%.2f	|	%.2f	|	%.2f	|	%.2f	|	%.2f	|
+---------------+-----------+-----------+-----------+-----------+-----------+

	`
	lines := make([]string, 0)
	for source, entries := range dic {
		innerLines := make([]string, 0)
		for _, entry := range entries {
			innerLines = append(innerLines, fmt.Sprintf(lineFormat, entry.Timestamp.Format("02/01/2006"), entry.Open, entry.High, entry.Low, entry.Close, entry.Volume))
		}
		lines = append(lines, fmt.Sprintf(format, source, strings.Join(innerLines, "")))
	}
	result := strings.Join(lines, "\n\n")

	raw := []byte(result)
	w.Write(raw)
}
