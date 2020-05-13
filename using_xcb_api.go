package main

import (
	"fmt"
	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"os"
	"strings"
)

const (
	StateNone = iota
	StateIncr
)

func main() {
	X, err := xgb.NewConn()
	if err != nil {
		fmt.Println(err)
		return
	}

	xinfo := xproto.Setup(X)

	/* Create CLIPBOARD atom */
	atomCookie := xproto.InternAtom(X, true, 9, "CLIPBOARD")
	var atomReply *xproto.InternAtomReply
	atomReply, err = atomCookie.Reply()
	if err != nil {
		fmt.Println(err)
		return
	}
	clipboardAtom := atomReply.Atom

	/* Create TARGETS atom */
	atomCookie = xproto.InternAtom(X, true, 7, "TARGETS")
	atomReply, err = atomCookie.Reply()
	if err != nil {
		fmt.Println(err)
		return
	}
	targetsAtom := atomReply.Atom

	/* Create INCR atom */
	atomCookie = xproto.InternAtom(X, true, 4, "INCR")
	atomReply, err = atomCookie.Reply()
	if err != nil {
		fmt.Println(err)
		return
	}
	incrAtom := atomReply.Atom

	/* Calculate chunk size */
	chunkSize := int(xinfo.MaximumRequestLength / 4)

	/* Create basic window to receive events */
	wid, _ := xproto.NewWindowId(X)
	screen := xinfo.DefaultScreen(X)
	xproto.CreateWindow(X, 0, wid, screen.Root, 0, 0, 1, 1, 0,
		xproto.WindowClassCopyFromParent, screen.RootVisual,
		xproto.CwEventMask, []uint32{xproto.EventMaskPropertyChange})

	xproto.SetSelectionOwner(X, wid, clipboardAtom, xproto.Timestamp(0))

	/* Wait to paste some stuff */
	msg := strings.Join(os.Args[1:], " ")

	/* NOTE: This is what xclip does, but not sure it's the best
	   thing. It might be using this across requests, which would
	   create a race condition */
	var (
		requestor xproto.Window
		property  xproto.Atom
		msgPos    int
	)

	state := StateNone
	for finished := false; ; {
		ev, xerr := X.WaitForEvent()
		if ev == nil && xerr == nil {
			fmt.Println("Both event and error are nil. Exiting...")
			return
		}
		if finished {
			return
		}

		if ev != nil {
			fmt.Printf("Event: %s\n", ev)

			switch state {
			case StateNone:
				req, ok := ev.(xproto.SelectionRequestEvent)
				if !ok {
					continue
				}

				requestor = req.Requestor
				property = req.Property
				if req.Target == targetsAtom {
					data := make([]byte, 8)
					xgb.Put32(data, uint32(targetsAtom))
					xgb.Put32(data[4:], uint32(xproto.AtomString))
					xproto.ChangeProperty(X, xproto.PropModeReplace,
						requestor, property, xproto.AtomAtom, byte(32),
						uint32(2), data)
				} else if len(msg) > chunkSize {
					xproto.ChangeProperty(X, xproto.PropModeReplace,
						requestor, property, incrAtom, byte(32), 0, nil)
					xproto.ChangeWindowAttributes(X, requestor,
						xproto.CwEventMask, []uint32{xproto.EventMaskPropertyChange})
					msgPos = 0

					state = StateIncr
				} else {
					xproto.ChangeProperty(X, xproto.PropModeReplace,
						requestor, property, xproto.AtomString,
						byte(8), uint32(len(msg)), []byte(msg))

					finished = true
				}

				res := xproto.SelectionNotifyEvent{
					Property:  property,
					Requestor: requestor,
					Selection: req.Selection,
					Target:    req.Target,
					Time:      req.Time,
				}
				xproto.SendEvent(X, false, requestor, 0, string(res.Bytes()))
			case StateIncr:
				req, ok := ev.(xproto.PropertyNotifyEvent)
				if !ok {
					continue
				}
				if req.State != xproto.PropertyDelete {
					continue
				}

				chunkLen := chunkSize
				if (msgPos + chunkLen) > len(msg) {
					chunkLen = len(msg) - msgPos
				}
				if msgPos > len(msg) {
					chunkLen = 0
				}

				var chunk []byte
				if chunkLen > 0 {
					chunk = []byte(msg)[msgPos:]
				}
				xproto.ChangeProperty(X, xproto.PropModeReplace,
					requestor, property, xproto.AtomString, byte(8),
					uint32(chunkLen), chunk)

				if chunkLen == 0 {
					state = StateNone
					finished = true
				}
				msgPos += chunkSize
			}
		}
		if xerr != nil {
			fmt.Printf("Error: %s\n", xerr)
		}
	}
}
