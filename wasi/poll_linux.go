// +build linux

package wasi

import (
	"os"
	"syscall"
	"time"
)

type osFS struct {
}

func newOSFS() *osFS {
	return &osFS{}
}

func (fs *osFS) Poll(subscriptions []Subscription) ([]Event, error) {
	epoll, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}
	defer syscall.Close(epoll)

	timeOrigin := time.Now()
	timeout := time.Duration(0)
	timeoutIndex := -1
	for i := range subscriptions {
		sub := &subscriptions[i]

		f, ok := sub.File.(*osFile)
		if !ok {
			return nil, os.ErrInvalid
		}

		var event syscall.EpollEvent
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
			event.Events = syscall.EPOLLIN | syscall.EPOLLRDHUP
		case SubscriptionWrite:
			event.Events = syscall.EPOLLOUT
		default:
			return nil, os.ErrInvalid
		}
		event.Events |= syscall.EPOLLERR | syscall.EPOLLHUP | syscall.EPOLLONESHOT

		event.Fd = int32(i)
		if err = syscall.EpollCtl(epoll, syscall.EPOLL_CTL_ADD, int(f.f.Fd()), &event); err != nil {
			return nil, err
		}
	}

	timeoutMilliseconds := -1
	if timeoutIndex != -1 {
		timeoutMilliseconds = int(timeout.Milliseconds())
	}

	epollEvents := make([]syscall.EpollEvent, len(subscriptions))
	n, err := syscall.EpollWait(epoll, epollEvents, timeoutMilliseconds)
	if err != nil {
		return nil, err
	}

	if n == 0 && timeoutIndex != -1 {
		sub := &subscriptions[timeoutIndex]
		return []Event{{
			Kind:     SubscriptionTimer,
			Userdata: sub.Userdata,
		}}, nil
	}

	events := make([]Event, n)
	for i, epollEvent := range epollEvents[:n] {
		event := &events[i]

		sub := &subscriptions[epollEvent.Fd]
		switch sub.Kind {
		case SubscriptionTimer:
			return nil, os.ErrInvalid
		case SubscriptionRead:
			event.Kind = SubscriptionRead
			event.Available = 1
			if epollEvent.Events&syscall.EPOLLRDHUP != 0 {
				event.Flags = EventHangup
			}
		case SubscriptionWrite:
			event.Kind = SubscriptionWrite
		}

		if epollEvent.Events&syscall.EPOLLERR != 0 {
			event.Error = ErrnoIO
		}
		if epollEvent.Events&syscall.EPOLLHUP != 0 {
			event.Flags = EventHangup
		}

		event.Userdata = sub.Userdata
	}

	return events, nil
}
