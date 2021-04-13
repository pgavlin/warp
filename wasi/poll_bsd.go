// +build darwin dragonfly freebsd netbsd openbsd

package wasi

import (
	"os"
	"syscall"
	"unsafe"
)

type osFS struct {
}

func newOSFS() *osFS {
	return &osFS{}
}

func (fs *osFS) Poll(subscriptions []Subscription) ([]Event, error) {
	kq, err := syscall.Kqueue()
	if err != nil {
		return nil, err
	}
	defer syscall.Close(kq)

	kevents := make([]syscall.Kevent_t, len(subscriptions))
	for i := range subscriptions {
		sub := &subscriptions[i]

		f, ok := sub.File.(*osFile)
		if !ok {
			return nil, os.ErrInvalid
		}

		event := &kevents[i]

		switch sub.Kind {
		case SubscriptionTimer:
			event.Ident = uint64(i)
			event.Filter = syscall.EVFILT_TIMER
			if !sub.Deadline.IsZero() {
				event.Fflags = syscall.NOTE_ABSOLUTE
				event.Data = sub.Deadline.UnixNano() / 1000000
			} else {
				event.Data = sub.Timeout.Milliseconds()
			}
		case SubscriptionRead:
			event.Ident = uint64(f.f.Fd())
			event.Filter = syscall.EVFILT_READ
		case SubscriptionWrite:
			event.Ident = uint64(f.f.Fd())
			event.Filter = syscall.EVFILT_WRITE
		default:
			return nil, os.ErrInvalid
		}

		event.Flags = syscall.EV_ENABLE | syscall.EV_ONESHOT
		event.Udata = (*byte)(unsafe.Pointer(uintptr(sub.Userdata)))
	}

	n, err := syscall.Kevent(int(kq), kevents, kevents, nil)
	if err != nil {
		return nil, err
	}

	events := make([]Event, n)
	for i, kevent := range kevents[:n] {
		event := &events[i]

		switch kevent.Filter {
		case syscall.EVFILT_TIMER:
			event.Kind = SubscriptionTimer
		case syscall.EVFILT_READ:
			event.Kind = SubscriptionRead
			event.Available = uint(kevent.Data)
			if kevent.Flags&syscall.EV_EOF != 0 {
				event.Flags = EventHangup
			}
		case syscall.EVFILT_WRITE:
			event.Kind = SubscriptionWrite
			event.Available = uint(kevent.Data)
			if kevent.Flags&syscall.EV_EOF != 0 {
				event.Flags = EventHangup
			}
		}

		if event.Flags&syscall.EV_ERROR != 0 {
			event.Error = int(kevent.Data)
		}

		event.Userdata = uint64(uintptr(unsafe.Pointer(kevent.Udata)))
	}

	return events, nil
}
