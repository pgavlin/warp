// +build windows

package wasi

import (
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

type osFS struct {
}

func newOSFS() *osFS {
	return &osFS{}
}

func (fs *osFS) Poll(subscriptions []Subscription) ([]Event, error) {
	const invalidHandle = ^windows.Handle(0)
	iocp, err := windows.CreateIoCompletionPort(invalidHandle, 0, 0, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(iocp)

	timeOrigin := time.Now()
	timeout := time.Duration(0)
	timeoutIndex := -1
	for i := range subscriptions {
		sub := &subscriptions[i]

		f, ok := sub.File.(*osFile)
		if !ok {
			return nil, os.ErrInvalid
		}

		switch sub.Kind {
		case SubscriptionTimer:
			var t time.Duration
			if !sub.Deadline.IsZero() {
				t = sub.Deadline.Sub(timeOrigin)
			} else {
				t = sub.Timeout
			}
			if timeoutIndex == -1 || t < timeout {
				timeout, timeoutIndex = t, i
			}
			continue
		case SubscriptionRead:
		case SubscriptionWrite:
		default:
			return nil, os.ErrInvalid
		}

		if _, err = windows.CreateIoCompletionPort(iocp, windows.Handle(f.f.Fd()), uintptr(i), 0); err != nil {
			return nil, err
		}
	}

	timeoutMilliseconds := -1
	if timeoutIndex != -1 {
		timeoutMilliseconds = int(timeout.Milliseconds())
	}

	var available uint32
	var key uintptr
	var overlapped *windows.Overlapped
	if err = windows.GetQueuedCompletionStatus(iocp, &available, &key, &overlapped, uint32(timeoutMilliseconds)); err != nil {
		if errno, ok := err.(syscall.Errno); ok && errno == syscall.WAIT_TIMEOUT && timeoutIndex != -1 {
			sub := &subscriptions[timeoutIndex]
			return []Event{{
				Kind:     SubscriptionTimer,
				Userdata: sub.Userdata,
			}}, nil
		}
		return nil, err
	}

	sub := &subscriptions[key]
	return []Event{{
		Kind:      sub.Kind,
		Available: uint(available),
		Userdata:  sub.Userdata,
	}}, nil
}
