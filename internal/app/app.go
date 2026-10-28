// Package app is the glue (doc/architecture.md, The five parts): it turns keys
// and provider events into actions and events for the reducer, carries out
// the commands the reducer returns, and hands the state to the renderer. It
// holds no dashboard state of its own beyond the interface's (focus, the
// text being edited).
package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/mock"
	"github.com/Evengelius/agent-command-center/internal/render"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Config is the command line.
type Config struct {
	Mock     string // a mock file; runs alone
	Provider string // prefills the agent CLI answer
	Project  string // prefills the project answer
}

// ParseArgs reads the command line (architecture.md, How it starts). --mock
// runs alone, and --provider accepts only the two supported CLIs: the mock
// provider is reachable through --mock alone (reliability.md, Mock mode).
func ParseArgs(args []string) (Config, error) {
	var c Config
	fs := flag.NewFlagSet("agent-command-center", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&c.Mock, "mock", "", "replay a mock file of normalised events")
	fs.StringVar(&c.Provider, "provider", "", "prefill the agent CLI: claude-code or codex")
	fs.StringVar(&c.Project, "project", "", "prefill the project directory")
	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("%w\nusage: agent-command-center [--mock <file>] [--provider claude-code|codex] [--project <path>]", err)
	}
	if fs.NArg() > 0 {
		return Config{}, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	if c.Mock != "" && (c.Provider != "" || c.Project != "") {
		return Config{}, errors.New("--mock runs alone: it cannot be combined with --provider or --project")
	}
	if c.Provider != "" && c.Provider != "claude-code" && c.Provider != "codex" {
		return Config{}, fmt.Errorf("--provider %q: use claude-code or codex", c.Provider)
	}
	return c, nil
}

// Options are what the application runs with. Tests replace the clock, the
// timers and the clipboard.
type Options struct {
	Config
	// Providers are the agent CLI adapters, in the startup screen's order.
	// A supported CLI without an adapter is shown as not usable.
	Providers []provider.Provider
	Now       func() time.Time
	Clipboard input.Clipboard
	// Getenv reads the environment (TMUX, SSH_*); nil is os.Getenv.
	Getenv func(string) string
	// After delivers msg after d; nil uses a real timer. Tests return nil
	// to switch timers off.
	After func(d time.Duration, msg tea.Msg) tea.Cmd
	// MockSpeed is the replay pace: 1 as recorded, 0 all at once.
	MockSpeed float64
	// Cwd is the project field's default and the base of a relative path;
	// "" reads the current directory.
	Cwd string
	// Home expands ~; "" reads the user's home directory.
	Home string
	// Drafts keeps the draft on disk; nil keeps it in memory only.
	Drafts *input.Drafts
	// Projects keeps the startup screen's project list (plan 15); nil reads
	// and writes nothing, and the screen has no list.
	Projects *input.Projects
	// Highlight colours a file's lines for the file viewer (plan 17); nil is
	// render.Highlight.
	Highlight func(name string, lines []string, from, to int) render.Highlighted
	// Git runs the Worktrees view's git (plan 16); nil is the git on the PATH.
	Git input.Git
	// Sessions records this dashboard's sessions for the Worktrees view;
	// nil registers nothing, and the view says other sessions are N/A.
	Sessions *input.Sessions
	// DetectTimeout bounds each adapter's detection; 0 is DetectTimeout.
	DetectTimeout time.Duration
	// Keys are the editor's send and newline keys read from the configuration
	// file (plan 21, item 5.1); a zero list is its default, so a run that
	// gives none has today's keys. KeysProblem is why the file's keys were not
	// used, "" when there is none: the startup screen shows it, and a mock
	// run's editor status row until the first message is sent.
	Keys        input.Keys
	KeysProblem string
	// CodexProxy is the proxy Codex is started through, config.json's
	// codex.proxy (plan 22, amendment 19), "" when there is none; the startup
	// screen shows it on a grey line. CodexProxyProblem is why the codex section
	// was not used, "" when there is none: the startup screen shows it under
	// KeysProblem. The adapter is given the proxy by the caller
	// (codex.Provider.Proxy); the application only shows it.
	CodexProxy        string
	CodexProxyProblem string
	// PickImages opens the system's Open dialog for images and gives the
	// files chosen, none when cancelled (plan 31); nil has no dialog: a click
	// on "Upload from computer" says so.
	PickImages func() ([]string, error)
	// PickFolder opens the system's folder dialog at start and gives the
	// folder chosen, "" when cancelled (plan 52); nil has no dialog: the
	// folder button and Ctrl+O say so on the startup screen.
	PickFolder func(start string) (string, error)
	// NerdIcons is true where the Explorer can draw Cascadia Mono NF's icons,
	// input.NerdIcons in the real run (plan 55, design D3); false keeps the
	// panel's own glyphs.
	NerdIcons bool
	// BranchWatch is how often the project's HEAD file is checked for a
	// switch made elsewhere (plan 59, design D9); 0 watches nothing.
	BranchWatch time.Duration
	// Location is the time zone the file viewer's blame shows dates in (plan
	// 60, design D2); nil is the machine's.
	Location *time.Location
	// GraphWatch is how often the Git Graph reads the references it follows
	// again while it shows (plan 62, design D4b); 0 watches nothing.
	GraphWatch time.Duration
}

