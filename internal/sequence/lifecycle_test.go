package sequence

import (
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/zkrebbekx/go-mermaid/internal/syntax"
)

func posErr(err error) *syntax.Error {
	t := &syntax.Error{}
	if !errors.As(err, &t) {
		return nil
	}
	return t
}

func TestCreateLifecycle(t *testing.T) {
	Convey("Given a created participant", t, func() {
		src := "sequenceDiagram\n" +
			"participant A\n" +
			"create participant W as Worker\n" +
			"A->>W: spawn\n" +
			"W-->>A: ack"

		Convey("When parsing", func() {
			d, err := Parse(src)

			Convey("Then the participant is born on its first message row", func() {
				So(err, ShouldBeNil)
				w := d.participant("W")
				So(w, ShouldNotBeNil)
				So(w.Label, ShouldEqual, "Worker")
				So(w.CreateRow, ShouldEqual, 0)
				So(w.DestroyRow, ShouldEqual, -1)
			})

			Convey("Then the create directive consumes no row", func() {
				So(d.Messages[0].Row, ShouldEqual, 0)
				So(d.Messages[1].Row, ShouldEqual, 1)
			})

			Convey("Then pre-existing participants keep a top-born lifeline", func() {
				So(d.participant("A").bornAtTop(), ShouldBeTrue)
				So(d.participant("W").bornAtTop(), ShouldBeFalse)
			})
		})
	})

	Convey("Given the short create form without a kind keyword", t, func() {
		Convey("When parsing `create W`", func() {
			d, err := Parse("sequenceDiagram\ncreate W\nA->>W: spawn")

			Convey("Then it is still treated as a created participant", func() {
				So(err, ShouldBeNil)
				So(d.participant("W").CreateRow, ShouldEqual, 0)
			})
		})
	})

	Convey("Given create actor syntax", t, func() {
		Convey("When parsing", func() {
			d, err := Parse("sequenceDiagram\ncreate actor W\nA->>W: spawn")

			Convey("Then the actor is registered like a participant", func() {
				So(err, ShouldBeNil)
				So(d.participant("W").CreateRow, ShouldEqual, 0)
			})
		})
	})
}

func TestDestroyLifecycle(t *testing.T) {
	Convey("Given a destroyed participant", t, func() {
		src := "sequenceDiagram\n" +
			"participant W\n" +
			"A->>W: work\n" +
			"destroy W\n" +
			"A->>A: continue"

		Convey("When parsing", func() {
			d, err := Parse(src)

			Convey("Then destroy occupies its own row and records the marker", func() {
				So(err, ShouldBeNil)
				So(d.participant("W").DestroyRow, ShouldEqual, 1)
				So(d.Messages[1].Row, ShouldEqual, 2)
			})
		})
	})

	Convey("Given an activation open at destroy time", t, func() {
		src := "sequenceDiagram\nparticipant W\nA->>+W: work\ndestroy W"

		Convey("When parsing", func() {
			d, err := Parse(src)

			Convey("Then the bar is closed at the destroy row, not the diagram end", func() {
				So(err, ShouldBeNil)
				So(len(d.Bars), ShouldEqual, 1)
				So(d.Bars[0].Participant, ShouldEqual, "W")
				So(d.Bars[0].StartRow, ShouldEqual, 0)
				So(d.Bars[0].EndRow, ShouldEqual, 1)
			})
		})
	})

	Convey("Given a created and then destroyed participant", t, func() {
		src := "sequenceDiagram\ncreate participant W\nA->>W: spawn\ndestroy W"

		Convey("When parsing", func() {
			d, err := Parse(src)

			Convey("Then both boundary rows are recorded", func() {
				So(err, ShouldBeNil)
				w := d.participant("W")
				So(w.CreateRow, ShouldEqual, 0)
				So(w.DestroyRow, ShouldEqual, 1)
			})
		})
	})
}

