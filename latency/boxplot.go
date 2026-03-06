package latency

import (
	"fmt"
	"strings"
)

const plotWidth = 40

// Boxplot renders an ASCII box plot from Stats.
// Returns a multi-line string with the plot and a five-number summary.
func Boxplot(s *Stats) string {
	if s == nil || s.Count == 0 {
		return "No latency data yet"
	}

	if s.Count < 4 {
		return fmt.Sprintf("min %s  median %s  max %s  (%d samples)",
			fmtMs(s.Min), fmtMs(s.Median), fmtMs(s.Max), s.Count)
	}

	// All values identical: degenerate case.
	if s.Min == s.Max {
		return fmt.Sprintf("all samples: %s  (%d samples)", fmtMs(s.Min), s.Count)
	}

	span := s.Max - s.Min
	pos := func(v float64) int {
		p := int((v - s.Min) / span * float64(plotWidth-1))
		if p < 0 {
			p = 0
		}
		if p >= plotWidth {
			p = plotWidth - 1
		}
		return p
	}

	q1Pos := pos(s.Q1)
	medPos := pos(s.Median)
	q3Pos := pos(s.Q3)

	// Ensure minimum widths for visibility.
	if medPos <= q1Pos {
		medPos = q1Pos + 1
	}
	if q3Pos <= medPos {
		q3Pos = medPos + 1
	}
	if q3Pos >= plotWidth {
		q3Pos = plotWidth - 1
		if medPos >= q3Pos {
			medPos = q3Pos - 1
		}
		if q1Pos >= medPos {
			q1Pos = medPos - 1
		}
	}

	// Build the plot line.
	line := make([]rune, plotWidth)
	for i := range line {
		switch {
		case i == 0:
			line[i] = '├'
		case i == plotWidth-1:
			line[i] = '┤'
		case i < q1Pos:
			line[i] = '─'
		case i == q1Pos:
			line[i] = '┤'
		case i == medPos:
			line[i] = '│'
		case i == q3Pos:
			line[i] = '├'
		case i > q1Pos && i < q3Pos:
			line[i] = '█'
		case i > q3Pos:
			line[i] = '─'
		default:
			line[i] = '─'
		}
	}

	var b strings.Builder
	b.WriteString(string(line))
	b.WriteByte('\n')
	b.WriteString(fmt.Sprintf("min %s  Q1 %s  med %s  Q3 %s  max %s  (%d samples)",
		fmtMs(s.Min), fmtMs(s.Q1), fmtMs(s.Median), fmtMs(s.Q3), fmtMs(s.Max), s.Count))
	return b.String()
}

// fmtMs formats milliseconds as a short latency string.
func fmtMs(ms float64) string {
	if ms < 1 {
		return fmt.Sprintf("%dus", int(ms*1000))
	}
	if ms < 10 {
		return fmt.Sprintf("%.1fms", ms)
	}
	return fmt.Sprintf("%dms", int(ms))
}