// Timeouts (events-and-state.md, timers).
const (
	DetectTimeout    = 10 * time.Second
	HandshakeTimeout = 60 * time.Second
	InterruptTimeout = 10 * time.Second
	CloseTimeout     = 5 * time.Second
	KillGrace        = 5 * time.Second // after the forced kill, leave anyway
	spinnerEvery     = 100 * time.Millisecond
	animEvery        = 50 * time.Millisecond // ASM-18, 20 frames a second while the thinking card animates (plan 22)
	clockEvery       = time.Second
	maxDrain         = 512 // events applied per screen update
)

// supported are the agent CLIs the startup screen lists.
var supported = []struct{ id, name string }{
	{"claude-code", "Claude Code"},
	{"codex", "OpenAI Codex"},
}

// updateHook runs first in every Update; nil outside TestPanic, which drives
// a session with it and injects a panic, and TestWin32InputMode.
var updateHook func(m *Model, msg tea.Msg) []tea.Cmd

// Model is the Bubble Tea model.
type Model struct {
	opt   Options
	s     *state.State
	ui    render.UI
	under scrollPos    // the scroll under the key overlay
	edit  input.Editor // zone B's input
	path  input.Editor // the startup screen's project field
	env   input.PathEnv

	detected map[string]provider.Detection
	sess     provider.Session
	cancel   context.CancelFunc // ends the session's context: the forced kill
	once     sync.Once
	pending  []tea.Cmd     // the launch's commands, for Init
	writes   chan writeJob // the session's writes, in order (writer.go)

	// The stored draft (draft.go): nil when drafts are off.
	draft      *draftStore
	draftReady bool   // read: saves may start
	draftTimer bool   // a save is due
	draftSaved string // the text last saved or read
	draftImgs  string // the images' paths last saved or read, joined (plan 31)
	draftGen   int
	unread     []string // the stored draft's images still being read back, as it lists them (plan 33)
	retrying   bool     // a failed save is due to be tried again (plan 33)

	// The project list (projects.go): nil when it is off.
	projects     *projectStore
	projectList  []input.Project
	projectsGen  int // the last read or change asked for
	projectsSeen int // the last one shown
	// The folder dialog is open (folder.go; plan 52).
	folderOpen bool
	// The Worktrees view (worktrees.go): nil when closed.
	wt    *worktrees
	wtGen int // the last git result asked for
	// The file viewer and the quick open (file.go): the generations of the
	// last read or colouring, search and listing asked for.
	fileGen, searchGen, finderGen int
	explorerGen                   int      // the Explorer's (explorer.go), its reads and git
	fileBytes                     int64    // the file shown, its size on disk
	finderAll                     []string // the project's files, as listed
	// The viewer's git blame (blame.go, plan 60): on for the session, and
	// the number of its latest read.
	blameOn  bool
	blameSeq int
	// The Git Graph (graph.go, plan 62): its scope, its read and its places.
	graph graphState
	// Source Control (source.go, plan 44): git's status reads, one at a time
	// (statusBusy; statusAgain asks one more), their generation and the last
	// one's time, the poll timed last; the snapshot read last, nil before one
	// or outside a repository; the open diff's entry, its sides' stamp, its
	// generation and the last comparison's time.
	statusGen, pollGen      int
	statusBusy, statusAgain bool
	statusTook, diffTook    time.Duration
	snap                    *input.GitSnapshot
	diffEntryOpen           input.GitEntry
	diffStamp               string
	diffGen                 int
	// The untracked folders' reads, numbered as issued, and each folder's
	// last applied: an older read never replaces a newer (plan 46, D3).
	folderSeq  int
	folderSeen map[string]int
	// The commit controls (commit.go, plan 45): the message field, the
	// operation running and its steps' generation, the question's offer.
	msg   input.Editor
	op    *gitOp
	opGen int
	opSeq int
	ask   *asked
	// This session's registration (plan 16, Sessions): nil when none.
	lock    *input.SessionLock
	locking bool
	// The branches popup (branches.go, plan 59): its reads' generation and
	// its search; the switch asked, nil when none; the fetch running, nil
	// when none, its stop asked, why a stop was refused, FetchLimit reached;
	// the HEAD watch's generation, file and time (design D9).
	branchGen                int
	branchFind               input.Editor
	sw                       *branchSwitch
	fetch                    *input.GitStop
	fetchStopping, fetchLate bool
	fetchHint                string
	headGen                  int
	headPath                 string
	headTime                 time.Time

	spinning, clocking bool
	framing            bool // an animMsg is pending (plan 22)

	pasting  bool // between a paste's start and end markers
	toastGen int  // the toast on screen; an older timer leaves it

	// The draft's attached images (attach.go, plan 31), and the Open dialog
	// open.
	attach  []input.Image
	picking bool

	// Pasted images (paste.go, plan 32): their PNG bytes and sums by key,
	// in memory and in the stored draft's folder (plan 33, design D8), and
	// the run's count for their names; the paste
	// keys: a Ctrl+V press seen, the terminal's pastes counted and the time
	// of the last, Ctrl+V's lone release waiting.
	pasted     map[string][]byte
	pastedSum  map[string][32]byte
	pasteN     int
	ctrlVDown  bool
	pasteCount int
	lastPaste  time.Time
	pasteWait  int
	waitPastes int

	// The preview (preview.go, plan 32): each attachment's thumbnail by key,
	// the last frame's thumbnails' signature and its generation, the
	// program's Send for the timers View sets, and, with no program (a
	// test), the drawings asked for.
	thumbs   map[string]thumbSixel
	thumbSig string
	thumbGen int
	drawn    []render.Thumb // where images were drawn last, to repaint once vacated
	send     func(tea.Msg)
	kicks    []int
	slide    slide // the effort slider's glide (plan 37)
	// pillLabels are the mode pill's titles as GDI drew them (plan 39).
	pillLabels map[string]*image.Alpha
	// liveLabel is a running operation's label as GDI drew it, the one that
	// shows: it changes every second (plan 45, design D15).
	liveLabel struct {
		key  string
		mask *image.Alpha
	}
	// The home screen's picture as last written, the glyphs it is drawn
	// with, and whether the drawing under way is the second (plan 51,
	// design D13).
	home       homePic
	glyphs     gdiGlyphs
	homeSecond bool
	resized    time.Time // the last size report: the picture waits for the size to settle (plan 51, design D13)
	// The banner's frames (plan 51, design D12): one is pending; the
	// terminal's window lacks the focus, which pauses them.
	bannering, blurred bool

	screen  *render.Screen // the last frame drawn: what the mouse points at
	repaint bool           // the palette changed: the next frame draws every row
	drag    drag
	click   clicks
	// motionDropped counts the motion reports motionFilter dropped before
	// Update and View (plan 43, design D4), for the tests.
	motionDropped int
	// clickFold is what the last click on a fold row of zone B changed, so a
	// double click there undoes it before opening the entry (plan 42).
	clickFold clickFold
}

