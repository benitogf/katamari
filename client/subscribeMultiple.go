package client

import "context"

// SubscribeMultiple2 subscribes to 2 paths with different types and a single callback.
// When any subscription updates, the callback receives ALL current states.
// Uses typed channels for type-safe, lock-free state management.
func SubscribeMultiple2[T1, T2 any](
	ctx context.Context,
	path1 Path,
	path2 Path,
	callback func([]Meta[T1], []Meta[T2]),
) {
	ch1 := make(chan []Meta[T1], 10)
	ch2 := make(chan []Meta[T2], 10)

	// State manager goroutine - single point of state mutation
	go func() {
		var state1 []Meta[T1]
		var state2 []Meta[T2]

		for {
			select {
			case <-ctx.Done():
				return
			case state1 = <-ch1:
				callback(state1, state2)
			case state2 = <-ch2:
				callback(state1, state2)
			}
		}
	}()

	go Subscribe(ctx, path1.Protocol, path1.Host, path1.Path, func(messages []Meta[T1]) {
		select {
		case ch1 <- messages:
		case <-ctx.Done():
		}
	})

	go Subscribe(ctx, path2.Protocol, path2.Host, path2.Path, func(messages []Meta[T2]) {
		select {
		case ch2 <- messages:
		case <-ctx.Done():
		}
	})
}

// SubscribeMultiple3 subscribes to 3 paths with different types and a single callback.
// When any subscription updates, the callback receives ALL current states.
// Uses typed channels for type-safe, lock-free state management.
func SubscribeMultiple3[T1, T2, T3 any](
	ctx context.Context,
	path1 Path,
	path2 Path,
	path3 Path,
	callback func([]Meta[T1], []Meta[T2], []Meta[T3]),
) {
	ch1 := make(chan []Meta[T1], 10)
	ch2 := make(chan []Meta[T2], 10)
	ch3 := make(chan []Meta[T3], 10)

	// State manager goroutine - single point of state mutation
	go func() {
		var state1 []Meta[T1]
		var state2 []Meta[T2]
		var state3 []Meta[T3]

		for {
			select {
			case <-ctx.Done():
				return
			case state1 = <-ch1:
				callback(state1, state2, state3)
			case state2 = <-ch2:
				callback(state1, state2, state3)
			case state3 = <-ch3:
				callback(state1, state2, state3)
			}
		}
	}()

	go Subscribe(ctx, path1.Protocol, path1.Host, path1.Path, func(messages []Meta[T1]) {
		select {
		case ch1 <- messages:
		case <-ctx.Done():
		}
	})

	go Subscribe(ctx, path2.Protocol, path2.Host, path2.Path, func(messages []Meta[T2]) {
		select {
		case ch2 <- messages:
		case <-ctx.Done():
		}
	})

	go Subscribe(ctx, path3.Protocol, path3.Host, path3.Path, func(messages []Meta[T3]) {
		select {
		case ch3 <- messages:
		case <-ctx.Done():
		}
	})
}

// SubscribeMultiple4 subscribes to 4 paths with different types and a single callback.
// When any subscription updates, the callback receives ALL current states.
// Uses typed channels for type-safe, lock-free state management.
func SubscribeMultiple4[T1, T2, T3, T4 any](
	ctx context.Context,
	path1 Path,
	path2 Path,
	path3 Path,
	path4 Path,
	callback func([]Meta[T1], []Meta[T2], []Meta[T3], []Meta[T4]),
) {
	ch1 := make(chan []Meta[T1], 10)
	ch2 := make(chan []Meta[T2], 10)
	ch3 := make(chan []Meta[T3], 10)
	ch4 := make(chan []Meta[T4], 10)

	// State manager goroutine - single point of state mutation
	go func() {
		var state1 []Meta[T1]
		var state2 []Meta[T2]
		var state3 []Meta[T3]
		var state4 []Meta[T4]

		for {
			select {
			case <-ctx.Done():
				return
			case state1 = <-ch1:
				callback(state1, state2, state3, state4)
			case state2 = <-ch2:
				callback(state1, state2, state3, state4)
			case state3 = <-ch3:
				callback(state1, state2, state3, state4)
			case state4 = <-ch4:
				callback(state1, state2, state3, state4)
			}
		}
	}()

	go Subscribe(ctx, path1.Protocol, path1.Host, path1.Path, func(messages []Meta[T1]) {
		select {
		case ch1 <- messages:
		case <-ctx.Done():
		}
	})

	go Subscribe(ctx, path2.Protocol, path2.Host, path2.Path, func(messages []Meta[T2]) {
		select {
		case ch2 <- messages:
		case <-ctx.Done():
		}
	})

	go Subscribe(ctx, path3.Protocol, path3.Host, path3.Path, func(messages []Meta[T3]) {
		select {
		case ch3 <- messages:
		case <-ctx.Done():
		}
	})

	go Subscribe(ctx, path4.Protocol, path4.Host, path4.Path, func(messages []Meta[T4]) {
		select {
		case ch4 <- messages:
		case <-ctx.Done():
		}
	})
}
