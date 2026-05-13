package bad

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// every line below is engineered to trigger exactly one rule.

var globalCounter int // expect: noglobals

type Color int // enum

const (
	Red Color = iota
	Green
	Blue
)

type Shape interface {
	area() float64 // unexported method -> sealed
}

type Circle struct{}

func (c Circle) area() float64 { return 0 }

type Square struct{}

func (s Square) area() float64 { return 0 }

func mayFail() error { return errors.New("nope") }

func split() (int, error) { return 0, errors.New("nope") }

func ignoresError() {
	mayFail() // expect: errcheck (dropped error)
}

func ignoresErrorWithBlank() {
	_ = mayFail() // expect: errcheck (blank without annotation)
}

func ignoresErrorWithAnnotation() {
	_ = mayFail() // safe-ignore: this fire-and-forget is deliberate
}

func ignoresErrInTuple() {
	x, _ := split() // expect: errcheck on the blank err
	_ = x
}

func shadowExample() {
	err := mayFail()
	if err != nil {
		err := mayFail() // expect: shadow
		_ = err          // safe-ignore: just to silence errcheck here
	}
	_ = err
}

func badAssert(v any) string { // expect: banany on the parameter
	return v.(string) // expect: forcetypeassert
}

func goodAssert(v any) string { // any-ok: this is a deliberate boundary func
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func swapBug(userID, orderID string) {} // declared with two strings

func callsSwap() {
	swapBug("u-1", "o-2") // expect: namedargs (two strings adjacent, no comments)
}

func callsSwapOK() {
	swapBug( /* userID */ "u-1" /* orderID */, "o-2")
}

func nonExhaustive(c Color) string {
	switch c {
	case Red:
		return "red"
	case Green:
		return "green"
	} // expect: exhaustive (missing Blue)
	return ""
}

func nonExhaustiveType(s Shape) string {
	switch s.(type) {
	case Circle:
		return "circle"
	} // expect: exhaustive (missing Square)
	return ""
}

func leakedBody() {
	resp, _ := http.Get("http://x") // safe-ignore: testing the body-close path
	_ = resp                        // expect: bodyclose
}

func notLeaked() {
	resp, _ := http.Get("http://x") // safe-ignore: testing the body-close path
	defer resp.Body.Close()         // safe-ignore: cleanup
}

func wrongContext(ctx context.Context) {
	fmt.Println(ctx)
	_ = context.Background() // expect: contextcheck
}

func spawnsBareGoroutine() {
	go fmt.Println("hi") // expect: goroutine (no lifecycle primitive)
}

func spawnsOKGoroutine() {
	go fmt.Println("hi") // goroutine-ok: deliberate fire-and-forget
}