// Messages.
type (
	actionMsg   struct{ a state.Action }
	detectedMsg struct{ found []provider.Detection }
	startedMsg  struct {
		sess   provider.Session
		cancel context.CancelFunc
		err    error
	}
	eventsMsg struct {
		sess   provider.Session
		evs    []event.Event
		closed bool
	}
	pasteMsg   struct{ text string }
	toastMsg   struct{ text string }
	spinMsg    struct{}
	animMsg    struct{} // a frame of zone B's thinking card, every animEvery (plan 22)
	clockMsg   struct{}
	leaveMsg   struct{}
	refreshMsg struct{ err error }
	bannerMsg  struct{} // a frame of the home screen's banner, every render.BannerTick (plan 51)
)

// New builds the model.
func New(opt Options) *Model {
	if opt.Now == nil {
		opt.Now = time.Now
	}
	if opt.Mock != "" {
		opt.Providers = nil // a mock run never reaches a CLI adapter
	}
	if opt.After == nil {
		opt.After = func(d time.Duration, msg tea.Msg) tea.Cmd {
			return tea.Tick(d, func(time.Time) tea.Msg { return msg })
		}
	}
	if opt.DetectTimeout == 0 {
		opt.DetectTimeout = DetectTimeout
	}
	if opt.Cwd == "" {
		opt.Cwd, _ = os.Getwd()
	}
	if opt.Home == "" {
		opt.Home, _ = os.UserHomeDir()
	}
	if opt.Getenv == nil {
		opt.Getenv = os.Getenv
	}
	if opt.Highlight == nil {
		opt.Highlight = render.Highlight
	}
	opt.Keys = opt.Keys.WithDefaults()
	m := &Model{opt: opt, s: state.New(), detected: map[string]provider.Detection{}}
	m.env = input.PathEnv{Windows: filepath.Separator == '\\', Home: opt.Home, Cwd: opt.Cwd}
	m.ui = render.UI{Width: render.MinWidth, Height: render.MinHeight, Now: opt.Now(), Focus: "B", Choice: opt.Provider, Wrap: render.NewWrapCache(), Memo: render.NewMemo(),
		KeysProblem: opt.KeysProblem, CodexProxy: opt.CodexProxy, CodexProxyProblem: opt.CodexProxyProblem, NerdIcons: opt.NerdIcons}
	m.ui.SendKeys, m.ui.NewlineKeys = opt.Keys.Display()
	m.ui.Version, m.ui.System = Version, systemName(runtime.GOOS) // the home screen's header (plan 51, design D5)
	m.path.SetText(m.startProject())
	// Launch now, not in Init's command: the first frame is drawn before any
	// message, and a mock run must show its badge from that frame on.
	m.pending = m.do(state.Launch{Mock: opt.Mock})
	if opt.Mock == "" {
		m.pending = append(m.pending, m.detect())
		if opt.Projects != nil {
			m.projects = &projectStore{p: *opt.Projects}
			m.pending = append(m.pending, m.projectsCmd(nil))
		}
	}
	m.syncUI()
	return m
}

// State is the current state, for tests.
func (m *Model) State() *state.State { return m.s }

// Init starts the application: detection, or the mock replay.
func (m *Model) Init() tea.Cmd {
	cmds := append(m.pending, m.timers()...)
	if win32Input {
		// win32-input-mode, then the device attributes request: the answer
		// says whether the terminal draws Sixel images (plan 32, design D4).
		cmds = append(cmds, tea.Raw(win32InputOn), tea.Raw(askAttributes))
	}
	m.pending = nil
	return tea.Batch(cmds...)
}

