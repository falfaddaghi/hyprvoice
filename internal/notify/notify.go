package notify

import (
	"context"
	"log"
	"os/exec"
	"sync"
	"time"
)

type Notifier interface {
	Send(mt MessageType)
	Error(msg string) // for dynamic errors (e.g., pipeline errors)
	StartAnimation(mt MessageType)
	StopAnimation()
}

// NewNotifier creates a notifier based on type with resolved messages
func NewNotifier(notifType string, messages map[MessageType]Message) Notifier {
	switch notifType {
	case "desktop":
		return NewDesktop(messages)
	case "log":
		return NewLog(messages)
	default:
		return &Nop{}
	}
}

// braille spinner frames
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type Desktop struct {
	messages  map[MessageType]Message
	animMu    sync.Mutex
	animCancel context.CancelFunc
}

func NewDesktop(messages map[MessageType]Message) *Desktop {
	return &Desktop{messages: messages}
}

func (d *Desktop) Send(mt MessageType) {
	msg, ok := d.messages[mt]
	if !ok {
		return
	}
	if msg.IsError {
		d.Error(msg.Body)
		return
	}
	d.notify(msg.Title, msg.Body)
}

func (d *Desktop) Error(msg string) {
	cmd := exec.Command("notify-send", "-a", "Hyprvoice", "-u", "critical",
		"-h", "string:x-canonical-private-synchronous:hyprvoice",
		"Hyprvoice Error", msg)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send error notification: %v", err)
	}
}

func (d *Desktop) StartAnimation(mt MessageType) {
	d.animMu.Lock()
	defer d.animMu.Unlock()

	// Stop any existing animation
	if d.animCancel != nil {
		d.animCancel()
		d.animCancel = nil
	}

	msg, ok := d.messages[mt]
	if !ok {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.animCancel = cancel

	go func() {
		i := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
				frame := spinnerFrames[i%len(spinnerFrames)]
				body := frame + " " + msg.Body
				d.notifyReplace(msg.Title, body)
				i++
				select {
				case <-ctx.Done():
					return
				case <-time.After(150 * time.Millisecond):
				}
			}
		}
	}()
}

func (d *Desktop) StopAnimation() {
	d.animMu.Lock()
	defer d.animMu.Unlock()

	if d.animCancel != nil {
		d.animCancel()
		d.animCancel = nil
	}
}

func (d *Desktop) notify(title, body string) {
	cmd := exec.Command("notify-send", "-a", "Hyprvoice",
		"-h", "string:x-canonical-private-synchronous:hyprvoice",
		title, body)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

func (d *Desktop) notifyReplace(title, body string) {
	cmd := exec.Command("notify-send", "-a", "Hyprvoice",
		"-h", "string:x-canonical-private-synchronous:hyprvoice",
		title, body)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

type Log struct {
	messages map[MessageType]Message
}

func NewLog(messages map[MessageType]Message) *Log {
	return &Log{messages: messages}
}

func (l *Log) Send(mt MessageType) {
	msg, ok := l.messages[mt]
	if !ok {
		return
	}
	if msg.IsError {
		l.Error(msg.Body)
		return
	}
	log.Printf("%s: %s", msg.Title, msg.Body)
}

func (l *Log) Error(msg string) {
	log.Printf("Hyprvoice Error: %s", msg)
}

func (l *Log) StartAnimation(mt MessageType) {
	msg, ok := l.messages[mt]
	if !ok {
		return
	}
	log.Printf("Processing: %s", msg.Body)
}

func (l *Log) StopAnimation() {
	log.Printf("Done")
}

type Nop struct{}

func (Nop) Send(mt MessageType)           {}
func (Nop) Error(msg string)              {}
func (Nop) StartAnimation(mt MessageType) {}
func (Nop) StopAnimation()                {}
