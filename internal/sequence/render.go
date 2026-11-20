package sequence

import (
	"fmt"
	"strings"

	"github.com/zkrebbekx/go-mermaid/internal/svgutil"
	"github.com/zkrebbekx/go-mermaid/internal/theme"
)

// RenderOptions controls sequence diagram appearance.
type RenderOptions struct {
	Theme    string
	FontFace string
	FontSize float64
	Padding  float64
	Title    string
}

// Render parses, lays out, and renders sequence diagram source to SVG.
func Render(src string, o RenderOptions) ([]byte, error) {
	d, err := Parse(src)
	if err != nil {
		return nil, err
	}
	lay := Compute(d, Options{FontSize: o.FontSize, Padding: o.Padding, FontFace: o.FontFace})
	return svg(lay, o), nil
}

func svg(lay *Layout, o RenderOptions) []byte {
	pal := theme.For(o.Theme)
	pad := o.Padding
	titleH := svgutil.TitleHeight(o.Title, o.FontSize)
	contentW := lay.Width
	face := svgutil.FaceFor(o.FontFace)
	if tw := face.Width(o.Title, o.FontSize); tw > contentW {
		contentW = tw
	}
	w := contentW + pad*2
	h := lay.Height + titleH + pad*2

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s" viewBox="0 0 %s %s" font-family="%s" font-size="%s">`,
		svgutil.Num(w), svgutil.Num(h), svgutil.Num(w), svgutil.Num(h), svgutil.Esc(o.FontFace), svgutil.Num(o.FontSize))
	b.WriteByte('\n')

	fmt.Fprintf(&b, `  <defs><marker id="seq-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0,0 L10,5 L0,10 z" fill="%s"/></marker></defs>`, pal.Edge)
	b.WriteByte('\n')
	fmt.Fprintf(&b, `  <rect width="100%%" height="100%%" fill="%s"/>`, pal.Background)
	b.WriteByte('\n')
	if o.Title != "" {
		fmt.Fprintf(&b, `  <text x="%s" y="%s" fill="%s" text-anchor="middle" font-weight="bold">%s</text>`,
			svgutil.Num(w/2), svgutil.Num(pad+o.FontSize), pal.Text, svgutil.Esc(o.Title))
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, `  <g transform="translate(%s,%s)">`, svgutil.Num(pad+lay.OffsetX), svgutil.Num(pad+titleH))
	b.WriteByte('\n')

	// Lifelines span each participant's live window only: a created
	// participant starts at its birth row and a destroyed one stops at the
	// destroy marker.
	for _, p := range lay.Diagram.Participants {
		writeLifeline(&b, p, lay, pal)
	}
	for _, bar := range lay.Diagram.Bars {
		writeBar(&b, bar, lay, pal)
	}
	for _, f := range lay.Diagram.Frames {
		writeFrame(&b, f, lay, pal, o)
	}
	// Static headers first; created participants carry their headers at the
	// birth row, drawn after messages so the box sits over the incoming line.
	for _, p := range lay.Diagram.Participants {
		if p.bornAtTop() {
			writeHeader(&b, p, lay, pal, o)
		}
	}
	for _, m := range lay.Diagram.Messages {
		writeMessage(&b, m, lay, pal)
	}
	for _, p := range lay.Diagram.Participants {
		if !p.bornAtTop() {
			writeHeader(&b, p, lay, pal, o)
		}
	}
	for _, n := range lay.Diagram.Notes {
		writeNote(&b, n, lay, pal, o)
	}
	for _, p := range lay.Diagram.Participants {
		if p.DestroyRow >= 0 {
			writeDestroy(&b, p, lay, pal)
		}
	}

	b.WriteString("  </g>\n</svg>\n")
	return []byte(b.String())
}

// writeLifeline draws the dashed line over the participant's live window.
func writeLifeline(b *strings.Builder, p *Participant, lay *Layout, pal theme.Palette) {
	y1 := lay.LifelineTop
	if !p.bornAtTop() {
		y1 = rowY(lay, p.CreateRow)
	}
	y2 := lay.LifelineEnd
	if p.DestroyRow >= 0 {
		y2 = rowY(lay, p.DestroyRow)
	}
	if y2 <= y1 {
		return
	}
	fmt.Fprintf(b, `    <line x1="%s" y1="%s" x2="%s" y2="%s" stroke="%s" stroke-dasharray="3,3"/>`,
		svgutil.Num(p.X), svgutil.Num(y1), svgutil.Num(p.X), svgutil.Num(y2), pal.Edge)
	b.WriteByte('\n')
}