// View draws the frame.
func (m *Model) View() tea.View {
	prev := m.screen
	if m.repaint {
		prev, m.repaint = nil, false // its rows hold the old palette's colours
	}
	m.screen = render.Draw(m.s, m.ui)
	// A popup the frame did not show closes: the selectors when it had no
	// room for them (a draft grown under the model selector, a turn's status
	// in the pill's place), any under a view (plan 49, design D7).
	m.closeUnshown(m.screen)
	v := tea.NewView(m.screen.StringAfter(prev))
	m.previewFrame() // the thumbnails drawn again after a frame that changed their rows (plan 32)
	v.AltScreen = true
	if m.s.Screen == state.ScreenStartup && !m.ui.MouseOff && m.wt == nil {
		v.MouseMode = tea.MouseModeCellMotion // the home screen's clicks (plan 51, design D11); F12 as on the dashboard
	}
	// The home screen's banner pauses while the window lacks the focus (plan
	// 51, design D12).
	v.ReportFocus = m.s.Screen == state.ScreenStartup
	if m.s.Screen == state.ScreenDashboard && !m.ui.MouseOff && m.wt == nil {
		v.MouseMode = tea.MouseModeCellMotion // F12 gives the mouse back to the terminal
		if m.screen.Layers.Top.Hover() || m.sideHoverable() || (m.ui.Source.Open || m.ui.Graph.Open) && m.screen.Panel.W > 0 {
			// The menu's or a selector's hover follows the pointer (plans 30, 34,
			// 39), the sidebar's tiles' too while the frame shows them (plan
			// 43, design D4), Source Control's rows (plan 44, design D5) and
			// the Git Graph's, for their card (plan 62, design D13);
			// motionFilter drops the reports that change none.
			v.MouseMode = tea.MouseModeAllMotion
		}
	}
	v.WindowTitle = "Agent Command Center"
	return v
}

// Update handles one message.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.ui.Now = m.opt.Now()
	var cmds []tea.Cmd
	if updateHook != nil {
		cmds = updateHook(m, msg)
	}
	if attrs, ok := deviceAttributes(msg); ok {
		m.attributesAnswered(attrs)
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ui.Width, m.ui.Height = msg.Width, msg.Height
		// Bubble Tea's renderer erases the screen on every size report, the
		// same size too: the images are drawn again after the next frame
		// (plan 43's review; previewFrame), the home screen's picture whole
		// (plan 51, design D13), its banner's scene going on where it fits.
		m.thumbSig, m.home, m.resized = "", homePic{banner: m.home.banner}, m.opt.Now()
	case tea.FocusMsg:
		m.blurred = false // the banner's frames again (plan 51, design D12)
	case tea.BlurMsg:
		m.blurred = true
	case bannerMsg:
		cmds = append(cmds, m.bannerFrame()...)
	case tea.ColorProfileMsg:
		// The terminal's colours, at startup and on an RGB capability report
		// (plan 5, item 13).
		if render.UsePalette(lipgloss.Complete(msg.Profile)) {
			m.repaint = true
		}
	case refusedMsg:
		cmds = append(cmds, m.do(msg.a)...) // the draft given back, then its changed file read again
		cmds = append(cmds, inspectImages([]string{msg.path}, inputLimits(m.limits()), false))
	case actionMsg:
		validating := m.s.Startup.Phase == state.Validating
		cmds = append(cmds, m.do(msg.a)...)
		if _, ok := msg.a.(state.ValidationFailed); ok {
			m.focusMessage()
		}
		if ok, isOK := msg.a.(state.ValidationOK); isOK && validating && !m.s.Context.Mock {
			cmds = append(cmds, m.recordProject(ok.Project)) // the validated path (plan 15)
			cmds = append(cmds, m.watchHead()...)            // its branch kept current (plan 59, design D9)
		}
	case projectsMsg:
		cmds = append(cmds, m.projectsDone(msg)...)
	case chosenFolderMsg:
		m.folderPicked(msg)
	case wtListMsg:
		m.worktreesListed(msg)
	case wtOpMsg:
		cmds = append(cmds, m.worktreeDone(msg)...)
	case fileMsg:
		cmds = append(cmds, m.fileRead(msg)...)
	case hlMsg:
		cmds = append(cmds, m.coloured(msg)...)
	case blameMsg:
		m.blameRead(msg)
	case graphRefsMsg:
		cmds = append(cmds, m.graphRefsRead(msg)...)
	case graphPageMsg:
		cmds = append(cmds, m.graphPaged(msg)...)
	case graphFilesMsg:
		m.graphFilesRead(msg)
	case graphMessageMsg:
		m.graphMessageRead(msg)
	case graphTickMsg:
		cmds = append(cmds, m.graphTick(msg)...)
	case searchMsg:
		cmds = append(cmds, m.searched(msg)...)
	case filesMsg:
		m.listed(msg)
	case dirMsg:
		cmds = append(cmds, m.dirRead(msg)...)
	case statusMsg:
		m.gitRead(msg)
	case snapshotMsg:
		cmds = append(cmds, m.snapshotRead(msg)...)
	case pollMsg:
		cmds = append(cmds, m.polled(msg)...)
	case folderMsg:
		m.folderRead(msg)
	case diffMsg:
		m.diffRead(msg)
	case opMsg:
		cmds = append(cmds, m.stepDone(msg)...)
	case opTickMsg:
		cmds = append(cmds, m.opTicked(msg)...)
	case wtDiffMsg:
		m.worktreeDiffed(msg)
	case branchesMsg:
		m.branchesRead(msg)
	case branchChangesMsg:
		cmds = append(cmds, m.branchChanges(msg)...)
	case branchSwitchedMsg:
		cmds = append(cmds, m.branchSwitched(msg)...)
	case fetchedMsg:
		cmds = append(cmds, m.fetched(msg)...)
	case fetchLimitMsg:
		cmds = append(cmds, m.fetchLimit(msg)...)
	case headFileMsg:
		cmds = append(cmds, m.headFound(msg)...)
	case headTickMsg:
		cmds = append(cmds, m.headTick(msg)...)
	case headCheckMsg:
		cmds = append(cmds, m.headChecked(msg)...)
	case lockMsg:
		cmds = append(cmds, m.locked(msg)...)
	case detectedMsg:
		cmds = append(cmds, m.detectionDone(msg.found)...)
	case startedMsg:
		cmds = append(cmds, m.started(msg)...)
	case eventsMsg:
		if msg.sess != m.sess {
			break // a previous session's leftovers
		}
		changed := false
		for _, ev := range msg.evs {
			cmds = append(cmds, m.run(m.s.Apply(ev))...)
			if ev.Kind == event.SessionSpawned {
				cmds = append(cmds, m.opt.After(HandshakeTimeout, actionMsg{state.HandshakeTimeout{}}))
			}
			// A tool call's or a turn's end may have changed the project's
			// files: Source Control reads git's status again (plan 44, design
			// D3).
			changed = changed || ev.Kind == event.ToolCompleted || ev.Kind == event.TurnCompleted
		}
		if changed {
			cmds = append(cmds, m.sourceEvent()...)
		}
		if !msg.closed {
			cmds = append(cmds, listen(m.sess))
		}
	case tea.PasteStartMsg:
		m.pasting = true
	case tea.PasteEndMsg:
		m.pasting = false
	case tea.PasteMsg:
		m.closeModels() // a paste goes on to the editor, as a key the selector does not take (plan 34)
		m.closeModes()  // and the modes selector's (plan 39)
		cmds = append(cmds, m.terminalPaste(msg.Content)...)
	case pasteMsg:
		m.paste(msg.text)
	case tea.KeyPressMsg:
		cmds = append(cmds, m.key(msg)...)
	case tea.KeyReleaseMsg:
		cmds = append(cmds, m.keyReleased(msg)...)
	case clipMsg:
		cmds = append(cmds, m.clipped(msg)...)
	case pastedMsg:
		cmds = append(cmds, m.pastedImage(msg)...)
	case pasteWaitMsg:
		cmds = append(cmds, m.pasteWaited(msg)...)
	case pathsMsg:
		cmds = append(cmds, m.pastedPaths(msg)...)
	case thumbDrawMsg:
		cmds = append(cmds, m.drawThumbs(msg)...)
	case slideMsg:
		cmds = append(cmds, m.drawSlide(msg)...)
	case tea.MouseMsg:
		cmds = append(cmds, m.mouse(msg)...)
	case copiedMsg:
		cmds = append(cmds, m.toast(msg.toast)...)
		if msg.seq != "" {
			cmds = append(cmds, tea.Raw(msg.seq))
		}
	case toastMsg:
		cmds = append(cmds, m.toast(msg.text)...)
	case toastDoneMsg:
		if msg.gen == m.toastGen {
			m.ui.Toast = ""
		}
	case spinMsg:
		m.spinning = false
		m.ui.Spinner++
	case animMsg:
		// A frame of the thinking card: ui.Now, set above, is all it changes. It
		// never counts the 100 ms spinner (plan 22, criterion 3.1).
		m.framing = false
	case clockMsg:
		m.clocking = false
		cmds = append(cmds, m.do(state.Tick{At: m.ui.Now})...)
	case draftLoadedMsg:
		cmds = append(cmds, m.draftLoaded(msg)...)
	case draftSaveMsg:
		cmds = append(cmds, m.draftDue()...)
	case draftFailedMsg:
		cmds = append(cmds, m.draftFailed(msg)...)
	case draftRetryMsg:
		cmds = append(cmds, m.draftRetried()...)
	case pickedMsg:
		cmds = append(cmds, m.picked(msg)...)
	case inspectedMsg:
		cmds = append(cmds, m.inspected(msg)...)
	case leaveMsg:
		m.stopGraphRead() // no git process left behind (plan 62, design D4)
		return m, tea.Quit
	case refreshMsg:
		// Refresh reads telemetry after a turn; a failed read leaves the
		// values as they are, each with its own age (no zero, no guess).
	}
	cmds = append(cmds, m.lockSync()...)
	m.pruneImages()
	m.syncUI()
	m.markLive()
	m.markActivity()
	cmds = append(cmds, m.timers()...)
	cmds = append(cmds, m.draftChanged()...)
	return m, tea.Batch(cmds...)
}

