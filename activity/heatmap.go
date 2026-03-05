package activity

import (
	"strings"
	"time"
)

// block characters for 5 intensity levels (0 = no activity).
var blocks = [5]rune{' ', '░', '▒', '▓', '█'}

// dayOrder maps display rows to time.Weekday, starting from Monday.
var dayOrder = [7]time.Weekday{
	time.Monday, time.Tuesday, time.Wednesday, time.Thursday,
	time.Friday, time.Saturday, time.Sunday,
}

// dayLabels are the 3-letter abbreviations for each row.
var dayLabels = [7]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

// Heatmap renders an ASCII heatmap of the activity matrix.
// Returns a multi-line string. If m is nil or all zeros, returns a "no data" message.
func Heatmap(m *Matrix) string {
	if m == nil || isEmpty(m) {
		return "No activity data yet"
	}

	max := maxCount(m)

	var b strings.Builder
	// Header with hour labels every 3 hours.
	b.WriteString("     0  3  6  9  12 15 18 21\n")
	for i, wd := range dayOrder {
		b.WriteString(dayLabels[i])
		b.WriteString("  ")
		for hr := 0; hr < 24; hr++ {
			level := quantize(m[wd][hr], max)
			b.WriteRune(blocks[level])
		}
		if i < 6 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// isEmpty returns true if all cells are zero.
func isEmpty(m *Matrix) bool {
	for _, row := range m {
		for _, v := range row {
			if v > 0 {
				return false
			}
		}
	}
	return true
}

// maxCount returns the maximum value in the matrix.
func maxCount(m *Matrix) uint32 {
	var max uint32
	for _, row := range m {
		for _, v := range row {
			if v > max {
				max = v
			}
		}
	}
	return max
}

// quantize maps a count to an intensity level 0-4.
func quantize(val, max uint32) int {
	if val == 0 || max == 0 {
		return 0
	}
	ratio := float64(val) / float64(max)
	switch {
	case ratio < 0.25:
		return 1
	case ratio < 0.50:
		return 2
	case ratio < 0.75:
		return 3
	default:
		return 4
	}
}