func TestLifecycleErrors(t *testing.T) {
	Convey("Given lifecycle misuse", t, func() {
		cases := []struct {
			name     string
			src      string
			wantLine int
			wantCol  int
		}{
			{
				"duplicate create of the same id",
				"sequenceDiagram\ncreate participant W\nA->>W: x\ncreate participant W",
				4, 20,
			},
			{
				"create of an existing declared participant",
				"sequenceDiagram\nparticipant W\ncreate participant W",
				3, 20,
			},
			{
				"destroy without create",
				"sequenceDiagram\ndestroy Ghost",
				2, 9,
			},
			{
				"double destroy",
				"sequenceDiagram\nparticipant W\ndestroy W\ndestroy W",
				4, 9,
			},
			{
				"message from a destroyed participant",
				"sequenceDiagram\nparticipant W\nW->>A: x\ndestroy W\nW->>A: late",
				5, 1,
			},
			{
				"message to a destroyed participant",
				"sequenceDiagram\nparticipant W\ndestroy W\nA->>W: late",
				4, 5,
			},
			{
				"note referencing a destroyed participant",
				"sequenceDiagram\nparticipant W\ndestroy W\nNote over W: gone",
				4, 11,
			},
			{
				"duplicate display label across participants",
				"sequenceDiagram\nparticipant A as Foo\ncreate participant B as Foo",
				3, 20,
			},
			{
				"redeclaration with a conflicting alias",
				"sequenceDiagram\nparticipant A as Alice\nparticipant A as Bob",
				3, 13,
			},
			{
				"created participant never used",
				"sequenceDiagram\nparticipant A\ncreate participant W\nA->>A: x",
				3, 20,
			},
			{
				"destroy a still-pending participant",
				"sequenceDiagram\ncreate participant W\ndestroy W",
				3, 9,
			},
			{
				"activate a destroyed participant",
				"sequenceDiagram\nparticipant W\ndestroy W\nactivate W",
				4, 10,
			},
			{
				"create without a name",
				"sequenceDiagram\ncreate",
				2, 1,
			},
			{
				"destroy without a name",
				"sequenceDiagram\ndestroy",
				2, 1,
			},
		}
		for _, c := range cases {
			c := c
			Convey("When "+c.name, func() {
				_, err := Parse(c.src)

				Convey("Then a positional error is returned", func() {
					So(err, ShouldNotBeNil)
					pe := posErr(err)
					So(pe, ShouldNotBeNil)
					So(pe.Line, ShouldEqual, c.wantLine)
					So(pe.Col, ShouldEqual, c.wantCol)
				})
			})
		}
	})

	Convey("Given an idempotent repeat declaration with the same label", t, func() {
		Convey("When the participant is declared twice identically", func() {
			_, err := Parse("sequenceDiagram\nparticipant A as Alice\nparticipant A as Alice\nA->>A: x")

			Convey("Then it is accepted", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestLifecycleInsideFrames(t *testing.T) {
	Convey("Given create and destroy nested inside an alt and a loop", t, func() {
		src := "sequenceDiagram\n" +
			"participant C\n" +
			"alt branch\n" +
			"create participant W\n" +
			"C->>W: spawn\n" +
			"loop retry\n" +
			"C->>W: ping\n" +
			"end\n" +
			"destroy W\n" +
			"end"

		Convey("When parsing", func() {
			d, err := Parse(src)

			Convey("Then lifecycle state is linear and survives the frames", func() {
				So(err, ShouldBeNil)
				w := d.participant("W")
				So(w.CreateRow, ShouldEqual, 0)
				So(w.DestroyRow, ShouldEqual, 2)
				So(len(d.Frames), ShouldEqual, 2)
			})
		})

		Convey("When a later branch references the destroyed participant", func() {
			later := src + "\nC->>W: after alt"
			_, err := Parse(later)

			Convey("Then the frame end does not restore the participant", func() {
				So(err, ShouldNotBeNil)
				So(posErr(err).Msg, ShouldContainSubstring, "destroyed")
			})
		})
	})
}