func writeHeader(b *strings.Builder, p *Participant, lay *Layout, pal theme.Palette, o RenderOptions) {
	top := 0.0
	if !p.bornAtTop() {
		top = rowY(lay, p.CreateRow)
	}
	x := p.X - p.Width/2
	fmt.Fprintf(b, `    <rect x="%s" y="%s" width="%s" height="%s" rx="3" fill="%s" stroke="%s"/>`,
		svgutil.Num(x), svgutil.Num(top), svgutil.Num(p.Width), svgutil.Num(lay.HeaderHeight), pal.NodeFill, pal.NodeStroke)
	b.WriteByte('\n')
	fmt.Fprintf(b, `    <text x="%s" y="%s" fill="%s" text-anchor="middle">%s</text>`,
		svgutil.Num(p.X), svgutil.Num(top+lay.HeaderHeight/2+o.FontSize*0.35), pal.Text, svgutil.Esc(p.Label))
	b.WriteByte('\n')
}

// writeDestroy draws the X termination marker at the end of a lifeline.
func writeDestroy(b *strings.Builder, p *Participant, lay *Layout, pal theme.Palette) {
	const s = 6
	cx, cy := p.X, rowY(lay, p.DestroyRow)
	fmt.Fprintf(b, `    <path d="M%s,%s L%s,%s M%s,%s L%s,%s" stroke="%s" stroke-width="1.5"/>`,
		svgutil.Num(cx-s), svgutil.Num(cy-s), svgutil.Num(cx+s), svgutil.Num(cy+s),
		svgutil.Num(cx-s), svgutil.Num(cy+s), svgutil.Num(cx+s), svgutil.Num(cy-s), pal.Edge)
	b.WriteByte('\n')
}

func writeMessage(b *strings.Builder, m *Message, lay *Layout, pal theme.Palette) {
	from := lay.Diagram.participant(m.From)
	to := lay.Diagram.participant(m.To)
	if from == nil || to == nil {
		return
	}
	dash := ""
	if m.Arrow.Dashed {
		dash = ` stroke-dasharray="6,4"`
	}
	marker := ""
	if m.Arrow.Head == HeadArrow {
		marker = ` marker-end="url(#seq-arrow)"`
	}

	label := MessageLabel(m)

	if m.From == m.To {
		// Self-message: a small loop to the right of the lifeline.
		x := from.X
		y := m.Y
		path := fmt.Sprintf("M%s,%s L%s,%s L%s,%s L%s,%s",
			svgutil.Num(x), svgutil.Num(y),
			svgutil.Num(x+selfLoopW), svgutil.Num(y),
			svgutil.Num(x+selfLoopW), svgutil.Num(y+12),
			svgutil.Num(x), svgutil.Num(y+12))
		fmt.Fprintf(b, `    <path d="%s" fill="none" stroke="%s"%s%s/>`, path, pal.Edge, dash, marker)
		b.WriteByte('\n')
		writeMsgText(b, label, x+selfLoopW+selfLabelGap, y+6, pal, "start")
		return
	}

	// On the birth row the header box occupies the center; stop the line at
	// its edge so the arrowhead stays visible instead of hiding under it.
	x1, x2 := from.X, to.X
	if from.CreateRow == m.Row {
		if x1 < x2 {
			x1 += from.Width / 2
		} else {
			x1 -= from.Width / 2
		}
	}
	if to.CreateRow == m.Row {
		if x2 > x1 {
			x2 -= to.Width / 2
		} else {
			x2 += to.Width / 2
		}
	}
	fmt.Fprintf(b, `    <line x1="%s" y1="%s" x2="%s" y2="%s" stroke="%s"%s%s/>`,
		svgutil.Num(x1), svgutil.Num(m.Y), svgutil.Num(x2), svgutil.Num(m.Y), pal.Edge, dash, marker)
	b.WriteByte('\n')

	if m.Arrow.Head == HeadCross {
		writeCross(b, x2, m.Y, x1 < x2, pal)
	}
	writeMsgText(b, label, (from.X+to.X)/2, m.Y-4, pal, "middle")
}

func writeMsgText(b *strings.Builder, text string, x, y float64, pal theme.Palette, anchor string) {
	if text == "" {
		return
	}
	fmt.Fprintf(b, `    <text x="%s" y="%s" fill="%s" text-anchor="%s">%s</text>`,
		svgutil.Num(x), svgutil.Num(y), pal.Text, anchor, svgutil.Esc(text))
	b.WriteByte('\n')
}

