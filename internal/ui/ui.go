// Package ui provides clack-style terminal output primitives.
// Aesthetic: vertical │ spine, ◆ step markers, consistent symbols, indigo brand.
package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// ─── environment ─────────────────────────────────────────────────────────────

var (
	IsTTY     = isatty.IsTerminal(os.Stdout.Fd())
	NoColor   = os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
	ForceJSON = !IsTTY
)

// ─── palette ─────────────────────────────────────────────────────────────────

var (
	Brand   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6366f1"))
	Spine   = lipgloss.NewStyle().Foreground(lipgloss.Color("#334155"))
	Step    = lipgloss.NewStyle().Foreground(lipgloss.Color("#818cf8"))
	Key     = lipgloss.NewStyle().Foreground(lipgloss.Color("#38bdf8"))
	Val     = lipgloss.NewStyle().Foreground(lipgloss.Color("#e2e8f0"))
	Muted   = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748b"))
	Success = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	Err     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
	Warn    = lipgloss.NewStyle().Foreground(lipgloss.Color("#f59e0b"))
	Accent  = lipgloss.NewStyle().Foreground(lipgloss.Color("#818cf8"))
	Accept  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#22c55e"))
	Reject  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ef4444"))
)

func bar() string   { return Spine.Render("│") }
func tick() string  { return Success.Render("✔") }
func cross() string { return Err.Render("✖") }
func dot() string   { return Step.Render("◆") }
func odot() string  { return Step.Render("◇") }

// ─── primitives ───────────────────────────────────────────────────────────────

func Intro(title, subtitle string) {
	fmt.Println()
	if subtitle != "" {
		fmt.Printf("  %s  %s\n", Brand.Render(title), Muted.Render(subtitle))
	} else {
		fmt.Printf("  %s\n", Brand.Render(title))
	}
	fmt.Printf("  %s\n", bar())
}

func Outro(msg string) {
	fmt.Printf("  %s\n", bar())
	fmt.Printf("  %s  %s\n", Success.Render("◆"), msg)
	fmt.Println()
}

func StepDone(msg string) {
	fmt.Printf("  %s\n", bar())
	fmt.Printf("  %s  %s\n", tick(), msg)
}

func StepActive(msg string) {
	fmt.Printf("  %s\n", bar())
	fmt.Printf("  %s  %s\n", dot(), msg)
}

func StepValue(label, value string, style lipgloss.Style) {
	fmt.Printf("  %s  %s  %s\n",
		bar(),
		Muted.Render(fmt.Sprintf("%-16s", label)),
		style.Render(value),
	)
}

func LogInfo(msg string) {
	fmt.Printf("  %s  %s\n", bar(), Muted.Render(msg))
}

func LogSuccess(msg string) {
	fmt.Printf("  %s  %s\n", bar(), Success.Render(msg))
}

func LogError(msg string) {
	fmt.Fprintf(os.Stderr, "  %s  %s\n", bar(), Err.Render(msg))
}

func Note(title string, lines []string) {
	fmt.Printf("  %s\n", bar())
	fmt.Printf("  %s  %s\n", dot(), Val.Render(title))
	for _, l := range lines {
		fmt.Printf("  %s  %s\n", Spine.Render("│ "), Muted.Render(l))
	}
}

func NextSteps(steps []string) {
	fmt.Printf("  %s\n", bar())
	fmt.Printf("  %s  %s\n", odot(), Muted.Render("Next steps"))
	for _, s := range steps {
		fmt.Printf("  %s  %s\n", Spine.Render("│"), Muted.Render("○  "+s))
	}
}

func ProgressStep(n, total int, msg string, done bool) {
	prefix := Muted.Render(fmt.Sprintf("[%d/%d]", n, total))
	sym := dot()
	if done {
		sym = tick()
	}
	fmt.Printf("  %s\n", bar())
	fmt.Printf("  %s  %s  %s\n", sym, prefix, msg)
}

func Bar() {
	fmt.Printf("  %s\n", bar())
}

// ─── errors ──────────────────────────────────────────────────────────────────

// AbortError is a sentinel error type used by Abort. Callers can type-assert
// to detect a user-facing abort vs an unexpected error.
type AbortError struct {
	Msg  string
	Hint string
	Docs string
}

func (e *AbortError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s (hint: %s)", e.Msg, e.Hint)
	}
	return e.Msg
}

// PrintAbort renders a human-readable error to stderr without exiting.
func PrintAbort(msg, hint, docs string) {
	fmt.Fprintf(os.Stderr, "  %s\n", bar())
	fmt.Fprintf(os.Stderr, "  %s  %s\n", cross(), Err.Render(msg))
	if hint != "" {
		fmt.Fprintf(os.Stderr, "  %s\n", Spine.Render("│"))
		fmt.Fprintf(os.Stderr, "  %s  %s  %s\n", Spine.Render("│"), Warn.Render("→"), Muted.Render(hint))
	}
	if docs != "" {
		fmt.Fprintf(os.Stderr, "  %s  %s %s\n", Spine.Render("│"), Muted.Render("docs:"), Accent.Render(docs))
	}
	fmt.Fprintln(os.Stderr)
}

