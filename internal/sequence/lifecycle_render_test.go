package sequence

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/zkrebbekx/go-mermaid/internal/svgutil"
)

// twoActorLifecycle is the canonical two-participant sample: a client that
// lives for the whole diagram and a worker created and destroyed inside an
// alt branch containing a loop.
const twoActorLifecycle = "sequenceDiagram\n" +
	"participant C as Client\n" +
	"C->>C: prepare\n" +
	"alt needs worker\n" +
	"create participant W as Worker\n" +
	"C->>W: spawn\n" +
	"W-->>C: ack\n" +
	"loop retry\n" +
	"C->>W: ping\n" +
	"end\n" +
	"destroy W\n" +
	"end\n" +
	"C->>C: done"

func TestLifecycleLayout(t *testing.T) {
	Convey("Given the two-actor lifecycle sample", t, func() {
		lay := layoutOf(twoActorLifecycle)
		d := lay.Diagram

		Convey("When computing the layout", func() {
			c := d.participant("C")
			w := d.participant("W")

			Convey("Then rows are assigned with create consuming no row", func() {
				// Rows: 0 prepare, 1 spawn, 2 ack, 3 ping, 4 destroy, 5 done.
				So(w.CreateRow, ShouldEqual, 1)
				So(w.DestroyRow, ShouldEqual, 4)
				So(d.Messages[0].Row, ShouldEqual, 0)
				So(d.Messages[3].Row, ShouldEqual, 3)
				So(d.Messages[4].Row, ShouldEqual, 5)
			})

			Convey("Then the worker lifeline spans only its live rows", func() {
				birthY := rowY(lay, w.CreateRow)
				deathY := rowY(lay, w.DestroyRow)
				So(birthY, ShouldEqual, d.Messages[1].Y)
				So(deathY, ShouldBeGreaterThan, birthY)
			})

			Convey("Then the client lifeline still spans the full canvas", func() {
				So(c.bornAtTop(), ShouldBeTrue)
				So(c.DestroyRow, ShouldEqual, -1)
			})
		})
	})
}

func TestLifecycleRender(t *testing.T) {
	Convey("Given the two-actor lifecycle sample", t, func() {
		out, err := Render(twoActorLifecycle, ropts)
		svg := string(out)

		Convey("When rendering", func() {
			Convey("Then it renders without error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the worker header is drawn at its birth row, not at the top", func() {
				lay := layoutOf(twoActorLifecycle)
				w := lay.Diagram.participant("W")
				birthTop := rowY(lay, w.CreateRow)
				// The created header rect uses the birth-row y coordinate.
				So(svg, ShouldContainSubstring,
					`<rect x="`+svgutil.Num(w.X-w.Width/2)+`" y="`+svgutil.Num(birthTop)+`"`)
				// No worker header at y=0.
				So(strings.Contains(svg, `y="0" width="`+svgutil.Num(w.Width)+`"`), ShouldBeFalse)
			})

			Convey("Then the client header stays at the top", func() {
				So(svg, ShouldContainSubstring, `y="0"`)
				So(svg, ShouldContainSubstring, ">Client<")
			})

			Convey("Then the worker lifeline starts at the birth row and ends at the destroy X", func() {
				lay := layoutOf(twoActorLifecycle)
				w := lay.Diagram.participant("W")
				birthY := rowY(lay, w.CreateRow)
				deathY := rowY(lay, w.DestroyRow)
				So(svg, ShouldContainSubstring,
					`x1="`+svgutil.Num(w.X)+`" y1="`+svgutil.Num(birthY)+`" x2="`+svgutil.Num(w.X)+`" y2="`+svgutil.Num(deathY)+`"`)
			})

			Convey("Then the destroy marker is an X centred on the lifeline", func() {
				lay := layoutOf(twoActorLifecycle)
				w := lay.Diagram.participant("W")
				deathY := rowY(lay, w.DestroyRow)
				So(svg, ShouldContainSubstring,
					`M`+svgutil.Num(w.X-6)+`,`+svgutil.Num(deathY-6))
			})

			Convey("Then the birth-row arrow stops at the header edge", func() {
				lay := layoutOf(twoActorLifecycle)
				w := lay.Diagram.participant("W")
				c := lay.Diagram.participant("C")
				spawn := lay.Diagram.Messages[1]
				So(spawn.Text, ShouldEqual, "spawn")
				edgeX := w.X - w.Width/2
				So(svg, ShouldContainSubstring,
					`x1="`+svgutil.Num(c.X)+`" y1="`+svgutil.Num(spawn.Y)+`" x2="`+svgutil.Num(edgeX)+`"`)
			})

			Convey("Then the frame boxes still wrap the branch and loop", func() {
				So(svg, ShouldContainSubstring, ">alt<")
				So(svg, ShouldContainSubstring, ">loop<")
				So(svg, ShouldContainSubstring, "[needs worker]")
				So(svg, ShouldContainSubstring, "[retry]")
			})
		})
	})

	Convey("Given a created participant that is also activated", t, func() {
		src := "sequenceDiagram\nparticipant A\ncreate participant W\nA->>+W: spawn\nW-->>-A: done\ndestroy W"

		Convey("When rendering", func() {
			out, err := Render(src, ropts)

			Convey("Then the activation bar stays inside the live window", func() {
				So(err, ShouldBeNil)
				lay := layoutOf(src)
				w := lay.Diagram.participant("W")
				So(len(lay.Diagram.Bars), ShouldEqual, 1)
				bar := lay.Diagram.Bars[0]
				So(bar.StartRow, ShouldBeGreaterThanOrEqualTo, w.CreateRow)
				So(bar.EndRow, ShouldBeLessThanOrEqualTo, w.DestroyRow)
				So(string(out), ShouldContainSubstring, "width=\"8\"")
			})
		})
	})
}
