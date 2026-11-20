// Package sequence parses, lays out, and renders Mermaid sequence diagrams
// to SVG. It is self-contained: source text in, SVG bytes out.
package sequence

// Head is the arrowhead style at the end of a message.
type Head int

const (
	// HeadNone is a line with no arrowhead (-> / -->).
	HeadNone Head = iota
	// HeadArrow is a solid triangular arrowhead (->> / -->>).
	HeadArrow
	// HeadCross is an X at the end, denoting a lost/async-end message (-x / --x).
	HeadCross
)

// Arrow describes a message line's style.
type Arrow struct {
	Dashed bool
	Head   Head
}

// Participant is an actor with a vertical lifeline.
type Participant struct {
	ID    string
	Label string

	// X is the lifeline's horizontal center; set during layout.
	X float64
	// Width is the header box width; set during layout.
	Width float64

	// CreateRow is the row where a `create`d participant first appears, or -1
	// for a participant whose lifeline starts at the top of the diagram.
	CreateRow int
	// DestroyRow is the row of a `destroy` marker, or -1 when the lifeline
	// runs to the bottom of the diagram.
	DestroyRow int
}

// bornAtTop reports whether the lifeline starts with the header at the top.
// A created participant's header is drawn at its first message instead.
func (p *Participant) bornAtTop() bool { return p.CreateRow < 0 }

// Message is a single arrow from one participant to another.
type Message struct {
	From  string
	To    string
	Text  string
	Arrow Arrow

	Num int     // autonumber sequence (0 = unnumbered)
	Row int     // ordinal row in the diagram (messages and notes share rows)
	Y   float64 // vertical position of the message line; set during layout
}

// NotePos is where a note sits relative to its participant(s).
type NotePos int

const (
	// NoteRight places the note to the right of a participant.
	NoteRight NotePos = iota
	// NoteLeft places the note to the left of a participant.
	NoteLeft
	// NoteOver spans the note over one or two participants.
	NoteOver
)

// Note is an annotation box occupying its own row.
type Note struct {
	Pos  NotePos
	Of   []string
	Text string

	Row int
	Y   float64
}

// Bar is an activation lifespan on a participant's lifeline, in rows.
type Bar struct {
	Participant string
	StartRow    int
	EndRow      int
}

// Section is an else/and divider inside a frame.
type Section struct {
	Row   int
	Label string
}

// Frame is a grouping box (loop/alt/opt/par/rect/critical/break) spanning rows.
type Frame struct {
	Type     string
	Label    string
	StartRow int
	EndRow   int
	Sections []*Section

	// Color is the background written as `rect rgb(0, 0, 255)`. It is empty
	// for every other frame kind.
	Color string
}

// Diagram is a parsed sequence diagram.
type Diagram struct {
	Participants []*Participant
	Messages     []*Message
	Notes        []*Note
	Bars         []*Bar
	Frames       []*Frame

	rows int // number of allocated rows
}

// participant returns the participant with id, or nil.
func (d *Diagram) participant(id string) *Participant {
	for _, p := range d.Participants {
		if p.ID == id {
			return p
		}
	}
	return nil
}