// syncUI copies what the renderer needs from the editors.
func (m *Model) syncUI() {
	m.settleCard()
	if m.ui.Focus == "R" && !render.PanelShown(m.ui) {
		m.ui.Focus = "B" // the docked panel hidden: zone B has the focus back (plan 43, design D5)
	}
	switch {
	case m.ui.Diff.Open && m.ui.Focus == "B" && m.ui.Card != "":
		// An approval's card given the focus (F2, from any zone): zone B,
		// where the card is, takes the diff's place back (plan 44, design D12).
		m.closeDiff()
	case m.ui.Diff.Open && m.ui.Focus == "B":
		m.ui.Focus = "D" // the diff in zone B's place takes its focus (plan 44, design D8)
	case !m.ui.Diff.Open && m.ui.Focus == "D":
		m.ui.Focus = "B"
	case m.ui.Focus == "R" && render.PanelUnderDiff(m.s, m.ui):
		m.closeDiff() // the panel given the focus in the diff's place: the panel shows (design D8)
	}
	if m.ui.Focus == "B" && render.PanelShown(m.ui) && render.PanelCompact(m.s, m.ui) {
		// Plan 43's compact rule: where the panel takes zone B's place, the
		// compact layout from 100 columns and the narrow one under 100
		// (design D1), zone B taking the focus hides the panel, whatever
		// gave it the focus (Esc on a top-level row, F6 or Tab, F2, Esc from
		// zone A or C, a click, a resize that leaves no room to dock it), so
		// typed text never goes into a draft that is not drawn; Source
		// Control's the same (plan 44, design D4).
		m.closePanels()
	}
	if m.ui.SideHover != 0 && !m.sideHoverable() {
		m.sideHover(0) // no sidebar under the pointer, or a window over it (plan 43, design D4)
	}
	if m.ui.Focus != "A" && m.ui.Focus != "C" {
		m.ui.Zoom = false // only a panel zooms; reaching B by F6, Tab or a click restores
	}
	m.ui.Draft = m.edit.Text()
	m.ui.Editor = m.editorView()
	m.syncCommit()   // the commit controls (plan 45)
	m.syncBranches() // the branches popup's git (plan 59)
	m.ui.Path = m.path.Text()
	r := input.ResolvePath(m.path.Text(), m.env)
	m.ui.Resolved, m.ui.PathProblem, m.ui.PathWarning = r.Path, r.Problem, r.Warning
	m.ui.Branch = ""
	if c := m.s.Context; c.Project != "" && (c.Project == r.Path || m.s.Screen == state.ScreenDashboard) {
		// Read on validation, for the path validated; on the dashboard the
		// session's, kept current by the HEAD watch (plan 59, design D9).
		m.ui.Branch = c.Branch
	}
	if m.s.Screen == state.ScreenStartup {
		m.ui.Projects = m.projectRows()
		m.ui.ProjectSel = max(min(m.ui.ProjectSel, len(m.projectList)-1), 0)
		if m.ui.Field == 3 && len(m.projectList) == 0 {
			m.ui.Field = 1
		}
	}
	m.ui.Worktrees = m.worktreesView()
}