// writeFrame draws a grouping frame (loop/alt/opt/...) with a label tab and
// any else/and section dividers.
func writeFrame(b *strings.Builder, f *Frame, lay *Layout, pal theme.Palette, o RenderOptions) {
	if f.EndRow < f.StartRow {
		return
	}
	if len(lay.Diagram.Participants) == 0 {
		return
	}
	minX, maxX := participantSpan(lay.Diagram)
	x, w := minX-frameInset, (maxX-minX)+frameInset*2
	top := rowY(lay, f.StartRow) - 16
	bot := rowY(lay, f.EndRow) + 14

	fill := "none"
	if f.Color != "" {
		fill = f.Color
	}
	fmt.Fprintf(b, `    <rect x="%s" y="%s" width="%s" height="%s" fill="%s" fill-opacity="0.15" stroke="%s" stroke-dasharray="2,2"/>`,
		svgutil.Num(x), svgutil.Num(top), svgutil.Num(w), svgutil.Num(bot-top), svgutil.Esc(fill), pal.NodeStroke)
	b.WriteByte('\n')

	tabW := svgutil.FaceFor(o.FontFace).Width(f.Type, o.FontSize) + 12
	fmt.Fprintf(b, `    <rect x="%s" y="%s" width="%s" height="%s" fill="%s" stroke="%s"/>`,
		svgutil.Num(x), svgutil.Num(top), svgutil.Num(tabW), svgutil.Num(o.FontSize+6), pal.NodeFill, pal.NodeStroke)
	b.WriteByte('\n')
	fmt.Fprintf(b, `    <text x="%s" y="%s" fill="%s" font-weight="bold">%s</text>`,
		svgutil.Num(x+6), svgutil.Num(top+o.FontSize), pal.Text, svgutil.Esc(f.Type))
	b.WriteByte('\n')
	if f.Label != "" {
		fmt.Fprintf(b, `    <text x="%s" y="%s" fill="%s">%s</text>`,
			svgutil.Num(x+tabW+6), svgutil.Num(top+o.FontSize), pal.Text, svgutil.Esc("["+f.Label+"]"))
		b.WriteByte('\n')
	}

	for _, s := range f.Sections {
		sy := rowY(lay, s.Row) - msgGap/2
		fmt.Fprintf(b, `    <line x1="%s" y1="%s" x2="%s" y2="%s" stroke="%s" stroke-dasharray="2,2"/>`,
			svgutil.Num(x), svgutil.Num(sy), svgutil.Num(x+w), svgutil.Num(sy), pal.NodeStroke)
		b.WriteByte('\n')
		if s.Label != "" {
			fmt.Fprintf(b, `    <text x="%s" y="%s" fill="%s">%s</text>`,
				svgutil.Num(x+6), svgutil.Num(sy+o.FontSize), pal.Text, svgutil.Esc("["+s.Label+"]"))
			b.WriteByte('\n')
		}
	}
}

// writeBar draws an activation bar on a participant's lifeline, clamped to
// the participant's live window.
func writeBar(b *strings.Builder, bar *Bar, lay *Layout, pal theme.Palette) {
	p := lay.Diagram.participant(bar.Participant)
	if p == nil {
		return
	}
	startRow, endRow := bar.StartRow, bar.EndRow
	if !p.bornAtTop() && startRow < p.CreateRow {
		startRow = p.CreateRow
	}
	if p.DestroyRow >= 0 && endRow > p.DestroyRow {
		endRow = p.DestroyRow
	}
	y1 := rowY(lay, startRow)
	y2 := rowY(lay, endRow)
	if y2-y1 < msgGap/2 {
		y2 = y1 + msgGap/2
	}
	fmt.Fprintf(b, `    <rect x="%s" y="%s" width="8" height="%s" fill="%s" stroke="%s"/>`,
		svgutil.Num(p.X-4), svgutil.Num(y1), svgutil.Num(y2-y1), pal.NodeFill, pal.NodeStroke)
	b.WriteByte('\n')
}

// writeNote draws a note box and its text.
func writeNote(b *strings.Builder, n *Note, lay *Layout, pal theme.Palette, o RenderOptions) {
	x, w := noteBox(lay.Diagram, n, svgutil.FaceFor(o.FontFace), o.FontSize)
	h := o.FontSize + 12
	y := n.Y - h/2
	fmt.Fprintf(b, `    <rect x="%s" y="%s" width="%s" height="%s" fill="%s" stroke="%s"/>`,
		svgutil.Num(x), svgutil.Num(y), svgutil.Num(w), svgutil.Num(h), "#FFF5AD", pal.NodeStroke)
	b.WriteByte('\n')
	fmt.Fprintf(b, `    <text x="%s" y="%s" fill="%s" text-anchor="middle">%s</text>`,
		svgutil.Num(x+w/2), svgutil.Num(y+h/2+o.FontSize*0.35), pal.Text, svgutil.Esc(n.Text))
	b.WriteByte('\n')
}

// writeCross draws an X at the receiving end of a -x / --x message.
func writeCross(b *strings.Builder, x, y float64, pointingRight bool, pal theme.Palette) {
	const s = 5
	cx := x + s
	if pointingRight {
		cx = x - s
	}
	fmt.Fprintf(b, `    <path d="M%s,%s L%s,%s M%s,%s L%s,%s" stroke="%s"/>`,
		svgutil.Num(cx-s), svgutil.Num(y-s), svgutil.Num(cx+s), svgutil.Num(y+s),
		svgutil.Num(cx-s), svgutil.Num(y+s), svgutil.Num(cx+s), svgutil.Num(y-s), pal.Edge)
	b.WriteByte('\n')
}
