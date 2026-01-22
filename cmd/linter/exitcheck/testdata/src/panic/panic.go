// Package panic содержит тестовые кейсы для проверки детекции panic.
package panic

func testPanic() {
	panic("this is a panic") // want "usage of builtin panic is discouraged"
}

func testPanicWithRecover() {
	defer func() {
		recover()
	}()
	panic("recovered")
}

func CustomPanic(msg string) {
}

func testCustomPanic() {
	msg := "test"
	_ = msg
}