// Abort prints a styled error to stderr and panics with AbortError.
// The panic is recovered in cmd.Execute() to produce a clean exit code.
func Abort(msg, hint, docs string) {
	PrintAbort(msg, hint, docs)
	panic(&AbortError{Msg: msg, Hint: hint, Docs: docs})
}

// ─── JSON output ──────────────────────────────────────────────────────────────

func PrintJSON(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

// ResolveOutput returns "json" if forced (non-TTY or explicit flag), else "plain".
// Returns an error if the flag value is invalid.
func ResolveOutput(flag string) (string, error) {
	if flag != "" {
		switch flag {
		case "plain", "json":
			return flag, nil
		default:
			return "", fmt.Errorf("invalid output format %q (must be \"plain\" or \"json\")", flag)
		}
	}
	if ForceJSON {
		return "json", nil
	}
	return "plain", nil
}

// DecisionBadge returns a styled decision string.
func DecisionBadge(d string) string {
	switch d {
	case "ACCEPT":
		return Accept.Render("● " + d)
	case "REJECT":
		return Reject.Render("● " + d)
	default:
		return Warn.Render("◌ " + d)
	}
}

// ─── spinner ─────────────────────────────────────────────────────────────────

type Spinner struct {
	mu      sync.Mutex
	frames  []string
	current int
	msg     string
	done    chan struct{}
	stopped chan struct{}
}

func NewSpinner(msg string) *Spinner {
	return &Spinner{
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		msg:     msg,
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

func (s *Spinner) Start() {
	if !IsTTY {
		fmt.Fprintf(os.Stderr, "  %s\n", Muted.Render(s.msg))
		close(s.stopped)
		return
	}
	go func() {
		defer close(s.stopped)
		for {
			select {
			case <-s.done:
				fmt.Printf("\r\033[2K")
				return
			default:
				s.mu.Lock()
				frame := s.frames[s.current%len(s.frames)]
				msg := s.msg
				s.mu.Unlock()
				fmt.Printf("\r  %s  %s",
					Step.Render(frame),
					Muted.Render(msg),
				)
				s.current++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

func (s *Spinner) Update(msg string) {
	s.mu.Lock()
	s.msg = msg
	s.mu.Unlock()
}

func (s *Spinner) Stop() {
	if !IsTTY {
		return
	}
	close(s.done)
	<-s.stopped
}

// ─── progress bar ─────────────────────────────────────────────────────────────

type ProgressBar struct {
	total   int
	current int
	width   int
	title   string
	start   time.Time
}

func NewProgressBar(total int, title string) *ProgressBar {
	return &ProgressBar{total: total, width: 24, title: title, start: time.Now()}
}

func (p *ProgressBar) Advance() {
	p.current++
	if IsTTY {
		p.render()
	}
}

func (p *ProgressBar) render() {
	pct := float64(p.current) / float64(p.total)
	filled := int(pct * float64(p.width))
	empty := p.width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	elapsed := time.Since(p.start).Round(time.Second)

	fmt.Printf("\r  %s %s %s %d/%d %s   ",
		Step.Render("·"),
		Muted.Render(p.title),
		Accent.Render(bar),
		p.current, p.total,
		Muted.Render(elapsed.String()),
	)
	if p.current >= p.total {
		fmt.Println()
	}
}

func (p *ProgressBar) Done() {
	if IsTTY && p.current < p.total {
		p.current = p.total
		p.render()
	}
}

// EventLine prints a single forwarded event line.
func EventLine(ts, component, eventName string, ok bool, detail string) {
	status := Success.Render("→")
	code := Success.Render("[200]")
	if !ok {
		status = Err.Render("✖")
		code = Err.Render("[ERR]")
	}
	fmt.Printf("  %s  %s  %s  %s%-35s  %s  %s\n",
		bar(),
		Muted.Render(ts),
		status,
		Muted.Render(fmt.Sprintf("%-8s", component)),
		Key.Render(eventName),
		code,
		Muted.Render(detail),
	)
}

// SuggestTypo returns the closest match by edit distance, or "".
func SuggestTypo(given string, choices []string) string {
	best := ""
	bestDist := 999
	for _, c := range choices {
		d := editDistance(strings.ToLower(given), strings.ToLower(c))
		thresh := max(2, len(given)/3)
		if d < bestDist && d <= thresh {
			bestDist = d
			best = c
		}
	}
	return best
}

func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	dp := make([]int, len(rb)+1)
	for i := range dp {
		dp[i] = i
	}
	for _, ca := range ra {
		ndp := make([]int, len(rb)+1)
		ndp[0] = dp[0] + 1
		for j, cb := range rb {
			ins := ndp[j] + 1
			del := dp[j+1] + 1
			sub := dp[j]
			if ca != cb {
				sub++
			}
			ndp[j+1] = minOf(ins, del, sub)
		}
		dp = ndp
	}
	return dp[len(rb)]
}

func minOf(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