// markLive records in ui.Live the thinking phases of the last exchange that
// are live now, by their ID (plan 22, design 7): only a phase an update
// left running was drawn live, so only it plays its exit; one that began and
// ended inside a single update (a replay, a resume) is never marked and shows
// as its summary row at once. A mark is never removed. The live phases are in
// the current exchange, so the scan from the end stops at the owner's prompt.
func (m *Model) markLive() {
	conv := m.s.Conversation
	for i := len(conv) - 1; i >= 0; i-- {
		e := &conv[i]
		if e.Role == "user" {
			return
		}
		if e.Role == "thinking" && e.Streaming && e.Thought != nil && !m.ui.Live[uint64(e.Thought.ID)] {
			if m.ui.Live == nil {
				m.ui.Live = map[uint64]bool{}
			}
			m.ui.Live[uint64(e.Thought.ID)] = true // by its ID, unique across a resume
		}
	}
}

// markActivity notes in ui.ActivitySince when the activity line's condition
// began, by the function the drawing uses (render.ActivityOn), and clears it
// when the condition does not hold: the line is drawn once the condition has
// held 200 ms, so the short gaps between a response and the next card, or
// before a turn's end, never show it. It runs after markLive, whose marks the
// condition reads, and before the timers, which ask for the line's frames. No
// timer of its own: the 100 ms spinner, which runs while a turn is under way,
// draws the line when it is due (plan 22, correction C6).
func (m *Model) markActivity() {
	switch {
	case !render.ActivityOn(m.s, m.ui):
		m.ui.ActivitySince = time.Time{}
	case m.ui.ActivitySince.IsZero():
		m.ui.ActivitySince = m.ui.Now
	}
}

// animating reports whether a spinner is on screen.
func (m *Model) animating() bool {
	switch m.s.Startup.Phase {
	case "", state.Detecting, state.Validating:
		return m.s.Screen == state.ScreenStartup
	}
	switch m.s.Session.Phase {
	case state.Spawning, state.Handshaking, state.Closing:
		return true
	}
	switch m.s.Turn.Phase {
	case state.Submitted, state.Running, state.Interrupting:
		return true
	}
	return false
}

// timers schedules the spinner while something animates, a frame every 50 ms
// while zone B's thinking card does (ASM-18, 20 frames a second; plan 22),
// the clock while the dashboard shows elapsed times, and the home screen's
// banner while it is drawn (plan 51, design D12); an idle dashboard, or a
// startup screen without a banner, schedules no frame.
func (m *Model) timers() []tea.Cmd {
	var cmds []tea.Cmd
	if !m.spinning && m.animating() {
		if c := m.opt.After(spinnerEvery, spinMsg{}); c != nil {
			m.spinning = true
			cmds = append(cmds, c)
		}
	}
	if !m.framing && render.Animating(m.s, m.ui) {
		if c := m.opt.After(animEvery, animMsg{}); c != nil {
			m.framing = true
			cmds = append(cmds, c)
		}
	}
	if !m.clocking && m.s.Screen == state.ScreenDashboard && m.s.Session.Phase != state.Closed {
		if c := m.opt.After(clockEvery, clockMsg{}); c != nil {
			m.clocking = true
			cmds = append(cmds, c)
		}
	}
	if !m.bannering && m.home.banner != nil && !m.blurred {
		// The home screen's banner while it is drawn and the window has the
		// focus (plan 51, design D12).
		if c := m.opt.After(render.BannerTick, bannerMsg{}); c != nil {
			m.bannering = true
			cmds = append(cmds, c)
		}
	}
	return cmds
}

// do applies an action and carries out its commands.
func (m *Model) do(a state.Action) []tea.Cmd { return m.run(m.s.Do(a)) }

// run carries out the reducer's commands.
func (m *Model) run(cs []state.Command) []tea.Cmd {
	var cmds []tea.Cmd
	for _, c := range cs {
		cmds = append(cmds, m.command(c))
	}
	return cmds
}

