package tui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"
)

type inputEvent struct {
	byte byte
	err  error
}

type keyReader struct {
	events <-chan inputEvent
}

func newKeyReader(in io.Reader) *keyReader {
	events := make(chan inputEvent, 8)
	reader := bufio.NewReader(in)
	go func() {
		defer close(events)
		for {
			value, err := reader.ReadByte()
			events <- inputEvent{byte: value, err: err}
			if err != nil {
				return
			}
		}
	}()
	return &keyReader{events: events}
}

func (r *keyReader) readKey() (string, error) {
	event, ok := <-r.events
	if !ok {
		return "", io.EOF
	}
	if event.err != nil {
		return "", event.err
	}
	if event.byte == 0x1b {
		select {
		case next := <-r.events:
			if next.err != nil || next.byte != '[' {
				return "esc", nil
			}
			last := <-r.events
			if last.err != nil {
				return "esc", nil
			}
			switch last.byte {
			case 'A':
				return "up", nil
			case 'B':
				return "down", nil
			}
			return "", nil
		case <-time.After(35 * time.Millisecond):
			return "esc", nil
		}
	}
	if event.byte == '\r' || event.byte == '\n' {
		return "enter", nil
	}
	return strings.ToLower(string(event.byte)), nil
}

func (r *keyReader) readLine(out io.Writer, label, initial string) (string, bool, error) {
	value := []byte(initial)
	fmt.Fprintf(out, "%s%s", label, string(value))
	for {
		event, ok := <-r.events
		if !ok {
			return "", false, io.EOF
		}
		if event.err != nil {
			return "", false, event.err
		}
		switch event.byte {
		case 0x1b:
			return "", false, nil
		case '\r', '\n':
			fmt.Fprintln(out)
			return string(value), true, nil
		case 0x7f, 0x08:
			if len(value) > 0 {
				_, size := utf8.DecodeLastRune(value)
				value = value[:len(value)-size]
				fmt.Fprint(out, "\b \b")
			}
		case 0x03:
			return "", false, io.EOF
		case 0x15:
			for len(value) > 0 {
				_, size := utf8.DecodeLastRune(value)
				value = value[:len(value)-size]
				fmt.Fprint(out, "\b \b")
			}
		default:
			value = append(value, event.byte)
			fmt.Fprint(out, string(event.byte))
		}
	}
}

func enableRawMode(in io.Reader) func() {
	file, ok := in.(*os.File)
	if !ok {
		return func() {}
	}
	stateCommand := exec.Command("stty", "-g")
	stateCommand.Stdin = file
	state, err := stateCommand.Output()
	if err != nil {
		return func() {}
	}
	command := exec.Command("stty", "-icanon", "-echo")
	command.Stdin = file
	if err := command.Run(); err != nil {
		return func() {}
	}
	return func() {
		restore := exec.Command("stty", strings.TrimSpace(string(state)))
		restore.Stdin = file
		_ = restore.Run()
	}
}
