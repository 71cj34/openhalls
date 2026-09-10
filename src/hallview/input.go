package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// readInputLine prints prompt and reads one line with basic line editing.
// Only the Backspace key goes back: on an empty buffer it returns
// back=true immediately without waiting for Enter.
// Left/Right move the cursor (byte-aware for pasted UTF-8);
// Home/End jump; Delete removes the character under the cursor.
// Ctrl+C exits. Returns back=true on back-out.
func readInputLine(prompt string) (string, bool) {
	fmt.Print(indent + prompt)

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", true
		}
		line = strings.TrimRight(line, "\r\n")
		if strings.ContainsRune(line, 0x7f) || strings.ContainsRune(line, 0x08) {
			fmt.Println()
			return "", true
		}
		return strings.TrimSpace(line), false
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", true
		}
		line = strings.TrimRight(line, "\r\n")
		if strings.ContainsRune(line, 0x7f) || strings.ContainsRune(line, 0x08) {
			fmt.Println()
			return "", true
		}
		return strings.TrimSpace(line), false
	}
	defer term.Restore(fd, oldState)

	var buf []byte
	pos := 0
	pushed := -1
	var pending []byte

	nextByte := func() (byte, bool) {
		if pushed >= 0 {
			b := byte(pushed)
			pushed = -1
			return b, true
		}
		var one [1]byte
		n, err := os.Stdin.Read(one[:])
		if err != nil || n == 0 {
			return 0, false
		}
		return one[0], true
	}

	redraw := func() {
		fmt.Print("\r\x1b[K" + indent + prompt + string(buf))
		if n := utf8.RuneCount(buf[pos:]); n > 0 {
			fmt.Printf("\x1b[%dD", n)
		}
	}
	moveLeft := func() {
		if pos > 0 {
			pos--
			for pos > 0 && buf[pos]&0xC0 == 0x80 {
				pos--
			}
			fmt.Print("\x1b[D")
		}
	}
	moveRight := func() {
		if pos < len(buf) {
			_, size := utf8.DecodeRune(buf[pos:])
			if size < 1 {
				size = 1
			}
			pos += size
			fmt.Print("\x1b[C")
		}
	}
	goHome := func() {
		if pos != 0 {
			pos = 0
			redraw()
		}
	}
	goEnd := func() {
		if pos != len(buf) {
			pos = len(buf)
			redraw()
		}
	}
	insertBytes := func(bs []byte) {
		nb := make([]byte, 0, len(buf)+len(bs))
		nb = append(nb, buf[:pos]...)
		nb = append(nb, bs...)
		nb = append(nb, buf[pos:]...)
		buf = nb
		pos += len(bs)
		redraw()
	}
	deleteBefore := func() {
		if pos > 0 {
			start := pos - 1
			for start > 0 && buf[start]&0xC0 == 0x80 {
				start--
			}
			buf = append(buf[:start], buf[pos:]...)
			pos = start
			redraw()
		}
	}
	deleteAt := func() {
		if pos < len(buf) {
			_, size := utf8.DecodeRune(buf[pos:])
			if size < 1 {
				size = 1
			}
			buf = append(buf[:pos], buf[pos+size:]...)
			redraw()
		}
	}
	applyCSI := func(params string, final byte) {
		switch csiAction(params, final) {
		case "left":
			moveLeft()
		case "right":
			moveRight()
		case "home":
			goHome()
		case "end":
			goEnd()
		case "del":
			deleteAt()
		}
	}
	// applyWinKey handles the two-byte Windows console sequences
	// (0xE0/0x00 prefix) some terminals emit instead of ANSI escapes.
	applyWinKey := func(c byte) {
		switch c {
		case 0x4B:
			moveLeft()
		case 0x4D:
			moveRight()
		case 0x47:
			goHome()
		case 0x4F:
			goEnd()
		case 0x53:
			deleteAt()
		}
	}

	for {
		b, ok := nextByte()
		if !ok {
			fmt.Println()
			return "", true
		}
		// Accumulate multi-byte UTF-8 so pasted text inserts as runes.
		// 0xE0 doubles as a Windows key prefix; disambiguate by peeking:
		// a continuation byte (0x80-0xBF) means UTF-8, else a key code.
		if len(pending) > 0 || b >= 0x80 {
			if b == 0xE0 && len(pending) == 0 {
				c, ok := nextByte()
				if !ok {
					fmt.Println()
					return "", true
				}
				if c < 0x80 || c >= 0xC0 {
					applyWinKey(c)
					continue
				}
				pending = append(pending, b, c)
			} else {
				pending = append(pending, b)
			}
			if !utf8.FullRune(pending) {
				continue
			}
			r, _ := utf8.DecodeRune(pending)
			pending = pending[:0]
			if r == utf8.RuneError {
				continue
			}
			var enc [utf8.UTFMax]byte
			insertBytes(enc[:utf8.EncodeRune(enc[:], r)])
			continue
		}
		switch {
		case b == '\r' || b == '\n':
			fmt.Print("\r\n")
			return strings.TrimSpace(string(buf)), false
		case b == 0x7f || b == 0x08:
			if len(buf) == 0 && pos == 0 {
				fmt.Print("\r\n")
				return "", true
			}
			deleteBefore()
		case b == 0x00:
			c, ok := nextByte()
			if !ok {
				fmt.Println()
				return "", true
			}
			applyWinKey(c)
		case b == 0x03:
			term.Restore(fd, oldState)
			fmt.Print("\r\n")
			os.Exit(0)
			return "", true
		case b == 0x1b:
			b2, ok := nextByte()
			if !ok {
				fmt.Println()
				return "", true
			}
			switch b2 {
			case 0x1b:
				pushed = int(b2)
			case '[':
				var params []byte
				final := byte(0)
				for len(params) < 16 {
					c, ok := nextByte()
					if !ok {
						fmt.Println()
						return "", true
					}
					if c >= 0x40 {
						final = c
						break
					}
					params = append(params, c)
				}
				if final != 0 {
					applyCSI(string(params), final)
				}
			case 'O':
				c, ok := nextByte()
				if !ok {
					fmt.Println()
					return "", true
				}
				applyCSI("", ss3Final(c))
			default:
				pushed = int(b2)
			}
		case b == 0x01:
			goHome()
		case b == 0x05:
			goEnd()
		case b == 0x02:
			moveLeft()
		case b == 0x06:
			moveRight()
		case b == 0x04:
			deleteAt()
		case b == 0x15:
			buf = nil
			pos = 0
			redraw()
		case b == 0x0b:
			buf = buf[:pos]
			redraw()
		case b < 0x20:
			continue
		default:
			insertBytes([]byte{b})
		}
	}
}

// csiAction maps a CSI sequence (params + final byte) to an editor
// action. Params carry modifiers (e.g. "1;5" for Ctrl+Left) which don't
// change the basic movement, so they are ignored except for "~".
func csiAction(params string, final byte) string {
	switch final {
	case 'D':
		return "left"
	case 'C':
		return "right"
	case 'A':
		return "up"
	case 'B':
		return "down"
	case 'H':
		return "home"
	case 'F':
		return "end"
	case '~':
		switch params {
		case "3":
			return "del"
		case "1", "7":
			return "home"
		case "4", "8":
			return "end"
		}
	}
	return "ignore"
}

// ss3Final maps an SS3 terminator (ESC O <c>) onto its CSI equivalent
// so both arrow encodings share one handler.
func ss3Final(c byte) byte {
	switch c {
	case 'A', 'B', 'C', 'D', 'H', 'F', 'P', 'Q', 'R', 'S':
		return c
	}
	return 0
}