func (m *Model) command(c state.Command) tea.Cmd {
	switch c.Kind {
	case state.CmdValidate:
		return m.validate()
	case state.CmdStartSession:
		if c.ID == "" {
			return tea.Batch(m.start(""), m.loadDraft())
		}
		return m.start(c.ID)
	case state.CmdWrite:
		m.edit.Clear()
		m.attach = nil
		sess, text, imgs, lim := m.sess, c.Text, c.Images, m.limits()
		pasted := m.pastedFor(imgs)
		return tea.Batch(m.draftWrite(), m.write(func() tea.Msg {
			if sess == nil {
				return actionMsg{state.WriteFailed{Message: "no session"}}
			}
			// The images read and checked again now, then sent with the text
			// (plan 31); a file gone or changed keeps the draft, a changed
			// one attached again as it is now (plan 33, design D7).
			msg, err := message(text, imgs, lim, pasted)
			if cf, ok := err.(changedFile); ok {
				return refusedMsg{state.WriteFailed{Message: err.Error()}, cf.path}
			}
			if err != nil {
				return actionMsg{state.WriteFailed{Message: err.Error()}}
			}
			if err := sess.Send(msg); err != nil {
				return actionMsg{state.WriteFailed{Message: err.Error()}}
			}
			return nil
		}, actionMsg{state.WriteFailed{Message: "too many writes waiting"}}))
	case state.CmdAnswer:
		sess, id, d, answers := m.sess, c.ID, c.Decision, c.Answers
		return m.write(func() tea.Msg {
			if sess == nil {
				return actionMsg{state.AnswerFailed{ID: id, Message: "no session"}}
			}
			if err := sendAnswer(sess, id, d, answers); err != nil {
				// A request the CLI withdrew first expires (plan 47, design D2).
				return actionMsg{state.AnswerFailed{ID: id, Message: err.Error(), Gone: errors.Is(err, provider.ErrNoPending)}}
			}
			return nil
		}, actionMsg{state.AnswerFailed{ID: id, Message: "too many writes waiting"}})
	case state.CmdSetApprovals:
		sess, mode := m.sess, c.Mode
		return m.write(func() tea.Msg {
			if sess == nil {
				return actionMsg{state.ApprovalFailed{Message: "no session"}}
			}
			if err := sess.SetApprovalMode(mode); err != nil {
				return actionMsg{state.ApprovalFailed{Message: err.Error()}}
			}
			return nil
		}, actionMsg{state.ApprovalFailed{Message: "too many writes waiting"}})
	case state.CmdModels:
		return m.modelsCmd(c.Change)
	case state.CmdInterrupt:
		sess := m.sess
		return tea.Batch(m.write(func() tea.Msg {
			if sess != nil {
				_ = sess.Interrupt() // no turn end within the timeout offers force stop
			}
			return nil
		}, nil), m.opt.After(InterruptTimeout, actionMsg{state.InterruptTimeout{}}))
	case state.CmdRefresh:
		// Through the write queue, in order with the owner's messages: the
		// Claude adapter's refresh writes "/usage", which must reach the CLI
		// before any prompt sent after the turn (plan 24, amendment 23).
		sess := m.sess
		return m.write(func() tea.Msg {
			if sess == nil {
				return nil
			}
			return refreshMsg{sess.Refresh()}
		}, nil)
	case state.CmdClose:
		return tea.Batch(m.close(), m.opt.After(CloseTimeout, actionMsg{state.CloseTimeout{}}))
	case state.CmdKill:
		if m.cancel != nil {
			m.cancel()
		}
		if m.s.Startup.Phase == state.Exited {
			return m.opt.After(KillGrace, leaveMsg{})
		}
	case state.CmdDraftAccepted:
		return m.draftAccepted()
	case state.CmdRestoreDraft:
		m.edit.SetText(joinDraft(c.Text, m.edit.Text())) // typed since the send: both kept
		m.restoreImages(c.Images)                        // and its images, before any attached since (plan 31)
	case state.CmdExit:
		return tea.Quit
	}
	return nil
}

// detect runs every adapter's detection in parallel, each bounded.
func (m *Model) detect() tea.Cmd {
	provs, limit := m.opt.Providers, m.opt.DetectTimeout
	return func() tea.Msg {
		found := make([]provider.Detection, len(provs))
		var wg sync.WaitGroup
		for i, p := range provs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), limit)
				defer cancel()
				done := make(chan provider.Detection, 1)
				go func() {
					// A panic in the detection is its CLI's problem, never the
					// program's end (plan 48, design D5; audit finding 6).
					defer func() {
						if v := recover(); v != nil {
							done <- provider.Detection{ID: p.ID(), Name: p.DisplayName(), Problem: "detection failed: " + provider.Recovered(v)}
						}
					}()
					done <- p.Detect(ctx)
				}()
				select {
				case d := <-done:
					found[i] = d
				case <-ctx.Done():
					found[i] = provider.Detection{ID: p.ID(), Name: p.DisplayName(), Problem: fmt.Sprintf("detection timed out after %g s", limit.Seconds())}
				}
			}()
		}
		wg.Wait()
		return detectedMsg{found}
	}
}

func (m *Model) detectionDone(found []provider.Detection) []tea.Cmd {
	byID := map[string]provider.Detection{}
	for _, d := range found {
		byID[d.ID] = d
	}
	var clis []state.CLI
	var installed []string
	for _, c := range supported {
		d, ok := byID[c.id]
		if !ok {
			clis = append(clis, state.CLI{ID: c.id, Name: c.name, Problem: "adapter not built yet"})
			continue
		}
		m.detected[c.id] = d
		clis = append(clis, state.CLI{ID: c.id, Name: c.name, Version: d.Version, Tested: d.Tested, Path: d.Path, Installed: d.Installed, Problem: d.Problem})
		if d.Installed {
			installed = append(installed, c.id)
		}
	}
	if !contains(installed, m.ui.Choice) {
		m.ui.Choice = ""
		if len(installed) > 0 {
			m.ui.Choice = installed[0]
		}
	}
	return m.do(state.DetectionDone{Installed: installed, CLIs: clis})
}

// start starts the session; resume is a CLI session id to resume.
func (m *Model) start(resume string) tea.Cmd {
	var p provider.Provider
	sc := provider.SessionContext{StartedAt: m.opt.Now(), ResumeID: resume}
	if m.s.Context.Mock {
		evs, err := mock.Load(m.opt.Mock)
		if err != nil {
			return func() tea.Msg { return actionMsg{state.SpawnFailed{Message: err.Error()}} }
		}
		p = &mock.Provider{Events: evs, Speed: m.opt.MockSpeed, Now: m.opt.Now}
		sc.Provider, sc.Name, sc.Mock = mock.ProviderID, p.DisplayName(), true
	} else {
		for _, q := range m.opt.Providers {
			if q.ID() == m.s.Context.Provider {
				p = q
			}
		}
		d := m.detected[m.s.Context.Provider]
		sc.Provider, sc.Name, sc.Version, sc.Executable = d.ID, d.Name, d.Version, d.Path
		sc.ProjectPath, sc.ProjectName = m.s.Context.Project, filepath.Base(m.s.Context.Project)
	}
	if p == nil {
		return func() tea.Msg { return actionMsg{state.SpawnFailed{Message: "no adapter for " + m.s.Context.Provider}} }
	}
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		sess, err := p.Start(ctx, sc)
		if err != nil {
			cancel()
		}
		return startedMsg{sess, cancel, err}
	}
}

func (m *Model) started(msg startedMsg) []tea.Cmd {
	if msg.err != nil {
		return m.do(state.SpawnFailed{Message: msg.err.Error()})
	}
	if m.sess != nil && m.cancel != nil {
		m.cancel() // a resumed session replaces the crashed one
	}
	m.sess, m.cancel = msg.sess, msg.cancel
	cmds := []tea.Cmd{listen(m.sess)}
	if m.s.Session.Phase == state.Closing {
		cmds = append(cmds, m.close()) // the owner quit while it started
	}
	return cmds
}

// listen waits for the next event, then takes whatever else is already
// waiting, so a burst is one screen update.
func listen(sess provider.Session) tea.Cmd {
	if sess == nil {
		return nil
	}
	ch := sess.Events()
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return eventsMsg{sess: sess, closed: true}
		}
		evs := []event.Event{ev}
		for len(evs) < maxDrain {
			select {
			case ev, ok := <-ch:
				if !ok {
					return eventsMsg{sess, evs, true}
				}
				evs = append(evs, ev)
			default:
				return eventsMsg{sess, evs, false}
			}
		}
		return eventsMsg{sess, evs, false}
	}
}

// close asks the session to close; the reducer's timers escalate. The close
// waits behind the writes queued before it, so the answers and the
// interrupt of a confirmed quit reach the CLI before its input closes.
func (m *Model) close() tea.Cmd {
	sess := m.sess
	if sess == nil {
		return nil // started() closes a session that arrives while closing
	}
	run := func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), CloseTimeout)
		defer cancel()
		_ = sess.Close(ctx) // its session.ended, or the close timeout, follows
		return nil
	}
	if m.writes != nil && len(m.writes) == cap(m.writes) {
		return run // the CLI reads nothing: closing comes before the order
	}
	return m.write(run, nil)
}

// Shutdown closes a session still open when the program ends another way
// than the reducer's exit (an error, a signal). Safe to call twice.
func (m *Model) Shutdown() {
	m.once.Do(func() {
		m.stopGraphRead() // the Git Graph's git process too (plan 62, design D4)
		if m.lock != nil {
			_ = m.lock.Release() // the program ends: its registration with it
			m.lock = nil
		}
		if m.sess == nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), CloseTimeout)
		defer cancel()
		_ = m.sess.Close(ctx)
		if m.cancel != nil {
			m.cancel()
		}
	})
}

func (m *Model) paste(text string) {
	// The layer on top takes the paste, as it takes the keys (plan 49,
	// design D7); the key overlay has no field, and what it hides takes
	// none.
	top := render.Layers(m.s, m.ui).Top
	switch {
	case m.wt != nil:
		if m.wt.mode == 'n' && m.wt.busy == "" {
			m.wt.branch.Insert(firstLine(text)) // a branch name is one line
		}
	case m.s.Screen == state.ScreenStartup && m.ui.Field == 1:
		m.path.Insert(input.CleanPath(text))
	case top == render.LayerKeys:
	case top == render.LayerWindow:
		m.filePaste(text) // the window takes it, or drops it
	case top == render.LayerBranches:
		m.branchPaste(text) // the search (plan 59, design D10)
	case top == render.LayerScope:
		m.scopePaste(text) // the scope popup's search (plan 62, design D3)
	case m.s.Screen == state.ScreenDashboard && m.msgFocused():
		m.msg.Paste(text) // the commit message (plan 45, design D2): never commits
	case m.s.Screen == state.ScreenDashboard && m.draftBeside():
		m.edit.Paste(text) // line breaks as LF, nothing else changed; never submits
	}
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// sendAnswer sends an approval's answer: a question's answers when it has some,
// else the decision (plan 14, item 1).
func sendAnswer(sess provider.Session, id string, decision event.Decision, answers map[string]string) error {
	if answers != nil {
		return sess.AnswerQuestion(id, answers)
	}
	return sess.Answer(id, decision)
}
